package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// ─────────────────────────────────────────────
// FBL / Complaint Records
// ─────────────────────────────────────────────

// GET /api/fbl?domain=&type=&sender=&days=30&limit=200
func (s *Server) handleListFBLRecords(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	feedbackType := r.URL.Query().Get("type")
	sender := r.URL.Query().Get("sender")

	days := 30
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	limit := 200
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 1000 {
		limit = l
	}

	since := time.Now().AddDate(0, 0, -days)
	records, err := s.Store.ListFBLRecords(domain, feedbackType, sender, since, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// GET /api/fbl/stats?days=30
func (s *Server) handleGetFBLStats(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	since := time.Now().AddDate(0, 0, -days)

	stats, err := s.Store.GetFBLStats(since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// DELETE /api/fbl/{id}
func (s *Server) handleDeleteFBLRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.Store.DeleteFBLRecord(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─────────────────────────────────────────────
// Bounce Classifications (DSN)
// ─────────────────────────────────────────────

// GET /api/fbl/bounces?domain=&category=&days=30&limit=200
func (s *Server) handleListBounceClassifications(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	category := r.URL.Query().Get("category")
	days := 30
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	limit := 200
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 1000 {
		limit = l
	}
	since := time.Now().AddDate(0, 0, -days)

	records, err := s.Store.ListBounceClassifications(domain, category, since, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// GET /api/fbl/bounces/summary?days=30
func (s *Server) handleGetBounceClassSummary(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && d > 0 {
		days = d
	}
	since := time.Now().AddDate(0, 0, -days)

	summary, err := s.Store.GetBounceClassificationSummary(since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// ─────────────────────────────────────────────
// VERP Configuration
// ─────────────────────────────────────────────

// GET /api/fbl/verp
func (s *Server) handleListVERPConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := s.Store.ListVERPConfigs()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// GET /api/fbl/verp/{domainID}
func (s *Server) handleGetVERPConfig(w http.ResponseWriter, r *http.Request) {
	domainID, err := strconv.ParseUint(chi.URLParam(r, "domainID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domainID"})
		return
	}

	domain, _ := s.Store.GetDomainByID(uint(domainID))
	domainName := ""
	if domain != nil {
		domainName = domain.Name
	}

	cfg, err := s.Store.GetVERPConfig(uint(domainID))
	if err != nil {
		// Return a sensible default if not configured yet
		writeJSON(w, http.StatusOK, models.VERPConfig{
			DomainID:     uint(domainID),
			Domain:       domainName,
			BounceDomain: "bounces." + domainName,
			IsEnabled:    false,
		})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// POST /api/fbl/verp/{domainID}
func (s *Server) handleSetVERPConfig(w http.ResponseWriter, r *http.Request) {
	domainID, err := strconv.ParseUint(chi.URLParam(r, "domainID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domainID"})
		return
	}

	var req models.VERPConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.DomainID = uint(domainID)

	domain, err := s.Store.GetDomainByID(uint(domainID))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "domain not found"})
		return
	}
	req.Domain = domain.Name
	if req.BounceDomain == "" {
		req.BounceDomain = "bounces." + domain.Name
	}

	// Preserve existing record ID for upsert
	if existing, _ := s.Store.GetVERPConfig(uint(domainID)); existing != nil {
		req.ID = existing.ID
		req.CreatedAt = existing.CreatedAt
	}
	req.UpdatedAt = time.Now()

	if err := s.Store.UpsertVERPConfig(&req); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// ─────────────────────────────────────────────
// Manual upload / testing endpoints
// ─────────────────────────────────────────────

// POST /api/fbl/upload  (multipart/form-data; field: "email")
// Accepts a raw .eml file and processes it as an FBL complaint report.
func (s *Server) handleUploadFBL(w http.ResponseWriter, r *http.Request) {
	raw, err := readUploadedEmail(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	fblSvc := core.NewFBLService(s.Store)
	record, err := fblSvc.ProcessFBL(raw, "manual-upload")
	if err != nil || record == nil {
		msg := "not a valid ARF/FBL report"
		if err != nil {
			msg += ": " + err.Error()
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// POST /api/fbl/upload-dsn  (multipart/form-data; field: "email")
// Accepts a raw .eml file and processes it as a DSN bounce notification.
func (s *Server) handleUploadDSN(w http.ResponseWriter, r *http.Request) {
	raw, err := readUploadedEmail(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	dsn, err := core.ParseDSN(raw)
	if err != nil || dsn == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "not a valid DSN message"})
		return
	}

	// Attempt VERP decode on the sender address
	secret, _ := core.GetEncryptionKey()
	verpDecoded := false
	localPart := core.ExtractLocalPart(dsn.OriginalSender)
	if core.IsVERPLocalPart(localPart) {
		if _, recipient, verr := core.VERPDecode(localPart, secret); verr == nil {
			if dsn.FinalRecipient == "" {
				dsn.FinalRecipient = recipient
			}
			verpDecoded = true
		}
	}

	provider := core.DetectProvider(core.ExtractDomain(dsn.FinalRecipient))
	bc := core.DSNToClassification(dsn, time.Now(), 0, provider, "manual-upload", false)
	bc.VERPDecoded = verpDecoded

	if err := s.Store.CreateBounceClassification(bc); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, bc)
}

// readUploadedEmail reads the "email" field from a multipart/form-data request
// or falls back to reading raw request body (for direct .eml POST).
func readUploadedEmail(r *http.Request) ([]byte, error) {
	ct := r.Header.Get("Content-Type")
	if len(ct) > 9 && ct[:9] == "multipart" {
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			return nil, err
		}
		file, _, err := r.FormFile("email")
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return io.ReadAll(io.LimitReader(file, 2<<20))
	}
	// Raw body upload
	return io.ReadAll(io.LimitReader(r.Body, 2<<20))
}
