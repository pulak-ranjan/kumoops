package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// GET /api/ippools
func (s *Server) handleListIPPools(w http.ResponseWriter, r *http.Request) {
	pools, err := s.Store.ListIPPools()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list ip pools"})
		return
	}
	writeJSON(w, http.StatusOK, pools)
}

// POST /api/ippools
func (s *Server) handleCreateIPPool(w http.ResponseWriter, r *http.Request) {
	var pool models.IPPool
	if err := json.NewDecoder(r.Body).Decode(&pool); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if pool.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	pool.ID = 0      // ensure insert, not update
	pool.Members = nil // members added separately via member endpoints
	if err := s.Store.CreateIPPool(&pool); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create ip pool"})
		return
	}

	writeJSON(w, http.StatusCreated, pool)
}

// PUT /api/ippools/{id}
func (s *Server) handleUpdateIPPool(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	existing, err := s.Store.GetIPPool(uint(id))
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ip pool not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch ip pool"})
		return
	}

	var req models.IPPool
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Preserve immutable fields
	req.ID = existing.ID
	req.CreatedAt = existing.CreatedAt
	req.Members = existing.Members // do not replace members via this endpoint
	if err := s.Store.UpdateIPPool(&req); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update ip pool"})
		return
	}

	writeJSON(w, http.StatusOK, req)
}

// DELETE /api/ippools/{id}
func (s *Server) handleDeleteIPPool(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := s.Store.DeleteIPPool(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete ip pool"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/ippools/{id}/members
func (s *Server) handleAddIPToPool(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	// Verify pool exists
	if _, err := s.Store.GetIPPool(uint(id)); err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ip pool not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch ip pool"})
		return
	}

	var req struct {
		IPValue string `json:"ip_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.IPValue == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ip_value is required"})
		return
	}

	if err := s.Store.AddIPToPool(uint(id), req.IPValue); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add ip to pool"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "added", "ip_value": req.IPValue})
}

// DELETE /api/ippools/{id}/members/{mid}
func (s *Server) handleRemoveIPFromPool(w http.ResponseWriter, r *http.Request) {
	midStr := chi.URLParam(r, "mid")
	mid, err := strconv.ParseUint(midStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	if err := s.Store.RemoveIPFromPool(uint(mid)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove ip from pool"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
