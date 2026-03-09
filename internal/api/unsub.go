package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
)

// GET /unsubscribe/{token} — web unsubscribe confirmation page (no auth required)
func (s *Server) handleUnsubscribePage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	_, _, ok := core.VerifyUnsubToken(token)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, unsubHTML("Invalid Link", "This unsubscribe link is invalid or has expired.", false))
		return
	}
	fmt.Fprint(w, unsubHTML("Unsubscribe", "Click the button below to unsubscribe from this mailing list.", true))
}

// POST /unsubscribe/{token} — one-click unsubscribe (RFC 8058, no auth required)
func (s *Server) handleUnsubscribePost(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	recip, err := core.ProcessUnsubscribe(s.Store, token)
	if err != nil {
		http.Error(w, "Invalid unsubscribe request: "+err.Error(), http.StatusBadRequest)
		return
	}
	// RFC 8058: respond with 200 OK for machine-initiated one-click
	accept := r.Header.Get("Content-Type")
	if accept == "application/x-www-form-urlencoded" || r.FormValue("List-Unsubscribe") == "One-Click" {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Browser-based: show confirmation page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, unsubHTML(
		"Unsubscribed",
		"You ("+recip.Email+") have been successfully unsubscribed. You will no longer receive emails from this sender.",
		false,
	))
}

// unsubHTML renders a minimal, clean unsubscribe HTML page.
func unsubHTML(title, message string, showButton bool) string {
	btn := ""
	if showButton {
		btn = `<form method="POST" style="margin-top:24px">
			<button type="submit" style="background:#e53e3e;color:#fff;border:none;padding:12px 32px;border-radius:8px;font-size:16px;cursor:pointer">
				Confirm Unsubscribe
			</button>
		</form>`
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s – KumoOps</title>
<style>
  body{font-family:system-ui,sans-serif;background:#f7f7f7;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
  .card{background:#fff;border-radius:12px;box-shadow:0 2px 16px rgba(0,0,0,.08);padding:48px 40px;max-width:440px;text-align:center}
  h1{margin:0 0 16px;font-size:24px;color:#1a202c}
  p{color:#4a5568;font-size:16px;line-height:1.6;margin:0}
</style></head>
<body>
<div class="card">
  <h1>%s</h1>
  <p>%s</p>
  %s
</div>
</body></html>`, title, title, message, btn)
}
