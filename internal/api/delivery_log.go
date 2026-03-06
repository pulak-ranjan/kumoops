package api

import (
	"net/http"
	"strconv"

	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// GET /api/delivery-log
// Returns a paginated, filtered list of per-recipient failure events.
func (s *Server) handleListDeliveryLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	hours, _ := strconv.Atoi(q.Get("hours"))

	filter := store.DeliveryEventFilter{
		Recipient: q.Get("recipient"),
		Domain:    q.Get("domain"),
		EventType: q.Get("type"),
		Hours:     hours,
	}

	events, total, err := s.Store.ListDeliveryEvents(page, limit, filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list events"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GET /api/delivery-log/summary
// Returns counts of Bounce and TransientFailure events for the given time window.
func (s *Server) handleDeliveryLogSummary(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 72
	}

	summary, err := s.Store.GetDeliveryEventSummary(hours)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get summary"})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// POST /api/delivery-log/refresh
// Parses KumoMTA logs and populates the DeliveryEvent table.
func (s *Server) handleRefreshDeliveryLog(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 72
	}

	if err := core.ParseDeliveryEvents(s.Store, hours); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}
