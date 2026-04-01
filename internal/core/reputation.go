package core

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// RBLEntry holds a blacklist zone with its severity level.
type RBLEntry struct {
	Zone     string
	Severity int // 3=critical, 2=high, 1=medium
}

// ipRBLs are DNS-based blacklists queried by reversing the IP octets.
// Note: dnsbl.sorbs.net (shut down June 2024) and ix.dnsbl.manitu.net (shut down Jan 2025) removed.
var ipRBLs = []RBLEntry{
	{"zen.spamhaus.org", 3},
	{"b.barracudacentral.org", 2},
	{"bl.spamcop.net", 3},
	{"truncate.gbudb.net", 2},
	{"dnsbl-1.uceprotect.net", 1},
	{"psbl.surriel.com", 1},
}

// domainRBLs are DNS-based blacklists queried with the domain prepended.
// Note: rhsbl.sorbs.net (shut down June 2024) removed.
var domainRBLs = []RBLEntry{
	{"dbl.spamhaus.org", 3},
	{"multi.surbl.org", 3},
	{"black.uribl.com", 3},
	{"dbl.nordspam.com", 1},
}

// DelistURLs maps RBL zone names to their delist/removal pages.
var DelistURLs = map[string]string{
	"zen.spamhaus.org":        "https://check.spamhaus.org/",
	"dbl.spamhaus.org":        "https://check.spamhaus.org/",
	"b.barracudacentral.org":  "https://www.barracudacentral.org/rbl/removal-request",
	"bl.spamcop.net":          "https://www.spamcop.net/bl.shtml",
	"truncate.gbudb.net":      "http://www.gbudb.com/truncate/",
	"dnsbl-1.uceprotect.net":  "https://www.uceprotect.net/en/index.php?m=7&s=0",
	"psbl.surriel.com":        "https://psbl.org/remove",
	"multi.surbl.org":         "https://www.surbl.org/surbl-analysis",
	"black.uribl.com":         "https://admin.uribl.com/",
	"dbl.nordspam.com":        "https://www.nordspam.com/delist/",
}

// rblResolver uses Go's built-in DNS client (PreferGo: true) to send queries
// directly from the server's own IP, avoiding public resolver false positives.
var rblResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 5 * time.Second}
		return d.DialContext(ctx, "udp", address)
	},
}

// RBLResult holds the result of a single DNSBL lookup.
type RBLResult struct {
	Zone     string `json:"zone"`
	Listed   bool   `json:"listed"`
	Severity int    `json:"severity"`
	Response string `json:"response,omitempty"` // the A record IP (e.g. 127.0.0.2)
	TXT      string `json:"txt,omitempty"`      // TXT record (listing reason/URL)
}

// isRealListing checks if a DNSBL A record response is a genuine listing.
//
// Response codes:
//
//	127.0.0.2 - 127.0.0.255 = real listing (varies by RBL)
//	127.0.0.1              = not listed / test response
//	127.255.255.252        = rate limited (Spamhaus)
//	127.255.255.254        = public resolver blocked (Spamhaus)
//	127.255.255.255        = typing error test (Spamhaus)
func isRealListing(addr string) bool {
	parts := strings.Split(addr, ".")
	if len(parts) != 4 || parts[0] != "127" {
		return false
	}
	// Only accept 127.0.0.x where x >= 2
	if parts[1] != "0" || parts[2] != "0" {
		return false
	}
	octet4 := 0
	fmt.Sscanf(parts[3], "%d", &octet4)
	return octet4 >= 2 && octet4 <= 255
}

// dnsblQuery performs a DNSBL lookup and returns the A record + TXT record.
// This matches the approach used in SpamCheckService.php's dnsBlacklistQuery().
func dnsblQuery(fqdn string) RBLResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := RBLResult{Zone: fqdn}

	// Step 1: DNS A record lookup
	addrs, err := rblResolver.LookupHost(ctx, fqdn)
	if err != nil || len(addrs) == 0 {
		return result // not listed
	}

	// Check each returned address
	for _, addr := range addrs {
		if isRealListing(addr) {
			result.Listed = true
			result.Response = addr
			break
		}
	}

	if !result.Listed {
		// Got a response but it's not a real listing (e.g. 127.255.255.254)
		if len(addrs) > 0 {
			result.Response = strings.Join(addrs, ",") // log for debugging
		}
		return result
	}

	// Step 2: DNS TXT record lookup (get listing reason/URL)
	txtRecords, err := rblResolver.LookupTXT(ctx, fqdn)
	if err == nil && len(txtRecords) > 0 {
		result.TXT = txtRecords[0]
	}

	return result
}

// checkIPAgainstRBL checks an IP against all IP-based RBLs.
func checkIPAgainstRBL(ip string) []RBLResult {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return nil
	}
	reversed := fmt.Sprintf("%s.%s.%s.%s", parts[3], parts[2], parts[1], parts[0])

	var mu sync.Mutex
	var results []RBLResult
	var wg sync.WaitGroup

	for _, rbl := range ipRBLs {
		wg.Add(1)
		go func(rbl RBLEntry) {
			defer wg.Done()
			fqdn := reversed + "." + rbl.Zone
			r := dnsblQuery(fqdn)
			r.Zone = rbl.Zone
			r.Severity = rbl.Severity
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(rbl)
	}
	wg.Wait()
	return results
}

// checkDomainAgainstRBL checks a domain against all domain-based RBLs.
func checkDomainAgainstRBL(domain string) []RBLResult {
	var mu sync.Mutex
	var results []RBLResult
	var wg sync.WaitGroup

	for _, rbl := range domainRBLs {
		wg.Add(1)
		go func(rbl RBLEntry) {
			defer wg.Done()
			fqdn := domain + "." + rbl.Zone
			r := dnsblQuery(fqdn)
			r.Zone = rbl.Zone
			r.Severity = rbl.Severity
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(rbl)
	}
	wg.Wait()
	return results
}

// CheckReputation checks all system IPs and configured domains, saves results to DB,
// and returns the full list of results.
func CheckReputation(st *store.Store) ([]models.ReputationCheck, error) {
	ips, err := st.ListSystemIPs()
	if err != nil {
		return nil, err
	}
	domains, err := st.ListDomains()
	if err != nil {
		return nil, err
	}

	// Purge stale DB entries for targets that no longer exist
	var activeIPs []string
	for _, ip := range ips {
		activeIPs = append(activeIPs, ip.Value)
	}
	var activeDomains []string
	for _, d := range domains {
		activeDomains = append(activeDomains, d.Name)
	}
	_ = st.PurgeStaleReputationTargets(activeIPs, activeDomains)

	var targets []struct {
		value      string
		targetType string
	}
	for _, ip := range ips {
		targets = append(targets, struct {
			value      string
			targetType string
		}{ip.Value, "ip"})
	}
	for _, d := range domains {
		targets = append(targets, struct {
			value      string
			targetType string
		}{d.Name, "domain"})
	}

	var mu sync.Mutex
	var results []models.ReputationCheck
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(value, targetType string) {
			defer wg.Done()

			var rblResults []RBLResult
			if targetType == "ip" {
				rblResults = checkIPAgainstRBL(value)
			} else {
				rblResults = checkDomainAgainstRBL(value)
			}

			var listedOn []string
			var listedDetails []string
			maxSeverity := 0

			for _, r := range rblResults {
				if r.Response != "" {
					log.Printf("[reputation] %s %s → %s = %s (listed=%v txt=%s)",
						targetType, value, r.Zone, r.Response, r.Listed, r.TXT)
				}
				if r.Listed {
					listedOn = append(listedOn, r.Zone)
					detail := r.Zone
					if r.TXT != "" {
						detail += ": " + r.TXT
					}
					listedDetails = append(listedDetails, detail)
					if r.Severity > maxSeverity {
						maxSeverity = r.Severity
					}
				}
			}

			status := "clean"
			listed := ""
			if len(listedOn) > 0 {
				status = "listed"
				listed = strings.Join(listedOn, ",")
			}

			rc := &models.ReputationCheck{
				Target:     value,
				TargetType: targetType,
				Status:     status,
				ListedOn:   listed,
			}
			_ = st.SaveReputationCheck(rc)

			mu.Lock()
			results = append(results, *rc)
			mu.Unlock()
		}(t.value, t.targetType)
	}
	wg.Wait()

	return results, nil
}
