package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
)

// ─────────────────────────────────────────────
// ISP Intelligence
// ─────────────────────────────────────────────

// GET /api/isp-intel/snapshots?domain=&isp=&days=7&limit=100
func (s *Server) handleListISPSnapshots(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	isp := r.URL.Query().Get("isp")
	days := 7
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	limit := 100
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	since := time.Now().AddDate(0, 0, -days)
	snaps, err := s.Store.ListISPSnapshots(domain, isp, since, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

// GET /api/isp-intel/snapshots/latest?domain=
func (s *Server) handleGetLatestISPSnapshots(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	snaps, err := s.Store.GetLatestISPSnapshots(domain)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

// POST /api/isp-intel/refresh — trigger a manual intelligence refresh
func (s *Server) handleRefreshISPIntel(w http.ResponseWriter, r *http.Request) {
	svc := core.NewISPIntelService(s.Store)
	go svc.RefreshAll() // run in background; returns immediately
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refresh_started"})
}

// GET /api/isp-intel/metrics?isp=Gmail&days=1
func (s *Server) handleGetISPMetrics(w http.ResponseWriter, r *http.Request) {
	isp := r.URL.Query().Get("isp")
	if isp == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "isp required"})
		return
	}
	days := 1
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	from := time.Now().AddDate(0, 0, -days)
	metrics, err := s.Store.GetISPDeliveryMetrics(isp, from, time.Now())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

// ─────────────────────────────────────────────
// Adaptive Throttle
// ─────────────────────────────────────────────

// GET /api/throttle/logs?isp=&days=7&limit=100
func (s *Server) handleListThrottleLogs(w http.ResponseWriter, r *http.Request) {
	isp := r.URL.Query().Get("isp")
	days := 7
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	limit := 100
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	since := time.Now().AddDate(0, 0, -days)
	logs, err := s.Store.ListThrottleAdjustmentLogs(isp, since, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

// POST /api/throttle/run — trigger a manual adaptive throttle cycle
func (s *Server) handleRunAdaptiveThrottle(w http.ResponseWriter, r *http.Request) {
	at := core.NewAdaptiveThrottler(s.Store)
	go at.Run()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "throttle_cycle_started"})
}

// ─────────────────────────────────────────────
// Anomaly Events
// ─────────────────────────────────────────────

// GET /api/anomalies?days=7&limit=100
func (s *Server) handleListAnomalyEvents(w http.ResponseWriter, r *http.Request) {
	days := 7
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	limit := 100
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	since := time.Now().AddDate(0, 0, -days)
	events, err := s.Store.ListAnomalyEvents(since, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// GET /api/anomalies/active
func (s *Server) handleListActiveAnomalies(w http.ResponseWriter, r *http.Request) {
	events, err := s.Store.ListActiveAnomalyEvents()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// POST /api/anomalies/{id}/resolve — manually resolve an anomaly event
func (s *Server) handleResolveAnomaly(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	active, err := s.Store.ListActiveAnomalyEvents()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	for i := range active {
		if active[i].ID == uint(id) {
			now := time.Now()
			active[i].ResolvedAt = &now
			active[i].AutoHealed = false
			if err := s.Store.ResolveAnomalyEvent(&active[i]); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "anomaly not found or already resolved"})
}
