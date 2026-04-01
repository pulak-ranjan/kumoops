package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// WebhookService handles sending notifications to Slack/Discord
type WebhookService struct {
	Store *store.Store
}

func NewWebhookService(st *store.Store) *WebhookService {
	return &WebhookService{Store: st}
}

// --- Payload Structures ---

type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string  `json:"color"`
	Title  string  `json:"title"`
	Text   string  `json:"text"`
	Fields []Field `json:"fields,omitempty"`
	Footer string  `json:"footer,omitempty"`
	Ts     int64   `json:"ts,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type DiscordMessage struct {
	Content  string         `json:"content,omitempty"`
	Username string         `json:"username,omitempty"`
	Embeds   []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Fields      []DiscordField `json:"fields,omitempty"`
	Footer      *DiscordFooter `json:"footer,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordFooter struct {
	Text string `json:"text"`
}

// --- Logic ---

func (ws *WebhookService) getSenderName() string {
	settings, err := ws.Store.GetSettings()
	if err == nil && settings != nil && settings.MainHostname != "" {
		return settings.MainHostname
	}
	return "KumoOps"
}

// 1. Audit Log (Task Modifications)
func (ws *WebhookService) SendAuditLog(action, details, user string) error {
	settings, err := ws.Store.GetSettings()
	if err != nil || settings == nil || !settings.WebhookEnabled || settings.WebhookURL == "" {
		return nil
	}

	isDiscord := strings.Contains(settings.WebhookURL, "discord.com")
	senderName := ws.getSenderName()

	var payload []byte
	if isDiscord {
		msg := DiscordMessage{
			Username: senderName,
			Embeds: []DiscordEmbed{{
				Title:       "🛠️ Audit Log",
				Description: fmt.Sprintf("**%s** performed action: %s", user, action),
				Color:       10181046, // Purple
				Fields: []DiscordField{
					{Name: "Details", Value: details, Inline: false},
				},
				Footer:    &DiscordFooter{Text: "System Audit"},
				Timestamp: time.Now().Format(time.RFC3339),
			}},
		}
		payload, _ = json.Marshal(msg)
	} else {
		msg := SlackMessage{
			Username:  senderName,
			IconEmoji: ":hammer_and_wrench:",
			Attachments: []Attachment{{
				Color:  "#9b59b6",
				Title:  "🛠️ Audit Log",
				Text:   fmt.Sprintf("*%s* performed action: %s\n> %s", user, action, details),
				Footer: "System Audit",
				Ts:     time.Now().Unix(),
			}},
		}
		payload, _ = json.Marshal(msg)
	}

	return ws.send(settings.WebhookURL, payload, "audit_log")
}

// 2. Blacklist Checker
// If forceReport is true, it sends a webhook even if no issues are found (for manual checks).
func (ws *WebhookService) CheckBlacklists(forceReport bool) error {
	// Delegate to the full reputation engine (8 IP RBLs + 5 domain DBLs, saves to DB)
	results, err := CheckReputation(ws.Store)
	if err != nil {
		return err
	}

	var issues []string
	for _, rc := range results {
		if rc.Status == "listed" {
			icon := "🌐"
			if rc.TargetType == "ip" {
				icon = "🖥️"
			}
			issues = append(issues, fmt.Sprintf("❌ %s **%s** listed on **%s**", icon, rc.Target, rc.ListedOn))
		}
	}

	if len(issues) > 0 {
		return ws.sendAlert("🚫 Blacklist Alert", "The following targets are blacklisted:", issues, 15158332)
	}

	if forceReport {
		ipCount := 0
		domainCount := 0
		for _, rc := range results {
			if rc.TargetType == "ip" {
				ipCount++
			} else {
				domainCount++
			}
		}
		return ws.sendAlert("✅ Blacklist Report",
			fmt.Sprintf("Scanned %d IPs and %d domains against 13 RBLs.", ipCount, domainCount),
			[]string{"All systems clean. No blacklistings detected."},
			3066993)
	}

	return nil
}

// 3. Security Audit
func (ws *WebhookService) RunSecurityAudit() error {
	var risks []string

	dbPath := os.Getenv("DB_DIR")
	if dbPath == "" {
		dbPath = "/var/lib/kumoops"
	}
	info, err := os.Stat(dbPath + "/panel.db")
	if err == nil {
		mode := info.Mode().Perm()
		if mode&0004 != 0 {
			risks = append(risks, "DB file is world-readable (chmod 600 required)")
		}
	}

	// Check for Debug Port (8000) - should be blocked by firewall
	if conn, err := net.DialTimeout("tcp", "0.0.0.0:8000", 1*time.Second); err == nil {
		conn.Close()
		risks = append(risks, "Port 8000 (HTTP) appears open locally/publicly")
	}

	settings, _ := ws.Store.GetSettings()
	// Check if AIAPIKey is set (encrypted or not, just check presence)
	if settings != nil && settings.AIAPIKey == "" {
		risks = append(risks, "AI API Key missing (Log Analysis disabled)")
	}

	if len(risks) > 0 {
		return ws.sendAlert("🔐 Security Alert", "Security issues detected", risks, 15105570) // Orange
	}
	return nil
}

// 4. Daily Summary
func (ws *WebhookService) SendDailySummary(stats map[string][]DayStats) error {
	settings, err := ws.Store.GetSettings()
	if err != nil || settings == nil || !settings.WebhookEnabled || settings.WebhookURL == "" {
		return nil
	}

	totalSent := int64(0)
	totalDelivered := int64(0)
	totalBounced := int64(0)

	for _, days := range stats {
		for _, d := range days {
			totalSent += d.Sent
			totalDelivered += d.Delivered
			totalBounced += d.Bounced
		}
	}

	summary := []string{
		fmt.Sprintf("Sent: %d", totalSent),
		fmt.Sprintf("Delivered: %d", totalDelivered),
		fmt.Sprintf("Bounced: %d", totalBounced),
	}

	return ws.sendAlert("📊 Daily Summary", "Traffic report for last 24h", summary, 3447003) // Blue
}

// 5. Test Webhook
func (ws *WebhookService) SendTestWebhook(webhookURL string) error {
	senderName := ws.getSenderName()
	isDiscord := strings.Contains(webhookURL, "discord.com")

	var payload []byte
	if isDiscord {
		msg := DiscordMessage{
			Username: senderName,
			Embeds: []DiscordEmbed{{
				Title:       "✅ Test Successful",
				Description: "Webhook is working correctly.",
				Color:       5763719, // Green
				Footer:      &DiscordFooter{Text: "KumoOps"},
				Timestamp:   time.Now().Format(time.RFC3339),
			}},
		}
		payload, _ = json.Marshal(msg)
	} else {
		msg := SlackMessage{
			Username: senderName,
			Text:     "✅ Test Successful! Webhook is working.",
		}
		payload, _ = json.Marshal(msg)
	}
	return ws.send(webhookURL, payload, "test")
}

// 6. Bounce Rates
func (ws *WebhookService) CheckBounceRates() error {
	// Reusing logic: get stats for today (1 day)
	stats, err := GetAllDomainsStats(1)
	if err != nil {
		return err
	}

	settings, err := ws.Store.GetSettings()
	if err != nil || settings == nil || !settings.WebhookEnabled {
		return nil
	}

	var alerts []string

	for domain, days := range stats {
		if len(days) == 0 {
			continue
		}
		today := days[len(days)-1]
		if today.Sent < 10 {
			continue
		} // Ignore low volume

		rate := float64(today.Bounced) / float64(today.Sent) * 100
		if rate >= settings.BounceAlertPct {
			alerts = append(alerts, fmt.Sprintf("**%s**: %.1f%% bounce rate (%d/%d)", domain, rate, today.Bounced, today.Sent))
		}
	}

	if len(alerts) > 0 {
		return ws.sendAlert("⚠️ High Bounce Rate", "Domains exceeding threshold:", alerts, 15158332)
	}
	return nil
}

// SendAlert sends a plain markdown message to both the configured webhook (Slack/Discord)
// and Telegram. Used by the anomaly detector and other autonomous subsystems.
func (ws *WebhookService) SendAlert(msg string) {
	// Webhook (Slack / Discord)
	settings, err := ws.Store.GetSettings()
	if err == nil && settings != nil && settings.WebhookEnabled && settings.WebhookURL != "" {
		isDiscord := strings.Contains(settings.WebhookURL, "discord.com")
		senderName := ws.getSenderName()
		var payload []byte
		if isDiscord {
			dm := DiscordMessage{
				Username: senderName,
				Embeds: []DiscordEmbed{{
					Description: msg,
					Color:       15158332, // orange-ish
					Footer:      &DiscordFooter{Text: "KumoOps Anomaly"},
					Timestamp:   time.Now().Format(time.RFC3339),
				}},
			}
			payload, _ = json.Marshal(dm)
		} else {
			sm := SlackMessage{
				Username:  senderName,
				IconEmoji: ":warning:",
				Text:      msg,
			}
			payload, _ = json.Marshal(sm)
		}
		_ = ws.send(settings.WebhookURL, payload, "anomaly_alert")
	}

	// Telegram
	if err == nil && settings != nil && settings.TelegramEnabled && settings.TelegramBotToken != "" {
		tb := &TelegramBot{Store: ws.Store}
		tb.broadcastAll(settings.TelegramBotToken, settings.TelegramChatID, label(settings, msg))
	}
}

// --- Internals ---

func (ws *WebhookService) sendAlert(title, desc string, items []string, color int) error {
	settings, err := ws.Store.GetSettings()
	if err != nil || settings == nil || !settings.WebhookEnabled || settings.WebhookURL == "" {
		return nil
	}

	isDiscord := strings.Contains(settings.WebhookURL, "discord.com")
	senderName := ws.getSenderName()

	itemList := strings.Join(items, "\n• ")
	if len(items) > 0 {
		itemList = "• " + itemList
	}

	var payload []byte
	if isDiscord {
		msg := DiscordMessage{
			Username: senderName,
			Embeds: []DiscordEmbed{{
				Title:       title,
				Description: desc,
				Color:       color,
				Fields: []DiscordField{
					{Name: "Details", Value: itemList, Inline: false},
				},
				Footer:    &DiscordFooter{Text: "KumoMTA Alert"},
				Timestamp: time.Now().Format(time.RFC3339),
			}},
		}
		payload, _ = json.Marshal(msg)
	} else {
		msg := SlackMessage{
			Username:  senderName,
			IconEmoji: ":warning:",
			Attachments: []Attachment{{
				Color:  fmt.Sprintf("#%06x", color),
				Title:  title,
				Text:   fmt.Sprintf("%s\n\n%s", desc, itemList),
				Footer: "KumoMTA Alert",
				Ts:     time.Now().Unix(),
			}},
		}
		payload, _ = json.Marshal(msg)
	}

	return ws.send(settings.WebhookURL, payload, "alert")
}

func (ws *WebhookService) send(url string, payload []byte, eventType string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ws.logWebhook(eventType, string(payload), 0, err.Error())
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ws.logWebhook(eventType, string(payload), resp.StatusCode, string(body))
	return nil
}

func (ws *WebhookService) logWebhook(eventType, payload string, status int, response string) {
	log := &models.WebhookLog{
		EventType: eventType,
		Payload:   payload,
		Status:    status,
		Response:  response,
		CreatedAt: time.Now(),
	}
	ws.Store.CreateWebhookLog(log)
}
