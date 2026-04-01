package api

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

type bounceDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // plain, only from client to set/update; never returned
	Domain   string `json:"domain"`
	Notes    string `json:"notes"`
}

// GET /api/bounces
func (s *Server) handleListBounceAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListBounceAccounts()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list bounce accounts"})
		return
	}

	out := make([]bounceDTO, 0, len(list))
	for _, b := range list {
		out = append(out, bounceDTO{
			ID:       b.ID,
			Username: b.Username,
			Password: "", // never return password
			Domain:   b.Domain,
			Notes:    b.Notes,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/bounces
// If dto.id == 0 => create; else update.
func (s *Server) handleSaveBounceAccount(w http.ResponseWriter, r *http.Request) {
	var dto bounceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	dto.Username = strings.TrimSpace(dto.Username)
	dto.Password = strings.TrimSpace(dto.Password)
	dto.Domain = strings.TrimSpace(dto.Domain)

	if dto.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	// CREATE
	if dto.ID == 0 {
		if dto.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password is required for new bounce account"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
			return
		}

		b := &models.BounceAccount{
			Username:     dto.Username,
			PasswordHash: string(hash),
			Domain:       dto.Domain,
			Notes:        dto.Notes,
		}

		if err := s.Store.CreateBounceAccount(b); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create bounce account"})
			return
		}

		// Apply to system immediately (create user + Maildir + set password)
		if err := core.EnsureBounceAccount(*b, dto.Password); err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "created in DB but failed to create system user: " + err.Error()})
			return
		}

		dto.ID = b.ID
		dto.Password = "" // do not echo password back
		writeJSON(w, http.StatusOK, dto)
		return
	}

	// UPDATE
	existing, err := s.Store.GetBounceAccountByID(dto.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "bounce account not found"})
			return
		}
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load bounce account"})
		return
	}

	existing.Username = dto.Username
	existing.Domain = dto.Domain
	existing.Notes = dto.Notes

	// If a new password is provided, re-hash and update system password.
	var plainToApply string
	if dto.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if err != nil {
			s.Store.LogError(err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash new password"})
			return
		}
		existing.PasswordHash = string(hash)
		plainToApply = dto.Password
	}

	if err := s.Store.UpdateBounceAccount(existing); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update bounce account"})
		return
	}

	// Ensure system user + Maildir; update password only if provided
	if err := core.EnsureBounceAccount(*existing, plainToApply); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "updated in DB but failed to apply to system: " + err.Error()})
		return
	}

	dto.Password = "" // never return password
	writeJSON(w, http.StatusOK, dto)
}

// DELETE /api/bounces/{bounceID}
func (s *Server) handleDeleteBounceAccount(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "bounceID")
	id, err := strconv.Atoi(raw)
	if err != nil || id < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bounce id"})
		return
	}

	// 1. Get the account first so we know the username
	account, err := s.Store.GetBounceAccountByID(uint(id))
	if err != nil {
		// If not found in DB, just return success or not found
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
		return
	}

	// 2. Delete from System (OS)
	if err := core.RemoveSystemUser(account.Username); err != nil {
		// Log error but continue to delete from DB so they aren't stuck
		s.Store.LogError(err)
	}

	// 3. Delete from Database
	if err := s.Store.DeleteBounceAccount(uint(id)); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete bounce account"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /api/bounces/{id}/messages
// Lists emails waiting in the Maildir for this bounce account (new/ and cur/).
func (s *Server) handleListMailboxMessages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	account, err := s.Store.GetBounceAccountByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
		return
	}

	type msgMeta struct {
		ID      string `json:"id"`
		Folder  string `json:"folder"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
		SizeB   int64  `json:"size_bytes"`
	}

	msgs := make([]msgMeta, 0)
	maildirBase := fmt.Sprintf("/home/%s/Maildir", account.Username)

	for _, folder := range []string{"new", "cur"} {
		dir := filepath.Join(maildirBase, folder)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fpath := filepath.Join(dir, entry.Name())
			fi, err := entry.Info()
			if err != nil {
				continue
			}
			from, subj, date := parseMailHeaders(fpath)
			msgs = append(msgs, msgMeta{
				ID:      entry.Name(),
				Folder:  folder,
				From:    from,
				Subject: subj,
				Date:    date,
				SizeB:   fi.Size(),
			})
		}
	}
	// newest first by date string (ISO)
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].Date > msgs[j].Date })
	writeJSON(w, http.StatusOK, msgs)
}

// GET /api/bounces/{id}/messages/{msgid}
// Returns the raw email content of a single message.
func (s *Server) handleGetMailboxMessage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	msgID := chi.URLParam(r, "msgid")
	// Sanitize: filename only, no path separators
	if strings.Contains(msgID, "/") || strings.Contains(msgID, "\\") || strings.Contains(msgID, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid message id"})
		return
	}

	account, err := s.Store.GetBounceAccountByID(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "account not found"})
		return
	}

	maildirBase := fmt.Sprintf("/home/%s/Maildir", account.Username)
	var fpath string
	for _, folder := range []string{"new", "cur"} {
		p := filepath.Join(maildirBase, folder, msgID)
		if _, err := os.Stat(p); err == nil {
			fpath = p
			break
		}
	}
	if fpath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "message not found"})
		return
	}

	raw, err := os.ReadFile(fpath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read message"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"raw": string(raw)})
}

// POST /api/bounces/create-required
// Creates standard required mailboxes (postmaster, abuse, fbl, noreply) for each domain.
func (s *Server) handleCreateRequiredInboxes(w http.ResponseWriter, r *http.Request) {
	domains, err := s.Store.ListDomains()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list domains"})
		return
	}
	if len(domains) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no domains configured"})
		return
	}

	var req struct {
		DomainID uint `json:"domain_id"` // 0 = all domains
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	type result struct {
		Username string `json:"username"`
		Domain   string `json:"domain"`
		Status   string `json:"status"`
		Error    string `json:"error,omitempty"`
	}
	results := make([]result, 0)

	required := []string{"postmaster", "abuse", "fbl", "noreply"}

	for _, d := range domains {
		if req.DomainID != 0 && d.ID != req.DomainID {
			continue
		}
		for _, prefix := range required {
			username := prefix + "-" + sanitizeDomain(d.Name)
			// Skip if already exists
			existing, _ := s.Store.ListBounceAccountsByUsername(username)
			if len(existing) > 0 {
				results = append(results, result{Username: username, Domain: d.Name, Status: "skipped", Error: "already exists"})
				continue
			}

			pass := generatePassword(16)
			hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			b := &models.BounceAccount{
				Username:     username,
				PasswordHash: string(hash),
				Domain:       d.Name,
				Notes:        fmt.Sprintf("Auto-created required inbox: %s@%s", prefix, d.Name),
			}
			if err := s.Store.CreateBounceAccount(b); err != nil {
				results = append(results, result{Username: username, Domain: d.Name, Status: "error", Error: err.Error()})
				continue
			}
			if err := core.EnsureBounceAccount(*b, pass); err != nil {
				results = append(results, result{Username: username, Domain: d.Name, Status: "db_ok_sys_err", Error: err.Error()})
				continue
			}
			results = append(results, result{Username: username, Domain: d.Name, Status: "created"})
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

// parseMailHeaders reads From, Subject, Date from first 60 lines of a .eml file.
func parseMailHeaders(path string) (from, subject, date string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() && lines < 60 {
		line := sc.Text()
		lines++
		if line == "" {
			break // end of headers
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "from:") && from == "" {
			from = strings.TrimSpace(line[5:])
		} else if strings.HasPrefix(lower, "subject:") && subject == "" {
			subject = strings.TrimSpace(line[8:])
		} else if strings.HasPrefix(lower, "date:") && date == "" {
			raw := strings.TrimSpace(line[5:])
			// Try parse to ISO, fall back to raw
			if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", raw); err == nil {
				date = t.Format(time.RFC3339)
			} else {
				date = raw
			}
		}
	}
	return
}

// sanitizeDomain replaces dots with dashes for use in Linux usernames.
func sanitizeDomain(domain string) string {
	return strings.NewReplacer(".", "-", "_", "-").Replace(domain)
}

// generatePassword creates a random alphanumeric password of given length.
func generatePassword(n int) string {
	chars := "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[idx.Int64()]
	}
	return string(b)
}

// POST /api/bounces/apply
// Ensures all bounce accounts exist at OS level with Maildir, without changing passwords.
func (s *Server) handleApplyBounceAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListBounceAccounts()
	if err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list bounce accounts"})
		return
	}

	if err := core.ApplyAllBounceAccounts(list); err != nil {
		s.Store.LogError(err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
