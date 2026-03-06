package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// AlertChecker runs in the background and evaluates alert rules every 5 minutes.
type AlertChecker struct {
	st *store.Store
}

func NewAlertChecker(st *store.Store) *AlertChecker {
	return &AlertChecker{st: st}
}

// Start launches the background monitoring goroutine.
func (ac *AlertChecker) Start() {
	go func() {
		// Initial delay before first check
		time.Sleep(2 * time.Minute)
		for {
			ac.checkAll()
			time.Sleep(5 * time.Minute)
		}
	}()
	log.Println("[ALERTING] Background alert checker started")
}

func (ac *AlertChecker) checkAll() {
	rules, err := ac.st.ListAlertRules()
	if err != nil {
		log.Println("[ALERTING] error loading rules:", err)
		return
	}
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}
		ac.evaluateRule(rule)
	}
}

func (ac *AlertChecker) evaluateRule(rule models.AlertRule) {
	// Check cooldown
	if rule.LastFired != nil {
		cooldown := time.Duration(rule.CooldownMin) * time.Minute
		if time.Since(*rule.LastFired) < cooldown {
			return
		}
	}

	value, err := ac.getMetricValue(rule.Type)
	if err != nil {
		log.Printf("[ALERTING] error getting metric %s: %v", rule.Type, err)
		return
	}

	triggered := false
	switch rule.Operator {
	case "gt":
		triggered = value > rule.Threshold
	case "lt":
		triggered = value < rule.Threshold
	default:
		triggered = value > rule.Threshold // default to gt
	}

	if !triggered {
		return
	}

	log.Printf("[ALERTING] Rule '%s' triggered: value=%.2f threshold=%.2f", rule.Name, value, rule.Threshold)
	if err := ac.fireAlert(rule, value); err != nil {
		log.Printf("[ALERTING] fire error for rule '%s': %v", rule.Name, err)
	}
}

func (ac *AlertChecker) getMetricValue(ruleType string) (float64, error) {
	switch ruleType {
	case "bounce_rate":
		stats, err := ac.st.GetTodayStats()
		if err != nil {
			return 0, err
		}
		var totalSent, totalBounced int64
		for _, s := range stats {
			totalSent += s.Sent
			totalBounced += s.Bounced
		}
		if totalSent == 0 {
			return 0, nil
		}
		return float64(totalBounced) / float64(totalSent) * 100, nil

	case "delivery_rate":
		stats, err := ac.st.GetTodayStats()
		if err != nil {
			return 0, err
		}
		var totalSent, totalDelivered int64
		for _, s := range stats {
			totalSent += s.Sent
			totalDelivered += s.Delivered
		}
		if totalSent == 0 {
			return 100, nil // no mail sent = 100% delivery rate (no failures)
		}
		return float64(totalDelivered) / float64(totalSent) * 100, nil

	case "queue_depth":
		stats, err := GetQueueStats()
		if err != nil {
			return 0, err
		}
		return float64(stats.Total), nil

	case "blacklist":
		return 0, nil // blacklist alerts fired externally via webhook
	}
	return 0, fmt.Errorf("unknown metric type: %s", ruleType)
}

func (ac *AlertChecker) fireAlert(rule models.AlertRule, value float64) error {
	msg := fmt.Sprintf("🚨 Alert: %s\nMetric: %s | Value: %.2f | Threshold: %.2f\nOperator: %s",
		rule.Name, rule.Type, value, rule.Threshold, rule.Operator)

	var sendErr error
	switch rule.Channel {
	case "slack":
		sendErr = postJSON(rule.Destination, map[string]string{"text": msg})
	default:
		sendErr = postJSON(rule.Destination, map[string]interface{}{
			"alert_name": rule.Name,
			"type":       rule.Type,
			"value":      value,
			"threshold":  rule.Threshold,
			"operator":   rule.Operator,
			"message":    msg,
			"fired_at":   time.Now().UTC().Format(time.RFC3339),
		})
	}

	now := time.Now()
	_ = ac.st.UpdateAlertRuleLastFired(rule.ID, now)

	event := models.AlertEvent{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Message:   msg,
		Value:     value,
		Threshold: rule.Threshold,
		Channel:   rule.Channel,
		FiredAt:   now,
	}
	if sendErr != nil {
		event.Status = "failed"
		event.Error = sendErr.Error()
	} else {
		event.Status = "sent"
	}
	_ = ac.st.CreateAlertEvent(&event)
	return sendErr
}

// postJSON sends a generic HTTP POST with JSON payload.
func postJSON(url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
