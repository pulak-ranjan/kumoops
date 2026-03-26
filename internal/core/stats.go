package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// --- CACHING STRUCTURES ---
var (
	cacheLock      sync.RWMutex
	cachedStats    map[string][]DayStats
	cacheExpiry    time.Time
	CACHE_DURATION = 60 * time.Second // Keep data in RAM for 60s
)

// KumoMTA log entry structure
type KumoLogEntry struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"event_time"`
	Sender    string    `json:"sender"`

	// NEW: Capture response for AI analysis
	Response struct {
		Code    int    `json:"code"`
		Content string `json:"content"`
	} `json:"response"`
}

const KumoLogDir = "/var/log/kumomta"

// --- FILE HELPERS ---

// openLogFile intelligently handles both Zstd compressed logs and plain JSON logs
// OpenLogFile is the exported version for use outside the core package.
func OpenLogFile(path string) (io.ReadCloser, error) { return openLogFile(path) }

func openLogFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Peek at the first 4 bytes to check for Zstd magic number (0x28 0xB5 0x2F 0xFD)
	br := bufio.NewReader(f)
	magic, _ := br.Peek(4)

	if len(magic) >= 4 && magic[0] == 0x28 && magic[1] == 0xB5 && magic[2] == 0x2F && magic[3] == 0xFD {
		// It is compressed (Zstd)
		decoder, err := zstd.NewReader(br)
		if err != nil {
			f.Close()
			return nil, err
		}
		return &compressedReader{f: f, d: decoder}, nil
	}

	// It is plain text (JSON)
	return &plainReader{f: f, r: br}, nil
}

type compressedReader struct {
	f *os.File
	d *zstd.Decoder
}

func (c *compressedReader) Read(p []byte) (int, error) { return c.d.Read(p) }
func (c *compressedReader) Close() error {
	c.d.Close()
	return c.f.Close()
}

type plainReader struct {
	f *os.File
	r *bufio.Reader
}

func (p *plainReader) Read(b []byte) (int, error) { return p.r.Read(b) }
func (p *plainReader) Close() error               { return p.f.Close() }

func extractDomain(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// --- PARALLEL PROCESSING ---

// ParseKumoLogs (Database Sync)
func ParseKumoLogs(st *store.Store, hoursBack int) error {
	days := hoursBack / 24
	if days < 1 {
		days = 1
	}

	stats, err := GetAllDomainsStats(days)
	if err != nil {
		return err
	}

	for domain, daysData := range stats {
		for _, day := range daysData {
			parsedDate, _ := time.Parse("2006-01-02", day.Date)
			dbStat := &models.EmailStats{
				Domain:    domain,
				Date:      parsedDate,
				Sent:      day.Sent,
				Delivered: day.Delivered,
				Bounced:   day.Bounced,
				Deferred:  day.Deferred,
			}
			st.SetEmailStats(dbStat)
		}
	}
	return nil
}

// GetDomainStatsFromLogs (Single Domain)
func GetDomainStatsFromLogs(domain string, days int) ([]DayStats, error) {
	all, err := GetAllDomainsStats(days)
	if err != nil {
		return nil, err
	}
	if d, ok := all[domain]; ok {
		return d, nil
	}
	empty := make([]DayStats, days)
	now := time.Now()
	for i := 0; i < days; i++ {
		empty[i] = DayStats{Date: now.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")}
	}
	return empty, nil
}

type DayStats struct {
	Date      string `json:"date"`
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	Bounced   int64  `json:"bounced"`
	Deferred  int64  `json:"deferred"`
}


// GetAllDomainsStats (Aggregated & Parallel)
func GetAllDomainsStats(days int) (map[string][]DayStats, error) {
	cacheLock.RLock()
	if time.Now().Before(cacheExpiry) && cachedStats != nil {
		res := cachedStats
		cacheLock.RUnlock()
		return res, nil
	}
	cacheLock.RUnlock()

	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*"))
	now := time.Now()
	cutoff := now.AddDate(0, 0, -days)

	var mu sync.Mutex
	tempStats := make(map[string]map[string]*DayStats)

	numWorkers := 50
	fileChan := make(chan string, len(files))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				processFile(file, cutoff, &mu, tempStats)
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

	result := make(map[string][]DayStats)
	for domain, dateMap := range tempStats {
		daysList := make([]DayStats, 0, len(dateMap))
		for _, stat := range dateMap {
			daysList = append(daysList, *stat)
		}
		for i := 0; i < len(daysList)-1; i++ {
			for j := i + 1; j < len(daysList); j++ {
				if daysList[i].Date > daysList[j].Date {
					daysList[i], daysList[j] = daysList[j], daysList[i]
				}
			}
		}
		result[domain] = daysList
	}

	cacheLock.Lock()
	cachedStats = result
	cacheExpiry = time.Now().Add(CACHE_DURATION)
	cacheLock.Unlock()

	return result, nil
}

func processFile(file string, cutoff time.Time, mu *sync.Mutex, tempStats map[string]map[string]*DayStats) {
	rc, err := openLogFile(file)
	if err != nil {
		return
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	localStats := make(map[string]map[string]*DayStats)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "{"); idx >= 0 {
			line = line[idx:]
		} else {
			continue
		}

		var entry KumoLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Timestamp.Before(cutoff) {
			continue
		}

		domain := extractDomain(entry.Sender)
		if domain == "" {
			continue
		}

		dateKey := entry.Timestamp.Format("2006-01-02")

		if localStats[domain] == nil {
			localStats[domain] = make(map[string]*DayStats)
		}
		if localStats[domain][dateKey] == nil {
			localStats[domain][dateKey] = &DayStats{Date: dateKey}
		}

		s := localStats[domain][dateKey]
		switch entry.Type {
		case "Reception":
			s.Sent++
		case "Delivery":
			s.Delivered++
		case "Bounce":
			s.Bounced++
		case "TransientFailure":
			s.Deferred++
		}
	}

	mu.Lock()
	for dom, days := range localStats {
		if tempStats[dom] == nil {
			tempStats[dom] = make(map[string]*DayStats)
		}
		for date, stat := range days {
			if tempStats[dom][date] == nil {
				tempStats[dom][date] = &DayStats{Date: date}
			}
			target := tempStats[dom][date]
			target.Sent += stat.Sent
			target.Delivered += stat.Delivered
			target.Bounced += stat.Bounced
			target.Deferred += stat.Deferred
		}
	}
	mu.Unlock()
}

// --- PER-PROVIDER DELIVERY STATS ---

// providerDomainMap maps recipient domains to named ISP providers.
var providerDomainMap = map[string]string{
	// Gmail
	"gmail.com": "Gmail", "googlemail.com": "Gmail",
	// Microsoft / Outlook
	"outlook.com": "Outlook", "hotmail.com": "Outlook", "hotmail.co.uk": "Outlook",
	"hotmail.fr": "Outlook", "hotmail.de": "Outlook", "hotmail.es": "Outlook",
	"hotmail.it": "Outlook", "live.com": "Outlook", "live.co.uk": "Outlook",
	"live.fr": "Outlook", "msn.com": "Outlook", "windowslive.com": "Outlook",
	// Yahoo
	"yahoo.com": "Yahoo", "yahoo.co.uk": "Yahoo", "yahoo.fr": "Yahoo",
	"yahoo.de": "Yahoo", "yahoo.es": "Yahoo", "yahoo.it": "Yahoo",
	"yahoo.ca": "Yahoo", "yahoo.co.jp": "Yahoo", "yahoo.com.au": "Yahoo",
	"ymail.com": "Yahoo", "rocketmail.com": "Yahoo",
	// AOL / Verizon Media
	"aol.com": "AOL", "aol.co.uk": "AOL", "aim.com": "AOL", "verizon.net": "AOL",
	// Apple
	"icloud.com": "Apple", "me.com": "Apple", "mac.com": "Apple",
}

// classifyProvider returns the ISP name for a given recipient domain.
func classifyProvider(recipientDomain string) string {
	if p, ok := providerDomainMap[strings.ToLower(recipientDomain)]; ok {
		return p
	}
	return "Other"
}

// categorizeDeferral maps a SMTP response code + message to a human-readable reason.
func categorizeDeferral(code int, content string) string {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "rate") || strings.Contains(lower, "too many") || strings.Contains(lower, "throttl"):
		return "Rate Limited"
	case strings.Contains(lower, "greylist") || strings.Contains(lower, "grey-list"):
		return "Greylisted"
	case strings.Contains(lower, "spam") || strings.Contains(lower, "reputation") || strings.Contains(lower, "block") || strings.Contains(lower, "reject"):
		return "Reputation Block"
	case strings.Contains(lower, "storage") || strings.Contains(lower, "quota") || strings.Contains(lower, "full") || code == 452:
		return "Mailbox Full"
	case strings.Contains(lower, "connection") || strings.Contains(lower, "timeout"):
		return "Connection Issue"
	case code == 421:
		return "Service Unavailable"
	case code >= 450 && code < 460:
		return "Mailbox Unavailable"
	default:
		if code > 0 {
			return fmt.Sprintf("Code %d", code)
		}
		return "Unknown"
	}
}

// ProviderStats holds aggregated delivery metrics for a single email provider.
type ProviderStats struct {
	Provider        string           `json:"provider"`
	Sent            int64            `json:"sent"`
	Delivered       int64            `json:"delivered"`
	Bounced         int64            `json:"bounced"`
	Deferred        int64            `json:"deferred"`
	DeliveryRate    float64          `json:"delivery_rate"`
	BounceRate      float64          `json:"bounce_rate"`
	DeferralRate    float64          `json:"deferral_rate"`
	DeferralReasons map[string]int64 `json:"deferral_reasons"`
}

// providerAccum is a mutable accumulator used during parallel log parsing.
type providerAccum struct {
	sent            int64
	delivered       int64
	bounced         int64
	deferred        int64
	deferralReasons map[string]int64
}

var (
	providerCacheLock   sync.RWMutex
	cachedProviderStats []ProviderStats
	providerCacheExpiry time.Time
)

// GetProviderStats parses KumoMTA logs and returns delivery stats per email provider.
func GetProviderStats(days int) ([]ProviderStats, error) {
	providerCacheLock.RLock()
	if time.Now().Before(providerCacheExpiry) && cachedProviderStats != nil {
		res := cachedProviderStats
		providerCacheLock.RUnlock()
		return res, nil
	}
	providerCacheLock.RUnlock()

	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*"))
	now := time.Now()
	cutoff := now.AddDate(0, 0, -days)

	var mu sync.Mutex
	tempStats := make(map[string]*providerAccum)
	for _, name := range []string{"Gmail", "Outlook", "Yahoo", "AOL", "Apple", "Other"} {
		tempStats[name] = &providerAccum{deferralReasons: make(map[string]int64)}
	}

	numWorkers := 50
	fileChan := make(chan string, len(files))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				processFileForProviders(file, cutoff, &mu, tempStats)
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

	providerOrder := map[string]int{"Gmail": 0, "Outlook": 1, "Yahoo": 2, "Apple": 3, "AOL": 4, "Other": 5}
	result := make([]ProviderStats, 0, len(tempStats))
	for provider, accum := range tempStats {
		ps := ProviderStats{
			Provider:        provider,
			Sent:            accum.sent,
			Delivered:       accum.delivered,
			Bounced:         accum.bounced,
			Deferred:        accum.deferred,
			DeferralReasons: accum.deferralReasons,
		}
		if ps.Sent > 0 {
			ps.DeliveryRate = float64(ps.Delivered) / float64(ps.Sent) * 100
			ps.BounceRate = float64(ps.Bounced) / float64(ps.Sent) * 100
			ps.DeferralRate = float64(ps.Deferred) / float64(ps.Sent) * 100
		}
		result = append(result, ps)
	}

	sort.Slice(result, func(i, j int) bool {
		oi := providerOrder[result[i].Provider]
		oj := providerOrder[result[j].Provider]
		if oi != oj {
			return oi < oj
		}
		return result[i].Sent > result[j].Sent
	})

	providerCacheLock.Lock()
	cachedProviderStats = result
	providerCacheExpiry = time.Now().Add(CACHE_DURATION)
	providerCacheLock.Unlock()

	return result, nil
}

// processFileForProviders parses a single log file and accumulates per-provider stats.
func processFileForProviders(file string, cutoff time.Time, mu *sync.Mutex, tempStats map[string]*providerAccum) {
	rc, err := openLogFile(file)
	if err != nil {
		return
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	type localEntry struct {
		Type      string    `json:"type"`
		Timestamp time.Time `json:"event_time"`
		Recipient string    `json:"recipient"`
		Response  struct {
			Code    int    `json:"code"`
			Content string `json:"content"`
		} `json:"response"`
	}

	localStats := make(map[string]*providerAccum)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "{"); idx >= 0 {
			line = line[idx:]
		} else {
			continue
		}

		var entry localEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Timestamp.Before(cutoff) {
			continue
		}
		if entry.Recipient == "" {
			continue
		}

		recipientDomain := extractDomain(entry.Recipient)
		if recipientDomain == "" {
			continue
		}

		provider := classifyProvider(recipientDomain)
		if localStats[provider] == nil {
			localStats[provider] = &providerAccum{deferralReasons: make(map[string]int64)}
		}

		acc := localStats[provider]
		switch entry.Type {
		case "Reception":
			acc.sent++
		case "Delivery":
			acc.delivered++
		case "Bounce":
			acc.bounced++
		case "TransientFailure":
			acc.deferred++
			reason := categorizeDeferral(entry.Response.Code, entry.Response.Content)
			acc.deferralReasons[reason]++
		}
	}

	mu.Lock()
	for provider, local := range localStats {
		if tempStats[provider] == nil {
			tempStats[provider] = &providerAccum{deferralReasons: make(map[string]int64)}
		}
		target := tempStats[provider]
		target.sent += local.sent
		target.delivered += local.delivered
		target.bounced += local.bounced
		target.deferred += local.deferred
		for reason, count := range local.deferralReasons {
			target.deferralReasons[reason] += count
		}
	}
	mu.Unlock()
}
