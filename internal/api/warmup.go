package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// DTO for the frontend table
type WarmupDTO struct {
	SenderID    uint      `json:"sender_id"`
	Email       string    `json:"email"`
	Domain      string    `json:"domain"`
	Enabled     bool      `json:"enabled"`
	Plan        string    `json:"plan"`
	Day         int       `json:"day"`
	CurrentRate string    `json:"current_rate"`
	LastUpdate  time.Time `json:"last_update"`
}

// GET /api/warmup
// Returns a list of all senders + their current warmup status/rate
func (s *Server) handleGetWarmupList(w http.ResponseWriter, r *http.Request) {
	domains, err := s.Store.ListDomains()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list domains"})
		return
	}

	var list []WarmupDTO

	for _, d := range domains {
		for _, snd := range d.Senders {
			// Calculate what the current rate is right now
			rate := core.GetSenderRate(snd)
			
			// Friendly display text
			if rate == "" && snd.WarmupEnabled {
				rate = "Complete"
			} else if rate == "" {
				rate = "Unlimited"
			}

			list = append(list, WarmupDTO{
				SenderID:    snd.ID,
				Email:       snd.Email,
				Domain:      d.Name,
				Enabled:     snd.WarmupEnabled,
				Plan:        snd.WarmupPlan,
				Day:         snd.WarmupDay,
				CurrentRate: rate,
				LastUpdate:  snd.WarmupLastUpdate,
			})
		}
	}

	writeJSON(w, http.StatusOK, list)
}

// POST /api/warmup/{id}
// Toggles warmup on/off or changes the plan for a specific sender
func (s *Server) handleUpdateWarmup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	var req struct {
		Enabled bool   `json:"enabled"`
		Plan    string `json:"plan"` // "standard", "conservative", "aggressive"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	// Update DB fields
	sender.WarmupEnabled = req.Enabled
	if req.Enabled {
		sender.WarmupPlan = req.Plan
		// Safety: If enabling for the first time, start at Day 1
		if sender.WarmupDay == 0 {
			sender.WarmupDay = 1
			sender.WarmupLastUpdate = time.Now()
		}
	}

	if err := s.Store.UpdateSender(sender); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		return
	}

	// Apply config immediately to enforce new rate (don't wait for daily cron)
	go func() {
		snap, _ := core.LoadSnapshot(s.Store)
		core.ApplyKumoConfig(snap)
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// POST /api/warmup/{id}/pause
func (s *Server) handlePauseWarmup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	oldRate := core.GetSenderRate(*sender)
	sender.WarmupEnabled = false
	if err := s.Store.UpdateSender(sender); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		return
	}

	_ = s.Store.CreateWarmupLog(&models.WarmupLog{
		SenderID: sender.ID,
		Email:    sender.Email,
		Event:    "paused",
		OldDay:   sender.WarmupDay,
		NewDay:   sender.WarmupDay,
		OldRate:  oldRate,
		NewRate:  oldRate,
		Reason:   req.Reason,
	})

	go func() {
		snap, _ := core.LoadSnapshot(s.Store)
		core.ApplyKumoConfig(snap)
	}()
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

// POST /api/warmup/{id}/resume
func (s *Server) handleResumeWarmup(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	oldRate := core.GetSenderRate(*sender)
	sender.WarmupEnabled = true
	if err := s.Store.UpdateSender(sender); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		return
	}

	newRate := core.GetSenderRate(*sender)
	_ = s.Store.CreateWarmupLog(&models.WarmupLog{
		SenderID: sender.ID,
		Email:    sender.Email,
		Event:    "resumed",
		OldDay:   sender.WarmupDay,
		NewDay:   sender.WarmupDay,
		OldRate:  oldRate,
		NewRate:  newRate,
	})

	go func() {
		snap, _ := core.LoadSnapshot(s.Store)
		core.ApplyKumoConfig(snap)
	}()
	writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
}

// GET /api/warmup/{id}/calendar
// Returns the full warmup schedule (volume per day) for visualization
func (s *Server) handleWarmupCalendar(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	sender, err := s.Store.GetSenderByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sender not found"})
		return
	}

	plan := core.GetWarmupPlan(sender.WarmupPlan)
	if plan == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sender_id": sender.ID,
			"email":     sender.Email,
			"schedule":  []interface{}{},
		})
		return
	}

	type DayEntry struct {
		Day     int    `json:"day"`
		Rate    string `json:"rate"`
		IsToday bool   `json:"is_today"`
		IsDone  bool   `json:"is_done"`
	}

	var schedule []DayEntry
	for _, entry := range plan {
		schedule = append(schedule, DayEntry{
			Day:     entry.Day,
			Rate:    entry.Rate,
			IsToday: entry.Day == sender.WarmupDay && sender.WarmupEnabled,
			IsDone:  entry.Day < sender.WarmupDay,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sender_id":   sender.ID,
		"email":       sender.Email,
		"plan":        sender.WarmupPlan,
		"current_day": sender.WarmupDay,
		"is_enabled":  sender.WarmupEnabled,
		"schedule":    schedule,
	})
}

// GET /api/warmup/{id}/logs
func (s *Server) handleWarmupLogs(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	logs, err := s.Store.ListWarmupLogs(uint(id), 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get warmup logs"})
		return
	}
	writeJSON(w, http.StatusOK, logs)
}
