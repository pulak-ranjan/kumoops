package api

// SMTP Relay Management API
//
// KumoOps exposes KumoMTA as a managed SMTP relay.
// Relay clients authenticate via SMTP AUTH using credentials managed here.
// KumoOps generates the auth.toml for KumoMTA containing relay credentials.
// This means relay clients send to KumoMTA port 25/587 with their credentials,
// KumoMTA authenticates via auth.toml, and applies the correct sender policy.
//
// Endpoints:
//   GET  /api/relay/status          — relay config + status
//   PUT  /api/relay/settings        — toggle relay on/off, set port/host
//   POST /api/relay/apply           — push updated auth.toml to KumoMTA

import (
	"encoding/json"
	"net/http"
)

// GET /api/relay/status
func (s *Server) handleGetRelayStatus(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Count how many senders have SMTP passwords (= relay credentials)
	var relayCount int64
	s.Store.DB.Model(&struct{}{}).Table("senders").
		Where("smtp_password != ''").Count(&relayCount)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":        settings.SMTPRelayEnabled,
		"port":           settings.SMTPRelayPort,
		"host":           settings.SMTPRelayHost,
		"relay_accounts": relayCount,
		"info": "KumoMTA's built-in SMTP listener acts as the relay. " +
			"Relay credentials are managed via Sender SMTP passwords and pushed via auth.toml.",
	})
}

// PUT /api/relay/settings
func (s *Server) handleUpdateRelaySettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool   `json:"enabled"`
		Port    int    `json:"port"`
		Host    string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	settings, err := s.Store.GetSettings()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	settings.SMTPRelayEnabled = body.Enabled
	if body.Port > 0 {
		settings.SMTPRelayPort = body.Port
	} else if settings.SMTPRelayPort == 0 {
		settings.SMTPRelayPort = 587
	}
	if body.Host != "" {
		settings.SMTPRelayHost = body.Host
	} else if settings.SMTPRelayHost == "" {
		settings.SMTPRelayHost = "0.0.0.0"
	}
	if err := s.Store.DB.Save(settings).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

// POST /api/relay/apply — regenerate and push auth.toml to KumoMTA
func (s *Server) handleApplyRelayConfig(w http.ResponseWriter, r *http.Request) {
	// Re-use the existing apply endpoint logic
	s.handleApplyConfig(w, r)
}
