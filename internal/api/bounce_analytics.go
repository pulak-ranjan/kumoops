package api

import (
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// BounceClassification categories
const (
	BounceHard          = "hard"          // 5xx permanent
	BounceSoft          = "soft"          // 4xx temporary
	BounceSpam          = "spam"          // spam rejection
	BounceQuota         = "quota"         // mailbox full
	BounceRateLimit     = "rate_limit"    // rate limit / too many connections
	BounceAuth          = "auth_failure"  // authentication
	BounceDNS           = "dns_failure"   // DNS resolution failure
	BounceTLS           = "tls_failure"   // TLS/cert issues
	BounceUnknown       = "unknown"
)

// smtpCode returns (hard/soft/category) for a given SMTP code
func classifyBounce(errorText string) (bounceType, category string) {
	lower := strings.ToLower(errorText)

	// Rate limiting
	if strings.Contains(lower, "rate") || strings.Contains(lower, "too many") ||
		strings.Contains(lower, "421") || strings.Contains(lower, "throttl") {
		return BounceSoft, BounceRateLimit
	}

	// Spam
	if strings.Contains(lower, "spam") || strings.Contains(lower, "blocked") ||
		strings.Contains(lower, "blacklist") || strings.Contains(lower, "5.7.1") ||
		strings.Contains(lower, "5.7.0") || strings.Contains(lower, "policy") ||
		strings.Contains(lower, "550") && strings.Contains(lower, "reject") {
		return BounceHard, BounceSpam
	}

	// Quota
	if strings.Contains(lower, "quota") || strings.Contains(lower, "full") ||
		strings.Contains(lower, "452") || strings.Contains(lower, "mailbox") && strings.Contains(lower, "full") {
		return BounceSoft, BounceQuota
	}

	// TLS
	if strings.Contains(lower, "tls") || strings.Contains(lower, "certificate") ||
		strings.Contains(lower, "ssl") || strings.Contains(lower, "starttls") {
		return BounceSoft, BounceTLS
	}

	// DNS
	if strings.Contains(lower, "dns") || strings.Contains(lower, "nxdomain") ||
		strings.Contains(lower, "no such host") || strings.Contains(lower, "lookup") {
		return BounceHard, BounceDNS
	}

	// Auth
	if strings.Contains(lower, "535") || strings.Contains(lower, "auth") && strings.Contains(lower, "fail") {
		return BounceHard, BounceAuth
	}

	// Hard bounce by SMTP code
	if strings.Contains(lower, "550") || strings.Contains(lower, "551") ||
		strings.Contains(lower, "552") || strings.Contains(lower, "553") ||
		strings.Contains(lower, "554") || strings.Contains(lower, "571") {
		return BounceHard, BounceHard
	}

	// Soft bounce by SMTP code
	if strings.Contains(lower, "4") {
		// Look for 4xx
		codeRe := regexp.MustCompile(`\b4\d\d\b`)
		if codeRe.MatchString(errorText) {
			return BounceSoft, BounceSoft
		}
	}

	return BounceUnknown, BounceUnknown
}

type bounceGroup struct {
	Error   string `json:"error"`
	Count   int    `json:"count"`
	Type    string `json:"type"`
	Category string `json:"category"`
}

// GET /api/bounce-analytics
func (s *Server) handleBounceAnalytics(w http.ResponseWriter, r *http.Request) {
	linesStr := r.URL.Query().Get("lines")
	lines := "2000"
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil && n > 0 {
			lines = strconv.Itoa(n)
		}
	}

	// Read KumoMTA logs
	out, err := exec.Command("journalctl", "-u", "kumomta", "--no-pager", "-n", lines, "--output=short").Output()
	logLines := []string{}
	if err == nil {
		logLines = strings.Split(string(out), "\n")
	}

	// Classify bounces
	hardCount := 0
	softCount := 0
	spamCount := 0
	deliveredCount := 0
	deferredCount := 0
	errorGroups := map[string]*bounceGroup{}

	for _, line := range logLines {
		lower := strings.ToLower(line)

		if strings.Contains(lower, "delivered") || strings.Contains(lower, "250") && strings.Contains(lower, "ok") {
			deliveredCount++
			continue
		}

		if strings.Contains(lower, "deferred") || strings.Contains(lower, "retry") {
			deferredCount++
		}

		// Look for bounce/failure lines
		isBounce := strings.Contains(lower, "bounce") || strings.Contains(lower, "failed") ||
			strings.Contains(lower, "rejected") || strings.Contains(lower, "refused")

		if !isBounce {
			// Check for 4xx/5xx codes
			codeRe := regexp.MustCompile(`\b[45]\d\d\b`)
			if !codeRe.MatchString(line) {
				continue
			}
		}

		bounceType, category := classifyBounce(line)

		switch bounceType {
		case BounceHard:
			hardCount++
		case BounceSoft:
			softCount++
		}
		if category == BounceSpam {
			spamCount++
		}

		// Group by error message (extract key part)
		errorKey := extractErrorKey(line)
		if errorKey != "" {
			if _, exists := errorGroups[errorKey]; !exists {
				errorGroups[errorKey] = &bounceGroup{
					Error:    errorKey,
					Count:    0,
					Type:     bounceType,
					Category: category,
				}
			}
			errorGroups[errorKey].Count++
		}
	}

	// Sort groups by count
	var groups []bounceGroup
	for _, g := range errorGroups {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Count > groups[j].Count
	})
	if len(groups) > 25 {
		groups = groups[:25]
	}

	// Get DB stats for overall numbers
	dbStats, _ := s.Store.GetTodayStats()
	var dbSent, dbBounced, dbDeferred int64
	for _, st := range dbStats {
		dbSent += st.Sent
		dbBounced += st.Bounced
		dbDeferred += st.Deferred
	}

	bounceRate := 0.0
	if dbSent > 0 {
		bounceRate = float64(dbBounced) / float64(dbSent) * 100
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary": map[string]interface{}{
			"total_sent":      dbSent,
			"hard_bounces":    hardCount,
			"soft_bounces":    softCount,
			"spam_rejections": spamCount,
			"delivered":       deliveredCount,
			"deferred":        deferredCount,
			"bounce_rate":     bounceRate,
		},
		"top_errors": groups,
		"by_type": map[string]int{
			"hard":        hardCount,
			"soft":        softCount,
			"spam":        spamCount,
			"rate_limit":  countByCategory(groups, BounceRateLimit),
			"quota":       countByCategory(groups, BounceQuota),
			"tls":         countByCategory(groups, BounceTLS),
			"dns":         countByCategory(groups, BounceDNS),
			"auth":        countByCategory(groups, BounceAuth),
		},
	})
}

func extractErrorKey(line string) string {
	// Extract meaningful error portion
	lower := strings.ToLower(line)
	markers := []string{"error:", "failed:", "rejected:", "refused:", "bounce:", "550", "421", "452", "535", "554"}
	for _, marker := range markers {
		idx := strings.Index(lower, marker)
		if idx >= 0 {
			end := idx + 80
			if end > len(line) {
				end = len(line)
			}
			key := strings.TrimSpace(line[idx:end])
			// Normalize: remove timestamps, IDs
			key = regexp.MustCompile(`\b[0-9a-f]{8,}\b`).ReplaceAllString(key, "***")
			key = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).ReplaceAllString(key, "x.x.x.x")
			if len(key) > 80 {
				key = key[:80]
			}
			return key
		}
	}
	return ""
}

func countByCategory(groups []bounceGroup, cat string) int {
	count := 0
	for _, g := range groups {
		if g.Category == cat {
			count += g.Count
		}
	}
	return count
}
