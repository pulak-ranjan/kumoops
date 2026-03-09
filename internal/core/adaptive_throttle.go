package core

// Adaptive Throttling Engine
//
// Learns each ISP's delivery behavior from recent history and automatically
// adjusts TrafficShapingRules to optimize throughput while protecting reputation.
//
// Algorithm (runs every 5 minutes):
//   For each ISP traffic shaping rule:
//     1. Collect last-1h delivery metrics for that ISP
//     2. Compute deferral_rate and acceptance_rate
//     3. Compare against 24h rolling baseline
//     4. Throttle DOWN if: deferral_rate > baseline × 1.5 OR acceptance_rate drops > 10%
//     5. Throttle UP if:   deferral_rate < baseline × 0.5 AND acceptance_rate stable > 95%
//     6. Apply adjustment, clamp to configured min/max bounds
//     7. Log every change to ThrottleAdjustmentLog

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// AdaptiveThrottler is the main adaptive throttling engine.
type AdaptiveThrottler struct {
	st *store.Store
}

// NewAdaptiveThrottler creates a new AdaptiveThrottler.
func NewAdaptiveThrottler(st *store.Store) *AdaptiveThrottler {
	return &AdaptiveThrottler{st: st}
}

// Run executes one cycle of the adaptive throttling algorithm.
// Called by the scheduler every 5 minutes.
func (at *AdaptiveThrottler) Run() {
	rules, err := at.st.ListEnabledShapingRules()
	if err != nil {
		log.Println("[THROTTLE] list rules:", err)
		return
	}

	for _, rule := range rules {
		at.adjustRule(&rule)
	}
}

// adjustRule evaluates and optionally adjusts one traffic shaping rule.
func (at *AdaptiveThrottler) adjustRule(rule *models.TrafficShapingRule) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	isp := rule.Provider

	// Collect recent metrics
	recent, err := at.st.GetISPDeliveryMetrics(isp, oneHourAgo, now)
	if err != nil || recent == nil || recent.Total == 0 {
		return // no data to act on
	}

	// Collect 24h baseline
	baseline, err := at.st.GetISPDeliveryMetrics(isp, oneDayAgo, now)
	if err != nil || baseline == nil || baseline.Total < 10 {
		return // need enough data for baseline
	}

	currentDeferral := float64(recent.Deferred) / float64(recent.Total)
	currentAcceptance := float64(recent.Delivered) / float64(max64(recent.Delivered+recent.Bounced, 1))
	baselineDeferral := float64(baseline.Deferred) / float64(baseline.Total)
	baselineAcceptance := float64(baseline.Delivered) / float64(max64(baseline.Delivered+baseline.Bounced, 1))

	// Parse current throttle settings
	currentRate, unit, err := parseRate(rule.MaxMessageRate)
	if err != nil || currentRate <= 0 {
		return
	}
	currentConns := rule.ConnectionLimit
	if currentConns <= 0 {
		currentConns = 5 // default
	}

	direction := "none"
	reason := ""
	newRate := currentRate
	newConns := currentConns

	// Decision logic
	deferralSpike := baselineDeferral > 0.01 && currentDeferral > baselineDeferral*1.5
	acceptanceDrop := baselineAcceptance > 0.9 && currentAcceptance < baselineAcceptance-0.10
	deferralLow := currentDeferral < baselineDeferral*0.5
	acceptanceGood := currentAcceptance >= 0.95

	switch {
	case deferralSpike || acceptanceDrop:
		// Throttle DOWN
		direction = "down"
		if deferralSpike {
			reason = fmt.Sprintf("deferral rate %.1f%% exceeds baseline %.1f%% by >50%%",
				currentDeferral*100, baselineDeferral*100)
		} else {
			reason = fmt.Sprintf("acceptance rate dropped %.1f%% from baseline %.1f%%",
				currentAcceptance*100, baselineAcceptance*100)
		}
		// Reduce rate by 20%, reduce connections by 1
		newRate = int(math.Round(float64(currentRate) * 0.80))
		newConns = max(currentConns-1, 1)

	case deferralLow && acceptanceGood:
		// Throttle UP — gradually, only 10% at a time
		direction = "up"
		reason = fmt.Sprintf("deferral %.1f%% well below baseline %.1f%%, acceptance %.1f%%",
			currentDeferral*100, baselineDeferral*100, currentAcceptance*100)
		newRate = int(math.Round(float64(currentRate) * 1.10))
		newConns = min(currentConns+1, 20)
	}

	if direction == "none" {
		return // no adjustment needed
	}

	// Apply min/max bounds
	newRate = clampRate(newRate, unit)

	// Skip if change is trivial (< 5% difference)
	if math.Abs(float64(newRate-currentRate))/float64(currentRate) < 0.05 && newConns == currentConns {
		return
	}

	oldRateStr := rule.MaxMessageRate
	newRateStr := formatRate(newRate, unit)

	// Apply to database
	rule.MaxMessageRate = newRateStr
	rule.ConnectionLimit = newConns
	if err := at.st.UpdateShapingRule(rule); err != nil {
		log.Printf("[THROTTLE] update rule %d: %v", rule.ID, err)
		return
	}

	// Log the adjustment
	adj := &models.ThrottleAdjustmentLog{
		CreatedAt:    now,
		ISP:          isp,
		RuleID:       rule.ID,
		RuleName:     rule.Provider,
		OldRate:      oldRateStr,
		OldConns:     currentConns,
		NewRate:      newRateStr,
		NewConns:     newConns,
		Direction:    direction,
		Reason:       reason,
		DeferralRate: currentDeferral,
		AcceptRate:   currentAcceptance,
	}
	if err := at.st.CreateThrottleAdjustmentLog(adj); err != nil {
		log.Printf("[THROTTLE] log adjustment: %v", err)
	}

	log.Printf("[THROTTLE] %s %s: rate %s→%s conns %d→%d | %s",
		direction, isp, oldRateStr, newRateStr, currentConns, newConns, reason)
}

// ─────────────────────────────────────────────
// Rate string parsing / formatting
// ─────────────────────────────────────────────

// parseRate parses a KumoMTA rate string like "100/h", "50/min", "10/s"
// into (value, unit, error).
func parseRate(s string) (int, string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "", fmt.Errorf("empty rate")
	}
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid rate format: %q", s)
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, "", fmt.Errorf("invalid rate value: %w", err)
	}
	return n, "/" + strings.TrimSpace(parts[1]), nil
}

// formatRate formats a rate value and unit back to a string.
func formatRate(value int, unit string) string {
	return fmt.Sprintf("%d%s", value, unit)
}

// clampRate ensures a rate value stays within reasonable bounds for its unit.
func clampRate(value int, unit string) int {
	minVal, maxVal := 1, 10000
	switch strings.TrimPrefix(unit, "/") {
	case "s":
		minVal, maxVal = 1, 100
	case "min":
		minVal, maxVal = 5, 1000
	case "h":
		minVal, maxVal = 10, 50000
	case "d":
		minVal, maxVal = 100, 500000
	}
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// ─────────────────────────────────────────────
// Utility
// ─────────────────────────────────────────────

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
