package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// GET /api/campaigns/{id}/ab-summary
func (s *Server) handleGetABSummary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid campaign ID"})
		return
	}
	summary, err := core.GetABTestSummary(s.Store, uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// GET /api/campaigns/{id}/variants
func (s *Server) handleListVariants(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid campaign ID"})
		return
	}
	vs, err := s.Store.ListCampaignVariants(uint(id))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, vs)
}

// POST /api/campaigns/{id}/variants
func (s *Server) handleCreateVariant(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid campaign ID"})
		return
	}
	var v models.CampaignVariant
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if v.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	v.CampaignID = uint(id)
	if err := s.Store.CreateCampaignVariant(&v); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

// DELETE /api/campaigns/{id}/variants/{vid}
func (s *Server) handleDeleteVariant(w http.ResponseWriter, r *http.Request) {
	vid, err := strconv.ParseUint(chi.URLParam(r, "vid"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid variant ID"})
		return
	}
	if err := s.Store.DeleteCampaignVariant(uint(vid)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/campaigns/{id}/variants/{vid}/set-winner — manual winner selection
func (s *Server) handleSetABWinner(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid campaign ID"})
		return
	}
	vid, err := strconv.ParseUint(chi.URLParam(r, "vid"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid variant ID"})
		return
	}
	if err := s.Store.SetABWinner(uint(id), uint(vid)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "winner_set"})
}
