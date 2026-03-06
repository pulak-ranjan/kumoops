package core

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// ipRBLs are DNS-based blacklists queried by reversing the IP octets.
var ipRBLs = []string{
	"zen.spamhaus.org",
	"b.barracudacentral.org",
	"bl.spamcop.net",
	"dnsbl.sorbs.net",
	"ix.dnsbl.manitu.net",
	"truncate.gbudb.net",
	"dnsbl-1.uceprotect.net",
	"psbl.surriel.com",
}

// domainRBLs are DNS-based blacklists queried with the domain prepended.
var domainRBLs = []string{
	"dbl.spamhaus.org",
	"multi.uribl.com",
	"black.uribl.com",
	"rhsbl.sorbs.net",
	"dbl.nordspam.com",
}

// checkIPAgainstRBL returns the list of RBL names the IP is listed on.
func checkIPAgainstRBL(ip string) []string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return nil
	}
	reversed := fmt.Sprintf("%s.%s.%s.%s", parts[3], parts[2], parts[1], parts[0])

	var mu sync.Mutex
	var listed []string
	var wg sync.WaitGroup

	for _, rbl := range ipRBLs {
		wg.Add(1)
		go func(rbl string) {
			defer wg.Done()
			lookup := reversed + "." + rbl
			addrs, err := net.LookupHost(lookup)
			if err == nil && len(addrs) > 0 {
				mu.Lock()
				listed = append(listed, rbl)
				mu.Unlock()
			}
		}(rbl)
	}
	wg.Wait()
	return listed
}

// checkDomainAgainstRBL returns the list of RBL names the domain is listed on.
func checkDomainAgainstRBL(domain string) []string {
	var mu sync.Mutex
	var listed []string
	var wg sync.WaitGroup

	for _, rbl := range domainRBLs {
		wg.Add(1)
		go func(rbl string) {
			defer wg.Done()
			lookup := domain + "." + rbl
			addrs, err := net.LookupHost(lookup)
			if err == nil && len(addrs) > 0 {
				mu.Lock()
				listed = append(listed, rbl)
				mu.Unlock()
			}
		}(rbl)
	}
	wg.Wait()
	return listed
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

			var listedOn []string
			if targetType == "ip" {
				listedOn = checkIPAgainstRBL(value)
			} else {
				listedOn = checkDomainAgainstRBL(value)
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
