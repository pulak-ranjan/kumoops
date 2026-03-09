package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// ─────────────────────────────────────────────
// Seed Mailboxes
// ─────────────────────────────────────────────

// GET /api/placement/mailboxes
func (s *Server) handleListSeedMailboxes(w http.ResponseWriter, r *http.Request) {
	ms, err := s.Store.ListSeedMailboxes()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// Strip encrypted passwords from response
	for i := range ms {
		ms[i].Password = ""
	}
	writeJSON(w, http.StatusOK, ms)
}

// POST /api/placement/mailboxes
func (s *Server) handleCreateSeedMailbox(w http.ResponseWriter, r *http.Request) {
	var m models.SeedMailbox
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if m.Email == "" || m.IMAPHost == "" || m.Username == "" || m.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email, imap_host, username, password required"})
		return
	}
	// Encrypt password
	enc, err := core.Encrypt(m.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encrypt: " + err.Error()})
		return
	}
	m.Password = enc
	if m.IMAPPort == 0 {
		m.IMAPPort = 993
	}
	m.IsActive = true
	if err := s.Store.CreateSeedMailbox(&m); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	m.Password = ""
	writeJSON(w, http.StatusCreated, m)
}

// DELETE /api/placement/mailboxes/{id}
func (s *Server) handleDeleteSeedMailbox(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := s.Store.DeleteSeedMailbox(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─────────────────────────────────────────────
// Placement Tests
// ─────────────────────────────────────────────

// GET /api/placement/tests?limit=20
func (s *Server) handleListPlacementTests(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	tests, err := s.Store.ListPlacementTests(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tests)
}

// POST /api/placement/tests — create + trigger a new placement test
func (s *Server) handleCreatePlacementTest(w http.ResponseWriter, r *http.Request) {
	var t models.PlacementTest
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if t.Name == "" || t.Subject == "" || t.SenderID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, subject, sender_id required"})
		return
	}
	t.Status = "pending"
	if err := s.Store.CreatePlacementTest(&t); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// Run async
	go core.NewPlacementService(s.Store).RunTest(&t)
	writeJSON(w, http.StatusCreated, t)
}

// GET /api/placement/tests/{id}
func (s *Server) handleGetPlacementTest(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	t, err := s.Store.GetPlacementTest(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "test not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}
