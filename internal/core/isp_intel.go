package core

// ISP Intelligence Engine
//
// Aggregates per-ISP delivery intelligence from three sources:
//   1. Local delivery data  — DeliveryEvent + EmailStats (always available)
//   2. Google Postmaster Tools — domain/IP reputation, spam rate (optional, OAuth2)
//   3. Microsoft SNDS        — per-IP filter result, trap rate (optional, API key)
//
// Data is refreshed on demand (manual) or on the hourly scheduler tick.
// Results are stored as ISPSnapshot records for trending and dashboard display.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// ISPIntelService collects and stores ISP intelligence snapshots.
type ISPIntelService struct {
	st     *store.Store
	client *http.Client
}

// NewISPIntelService creates a new ISPIntelService.
func NewISPIntelService(st *store.Store) *ISPIntelService {
	return &ISPIntelService{
		st:     st,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// RefreshAll collects a fresh ISP snapshot for every sending domain.
// Runs local metrics always; Postmaster/SNDS only if credentials are configured.
func (svc *ISPIntelService) RefreshAll() {
	settings, err := svc.st.GetSettings()
	if err != nil {
		log.Println("[ISP] cannot load settings:", err)
		return
	}

	domains, err := svc.st.ListDomains()
	if err != nil {
		log.Println("[ISP] cannot list domains:", err)
		return
	}

	now := time.Now().Truncate(time.Hour) // hourly granularity

	// For each sending domain, build snapshots per ISP
	for _, domain := range domains {
		svc.refreshDomain(domain.Name, now, settings)
	}
}

// refreshDomain collects and stores ISP snapshots for one sending domain.
func (svc *ISPIntelService) refreshDomain(domain string, at time.Time, settings *models.AppSettings) {
	// --- 1. Aggregate local delivery metrics per ISP ---
	localMetrics, err := svc.st.GetISPMetricsForDomain(domain, at.Add(-24*time.Hour), at)
	if err != nil {
		log.Printf("[ISP] local metrics for %s: %v", domain, err)
		return
	}

	// --- 2. FBL complaint rates ---
	fblCounts, _ := svc.st.GetFBLCountsByISP(domain, at.Add(-24*time.Hour))

	// --- 3. Google Postmaster Tools (optional) ---
	gptData := map[string]*GPTDomainData{}
	if settings.GooglePostmasterEnabled && settings.GooglePostmasterCredentials != "" {
		creds, err := Decrypt(settings.GooglePostmasterCredentials)
		if err == nil && creds != "" {
			gptData, _ = svc.fetchGooglePostmaster(domain, creds)
		}
	}

	// --- 4. Microsoft SNDS (optional) ---
	sndsData := map[string]*SNDSData{}
	if settings.MicrosoftSNDSKey != "" {
		key, err := Decrypt(settings.MicrosoftSNDSKey)
		if err == nil && key != "" {
			// Get sending IPs for this domain from sender records
			senders, _ := svc.st.ListSendersByDomainName(domain)
			for _, sender := range senders {
				if sender.IP != "" {
					if data, err := svc.fetchSNDS(sender.IP, key); err == nil {
						sndsData[sender.IP] = data
					}
				}
			}
		}
	}

	// Build one snapshot per ISP based on local metrics
	ispNames := []string{"Gmail", "Yahoo", "Outlook", "AOL", "Apple", "Other"}
	for _, isp := range ispNames {
		m := localMetrics[isp]

		snap := &models.ISPSnapshot{
			ISP:        isp,
			Domain:     domain,
			CapturedAt: at,
		}

		// Local metrics
		if m != nil {
			snap.TotalSent = m.Sent
			snap.TotalDelivered = m.Delivered
			snap.TotalBounced = m.Bounced
			snap.TotalDeferred = m.Deferred
			total := m.Delivered + m.Bounced
			if total > 0 {
				snap.AcceptanceRate = float64(m.Delivered) / float64(total) * 100
				snap.BounceRate = float64(m.Bounced) / float64(total) * 100
			}
			if snap.TotalSent > 0 {
				snap.DeferralRate = float64(m.Deferred) / float64(snap.TotalSent) * 100
			}
		}

		// Complaint rate from FBL
		if sent := snap.TotalSent; sent > 0 {
			if complaints, ok := fblCounts[isp]; ok && complaints > 0 {
				snap.ComplaintRate = float64(complaints) / float64(sent) * 100
			}
		}

		// Google Postmaster
		if gd, ok := gptData[domain]; ok {
			snap.GPTDomainReputation = gd.DomainReputation
			snap.GPTIPReputation = gd.IPReputation
			snap.GPTSpamRate = gd.SpamRate
			snap.GPTDeliveryErrors = gd.DeliveryErrors
			snap.GPTEnabled = true
		}

		// SNDS — aggregate across all IPs for this domain
		if len(sndsData) > 0 {
			snap.SNDSEnabled = true
			var trapRates []float64
			results := []string{}
			for _, sd := range sndsData {
				trapRates = append(trapRates, sd.TrapRate)
				results = append(results, sd.FilterResult)
			}
			if len(trapRates) > 0 {
				var sum float64
				for _, r := range trapRates {
					sum += r
				}
				snap.SNDSTrapRate = sum / float64(len(trapRates))
			}
			snap.SNDSFilterResult = worstSNDSResult(results)
		}

		snap.HealthScore = computeHealthScore(snap)

		if err := svc.st.UpsertISPSnapshot(snap); err != nil {
			log.Printf("[ISP] store snapshot %s/%s: %v", domain, isp, err)
		}
	}
	log.Printf("[ISP] Refreshed snapshots for domain: %s", domain)
}

// ─────────────────────────────────────────────
// Google Postmaster Tools API client
// ─────────────────────────────────────────────

// GPTDomainData holds data fetched from Google Postmaster Tools for one domain.
type GPTDomainData struct {
	DomainReputation string
	IPReputation     string
	SpamRate         float64
	DeliveryErrors   int64
}

// fetchGooglePostmaster queries the Google Postmaster Tools API for the given domain.
// credentialsJSON is the raw content of a Google service account JSON file.
// Returns a map keyed by domain name (usually just one entry for the queried domain).
func (svc *ISPIntelService) fetchGooglePostmaster(domain, credentialsJSON string) (map[string]*GPTDomainData, error) {
	token, err := getGoogleAccessToken(credentialsJSON, svc.client)
	if err != nil {
		return nil, fmt.Errorf("google postmaster auth: %w", err)
	}

	// Date: yesterday in YYYYMMDD
	yesterday := time.Now().AddDate(0, 0, -1)
	dateStr := yesterday.Format("20060102")
	resourceName := fmt.Sprintf("domains/%s", domain)
	url := fmt.Sprintf("https://gmailpostmastertools.googleapis.com/v1/%s/domainReputations?startDate.year=%s&startDate.month=%s&startDate.day=%s",
		resourceName,
		yesterday.Format("2006"),
		fmt.Sprintf("%d", int(yesterday.Month())),
		yesterday.Format("02"),
	)
	_ = dateStr

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := svc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("postmaster tools API %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		DomainReputations []struct {
			Name             string  `json:"name"`
			Date             struct{ Year, Month, Day int } `json:"date"`
			DomainReputation string  `json:"domainReputation"`
			SpamRate         float64 `json:"spamRateInRange"`
		} `json:"domainReputations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	out := map[string]*GPTDomainData{}
	for _, dr := range result.DomainReputations {
		out[domain] = &GPTDomainData{
			DomainReputation: dr.DomainReputation,
			SpamRate:         dr.SpamRate,
		}
	}
	return out, nil
}

// getGoogleAccessToken exchanges a service account JSON credential for a short-lived access token
// using the OAuth2 JWT flow (google.golang.org/api not required — pure stdlib).
func getGoogleAccessToken(serviceAccountJSON string, client *http.Client) (string, error) {
	var sa struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		TokenURI    string `json:"token_uri"`
	}
	if err := json.Unmarshal([]byte(serviceAccountJSON), &sa); err != nil {
		return "", fmt.Errorf("parse service account: %w", err)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}

	// Build JWT
	now := time.Now().Unix()
	scope := "https://www.googleapis.com/auth/postmaster.readonly"
	claimsJSON, _ := json.Marshal(map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": scope,
		"aud":   sa.TokenURI,
		"exp":   now + 3600,
		"iat":   now,
	})

	jwt, err := buildGoogleJWT(sa.PrivateKey, claimsJSON)
	if err != nil {
		return "", err
	}

	// Exchange JWT for access token
	resp, err := client.PostForm(sa.TokenURI, map[string][]string{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.Error != "" {
		return "", fmt.Errorf("token exchange: %s", tok.Error)
	}
	return tok.AccessToken, nil
}

// ─────────────────────────────────────────────
// Microsoft SNDS client
// ─────────────────────────────────────────────

// SNDSData holds data from Microsoft's Smart Network Data Services for one IP.
type SNDSData struct {
	IP           string
	FilterResult string  // "GREEN", "YELLOW", "RED"
	TrapRate     float64 // fraction 0.0–1.0
	TrapCount    int64
	MessageCount int64
}

// fetchSNDS queries the Microsoft SNDS API for a single IP address.
// apiKey is the SNDS access key obtained from https://sendersupport.olc.protection.outlook.com/snds/
func (svc *ISPIntelService) fetchSNDS(ip, apiKey string) (*SNDSData, error) {
	url := fmt.Sprintf("https://sendersupport.olc.protection.outlook.com/snds/data.aspx?key=%s&ip=%s", apiKey, ip)
	resp, err := svc.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// SNDS returns CSV: IP, ActivityStart, ActivityEnd, SpamRate, TrapMessageCount, TrapHitsCount, FilterResult, RCPT, DATA, MessageCount
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty SNDS response")
	}

	data := &SNDSData{IP: ip}
	// Parse last line (most recent data point)
	last := lines[len(lines)-1]
	fields := strings.Split(last, ",")
	if len(fields) < 7 {
		return nil, fmt.Errorf("unexpected SNDS CSV format: %q", last)
	}

	data.FilterResult = strings.ToUpper(strings.TrimSpace(fields[6]))

	// Parse trap rate from fields[3] which is a percentage string like "0.00%"
	trapStr := strings.TrimSuffix(strings.TrimSpace(fields[3]), "%")
	var trapPct float64
	fmt.Sscanf(trapStr, "%f", &trapPct)
	data.TrapRate = trapPct / 100.0

	// Message count is fields[9] if present
	if len(fields) > 9 {
		fmt.Sscanf(strings.TrimSpace(fields[9]), "%d", &data.MessageCount)
	}
	return data, nil
}

// ─────────────────────────────────────────────
// Health scoring
// ─────────────────────────────────────────────

// computeHealthScore returns a 0–100 health score for an ISP snapshot.
// 100 = perfect delivery, 0 = completely blocked.
func computeHealthScore(s *models.ISPSnapshot) int {
	score := 100.0

	// Acceptance rate (max -40 points)
	if s.AcceptanceRate < 95 {
		score -= (95 - s.AcceptanceRate) * 2 // each % below 95 costs 2 points
	}

	// Bounce rate (max -20 points)
	if s.BounceRate > 2 {
		score -= (s.BounceRate - 2) * 4
	}

	// Deferral rate (max -20 points)
	if s.DeferralRate > 5 {
		score -= (s.DeferralRate - 5) * 2
	}

	// Complaint rate (max -30 points — weighted heavily)
	if s.ComplaintRate > 0.05 {
		score -= (s.ComplaintRate - 0.05) * 50
	}

	// Google Postmaster reputation penalty
	switch s.GPTDomainReputation {
	case "LOW":
		score -= 15
	case "BAD":
		score -= 35
	}

	// SNDS filter result penalty
	switch s.SNDSFilterResult {
	case "YELLOW":
		score -= 10
	case "RED":
		score -= 30
	}

	if score < 0 {
		score = 0
	}
	return int(score)
}

// worstSNDSResult returns the most severe result from a list of SNDS filter results.
func worstSNDSResult(results []string) string {
	worst := "GREEN"
	for _, r := range results {
		switch r {
		case "RED":
			return "RED"
		case "YELLOW":
			worst = "YELLOW"
		}
	}
	return worst
}
