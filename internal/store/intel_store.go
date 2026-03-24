package store

import (
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
)

// ─────────────────────────────────────────────
// ISP Snapshots
// ─────────────────────────────────────────────

// UpsertISPSnapshot inserts or replaces an ISP snapshot (keyed by ISP+Domain+hour).
func (s *Store) UpsertISPSnapshot(snap *models.ISPSnapshot) error {
	// Delete existing snapshot for the same ISP+Domain+hour to avoid duplicates
	hour := snap.CapturedAt.Truncate(time.Hour)
	s.DB.Where("isp = ? AND domain = ? AND captured_at = ?", snap.ISP, snap.Domain, hour).
		Delete(&models.ISPSnapshot{})
	snap.CapturedAt = hour
	return s.DB.Create(snap).Error
}

// ListISPSnapshots returns ISP snapshots filtered by optional domain/ISP, newest first.
func (s *Store) ListISPSnapshots(domain, isp string, since time.Time, limit int) ([]models.ISPSnapshot, error) {
	q := s.DB.Model(&models.ISPSnapshot{}).Order("captured_at DESC")
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	if isp != "" {
		q = q.Where("isp = ?", isp)
	}
	if !since.IsZero() {
		q = q.Where("captured_at >= ?", since)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	snaps := make([]models.ISPSnapshot, 0)
	return snaps, q.Find(&snaps).Error
}

// GetLatestISPSnapshots returns the most recent snapshot per ISP for a domain.
func (s *Store) GetLatestISPSnapshots(domain string) ([]models.ISPSnapshot, error) {
	snaps := make([]models.ISPSnapshot, 0)
	err := s.DB.Raw(`
		SELECT * FROM isp_snapshots s
		WHERE domain = ? AND captured_at = (
			SELECT MAX(captured_at) FROM isp_snapshots
			WHERE isp = s.isp AND domain = s.domain
		)
		ORDER BY isp
	`, domain).Scan(&snaps).Error
	return snaps, err
}

// ISPDeliveryMetrics holds aggregated delivery counts for one ISP over a time window.
type ISPDeliveryMetrics struct {
	ISP       string
	Total     int64
	Delivered int64
	Bounced   int64
	Deferred  int64
	Sent      int64
}

// GetISPDeliveryMetrics aggregates DeliveryEvent data for one ISP over a time window.
// "Delivered" is estimated as Sent - Bounced - Deferred (since we only log failures).
func (s *Store) GetISPDeliveryMetrics(isp string, from, to time.Time) (*ISPDeliveryMetrics, error) {
	type row struct {
		EventType string
		Count     int64
	}
	var rows []row
	err := s.DB.Raw(`
		SELECT event_type, COUNT(*) as count
		FROM delivery_events
		WHERE provider = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY event_type
	`, isp, from, to).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	m := &ISPDeliveryMetrics{ISP: isp}
	for _, r := range rows {
		switch r.EventType {
		case "Bounce":
			m.Bounced = r.Count
		case "TransientFailure":
			m.Deferred = r.Count
		}
	}
	m.Total = m.Bounced + m.Deferred
	return m, nil
}

// GetISPMetricsForDomain aggregates local delivery metrics per ISP from DeliveryEvent for a domain.
func (s *Store) GetISPMetricsForDomain(domain string, from, to time.Time) (map[string]*ISPDeliveryMetrics, error) {
	type row struct {
		Provider  string
		EventType string
		Count     int64
	}
	var rows []row
	err := s.DB.Raw(`
		SELECT provider, event_type, COUNT(*) as count
		FROM delivery_events
		WHERE domain = ? AND timestamp >= ? AND timestamp <= ?
		GROUP BY provider, event_type
	`, domain, from, to).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := map[string]*ISPDeliveryMetrics{}
	for _, r := range rows {
		if _, ok := result[r.Provider]; !ok {
			result[r.Provider] = &ISPDeliveryMetrics{ISP: r.Provider}
		}
		m := result[r.Provider]
		switch r.EventType {
		case "Bounce":
			m.Bounced += r.Count
		case "TransientFailure":
			m.Deferred += r.Count
		}
		m.Total += r.Count
	}
	return result, nil
}

// GetFBLCountsByISP returns complaint counts grouped by ISP (inferred from recipient domain) for a domain/time window.
func (s *Store) GetFBLCountsByISP(domain string, since time.Time) (map[string]int64, error) {
	type row struct {
		ReportingMTA string
		Count        int64
	}
	var rows []row
	err := s.DB.Raw(`
		SELECT reporting_mta, COUNT(*) as count
		FROM fbl_records
		WHERE domain = ? AND received_at >= ?
		GROUP BY reporting_mta
	`, domain, since).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := map[string]int64{}
	for _, r := range rows {
		isp := mapMTAtoISP(r.ReportingMTA)
		result[isp] += r.Count
	}
	return result, nil
}

// GetDomainStats24h returns EmailStats sum for a domain over the last 24 hours.
func (s *Store) GetDomainStats24h(domain string) (*models.EmailStats, error) {
	var stat models.EmailStats
	err := s.DB.Raw(`
		SELECT COALESCE(SUM(sent),0) as sent, COALESCE(SUM(delivered),0) as delivered,
		       COALESCE(SUM(bounced),0) as bounced, COALESCE(SUM(deferred),0) as deferred
		FROM email_stats
		WHERE domain = ? AND date >= ?
	`, domain, time.Now().Add(-24*time.Hour)).Scan(&stat).Error
	return &stat, err
}

// ListSendersByDomainName returns Sender records for a given domain name string.
// Used by ISP Intel service; distinct from ListSendersByDomain(domainID uint) in db.go.
func (s *Store) ListSendersByDomainName(domainName string) ([]models.Sender, error) {
	senders := make([]models.Sender, 0)
	err := s.DB.
		Joins("JOIN domains ON domains.id = senders.domain_id").
		Where("domains.name = ?", domainName).
		Find(&senders).Error
	return senders, err
}

// mapMTAtoISP maps a reporting MTA domain to a well-known ISP name.
func mapMTAtoISP(mta string) string {
	switch {
	case contains(mta, "google", "gmail"):
		return "Gmail"
	case contains(mta, "yahoo", "ymail"):
		return "Yahoo"
	case contains(mta, "hotmail", "outlook", "microsoft", "live.com", "msn"):
		return "Outlook"
	case contains(mta, "aol", "aim.com"):
		return "AOL"
	default:
		return "Other"
	}
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// ─────────────────────────────────────────────
// Throttle Adjustment Log
// ─────────────────────────────────────────────

func (s *Store) CreateThrottleAdjustmentLog(log *models.ThrottleAdjustmentLog) error {
	return s.DB.Create(log).Error
}

func (s *Store) ListThrottleAdjustmentLogs(isp string, since time.Time, limit int) ([]models.ThrottleAdjustmentLog, error) {
	q := s.DB.Model(&models.ThrottleAdjustmentLog{}).Order("created_at DESC")
	if isp != "" {
		q = q.Where("isp = ?", isp)
	}
	if !since.IsZero() {
		q = q.Where("created_at >= ?", since)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	logs := make([]models.ThrottleAdjustmentLog, 0)
	return logs, q.Find(&logs).Error
}

// ─────────────────────────────────────────────
// Traffic Shaping Rule helpers (for adaptive throttle)
// ─────────────────────────────────────────────

func (s *Store) ListEnabledShapingRules() ([]models.TrafficShapingRule, error) {
	rules := make([]models.TrafficShapingRule, 0)
	return rules, s.DB.Where("is_enabled = ?", true).Find(&rules).Error
}

func (s *Store) GetShapingRuleByProvider(provider string) (*models.TrafficShapingRule, error) {
	var rule models.TrafficShapingRule
	err := s.DB.Where("provider = ? AND is_enabled = ?", provider, true).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// ─────────────────────────────────────────────
// Anomaly Events
// ─────────────────────────────────────────────

func (s *Store) CreateAnomalyEvent(e *models.AnomalyEvent) error {
	return s.DB.Create(e).Error
}

// GetActiveAnomaly returns an unresolved anomaly of the given type+isp+domain, if any.
func (s *Store) GetActiveAnomaly(anomalyType, isp, domain string) (*models.AnomalyEvent, error) {
	var e models.AnomalyEvent
	err := s.DB.Where("type = ? AND isp = ? AND domain = ? AND resolved_at IS NULL", anomalyType, isp, domain).
		Order("detected_at DESC").First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) ListActiveAnomalyEvents() ([]models.AnomalyEvent, error) {
	events := make([]models.AnomalyEvent, 0)
	return events, s.DB.Where("resolved_at IS NULL").Order("detected_at DESC").Find(&events).Error
}

func (s *Store) ListAnomalyEvents(since time.Time, limit int) ([]models.AnomalyEvent, error) {
	q := s.DB.Model(&models.AnomalyEvent{}).Order("detected_at DESC")
	if !since.IsZero() {
		q = q.Where("detected_at >= ?", since)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	events := make([]models.AnomalyEvent, 0)
	return events, q.Find(&events).Error
}

func (s *Store) ResolveAnomalyEvent(e *models.AnomalyEvent) error {
	return s.DB.Save(e).Error
}
