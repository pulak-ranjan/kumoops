package core

// Anomaly Detection + Self-Healing Engine
//
// Monitors key delivery metrics every 5 minutes and detects statistical
// anomalies compared to rolling baselines. When anomalies are detected:
//   - "warning" severity: alert only
//   - "critical" severity: alert + auto-pause affected sender + reduce rate
//
// Self-healing: Every 5 minutes, checks if active anomalies have resolved.
// If metrics return to normal for 15+ minutes, auto-resumes paused senders.

import (
	"fmt"
	"log"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// AnomalyDetector monitors delivery health and self-heals when problems resolve.
type AnomalyDetector struct {
	st *store.Store
	ws *WebhookService
}

// NewAnomalyDetector creates an AnomalyDetector.
func NewAnomalyDetector(st *store.Store, ws *WebhookService) *AnomalyDetector {
	return &AnomalyDetector{st: st, ws: ws}
}

// thresholds defines detection thresholds per anomaly type.
var anomalyThresholds = map[string]struct{ warning, critical float64 }{
	"bounce_rate":      {5.0, 10.0},   // % of total sends
	"complaint_rate":   {0.05, 0.10},  // % of total sends (ISP threshold × 0.5 / 1.0)
	"deferral_rate":    {20.0, 40.0},  // % of total attempts
	"acceptance_drop":  {5.0, 15.0},   // % drop from baseline
}

// Run executes one detection + healing cycle.
// Should be called every 5 minutes by the scheduler.
func (ad *AnomalyDetector) Run() {
	ad.detectAnomalies()
	ad.healResolvedAnomalies()
}

// detectAnomalies scans all senders and ISPs for current anomalies.
func (ad *AnomalyDetector) detectAnomalies() {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	// Get per-ISP metrics for the last hour
	isps := []string{"Gmail", "Yahoo", "Outlook", "AOL"}
	for _, isp := range isps {
		recent, err := ad.st.GetISPDeliveryMetrics(isp, oneHourAgo, now)
		if err != nil || recent == nil || recent.Total < 5 {
			continue
		}

		baseline, err := ad.st.GetISPDeliveryMetrics(isp, oneDayAgo, now)
		if err != nil || baseline == nil || baseline.Total < 20 {
			continue
		}

		// Bounce rate spike
		if recent.Total > 0 {
			bounceRate := float64(recent.Bounced) / float64(recent.Total) * 100
			baselineBounce := float64(baseline.Bounced) / float64(baseline.Total) * 100
			thresholds := anomalyThresholds["bounce_rate"]

			if bounceRate > thresholds.critical && bounceRate > baselineBounce*2 {
				ad.recordAnomaly("bounce_spike", "critical", isp, "", "",
					bounceRate, thresholds.critical, baselineBounce,
					"Bounce rate exceeds critical threshold")
			} else if bounceRate > thresholds.warning && bounceRate > baselineBounce*1.5 {
				ad.recordAnomaly("bounce_spike", "warning", isp, "", "",
					bounceRate, thresholds.warning, baselineBounce,
					"Bounce rate above warning threshold")
			}
		}

		// Deferral rate spike
		if recent.Total > 0 {
			deferralRate := float64(recent.Deferred) / float64(recent.Total) * 100
			baselineDeferral := float64(baseline.Deferred) / float64(baseline.Total) * 100
			thresholds := anomalyThresholds["deferral_rate"]

			if deferralRate > thresholds.critical && deferralRate > baselineDeferral*2 {
				ad.recordAnomaly("deferral_spike", "critical", isp, "", "",
					deferralRate, thresholds.critical, baselineDeferral,
					"ISP deferral rate critically high — possible throttling or reputation issue")
			} else if deferralRate > thresholds.warning && deferralRate > baselineDeferral*1.5 {
				ad.recordAnomaly("deferral_spike", "warning", isp, "", "",
					deferralRate, thresholds.warning, baselineDeferral,
					"ISP deferral rate elevated above baseline")
			}
		}

		// Acceptance rate drop
		if recent.Delivered+recent.Bounced > 0 && baseline.Delivered+baseline.Bounced > 0 {
			recentAccept := float64(recent.Delivered) / float64(recent.Delivered+recent.Bounced) * 100
			baselineAccept := float64(baseline.Delivered) / float64(baseline.Delivered+baseline.Bounced) * 100
			drop := baselineAccept - recentAccept
			thresholds := anomalyThresholds["acceptance_drop"]

			if drop > thresholds.critical {
				ad.recordAnomaly("acceptance_drop", "critical", isp, "", "",
					drop, thresholds.critical, baselineAccept,
					fmt.Sprintf("Acceptance rate dropped %.1f%% from baseline %.1f%%", drop, baselineAccept))
			} else if drop > thresholds.warning {
				ad.recordAnomaly("acceptance_drop", "warning", isp, "", "",
					drop, thresholds.warning, baselineAccept,
					fmt.Sprintf("Acceptance rate dropped %.1f%% from baseline", drop))
			}
		}
	}

	// Per-domain complaint rate check
	domains, _ := ad.st.ListDomains()
	for _, domain := range domains {
		oneHourSince := now.Add(-1 * time.Hour)
		complaintsNow, _ := ad.st.CountFBLRecordsSince(domain.Name, oneHourSince)
		if complaintsNow == 0 {
			continue
		}

		// Get approximate sent count from EmailStats
		stats, err := ad.st.GetDomainStats24h(domain.Name)
		if err != nil || stats == nil || stats.Sent == 0 {
			continue
		}
		// Scale hourly sent estimate
		hourlySent := stats.Sent / 24
		if hourlySent == 0 {
			hourlySent = 1
		}
		complaintRate := float64(complaintsNow) / float64(hourlySent) * 100
		thresholds := anomalyThresholds["complaint_rate"]

		if complaintRate > thresholds.critical {
			ad.recordAnomaly("complaint_spike", "critical", "", "", domain.Name,
				complaintRate, thresholds.critical, 0,
				fmt.Sprintf("%.2f%% complaint rate in last hour — approaching ISP block threshold", complaintRate))
		} else if complaintRate > thresholds.warning {
			ad.recordAnomaly("complaint_spike", "warning", "", "", domain.Name,
				complaintRate, thresholds.warning, 0,
				"Complaint rate elevated above warning threshold")
		}
	}
}

// recordAnomaly stores an anomaly event and takes appropriate self-healing action.
// Deduplicates: does not record a new event if a same-type+isp+domain event is already active.
func (ad *AnomalyDetector) recordAnomaly(
	anomalyType, severity, isp, senderEmail, domain string,
	value, threshold, baseline float64,
	notes string,
) {
	// Check for existing active anomaly of same type
	existing, _ := ad.st.GetActiveAnomaly(anomalyType, isp, domain)
	if existing != nil {
		return // already tracking this anomaly
	}

	action := "none"

	// Critical anomalies trigger automatic throttle reduction
	if severity == "critical" {
		if isp != "" {
			// Reduce rate for this ISP's shaping rule
			rules, err := ad.st.GetShapingRuleByProvider(isp)
			if err == nil && rules != nil {
				rate, unit, err := parseRate(rules.MaxMessageRate)
				if err == nil {
					rules.MaxMessageRate = formatRate(int(float64(rate)*0.50), unit)
					rules.ConnectionLimit = max(rules.ConnectionLimit-2, 1)
					ad.st.UpdateShapingRule(rules)
					action = fmt.Sprintf("rate_reduced_to_%s", rules.MaxMessageRate)
				}
			}
		}
	}

	event := &models.AnomalyEvent{
		DetectedAt:  time.Now(),
		Type:        anomalyType,
		Severity:    severity,
		ISP:         isp,
		SenderEmail: senderEmail,
		Domain:      domain,
		MetricValue: value,
		Threshold:   threshold,
		BaselineVal: baseline,
		ActionTaken: action,
		Notes:       notes,
	}

	if err := ad.st.CreateAnomalyEvent(event); err != nil {
		log.Printf("[ANOMALY] store event: %v", err)
		return
	}

	log.Printf("[ANOMALY] %s detected: type=%s severity=%s isp=%s domain=%s value=%.2f action=%s",
		severity, anomalyType, severity, isp, domain, value, action)

	// Send alert via webhook/Telegram
	msg := fmt.Sprintf("⚠️ *Anomaly Detected*\nType: `%s`\nSeverity: `%s`\nISP: %s\nDomain: %s\nValue: %.2f%%\nThreshold: %.2f%%\nAction: %s\n%s",
		anomalyType, severity, isp, domain, value, threshold, action, notes)
	ad.ws.SendAlert(msg)
}

// healResolvedAnomalies checks active anomalies and resolves those whose metrics
// have returned to normal for at least 15 minutes.
func (ad *AnomalyDetector) healResolvedAnomalies() {
	active, err := ad.st.ListActiveAnomalyEvents()
	if err != nil {
		return
	}

	now := time.Now()
	fifteenMinAgo := now.Add(-15 * time.Minute)

	for _, event := range active {
		// Only consider healing anomalies that are at least 15 minutes old
		if event.DetectedAt.After(fifteenMinAgo) {
			continue
		}

		healed := ad.checkIfHealed(&event)
		if !healed {
			continue
		}

		// Mark as resolved
		resolvedAt := now
		event.ResolvedAt = &resolvedAt
		event.AutoHealed = true
		if err := ad.st.ResolveAnomalyEvent(&event); err != nil {
			log.Printf("[ANOMALY] resolve event %d: %v", event.ID, err)
			continue
		}

		log.Printf("[ANOMALY] Auto-healed: type=%s isp=%s domain=%s", event.Type, event.ISP, event.Domain)
		msg := fmt.Sprintf("✅ *Anomaly Resolved*\nType: `%s`\nISP: %s\nDomain: %s\nAuto-healed after %.0f minutes",
			event.Type, event.ISP, event.Domain, now.Sub(event.DetectedAt).Minutes())
		ad.ws.SendAlert(msg)
	}
}

// checkIfHealed returns true if the metrics for an anomaly event have normalized.
func (ad *AnomalyDetector) checkIfHealed(event *models.AnomalyEvent) bool {
	now := time.Now()
	recentWindow := now.Add(-15 * time.Minute)

	switch event.Type {
	case "bounce_spike", "deferral_spike", "acceptance_drop":
		if event.ISP == "" {
			return false
		}
		m, err := ad.st.GetISPDeliveryMetrics(event.ISP, recentWindow, now)
		if err != nil || m == nil || m.Total < 3 {
			return false
		}
		bounceRate := float64(m.Bounced) / float64(m.Total) * 100
		deferralRate := float64(m.Deferred) / float64(m.Total) * 100
		// Healed if both rates are below warning thresholds
		return bounceRate < anomalyThresholds["bounce_rate"].warning &&
			deferralRate < anomalyThresholds["deferral_rate"].warning

	case "complaint_spike":
		since := now.Add(-15 * time.Minute)
		count, err := ad.st.CountFBLRecordsSince(event.Domain, since)
		if err != nil {
			return false
		}
		return count == 0 // healed if no new complaints in last 15 min

	default:
		// Conservative: don't auto-heal unknown types
		return false
	}
}
