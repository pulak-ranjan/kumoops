package api

import (
	"net/http"
	"sync/atomic"

	"github.com/pulak-ranjan/kumoops/internal/core"
)

// checkInProgress is a simple flag to prevent concurrent reputation scans.
var checkInProgress int32

// GET /api/reputation
// Returns the latest reputation check result for every tracked target.
func (s *Server) handleGetReputation(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.GetLatestReputationChecks()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// POST /api/reputation/check
// Triggers an async reputation scan of all IPs and domains.
func (s *Server) handleRunReputationCheck(w http.ResponseWriter, r *http.Request) {
	if !atomic.CompareAndSwapInt32(&checkInProgress, 0, 1) {
		writeJSON(w, http.StatusConflict, map[string]string{"status": "already_running"})
		return
	}
	go func() {
		defer atomic.StoreInt32(&checkInProgress, 0)
		_, err := core.CheckReputation(s.Store)
		if err != nil {
			s.Store.LogError(err)
		}
	}()
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

// GET /api/reputation/status
// Returns whether a check is currently running.
func (s *Server) handleReputationStatus(w http.ResponseWriter, r *http.Request) {
	running := atomic.LoadInt32(&checkInProgress) == 1
	writeJSON(w, http.StatusOK, map[string]bool{"running": running})
}

// GET /api/reputation/delist-urls
// Returns the delist URL for each RBL zone.
func (s *Server) handleDelistURLs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, core.DelistURLs)
}
