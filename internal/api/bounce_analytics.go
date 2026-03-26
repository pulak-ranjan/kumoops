package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/core"
)

// BounceClassification categories
const (
	BounceHard      = "hard"         // 5xx permanent
	BounceSoft      = "soft"         // 4xx temporary
	BounceSpam      = "spam"         // spam rejection
	BounceQuota     = "quota"        // mailbox full
	BounceRateLimit = "rate_limit"   // rate limit / too many connections
	BounceAuth      = "auth_failure" // authentication
	BounceDNS       = "dns_failure"  // DNS resolution failure
	BounceTLS       = "tls_failure"  // TLS/cert issues
	BounceUnknown   = "unknown"
)

// smtpCode returns (hard/soft/category) for a given SMTP code
func classifyBounce(code int, errorText string) (bounceType, category string) {
	lower := strings.ToLower(errorText)

	// Rate limiting
	if code == 421 || strings.Contains(lower, "rate") || strings.Contains(lower, "too many") ||
		strings.Contains(lower, "throttl") {
		return BounceSoft, BounceRateLimit
	}

	// Spam
	if strings.Contains(lower, "spam") || strings.Contains(lower, "blocked") ||
		strings.Contains(lower, "blacklist") || strings.Contains(lower, "5.7.1") ||
		strings.Contains(lower, "5.7.0") || strings.Contains(lower, "policy") ||
		(code == 550 && strings.Contains(lower, "reject")) {
		return BounceHard, BounceSpam
	}

	// Quota
	if code == 452 || strings.Contains(lower, "quota") ||
		(strings.Contains(lower, "mailbox") && strings.Contains(lower, "full")) {
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
	if code == 535 || (strings.Contains(lower, "auth") && strings.Contains(lower, "fail")) {
		return BounceHard, BounceAuth
	}

	// Hard bounce by SMTP code
	if code >= 550 && code <= 599 {
		return BounceHard, BounceHard
	}

	// Soft bounce by SMTP code
	if code >= 400 && code <= 499 {
		return BounceSoft, BounceSoft
	}

	// Try to guess from text
	codeRe := regexp.MustCompile(`\b5\d\d\b`)
	if codeRe.MatchString(errorText) {
		return BounceHard, BounceHard
	}
	codeRe4 := regexp.MustCompile(`\b4\d\d\b`)
	if codeRe4.MatchString(errorText) {
		return BounceSoft, BounceSoft
	}

	return BounceUnknown, BounceUnknown
}

type bounceGroup struct {
	Error    string `json:"error"`
	Count    int    `json:"count"`
	Type     string `json:"type"`
	Category string `json:"category"`
}

// kumoLogEntry is the JSON structure written by KumoMTA to /var/log/kumomta/
type kumoLogEntry struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"event_time"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Queue     string    `json:"queue"`
	Site      string    `json:"site"`
	Response  struct {
		Code         int    `json:"code"`
		EnhancedCode string `json:"enhanced_code"`
		Content      string `json:"content"`
	} `json:"response"`
}

type analyticsAccum struct {
	received  int
	delivered int
	hard      int
	soft      int
	spam      int
	deferred  int
	errorGroups map[string]*bounceGroup
	byProvider  map[string]*providerStats
}

type providerStats struct {
	Provider  string `json:"provider"`
	Delivered int    `json:"delivered"`
	Bounced   int    `json:"bounced"`
	Deferred  int    `json:"deferred"`
}

func newAccum() *analyticsAccum {
	return &analyticsAccum{
		errorGroups: make(map[string]*bounceGroup),
		byProvider:  make(map[string]*providerStats),
	}
}

func (a *analyticsAccum) merge(b *analyticsAccum) {
	a.received += b.received
	a.delivered += b.delivered
	a.hard += b.hard
	a.soft += b.soft
	a.spam += b.spam
	a.deferred += b.deferred
	for k, g := range b.errorGroups {
		if existing, ok := a.errorGroups[k]; ok {
			existing.Count += g.Count
		} else {
			cp := *g
			a.errorGroups[k] = &cp
		}
	}
	for k, p := range b.byProvider {
		if existing, ok := a.byProvider[k]; ok {
			existing.Delivered += p.Delivered
			existing.Bounced += p.Bounced
			existing.Deferred += p.Deferred
		} else {
			cp := *p
			a.byProvider[k] = &cp
		}
	}
}

func processLogFileForAnalytics(file string, cutoff time.Time) *analyticsAccum {
	rc, err := core.OpenLogFile(file)
	if err != nil {
		return newAccum()
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	acc := newAccum()

	for scanner.Scan() {
		line := scanner.Text()
		// KumoMTA logs are JSON lines; skip anything before '{'
		if idx := strings.Index(line, "{"); idx >= 0 {
			line = line[idx:]
		} else {
			continue
		}

		var e kumoLogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Timestamp.Before(cutoff) {
			continue
		}

		// Determine receiving domain / provider
		domain := ""
		if e.Recipient != "" {
			parts := strings.SplitN(e.Recipient, "@", 2)
			if len(parts) == 2 {
				domain = strings.ToLower(parts[1])
			}
		}
		provider := classifyProviderDomain(domain)

		switch e.Type {
		case "Reception":
			acc.received++
		case "Delivery":
			acc.delivered++
			if provider != "" {
				if acc.byProvider[provider] == nil {
					acc.byProvider[provider] = &providerStats{Provider: provider}
				}
				acc.byProvider[provider].Delivered++
			}
		case "Bounce":
			bt, cat := classifyBounce(e.Response.Code, e.Response.Content)
			if bt == BounceHard {
				acc.hard++
			} else {
				acc.soft++
			}
			if cat == BounceSpam {
				acc.spam++
			}
			errorKey := normalizeErrorKey(e.Response.Code, e.Response.Content)
			if errorKey != "" {
				if acc.errorGroups[errorKey] == nil {
					acc.errorGroups[errorKey] = &bounceGroup{
						Error:    errorKey,
						Type:     bt,
						Category: cat,
					}
				}
				acc.errorGroups[errorKey].Count++
			}
			if provider != "" {
				if acc.byProvider[provider] == nil {
					acc.byProvider[provider] = &providerStats{Provider: provider}
				}
				acc.byProvider[provider].Bounced++
			}
		case "TransientFailure":
			acc.deferred++
			if provider != "" {
				if acc.byProvider[provider] == nil {
					acc.byProvider[provider] = &providerStats{Provider: provider}
				}
				acc.byProvider[provider].Deferred++
			}
		}
	}

	return acc
}

// normalizeErrorKey creates a clean, deduplicated key from SMTP response.
func normalizeErrorKey(code int, content string) string {
	if content == "" {
		return ""
	}
	// Trim to first 120 chars, remove IPs/hashes
	key := content
	if len(key) > 120 {
		key = key[:120]
	}
	key = regexp.MustCompile(`\b[0-9a-f]{8,}\b`).ReplaceAllString(key, "***")
	key = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).ReplaceAllString(key, "x.x.x.x")
	key = strings.TrimSpace(key)
	if code > 0 {
		codeStr := strconv.Itoa(code)
		if !strings.Contains(key, codeStr) {
			key = codeStr + " " + key
		}
	}
	return key
}

// classifyProviderDomain maps receiving domains to friendly provider names.
func classifyProviderDomain(domain string) string {
	d := strings.ToLower(domain)
	switch {
	case strings.Contains(d, "gmail") || strings.Contains(d, "googlemail"):
		return "Gmail"
	case strings.Contains(d, "yahoo") || strings.Contains(d, "ymail"):
		return "Yahoo"
	case strings.Contains(d, "hotmail") || strings.Contains(d, "outlook") ||
		strings.Contains(d, "live.") || strings.Contains(d, "msn."):
		return "Microsoft"
	case strings.Contains(d, "icloud") || strings.Contains(d, "me.com") || strings.Contains(d, "mac.com"):
		return "Apple"
	case strings.Contains(d, "aol."):
		return "AOL"
	case strings.Contains(d, "proton"):
		return "ProtonMail"
	case strings.Contains(d, "zoho"):
		return "Zoho"
	default:
		if d != "" {
			return d
		}
		return "unknown"
	}
}

// GET /api/bounce-analytics
func (s *Server) handleBounceAnalytics(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if n, err := strconv.Atoi(hoursStr); err == nil && n > 0 && n <= 168 {
			hours = n
		}
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	files, _ := filepath.Glob(filepath.Join(core.KumoLogDir, "*"))

	var mu sync.Mutex
	total := newAccum()

	numWorkers := 10
	fileChan := make(chan string, len(files)+1)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				local := processLogFileForAnalytics(file, cutoff)
				mu.Lock()
				total.merge(local)
				mu.Unlock()
			}
		}()
	}

	for _, file := range files {
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			if info.ModTime().After(cutoff) {
				fileChan <- file
			}
		}
	}
	close(fileChan)
	wg.Wait()

	// Sort error groups by count
	var groups []bounceGroup
	for _, g := range total.errorGroups {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Count > groups[j].Count
	})
	if len(groups) > 25 {
		groups = groups[:25]
	}

	// Build provider list
	var providerList []providerStats
	for _, p := range total.byProvider {
		providerList = append(providerList, *p)
	}
	sort.Slice(providerList, func(i, j int) bool {
		ti := providerList[i].Delivered + providerList[i].Bounced + providerList[i].Deferred
		tj := providerList[j].Delivered + providerList[j].Bounced + providerList[j].Deferred
		return ti > tj
	})
	if len(providerList) > 20 {
		providerList = providerList[:20]
	}

	// Get DB stats for overall numbers (fallback / supplement)
	dbStats, _ := s.Store.GetTodayStats()
	var dbSent, dbBounced, dbDeferred int64
	for _, st := range dbStats {
		dbSent += st.Sent
		dbBounced += st.Bounced
		dbDeferred += st.Deferred
	}

	totalBounces := total.hard + total.soft
	bounceRate := 0.0
	if total.received > 0 {
		bounceRate = float64(totalBounces) / float64(total.received) * 100
	} else if dbSent > 0 {
		bounceRate = float64(dbBounced) / float64(dbSent) * 100
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary": map[string]interface{}{
			"total_received":  total.received,
			"total_sent":      dbSent,
			"delivered":       total.delivered,
			"hard_bounces":    total.hard,
			"soft_bounces":    total.soft,
			"spam_rejections": total.spam,
			"deferred":        total.deferred,
			"bounce_rate":     bounceRate,
			"hours":           hours,
		},
		"top_errors": groups,
		"by_type": map[string]int{
			"hard":       total.hard,
			"soft":       total.soft,
			"spam":       total.spam,
			"rate_limit": countByCategory(groups, BounceRateLimit),
			"quota":      countByCategory(groups, BounceQuota),
			"tls":        countByCategory(groups, BounceTLS),
			"dns":        countByCategory(groups, BounceDNS),
			"auth":       countByCategory(groups, BounceAuth),
		},
		"by_provider": providerList,
	})
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
