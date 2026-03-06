package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// GET /api/alerts/rules
func (s *Server) handleListAlertRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.Store.ListAlertRules()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list alert rules"})
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// POST /api/alerts/rules
func (s *Server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule models.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if rule.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if rule.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type is required"})
		return
	}
	rule.ID = 0
	if err := s.Store.CreateAlertRule(&rule); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create alert rule"})
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

// PUT /api/alerts/rules/{id}
func (s *Server) handleUpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	existing, err := s.Store.GetAlertRule(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}
	var req models.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.ID = existing.ID
	req.CreatedAt = existing.CreatedAt
	req.LastFired = existing.LastFired
	if err := s.Store.UpdateAlertRule(&req); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update alert rule"})
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// DELETE /api/alerts/rules/{id}
func (s *Server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.Store.DeleteAlertRule(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete alert rule"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /api/alerts/events?limit=50
func (s *Server) handleListAlertEvents(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	events, err := s.Store.ListAlertEvents(limit)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list alert events"})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// POST /api/alerts/test/{id}
func (s *Server) handleTestAlert(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	rule, err := s.Store.GetAlertRule(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}

	msg := fmt.Sprintf("🔔 TEST ALERT: %s\nType: %s | Threshold: %.2f | Channel: %s\nThis is a test notification.", rule.Name, rule.Type, rule.Threshold, rule.Channel)

	var sendErr error
	switch rule.Channel {
	case "slack":
		sendErr = sendSlackMessage(rule.Destination, msg)
	case "webhook", "email":
		sendErr = sendWebhookAlert(rule.Destination, map[string]interface{}{
			"alert_name": rule.Name,
			"type":       rule.Type,
			"message":    msg,
			"test":       true,
		})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown channel type"})
		return
	}

	event := models.AlertEvent{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Message:   msg,
		Value:     0,
		Threshold: rule.Threshold,
		Channel:   rule.Channel,
	}
	if sendErr != nil {
		event.Status = "failed"
		event.Error = sendErr.Error()
	} else {
		event.Status = "sent"
	}
	_ = s.Store.CreateAlertEvent(&event)

	if sendErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"status": "failed",
			"error":  sendErr.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
