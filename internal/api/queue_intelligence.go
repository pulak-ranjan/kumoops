package api

import (
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/core"
)

// GET /api/queue/providers
// Returns per-destination-domain queue depth breakdown
func (s *Server) handleQueueByProvider(w http.ResponseWriter, r *http.Request) {
	msgs, err := core.GetQueueMessages(500)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read queue"})
		return
	}

	// Group by recipient domain
	type providerStat struct {
		Domain   string `json:"domain"`
		Total    int    `json:"total"`
		Queued   int    `json:"queued"`
		Deferred int    `json:"deferred"`
		Oldest   string `json:"oldest,omitempty"`
	}

	grouped := map[string]*providerStat{}
	for _, msg := range msgs {
		// Extract recipient domain
		rcptDomain := "unknown"
		if at := strings.LastIndex(msg.Recipient, "@"); at >= 0 {
			rcptDomain = strings.ToLower(msg.Recipient[at+1:])
		}

		if _, ok := grouped[rcptDomain]; !ok {
			grouped[rcptDomain] = &providerStat{Domain: rcptDomain}
		}
		stat := grouped[rcptDomain]
		stat.Total++
		if msg.Attempts > 0 {
			stat.Deferred++
		} else {
			stat.Queued++
		}
		if stat.Oldest == "" || msg.CreatedAt.Before(parseTime(stat.Oldest)) {
			stat.Oldest = msg.CreatedAt.Format(time.RFC3339)
		}
	}

	// Sort by total descending
	var result []*providerStat
	for _, v := range grouped {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Total > result[j].Total
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers":   result,
		"total_queue": len(msgs),
	})
}

// GET /api/queue/stuck
// Returns messages that have been deferred more than N times or older than 24h
func (s *Server) handleStuckMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := core.GetQueueMessages(500)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read queue"})
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	var stuck []interface{}
	for _, msg := range msgs {
		isStuck := msg.Attempts >= 5 || (!msg.CreatedAt.IsZero() && msg.CreatedAt.Before(cutoff))
		if isStuck {
			stuck = append(stuck, map[string]interface{}{
				"id":         msg.ID,
				"sender":     msg.Sender,
				"recipient":  msg.Recipient,
				"attempts":   msg.Attempts,
				"age_hours":  time.Since(msg.CreatedAt).Hours(),
				"last_error": msg.ErrorMsg,
				"created":    msg.CreatedAt.Format(time.RFC3339),
			})
		}
	}
	if stuck == nil {
		stuck = []interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stuck_count": len(stuck),
		"messages":    stuck,
	})
}

// GET /api/logs/search?service=kumomta&q=bounce&lines=200
func (s *Server) handleLogSearch(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	query := r.URL.Query().Get("q")
	linesStr := r.URL.Query().Get("lines")
	lines := "500"
	if linesStr != "" {
		lines = linesStr
	}

	unitMap := map[string]string{
		"kumomta":  "kumomta",
		"dovecot":  "dovecot",
		"fail2ban": "fail2ban",
		"nginx":    "nginx",
	}
	unit, ok := unitMap[service]
	if !ok {
		unit = "kumomta"
	}

	args := []string{"-u", unit, "--no-pager", "-n", lines, "--output=short"}
	out, err := exec.Command("journalctl", args...).Output()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read logs"})
		return
	}

	allLines := strings.Split(string(out), "\n")
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"service": unit,
			"query":   query,
			"lines":   allLines,
			"count":   len(allLines),
		})
		return
	}

	queryLower := strings.ToLower(query)
	var matched []string
	for _, line := range allLines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			matched = append(matched, line)
		}
	}
	if matched == nil {
		matched = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service": unit,
		"query":   query,
		"lines":   matched,
		"count":   len(matched),
	})
}

// GET /api/logs/patterns?service=kumomta
// Detects common patterns and error frequencies
func (s *Server) handleLogPatterns(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		service = "kumomta"
	}

	args := []string{"-u", service, "--no-pager", "-n", "1000", "--output=short"}
	out, err := exec.Command("journalctl", args...).Output()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read logs"})
		return
	}

	lines := strings.Split(string(out), "\n")

	// Count pattern occurrences
	patterns := map[string]int{
		"bounce":        0,
		"deferred":      0,
		"refused":       0,
		"timeout":       0,
		"tls":           0,
		"connection":    0,
		"rate limit":    0,
		"550":           0,
		"421":           0,
		"451":           0,
		"authentication": 0,
		"delivered":     0,
		"error":         0,
	}

	for _, line := range lines {
		lower := strings.ToLower(line)
		for pat := range patterns {
			if strings.Contains(lower, pat) {
				patterns[pat]++
			}
		}
	}

	// Build sorted result
	type patternResult struct {
		Pattern string `json:"pattern"`
		Count   int    `json:"count"`
	}
	var results []patternResult
	for pat, count := range patterns {
		if count > 0 {
			results = append(results, patternResult{Pattern: pat, Count: count})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Count > results[j].Count
	})

	// Extract recent errors (lines with ERROR or error keywords)
	var recentErrors []string
	for i := len(lines) - 1; i >= 0 && len(recentErrors) < 20; i-- {
		lower := strings.ToLower(lines[i])
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "bounce") {
			if strings.TrimSpace(lines[i]) != "" {
				recentErrors = append(recentErrors, lines[i])
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":       service,
		"total_lines":   len(lines),
		"patterns":      results,
		"recent_errors": recentErrors,
	})
}

// parseTime parses RFC3339 time, returns zero time on error
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

