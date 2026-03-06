package api

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// GET /api/suppression
func (s *Server) handleListSuppression(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	search := r.URL.Query().Get("search")

	page := 1
	pageSize := 50
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 500 {
		pageSize = ps
	}

	results, total, err := s.Store.ListSuppressed(page, pageSize, search)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list suppression"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// POST /api/suppression
func (s *Server) handleAddSuppression(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email      string `json:"email"`
		Reason     string `json:"reason"`
		SourceInfo string `json:"source_info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required"})
		return
	}
	if req.Reason == "" {
		req.Reason = "manual"
	}
	if req.SourceInfo == "" {
		req.SourceInfo = "admin"
	}
	if err := s.Store.AddSuppression(req.Email, req.Reason, req.SourceInfo); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add suppression"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "suppressed", "email": req.Email})
}

// DELETE /api/suppression/{id}
func (s *Server) handleRemoveSuppression(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.Store.RemoveSuppression(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove suppression"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// POST /api/suppression/bulk
func (s *Server) handleBulkSuppression(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Emails []string `json:"emails"`
		Reason string   `json:"reason"`
		Source string   `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if len(req.Emails) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "emails list is empty"})
		return
	}
	if req.Reason == "" {
		req.Reason = "manual"
	}
	if req.Source == "" {
		req.Source = "bulk_import"
	}
	// Normalize emails
	for i, e := range req.Emails {
		req.Emails[i] = strings.ToLower(strings.TrimSpace(e))
	}
	count, err := s.Store.BulkAddSuppression(req.Emails, req.Reason, req.Source)
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "bulk suppression failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"added": count})
}

// GET /api/suppression/export
func (s *Server) handleExportSuppression(w http.ResponseWriter, r *http.Request) {
	results, _, err := s.Store.ListSuppressed(1, 100000, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "export failed"})
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="suppression_list.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"email", "reason", "domain", "source_info", "created_at"})
	for _, entry := range results {
		_ = cw.Write([]string{
			entry.Email,
			entry.Reason,
			entry.Domain,
			entry.SourceInfo,
			entry.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	cw.Flush()
}

// POST /api/suppression/import
func (s *Server) handleImportSuppression(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse form"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file field required"})
		return
	}
	defer file.Close()

	cr := csv.NewReader(file)
	cr.FieldsPerRecord = -1 // allow variable fields
	records, err := cr.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid csv"})
		return
	}

	var emails []string
	reason := "import"
	for i, row := range records {
		if i == 0 && len(row) > 0 && strings.ToLower(row[0]) == "email" {
			continue // skip header
		}
		if len(row) == 0 {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(row[0]))
		if email != "" {
			emails = append(emails, email)
		}
		if i == 0 && len(row) > 1 {
			// if no header skip, use second column as reason for first row hint
		}
	}
	added, _ := s.Store.BulkAddSuppression(emails, reason, "csv_import")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"added":    added,
		"parsed":   len(emails),
	})
}

// GET /api/suppression/check?email=x@y.com
func (s *Server) handleCheckSuppressed(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email param required"})
		return
	}
	suppressed := s.Store.IsSuppressed(email)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"email":      email,
		"suppressed": suppressed,
	})
}

// Ensure models import is used
var _ = models.SuppressedEmail{}
