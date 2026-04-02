package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// GET /api/authtools/bimi/{domain}
func (s *Server) handleGetBIMI(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	bimi, err := s.Store.GetBIMI(domain)
	if err == store.ErrNotFound {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domain":     domain,
			"is_enabled": false,
			"logo_url":   "",
			"vmc_url":    "",
			"dns_record": "",
		})
		return
	}
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get BIMI"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         bimi.ID,
		"domain":     bimi.Domain,
		"is_enabled": bimi.IsEnabled,
		"logo_url":   bimi.LogoURL,
		"vmc_url":    bimi.VMCURL,
		"dns_record": buildBIMIDNSRecord(bimi),
	})
}

// POST /api/authtools/bimi/{domain}
func (s *Server) handleSetBIMI(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	var req models.BIMIRecord
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.Domain = domain
	if req.LogoURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "logo_url is required"})
		return
	}
	if !strings.HasPrefix(req.LogoURL, "https://") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "logo_url must use HTTPS"})
		return
	}
	if err := s.Store.UpsertBIMI(&req); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save BIMI"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":     req.Domain,
		"is_enabled": req.IsEnabled,
		"logo_url":   req.LogoURL,
		"vmc_url":    req.VMCURL,
		"dns_record": buildBIMIDNSRecord(&req),
	})
}

// GET /api/authtools/mtasts/{domain}
func (s *Server) handleGetMTASTS(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	p, err := s.Store.GetMTASTS(domain)
	if err == store.ErrNotFound {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domain":      domain,
			"is_enabled":  false,
			"mode":        "testing",
			"max_age":     86400,
			"mx_hosts":    []string{},
			"policy_file": "",
			"dns_record":  "",
		})
		return
	}
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get MTA-STS"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":          p.ID,
		"domain":      p.Domain,
		"is_enabled":  p.IsEnabled,
		"mode":        p.Mode,
		"max_age":     p.MaxAge,
		"mx_hosts":    splitMXHosts(p.MXHosts),
		"policy_file": buildMTASTSPolicyFile(p),
		"dns_record":  buildMTASTSDNSRecord(p),
	})
}

// POST /api/authtools/mtasts/{domain}
func (s *Server) handleSetMTASTS(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	var req models.MTASTSPolicy
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.Domain = domain
	if req.Mode == "" {
		req.Mode = "testing"
	}
	if req.MaxAge == 0 {
		req.MaxAge = 86400
	}
	if err := s.Store.UpsertMTASTS(&req); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save MTA-STS"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":      req.Domain,
		"is_enabled":  req.IsEnabled,
		"mode":        req.Mode,
		"max_age":     req.MaxAge,
		"mx_hosts":    splitMXHosts(req.MXHosts),
		"policy_file": buildMTASTSPolicyFile(&req),
		"dns_record":  buildMTASTSDNSRecord(&req),
	})
}

// GET /api/authtools/check/{domain}
// Returns a comprehensive auth health check for the domain with real DNS lookups
// and deep validation (SPF IP matching, DKIM key details, DMARC policy parsing).
func (s *Server) handleCheckAuthTools(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")

	// Get server IP for SPF validation
	serverIP := ""
	if settings, err := s.Store.GetSettings(); err == nil && settings != nil {
		serverIP = settings.MainServerIP
	}

	// ── SPF — deep validation ──
	spfConfigured := false
	spfRecord := ""
	spfDesc := "No SPF record found"
	spfDetails := make([]map[string]interface{}, 0)
	if txts, err := net.LookupTXT(domain); err == nil {
		for _, txt := range txts {
			if strings.HasPrefix(strings.ToLower(txt), "v=spf1") {
				spfConfigured = true
				spfRecord = txt
				spfDesc = txt
				spfDetails = parseSPFRecord(txt, domain, serverIP)
				break
			}
		}
	}

	// ── DKIM — check common selectors + sender selectors ──
	dkimConfigured := false
	dkimDesc := "No DKIM record found"
	dkimDetails := make([]map[string]interface{}, 0)
	allSelectors := []string{"default", "google", "s1", "s2", "k1", "k2", "dkim", "mail"}

	for _, sel := range allSelectors {
		dkimHost := sel + "._domainkey." + domain
		if txts, err := net.LookupTXT(dkimHost); err == nil {
			for _, txt := range txts {
				if strings.Contains(strings.ToLower(txt), "v=dkim1") || strings.Contains(txt, "p=") {
					dkimConfigured = true
					keyLen := estimateDKIMKeyLength(txt)
					detail := map[string]interface{}{
						"selector": sel,
						"type":     "TXT",
						"record":   txt,
						"key_bits": keyLen,
						"status":   "valid",
					}
					if keyLen > 0 && keyLen < 1024 {
						detail["status"] = "weak"
						detail["warning"] = fmt.Sprintf("%d-bit key is too short, use 2048-bit", keyLen)
					}
					dkimDetails = append(dkimDetails, detail)
					if dkimDesc == "No DKIM record found" {
						dkimDesc = fmt.Sprintf("Found: %s._domainkey (%d-bit)", sel, keyLen)
					}
					break
				}
			}
		}
		// Also check CNAME (KumoMTA uses CNAME for DKIM)
		if cname, err := net.LookupCNAME(dkimHost); err == nil && cname != "" && cname != dkimHost+"." {
			dkimConfigured = true
			dkimDetails = append(dkimDetails, map[string]interface{}{
				"selector": sel,
				"type":     "CNAME",
				"target":   cname,
				"status":   "valid",
			})
			if dkimDesc == "No DKIM record found" {
				dkimDesc = fmt.Sprintf("CNAME: %s._domainkey → %s", sel, cname)
			}
		}
	}

	// ── DMARC — parse policy details ──
	dmarcConfigured := false
	dmarcDesc := "No DMARC record found"
	dmarcDetails := map[string]interface{}{}
	if txts, err := net.LookupTXT("_dmarc." + domain); err == nil {
		for _, txt := range txts {
			if strings.HasPrefix(strings.ToLower(txt), "v=dmarc1") {
				dmarcConfigured = true
				dmarcDesc = txt
				dmarcDetails = parseDMARCRecord(txt)
				break
			}
		}
	}

	// ── MX Records ──
	mxRecords := make([]map[string]interface{}, 0)
	if mxs, err := net.LookupMX(domain); err == nil {
		for _, mx := range mxs {
			host := strings.TrimSuffix(mx.Host, ".")
			ips := make([]string, 0)
			if addrs, err := net.LookupHost(host); err == nil {
				ips = addrs
			}
			mxRecords = append(mxRecords, map[string]interface{}{
				"host":     host,
				"priority": mx.Pref,
				"ips":      ips,
			})
		}
	}

	// ── TLS-RPT ──
	tlsrptConfigured := false
	tlsrptDesc := "No TLS-RPT record found"
	if txts, err := net.LookupTXT("_smtp._tls." + domain); err == nil {
		for _, txt := range txts {
			if strings.Contains(strings.ToLower(txt), "v=tlsrptv1") {
				tlsrptConfigured = true
				tlsrptDesc = txt
				break
			}
		}
	}

	// ── BIMI ──
	bimiEnabled := false
	bimiDesc := "Not configured"
	if b, err := s.Store.GetBIMI(domain); err == nil && b.IsEnabled {
		bimiEnabled = true
		bimiDesc = "Configured: " + b.LogoURL
	}
	if txts, err := net.LookupTXT("default._bimi." + domain); err == nil {
		for _, txt := range txts {
			if strings.Contains(strings.ToLower(txt), "v=bimi1") {
				bimiEnabled = true
				bimiDesc = txt
				break
			}
		}
	}

	// ── MTA-STS ──
	mtastsEnabled := false
	mtastsDesc := "Not configured"
	mtastsMode := ""
	if p, err := s.Store.GetMTASTS(domain); err == nil && p.IsEnabled {
		mtastsEnabled = true
		mtastsMode = p.Mode
		mtastsDesc = fmt.Sprintf("Mode: %s", p.Mode)
	}
	if txts, err := net.LookupTXT("_mta-sts." + domain); err == nil {
		for _, txt := range txts {
			if strings.Contains(strings.ToLower(txt), "v=stsv1") {
				mtastsEnabled = true
				if mtastsDesc == "Not configured" {
					mtastsDesc = txt
				}
				break
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":    domain,
		"server_ip": serverIP,
		"bimi": map[string]interface{}{
			"configured": bimiEnabled,
		},
		"mta_sts": map[string]interface{}{
			"configured": mtastsEnabled,
			"mode":       mtastsMode,
		},
		"mx_records": mxRecords,
		"spf_details": spfDetails,
		"spf_record":  spfRecord,
		"dkim_details": dkimDetails,
		"dmarc_details": dmarcDetails,
		"checklist": []map[string]interface{}{
			{"name": "SPF", "configured": spfConfigured, "description": spfDesc},
			{"name": "DKIM", "configured": dkimConfigured, "description": dkimDesc},
			{"name": "DMARC", "configured": dmarcConfigured, "description": dmarcDesc},
			{"name": "BIMI", "configured": bimiEnabled, "description": bimiDesc},
			{"name": "MTA-STS", "configured": mtastsEnabled, "description": mtastsDesc},
			{"name": "TLS-RPT", "configured": tlsrptConfigured, "description": tlsrptDesc},
		},
	})
}

// parseSPFRecord breaks down an SPF record and validates each mechanism against the server IP.
func parseSPFRecord(record, domain, serverIP string) []map[string]interface{} {
	parts := strings.Fields(record)
	results := make([]map[string]interface{}, 0, len(parts))

	for _, part := range parts {
		if strings.ToLower(part) == "v=spf1" {
			continue
		}

		entry := map[string]interface{}{
			"mechanism": part,
			"match":     false,
		}

		lower := strings.ToLower(part)
		// Strip qualifier prefix
		qualifier := "+"
		mech := lower
		if len(mech) > 0 && (mech[0] == '+' || mech[0] == '-' || mech[0] == '~' || mech[0] == '?') {
			qualifier = string(mech[0])
			mech = mech[1:]
		}
		entry["qualifier"] = qualifier

		switch {
		case mech == "a" || strings.HasPrefix(mech, "a:"):
			host := domain
			if strings.HasPrefix(mech, "a:") {
				host = mech[2:]
			}
			ips, _ := net.LookupHost(host)
			entry["resolved_ips"] = ips
			entry["explanation"] = fmt.Sprintf("Allow IPs of %s → %s", host, strings.Join(ips, ", "))
			if serverIP != "" {
				for _, ip := range ips {
					if ip == serverIP {
						entry["match"] = true
					}
				}
			}

		case mech == "mx" || strings.HasPrefix(mech, "mx:"):
			host := domain
			if strings.HasPrefix(mech, "mx:") {
				host = mech[3:]
			}
			mxIPs := make([]string, 0)
			if mxs, err := net.LookupMX(host); err == nil {
				for _, mx := range mxs {
					if addrs, err := net.LookupHost(strings.TrimSuffix(mx.Host, ".")); err == nil {
						mxIPs = append(mxIPs, addrs...)
					}
				}
			}
			entry["resolved_ips"] = mxIPs
			entry["explanation"] = fmt.Sprintf("Allow MX servers of %s → %s", host, strings.Join(mxIPs, ", "))
			if serverIP != "" {
				for _, ip := range mxIPs {
					if ip == serverIP {
						entry["match"] = true
					}
				}
			}

		case strings.HasPrefix(mech, "ip4:") || strings.HasPrefix(mech, "ip6:"):
			cidr := mech[4:]
			entry["explanation"] = fmt.Sprintf("Allow IP/range %s", cidr)
			if serverIP != "" {
				if strings.Contains(cidr, "/") {
					if _, ipNet, err := net.ParseCIDR(cidr); err == nil {
						if ipNet.Contains(net.ParseIP(serverIP)) {
							entry["match"] = true
						}
					}
				} else if cidr == serverIP {
					entry["match"] = true
				}
			}

		case strings.HasPrefix(mech, "include:"):
			included := mech[8:]
			entry["explanation"] = fmt.Sprintf("Include SPF of %s", included)
			// Check if the include resolves
			if txts, err := net.LookupTXT(included); err == nil {
				for _, txt := range txts {
					if strings.HasPrefix(strings.ToLower(txt), "v=spf1") {
						entry["included_record"] = txt
						break
					}
				}
			}

		case strings.HasPrefix(mech, "redirect="):
			target := mech[9:]
			entry["explanation"] = fmt.Sprintf("Redirect to SPF of %s", target)

		case mech == "all":
			qualLabels := map[string]string{"+": "pass (allow all)", "-": "fail (reject)", "~": "softfail", "?": "neutral"}
			entry["explanation"] = fmt.Sprintf("Default policy: %s", qualLabels[qualifier])

		default:
			entry["explanation"] = part
		}

		results = append(results, entry)
	}
	return results
}

// parseDMARCRecord extracts policy fields from a DMARC record.
func parseDMARCRecord(record string) map[string]interface{} {
	result := map[string]interface{}{
		"raw": record,
	}
	for _, part := range strings.Split(record, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "p":
			result["policy"] = val
			labels := map[string]string{"none": "Monitor only", "quarantine": "Send to spam", "reject": "Reject failures"}
			result["policy_label"] = labels[val]
		case "sp":
			result["subdomain_policy"] = val
		case "rua":
			result["aggregate_reports"] = val
		case "ruf":
			result["forensic_reports"] = val
		case "pct":
			result["percentage"] = val
		case "adkim":
			align := map[string]string{"r": "Relaxed", "s": "Strict"}
			result["dkim_alignment"] = align[val]
		case "aspf":
			align := map[string]string{"r": "Relaxed", "s": "Strict"}
			result["spf_alignment"] = align[val]
		case "fo":
			result["failure_options"] = val
		}
	}
	return result
}

// estimateDKIMKeyLength guesses key length from base64-encoded public key in a DKIM TXT record.
func estimateDKIMKeyLength(record string) int {
	for _, part := range strings.Split(record, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "p=") {
			b64 := strings.TrimPrefix(part, "p=")
			b64 = strings.ReplaceAll(b64, " ", "")
			// Base64 bytes ≈ raw bytes × 4/3; RSA key length ≈ raw bytes × 8
			rawBytes := len(b64) * 3 / 4
			bits := rawBytes * 8
			// Round to nearest standard key size
			if bits >= 3800 {
				return 4096
			}
			if bits >= 1800 {
				return 2048
			}
			if bits >= 900 {
				return 1024
			}
			if bits >= 400 {
				return 512
			}
			return bits
		}
	}
	return 0
}

// buildBIMIDNSRecord generates the DNS TXT record value for BIMI
func buildBIMIDNSRecord(b *models.BIMIRecord) string {
	if b == nil || !b.IsEnabled {
		return ""
	}
	record := fmt.Sprintf("v=BIMI1; l=%s", b.LogoURL)
	if b.VMCURL != "" {
		record += fmt.Sprintf("; a=%s", b.VMCURL)
	}
	return fmt.Sprintf("Name: default._bimi.%s\nType: TXT\nValue: %s", b.Domain, record)
}

// buildMTASTSPolicyFile generates the .well-known/mta-sts.txt policy file content
func buildMTASTSPolicyFile(p *models.MTASTSPolicy) string {
	if p == nil {
		return ""
	}
	lines := []string{
		"version: STSv1",
		fmt.Sprintf("mode: %s", p.Mode),
	}
	for _, mx := range strings.Split(p.MXHosts, "\n") {
		mx = strings.TrimSpace(mx)
		if mx != "" {
			lines = append(lines, fmt.Sprintf("mx: %s", mx))
		}
	}
	lines = append(lines, fmt.Sprintf("max_age: %d", p.MaxAge))
	return strings.Join(lines, "\n")
}

// splitMXHosts converts newline-separated MXHosts string to a slice
func splitMXHosts(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, "\n")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}

// buildMTASTSDNSRecord generates the DNS TXT record for MTA-STS
func buildMTASTSDNSRecord(p *models.MTASTSPolicy) string {
	if p == nil || !p.IsEnabled {
		return ""
	}
	policyID := fmt.Sprintf("%d", p.UpdatedAt.Unix())
	return fmt.Sprintf("Name: _mta-sts.%s\nType: TXT\nValue: v=STSv1; id=%s\n\nAlso needed:\nName: _smtp._tls.%s\nType: TXT\nValue: v=TLSRPTv1; rua=mailto:tls-rpt@%s",
		p.Domain, policyID, p.Domain, p.Domain)
}
