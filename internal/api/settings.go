package api

import (
	"encoding/json"
	"net/http"

	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

type settingsDTO struct {
	MainHostname string `json:"main_hostname"`
	MainServerIP string `json:"main_server_ip"`
	RelayIPs     string `json:"relay_ips"`
	TLSCertPath  string `json:"tls_cert_path"`
	TLSKeyPath   string `json:"tls_key_path"`

	// AI
	AIProvider    string `json:"ai_provider"`
	AIAPIKey      string `json:"ai_api_key,omitempty"`
	OllamaBaseURL string `json:"ollama_base_url"`
	OllamaModel   string `json:"ollama_model"`

	// Telegram
	TelegramBotToken   string `json:"telegram_bot_token,omitempty"`
	TelegramChatID     string `json:"telegram_chat_id"`
	TelegramEnabled    bool   `json:"telegram_enabled"`
	TelegramDigestHour int    `json:"telegram_digest_hour"`

	// Discord Webhook
	DiscordWebhookURL string `json:"discord_webhook_url"`
	DiscordEnabled    bool   `json:"discord_enabled"`

	// Discord Bot
	DiscordBotToken      string `json:"discord_bot_token,omitempty"`
	DiscordApplicationID string `json:"discord_application_id"`
	DiscordPublicKey     string `json:"discord_public_key"`
	DiscordBotEnabled    bool   `json:"discord_bot_enabled"`

	// Server label
	ServerLabel string `json:"server_label"`
}

// GET /api/settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Store.GetSettings()
	if err != nil && err != store.ErrNotFound {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	if st == nil {
		writeJSON(w, http.StatusOK, settingsDTO{})
		return
	}

	writeJSON(w, http.StatusOK, settingsDTO{
		MainHostname:         st.MainHostname,
		MainServerIP:         st.MainServerIP,
		RelayIPs:             st.MailWizzIP,
		TLSCertPath:          st.TLSCertPath,
		TLSKeyPath:           st.TLSKeyPath,
		AIProvider:           st.AIProvider,
		OllamaBaseURL:        st.OllamaBaseURL,
		OllamaModel:          st.OllamaModel,
		// AIAPIKey intentionally omitted on GET — write-only
		TelegramChatID:       st.TelegramChatID,
		TelegramEnabled:      st.TelegramEnabled,
		TelegramDigestHour:   st.TelegramDigestHour,
		// TelegramBotToken omitted on GET — write-only
		DiscordWebhookURL:    st.DiscordWebhookURL,
		DiscordEnabled:       st.DiscordEnabled,
		DiscordApplicationID: st.DiscordApplicationID,
		DiscordPublicKey:     st.DiscordPublicKey,
		DiscordBotEnabled:    st.DiscordBotEnabled,
		// DiscordBotToken omitted on GET — write-only
		ServerLabel:          st.ServerLabel,
	})
}

// POST /api/settings
func (s *Server) handleSetSettings(w http.ResponseWriter, r *http.Request) {
	var dto settingsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	existing, err := s.Store.GetSettings()
	if err != nil && err != store.ErrNotFound {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load settings"})
		return
	}

	if existing == nil {
		existing = &models.AppSettings{}
	}

	existing.MainHostname = dto.MainHostname
	existing.MainServerIP = dto.MainServerIP
	existing.MailWizzIP = dto.RelayIPs
	existing.TLSCertPath = dto.TLSCertPath
	existing.TLSKeyPath = dto.TLSKeyPath
	existing.AIProvider = dto.AIProvider
	existing.OllamaBaseURL = dto.OllamaBaseURL
	existing.OllamaModel = dto.OllamaModel

	// Telegram
	existing.TelegramChatID = dto.TelegramChatID
	existing.TelegramEnabled = dto.TelegramEnabled
	existing.TelegramDigestHour = dto.TelegramDigestHour
	if dto.TelegramBotToken != "" {
		existing.TelegramBotToken = dto.TelegramBotToken
	}

	// Discord
	existing.DiscordWebhookURL = dto.DiscordWebhookURL
	existing.DiscordEnabled = dto.DiscordEnabled
	existing.DiscordApplicationID = dto.DiscordApplicationID
	existing.DiscordPublicKey = dto.DiscordPublicKey
	existing.DiscordBotEnabled = dto.DiscordBotEnabled
	if dto.DiscordBotToken != "" {
		existing.DiscordBotToken = dto.DiscordBotToken
	}

	// Server label
	existing.ServerLabel = dto.ServerLabel

	// Encrypt sensitive keys
	if dto.AIAPIKey != "" {
		enc, err := core.Encrypt(dto.AIAPIKey)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encrypt key"})
			return
		}
		existing.AIAPIKey = enc
	}

	if err := s.Store.UpsertSettings(existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
