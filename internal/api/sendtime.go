package api

import (
	"net/http"
	"strconv"

	"github.com/pulak-ranjan/kumoops/internal/core"
)

// GET /api/analytics/send-time?domain=&days=90
func (s *Server) handleSendTimeHeatmap(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	days := 90
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	heatmap, err := core.GetSendTimeHeatmap(s.Store, domain, days)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, heatmap)
}
