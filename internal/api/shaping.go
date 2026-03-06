package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// GET /api/shaping
func (s *Server) handleListShaping(w http.ResponseWriter, r *http.Request) {
	rules, err := s.Store.ListShapingRules()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list shaping rules"})
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// POST /api/shaping
func (s *Server) handleCreateShaping(w http.ResponseWriter, r *http.Request) {
	var rule models.TrafficShapingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if rule.Pattern == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pattern is required"})
		return
	}

	rule.ID = 0 // ensure insert, not update
	if err := s.Store.CreateShapingRule(&rule); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create shaping rule"})
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

// PUT /api/shaping/{id}
func (s *Server) handleUpdateShaping(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	existing, err := s.Store.GetShapingRule(uint(id))
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "shaping rule not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch shaping rule"})
		return
	}

	var req models.TrafficShapingRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Apply updates, preserving the ID
	req.ID = existing.ID
	req.CreatedAt = existing.CreatedAt
	if err := s.Store.UpdateShapingRule(&req); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update shaping rule"})
		return
	}

	writeJSON(w, http.StatusOK, req)
}

// DELETE /api/shaping/{id}
func (s *Server) handleDeleteShaping(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.Store.DeleteShapingRule(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete shaping rule"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/shaping/seed
func (s *Server) handleSeedShaping(w http.ResponseWriter, r *http.Request) {
	// Delete all existing rules first, then re-seed defaults
	if err := s.Store.DB.Where("1 = 1").Delete(&models.TrafficShapingRule{}).Error; err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to clear existing rules"})
		return
	}

	if err := s.Store.SeedDefaultShapingRules(); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to seed shaping rules"})
		return
	}

	rules, err := s.Store.ListShapingRules()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "seeded but failed to list rules"})
		return
	}

	writeJSON(w, http.StatusOK, rules)
}
