package api

import (
	"encoding/json"
	"net/http"

	"github.com/pulak-ranjan/kumoops/internal/core"
)

// POST /api/notify/test-telegram
func (s *Server) handleTestTelegram(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token  string `json:"token"`
		ChatID string `json:"chat_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Token == "" || req.ChatID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token and chat_id required"})
		return
	}
	if err := core.TestTelegramConnection(req.Token, req.ChatID); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// POST /api/notify/test-discord
func (s *Server) handleTestDiscord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url required"})
		return
	}
	if err := core.TestDiscordConnection(req.URL); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
