package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// HourStats holds aggregated counts for a single hour bucket.
type HourStats struct {
	Hour      string `json:"hour"`      // display label e.g. "03-06 14:00"
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	Bounced   int64  `json:"bounced"`
	Deferred  int64  `json:"deferred"`
}

// GetHourlyStats parses KumoMTA logs and returns per-hour aggregated stats
// for the last `hours` hours (1–168).
func GetHourlyStats(hours int) ([]HourStats, error) {
	if hours < 1 || hours > 168 {
		hours = 24
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*"))

	var mu sync.Mutex
	tempStats := make(map[string]*HourStats)

	numWorkers := 20
	fileChan := make(chan string, len(files)+1)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				processFileHourly(file, cutoff, &mu, tempStats)
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

	// Build full result covering every hour in the window (fill zeros for missing hours)
	now := time.Now()
	result := make([]HourStats, 0, hours)
	for i := hours - 1; i >= 0; i-- {
		h := now.Add(-time.Duration(i) * time.Hour).Truncate(time.Hour)
		key := h.Format("2006-01-02T15")
		label := h.Format("01-02 15:00")
		if s, ok := tempStats[key]; ok {
			s.Hour = label
			result = append(result, *s)
		} else {
			result = append(result, HourStats{Hour: label})
		}
	}
	return result, nil
}

func processFileHourly(file string, cutoff time.Time, mu *sync.Mutex, tempStats map[string]*HourStats) {
	rc, err := openLogFile(file)
	if err != nil {
		return
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	type entry struct {
		Type      string    `json:"type"`
		Timestamp time.Time `json:"event_time"`
	}

	localStats := make(map[string]*HourStats)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "{"); idx >= 0 {
			line = line[idx:]
		} else {
			continue
		}

		var e entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Timestamp.Before(cutoff) {
			continue
		}

		key := e.Timestamp.Truncate(time.Hour).Format("2006-01-02T15")
		if localStats[key] == nil {
			localStats[key] = &HourStats{Hour: key}
		}
		s := localStats[key]
		switch e.Type {
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
	for key, local := range localStats {
		if tempStats[key] == nil {
			tempStats[key] = &HourStats{Hour: key}
		}
		t := tempStats[key]
		t.Sent += local.Sent
		t.Delivered += local.Delivered
		t.Bounced += local.Bounced
		t.Deferred += local.Deferred
	}
	mu.Unlock()
}

// ParseDeliveryEvents reads KumoMTA logs for the last hoursBack hours,
// extracts per-recipient Bounce and TransientFailure events, and saves them
// to the DeliveryEvent table (replacing any existing records for that window).
func ParseDeliveryEvents(st *store.Store, hoursBack int) error {
	if hoursBack < 1 || hoursBack > 168 {
		hoursBack = 72
	}

	cutoff := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	files, _ := filepath.Glob(filepath.Join(KumoLogDir, "*"))

	var mu sync.Mutex
	var events []models.DeliveryEvent

	numWorkers := 20
	fileChan := make(chan string, len(files)+1)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				local := parseFileForDeliveryEvents(file, cutoff)
				if len(local) > 0 {
					mu.Lock()
					events = append(events, local...)
					mu.Unlock()
				}
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

	// Prune stale events beyond our window before inserting fresh data
	if err := st.PruneDeliveryEvents(hoursBack + 24); err != nil {
		fmt.Printf("[delivery_events] prune warning: %v\n", err)
	}

	return st.BulkInsertDeliveryEvents(events, cutoff)
}

func parseFileForDeliveryEvents(file string, cutoff time.Time) []models.DeliveryEvent {
	rc, err := openLogFile(file)
	if err != nil {
		return nil
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	type entry struct {
		Type      string    `json:"type"`
		Timestamp time.Time `json:"event_time"`
		Sender    string    `json:"sender"`
		Recipient string    `json:"recipient"`
		Response  struct {
			Code    int    `json:"code"`
			Content string `json:"content"`
		} `json:"response"`
	}

	var results []models.DeliveryEvent

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "{"); idx >= 0 {
			line = line[idx:]
		} else {
			continue
		}

		var e entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Timestamp.Before(cutoff) {
			continue
		}
		// Only track failure events
		if e.Type != "Bounce" && e.Type != "TransientFailure" {
			continue
		}
		if e.Recipient == "" {
			continue
		}

		domain := extractDomain(e.Recipient)
		provider := classifyProvider(domain)

		results = append(results, models.DeliveryEvent{
			Timestamp: e.Timestamp,
			EventType: e.Type,
			Sender:    e.Sender,
			Recipient: e.Recipient,
			Domain:    domain,
			ErrorCode: e.Response.Code,
			ErrorMsg:  e.Response.Content,
			Provider:  provider,
		})
	}

	return results
}
