package api

import (
	"io"
	"net/http"

	"github.com/pulak-ranjan/kumoops/internal/core"
)

// POST /api/discord/interactions
// Public endpoint — Discord POSTs here for every slash command and button click.
// Authentication is via Ed25519 signature, NOT the session token.
func (s *Server) handleDiscordInteractions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify Ed25519 signature (required by Discord — must return 401 if invalid)
	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || !settings.DiscordBotEnabled || settings.DiscordPublicKey == "" {
		http.Error(w, "discord bot not configured", http.StatusForbidden)
		return
	}

	signature := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")
	if !core.VerifyDiscordSignature(settings.DiscordPublicKey, signature, timestamp, body) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	resp, err := s.DiscordBot.Handle(body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// POST /api/discord/register-commands
// Protected — call once after configuring Discord bot credentials to register slash commands.
func (s *Server) handleDiscordRegisterCommands(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load settings"})
		return
	}
	if settings.DiscordApplicationID == "" || settings.DiscordBotToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "discord_application_id and discord_bot_token required"})
		return
	}
	if err := core.RegisterSlashCommands(settings.DiscordApplicationID, settings.DiscordBotToken); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "commands registered"})
}
