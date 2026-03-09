package api

// HTTP Sending API — Mailgun/SendGrid-compatible REST endpoint
//
// POST /api/v1/messages
//   Authorization: Bearer <api_key>   OR   ?api_key=<key>
//   Body: JSON { from, to, subject, html, text, sender_id, vars }

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

type sendMsgReq struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	HTML     string            `json:"html"`
	Text     string            `json:"text"`
	SenderID uint              `json:"sender_id"`
	Vars     map[string]string `json:"vars"`
}

// POST /api/v1/messages
func (s *Server) handleAPISendMessage(w http.ResponseWriter, r *http.Request) {
	apiKey := pickAPIKey(r)
	if apiKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "API key required"})
		return
	}
	var keyRec models.APIKey
	if err := s.Store.DB.Where("`key` = ?", apiKey).First(&keyRec).Error; err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
		return
	}
	s.Store.DB.Model(&keyRec).Update("last_used", time.Now())

	var req sendMsgReq
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
	} else {
		r.ParseForm()
		req.From = r.FormValue("from")
		req.To = r.FormValue("to")
		req.Subject = r.FormValue("subject")
		req.HTML = r.FormValue("html")
		req.Text = r.FormValue("text")
	}

	if req.To == "" || req.Subject == "" || (req.HTML == "" && req.Text == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "to, subject, and html or text required"})
		return
	}

	var sender models.Sender
	if req.SenderID > 0 {
		if err := s.Store.DB.First(&sender, req.SenderID).Error; err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sender not found"})
			return
		}
	} else if req.From != "" {
		if err := s.Store.DB.Where("email = ?", req.From).First(&sender).Error; err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no sender matches from; provide sender_id"})
			return
		}
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sender_id or from required"})
		return
	}

	html := tmplReplace(req.HTML, req.Vars)
	if html == "" {
		html = "<pre>" + tmplReplace(req.Text, req.Vars) + "</pre>"
	}
	subject := tmplReplace(req.Subject, req.Vars)

	cs := core.NewCampaignService(s.Store)
	var errs []string
	recipients := splitAddrs(req.To)
	for _, to := range recipients {
		if err := cs.SendSingleEmail(to, subject, html, sender.ID); err != nil {
			errs = append(errs, to+": "+err.Error())
		}
	}
	if len(errs) > 0 {
		writeJSON(w, http.StatusMultiStatus, map[string]interface{}{
			"message": fmt.Sprintf("%d/%d sent", len(recipients)-len(errs), len(recipients)),
			"errors":  errs,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"id":      fmt.Sprintf("<v1.%d.%d@kumoops>", sender.ID, time.Now().UnixNano()),
		"message": fmt.Sprintf("Queued. %d recipient(s).", len(recipients)),
	})
}

func tmplReplace(text string, vars map[string]string) string {
	if len(vars) == 0 || text == "" {
		return text
	}
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	return re.ReplaceAllStringFunc(text, func(m string) string {
		if v, ok := vars[m[2:len(m)-2]]; ok {
			return v
		}
		return m
	})
}

func splitAddrs(s string) []string {
	var out []string
	for _, e := range strings.Split(s, ",") {
		if t := strings.TrimSpace(e); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func pickAPIKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("api_key")
}
