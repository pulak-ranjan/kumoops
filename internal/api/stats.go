package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// dbEmailStatsToMap converts DB EmailStats rows into the map[domain][]DayStats format.
func dbEmailStatsToMap(rows []models.EmailStats) map[string][]core.DayStats {
	temp := make(map[string]map[string]*core.DayStats)
	for _, r := range rows {
		d := r.Date.Format("2006-01-02")
		if temp[r.Domain] == nil {
			temp[r.Domain] = make(map[string]*core.DayStats)
		}
		temp[r.Domain][d] = &core.DayStats{
			Date:      d,
			Sent:      r.Sent,
			Delivered: r.Delivered,
			Bounced:   r.Bounced,
			Deferred:  r.Deferred,
		}
	}
	out := make(map[string][]core.DayStats)
	for domain, dateMap := range temp {
		list := make([]core.DayStats, 0, len(dateMap))
		for _, s := range dateMap {
			list = append(list, *s)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
		out[domain] = list
	}
	return out
}

// GET /api/stats/domains
// Returns aggregated stats for all domains over the requested period (default 7 days).
// Falls back to database when no KumoMTA log files are present.
func (s *Server) handleGetDomainStats(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
		days = d
	}

	stats, err := core.GetAllDomainsStats(days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	// Fall back to DB when log files are absent or empty
	if len(stats) == 0 {
		rows, dbErr := s.Store.GetEmailStatsAll(days)
		if dbErr == nil && len(rows) > 0 {
			stats = dbEmailStatsToMap(rows)
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

// GET /api/stats/domains/{domain}
// Returns detailed daily stats for a specific domain.
func (s *Server) handleGetSingleDomainStats(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain required"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
		days = d
	}

	stats, err := core.GetDomainStatsFromLogs(domain, days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GET /api/stats/summary
// Returns a high-level summary (Sent/Bounced/Delivered) for the Dashboard.
func (s *Server) handleGetStatsSummary(w http.ResponseWriter, r *http.Request) {
	// Get today's stats (1 day) for the quick view
	stats, err := core.GetAllDomainsStats(1)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get stats"})
		return
	}

	summary := struct {
		TotalSent      int64   `json:"total_sent"`
		TotalDelivered int64   `json:"total_delivered"`
		TotalBounced   int64   `json:"total_bounced"`
		TotalDeferred  int64   `json:"total_deferred"`
		DeliveryRate   float64 `json:"delivery_rate"`
		BounceRate     float64 `json:"bounce_rate"`
		DomainsActive  int     `json:"domains_active"`
	}{}

	for _, domainStats := range stats {
		for _, day := range domainStats {
			summary.TotalSent += day.Sent
			summary.TotalDelivered += day.Delivered
			summary.TotalBounced += day.Bounced
			summary.TotalDeferred += day.Deferred
		}
	}

	summary.DomainsActive = len(stats)

	if summary.TotalSent > 0 {
		summary.DeliveryRate = float64(summary.TotalDelivered) / float64(summary.TotalSent) * 100
		summary.BounceRate = float64(summary.TotalBounced) / float64(summary.TotalSent) * 100
	}

	writeJSON(w, http.StatusOK, summary)
}

// GET /api/stats/providers
// Returns delivery stats broken down by email provider (ISP).
func (s *Server) handleGetProviderStats(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
		days = d
	}

	stats, err := core.GetProviderStats(days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get provider stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// POST /api/stats/refresh
// Triggers a manual parsing of logs to update the database cache.
func (s *Server) handleRefreshStats(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
		hours = h
	}

	// This now uses the Zstd-capable, parallel parser
	if err := core.ParseKumoLogs(s.Store, hours); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to parse logs"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}

// GET /api/stats/hourly
// Returns per-hour aggregated stats for the last N hours (default 24, max 168).
func (s *Server) handleGetHourlyStats(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
		hours = h
	}

	stats, err := core.GetHourlyStats(hours)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get hourly stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
