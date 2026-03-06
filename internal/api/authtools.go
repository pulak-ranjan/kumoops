package api

import (
	"encoding/json"
	"fmt"
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
			"mx_hosts":    "",
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
		"mx_hosts":    p.MXHosts,
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
		"mx_hosts":    req.MXHosts,
		"policy_file": buildMTASTSPolicyFile(&req),
		"dns_record":  buildMTASTSDNSRecord(&req),
	})
}

// GET /api/authtools/check/{domain}
// Returns a comprehensive auth health check for the domain
func (s *Server) handleCheckAuthTools(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")

	bimiEnabled := false
	bimiLogoURL := ""
	if b, err := s.Store.GetBIMI(domain); err == nil {
		bimiEnabled = b.IsEnabled
		bimiLogoURL = b.LogoURL
	}

	mtastsEnabled := false
	mtastsMode := ""
	if p, err := s.Store.GetMTASTS(domain); err == nil {
		mtastsEnabled = p.IsEnabled
		mtastsMode = p.Mode
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain": domain,
		"bimi": map[string]interface{}{
			"configured": bimiEnabled,
			"logo_url":   bimiLogoURL,
		},
		"mta_sts": map[string]interface{}{
			"configured": mtastsEnabled,
			"mode":       mtastsMode,
		},
		"checklist": []map[string]interface{}{
			{"name": "SPF", "description": "Publish SPF TXT record at DNS"},
			{"name": "DKIM", "description": "Generate DKIM keys and publish CNAME/TXT"},
			{"name": "DMARC", "description": "Set DMARC policy (p=quarantine or p=reject)"},
			{"name": "BIMI", "description": "Publish BIMI record with brand logo", "configured": bimiEnabled},
			{"name": "MTA-STS", "description": "Publish MTA-STS policy for TLS enforcement", "configured": mtastsEnabled},
			{"name": "TLS-RPT", "description": "Add SMTP TLS reporting DNS record"},
		},
	})
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

// buildMTASTSDNSRecord generates the DNS TXT record for MTA-STS
func buildMTASTSDNSRecord(p *models.MTASTSPolicy) string {
	if p == nil || !p.IsEnabled {
		return ""
	}
	policyID := fmt.Sprintf("%d", p.UpdatedAt.Unix())
	return fmt.Sprintf("Name: _mta-sts.%s\nType: TXT\nValue: v=STSv1; id=%s\n\nAlso needed:\nName: _smtp._tls.%s\nType: TXT\nValue: v=TLSRPTv1; rua=mailto:tls-rpt@%s",
		p.Domain, policyID, p.Domain, p.Domain)
}
