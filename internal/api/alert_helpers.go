package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// sendSlackMessage sends a Slack incoming webhook message.
func sendSlackMessage(webhookURL, text string) error {
	payload := map[string]string{"text": text}
	return sendWebhookAlert(webhookURL, payload)
}

// sendWebhookAlert sends a generic HTTP POST with JSON payload.
func sendWebhookAlert(url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("webhook post failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
