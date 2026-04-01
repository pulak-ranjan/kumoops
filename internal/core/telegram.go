package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// ─── Telegram API types ───────────────────────────────────────────────────────

type tgUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type tgGetUpdatesResp struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

// ─── Pending confirmation ─────────────────────────────────────────────────────

type pendingConfirm struct {
	description string
	execute     func()
	expiresAt   time.Time
}

// ─── TelegramBot ─────────────────────────────────────────────────────────────

type TelegramBot struct {
	Store    *store.Store
	mu       sync.Mutex
	confirms map[string]*pendingConfirm // keyed by chatID
}

func NewTelegramBot(st *store.Store) *TelegramBot {
	tb := &TelegramBot{
		Store:    st,
		confirms: make(map[string]*pendingConfirm),
	}
	go tb.expireConfirms()
	return tb
}

func (tb *TelegramBot) expireConfirms() {
	t := time.NewTicker(15 * time.Second)
	for range t.C {
		tb.mu.Lock()
		now := time.Now()
		for chatID, p := range tb.confirms {
			if now.After(p.expiresAt) {
				delete(tb.confirms, chatID)
			}
		}
		tb.mu.Unlock()
	}
}

func (tb *TelegramBot) setPending(chatID string, p *pendingConfirm) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.confirms[chatID] = p
}

func (tb *TelegramBot) takePending(chatID string) (*pendingConfirm, bool) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	p, ok := tb.confirms[chatID]
	if ok {
		delete(tb.confirms, chatID)
	}
	return p, ok
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// SendMessage sends a Markdown message to a single chatID.
func (tb *TelegramBot) SendMessage(token, chatID, text string) error {
	if token == "" || chatID == "" {
		return nil
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body, _ := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// broadcastAll sends a message to every chat ID in the comma-separated list.
// This is used for notifications/alerts so the whole team sees them.
func (tb *TelegramBot) broadcastAll(token, chatIDs, text string) {
	if token == "" || chatIDs == "" {
		return
	}
	for _, id := range strings.Split(chatIDs, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			tb.SendMessage(token, id, text) //nolint:errcheck
		}
	}
}

// label prepends the server label to a message so recipients know which VPS sent it.
func label(settings *models.AppSettings, text string) string {
	if settings.ServerLabel == "" {
		return text
	}
	return fmt.Sprintf("🖥 *[%s]*\n\n%s", settings.ServerLabel, text)
}

// isAllowed checks whether chatID is permitted to use the bot.
// Allowed chats = the configured TelegramChatID (comma-separated list).
func isAllowed(settings *models.AppSettings, chatID string) bool {
	if settings.TelegramChatID == "" {
		return true // open — no restriction configured
	}
	for _, id := range strings.Split(settings.TelegramChatID, ",") {
		if strings.TrimSpace(id) == chatID {
			return true
		}
	}
	return false
}

// kumoAPI calls the KumoMTA management API.
func kumoAPI(method, path string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "http://127.0.0.1:8000"+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ─── Polling loop ─────────────────────────────────────────────────────────────

// deleteWebhook removes any active webhook so getUpdates polling works.
// Telegram ignores getUpdates while a webhook is set.
func deleteWebhook(token string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteWebhook", token)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[Telegram] deleteWebhook failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Println("[Telegram] Webhook cleared, polling mode active")
}

// StartPolling begins long-polling for Telegram bot commands. Runs forever.
func (tb *TelegramBot) StartPolling() {
	log.Println("[Telegram] Bot polling started")
	offset := 0
	webhookCleared := false

	for {
		settings, err := tb.Store.GetSettings()
		if err != nil || settings == nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
			if err != nil {
				log.Printf("[Telegram] Settings load error: %v", err)
			}
			time.Sleep(30 * time.Second)
			webhookCleared = false
			continue
		}

		token := settings.TelegramBotToken

		// Clear any existing webhook on first successful config load
		if !webhookCleared {
			deleteWebhook(token)
			webhookCleared = true
		}

		apiURL := fmt.Sprintf(
			"https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=25",
			token, offset,
		)

		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("[Telegram] getUpdates error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("[Telegram] getUpdates HTTP %d: %s", resp.StatusCode, string(data))
			time.Sleep(10 * time.Second)
			continue
		}

		var updates tgGetUpdatesResp
		if err := json.Unmarshal(data, &updates); err != nil || !updates.OK {
			log.Printf("[Telegram] getUpdates parse error: err=%v ok=%v body=%s", err, updates.OK, string(data))
			time.Sleep(5 * time.Second)
			continue
		}

		for _, upd := range updates.Result {
			offset = upd.UpdateID + 1
			if upd.Message == nil || upd.Message.Text == "" {
				continue
			}
			chatID := strconv.FormatInt(upd.Message.Chat.ID, 10)
			username := upd.Message.From.Username
			text := strings.TrimSpace(upd.Message.Text)
			tb.handleCommand(token, chatID, username, text, settings)
		}
	}
}

// ─── Command router ───────────────────────────────────────────────────────────

func (tb *TelegramBot) handleCommand(token, chatID, username, text string, settings *models.AppSettings) {
	// Security: check allowed chats
	if !isAllowed(settings, chatID) {
		tb.SendMessage(token, chatID, "🚫 Unauthorized.")
		return
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return
	}
	cmd := strings.ToLower(fields[0])

	// Strip bot @mention suffix (e.g. /stats@MyBot → /stats)
	if i := strings.Index(cmd, "@"); i != -1 {
		cmd = cmd[:i]
	}

	// Audit: log the command
	tb.Store.SaveChatLog("bot-command", fmt.Sprintf("[%s] %s: %s", chatID, username, text))

	// ── /confirm handler ──────────────────────────────────────────────────
	if cmd == "/confirm" || cmd == "/yes" {
		p, ok := tb.takePending(chatID)
		if !ok {
			tb.SendMessage(token, chatID, "⚠️ Nothing pending confirmation.")
			return
		}
		if time.Now().After(p.expiresAt) {
			tb.SendMessage(token, chatID, "⏱ Confirmation expired.")
			return
		}
		tb.SendMessage(token, chatID, fmt.Sprintf("⚙️ Running: _%s_…", p.description))
		go p.execute()
		return
	}
	if cmd == "/cancel" {
		if _, ok := tb.takePending(chatID); ok {
			tb.SendMessage(token, chatID, "✅ Cancelled.")
		} else {
			tb.SendMessage(token, chatID, "Nothing to cancel.")
		}
		return
	}

	// ── Read-only commands ────────────────────────────────────────────────
	switch cmd {

	case "/start", "/help":
		tb.SendMessage(token, chatID,
			"🤖 *KumoMTA Bot — Commands*\n\n"+
				"*📊 Stats*\n"+
				"`/stats` — Today's delivery stats\n"+
				"`/bounces` — Bounce summary (24h)\n"+
				"`/tail [n]` — Last N log lines (default 20)\n\n"+
				"*📬 Queue*\n"+
				"`/queue` — Current queue depth\n"+
				"`/flush` — ⚠️ Flush all deferred messages\n"+
				"`/drop-bounced` — ⚠️ Drop all bounced messages\n"+
				"`/retry-all` — ⚠️ Force retry all deferred\n\n"+
				"*🌐 Reputation*\n"+
				"`/reputation` — Blacklist check status\n"+
				"`/check` — Run a fresh scan\n\n"+
				"*🚀 Campaigns*\n"+
				"`/campaigns` — List active campaigns\n"+
				"`/pause-campaign <id>` — ⚠️ Pause a campaign\n"+
				"`/resume-campaign <id>` — Resume a campaign\n\n"+
				"*🔥 Warmup*\n"+
				"`/warmup` — Show warmup status per sender\n\n"+
				"*⚙️ System*\n"+
				"`/disk` — Disk usage\n"+
				"`/mem` — Memory & CPU\n"+
				"`/reload` — ⚠️ Reload KumoMTA config\n"+
				"`/restart` — ⚠️ Restart KumoMTA service\n\n"+
				"⚠️ = requires `/confirm` within 60s\n"+
				"`/cancel` — Cancel pending action",
		)

	case "/stats":
		stats, err := GetAllDomainsStats(1)
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Could not fetch stats.")
			return
		}
		var sent, delivered, bounced, deferred int64
		for _, days := range stats {
			for _, d := range days {
				sent += d.Sent
				delivered += d.Delivered
				bounced += d.Bounced
				deferred += d.Deferred
			}
		}
		rate := 0.0
		if sent > 0 {
			rate = float64(delivered) / float64(sent) * 100
		}
		indicator := "✅"
		if rate < 80 {
			indicator = "🚨"
		} else if rate < 95 {
			indicator = "⚠️"
		}
		tb.SendMessage(token, chatID, fmt.Sprintf(
			"📊 *Stats — Today*\n\n"+
				"📤 Sent: `%d`\n"+
				"✅ Delivered: `%d`\n"+
				"❌ Bounced: `%d`\n"+
				"⏳ Deferred: `%d`\n"+
				"%s Rate: `%.1f%%`",
			sent, delivered, bounced, deferred, indicator, rate,
		))

	case "/queue":
		msgs, err := GetQueueMessages(1000)
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Could not read queue.")
			return
		}
		// Break down by domain (extract from recipient)
		domains := map[string]int{}
		for _, m := range msgs {
			domain := m.Recipient
			if i := strings.LastIndex(m.Recipient, "@"); i != -1 {
				domain = m.Recipient[i+1:]
			}
			domains[domain]++
		}
		lines := []string{fmt.Sprintf("📬 *Queue — %d message(s)*\n", len(msgs))}
		for domain, count := range domains {
			lines = append(lines, fmt.Sprintf("  • `%s`: %d", domain, count))
		}
		tb.SendMessage(token, chatID, strings.Join(lines, "\n"))

	case "/tail":
		n := 20
		if len(fields) >= 2 {
			if v, err := strconv.Atoi(fields[1]); err == nil && v > 0 && v <= 100 {
				n = v
			}
		}
		out, err := exec.Command("journalctl", "-u", "kumomta", "-n", strconv.Itoa(n), "--no-pager", "--output=short").CombinedOutput()
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Could not read logs.")
			return
		}
		tb.SendMessage(token, chatID, "```\n"+truncate(string(out), 3800)+"\n```")

	case "/reputation":
		rows, err := tb.Store.GetLatestReputationChecks()
		if err != nil || len(rows) == 0 {
			tb.SendMessage(token, chatID, "⚠️ No reputation data. Send /check to run a scan.")
			return
		}
		var listed, clean []string
		for _, r := range rows {
			if r.Status == "listed" {
				listed = append(listed, fmt.Sprintf("  🔴 `%s` → _%s_", r.Target, r.ListedOn))
			} else {
				clean = append(clean, r.Target)
			}
		}
		if len(listed) == 0 {
			tb.SendMessage(token, chatID, fmt.Sprintf("✅ All %d targets are clean!", len(clean)))
		} else {
			tb.SendMessage(token, chatID, fmt.Sprintf(
				"🚨 *Blacklisted — %d/%d*\n\n%s\n\n✅ Clean: %d",
				len(listed), len(rows), strings.Join(listed, "\n"), len(clean),
			))
		}

	case "/check":
		tb.SendMessage(token, chatID, "🔍 Running reputation scan…")
		go func() {
			results, err := CheckReputation(tb.Store)
			if err != nil {
				tb.SendMessage(token, chatID, "❌ Scan failed: "+err.Error())
				return
			}
			listed := 0
			for _, r := range results {
				if r.Status == "listed" {
					listed++
				}
			}
			if listed == 0 {
				tb.SendMessage(token, chatID, fmt.Sprintf("✅ All %d targets clean.", len(results)))
			} else {
				tb.SendMessage(token, chatID, fmt.Sprintf(
					"🚨 *%d/%d targets blacklisted!* Use /reputation for details.", listed, len(results),
				))
			}
		}()

	case "/bounces":
		events, _, err := tb.Store.ListDeliveryEvents(1, 200, store.DeliveryEventFilter{
			EventType: "Bounce",
			Hours:     24,
		})
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Could not fetch bounces.")
			return
		}
		if len(events) == 0 {
			tb.SendMessage(token, chatID, "✅ No bounces in the last 24h!")
			return
		}
		domains := map[string]int{}
		for _, e := range events {
			domains[e.Domain]++
		}
		lines := []string{fmt.Sprintf("📉 *Bounces (24h) — %d total*\n", len(events))}
		for domain, count := range domains {
			lines = append(lines, fmt.Sprintf("  • `%s`: %d", domain, count))
		}
		tb.SendMessage(token, chatID, strings.Join(lines, "\n"))

	case "/campaigns":
		var campaigns []models.Campaign
		if err := tb.Store.DB.Order("created_at desc").Limit(10).Find(&campaigns).Error; err != nil {
			tb.SendMessage(token, chatID, "❌ Could not fetch campaigns.")
			return
		}
		if len(campaigns) == 0 {
			tb.SendMessage(token, chatID, "No campaigns found.")
			return
		}
		lines := []string{"🚀 *Campaigns (last 10)*\n"}
		statusIcon := map[string]string{
			"sending": "▶️", "paused": "⏸", "completed": "✅",
			"failed": "❌", "draft": "📝", "scheduled": "⏰",
		}
		for _, c := range campaigns {
			icon := statusIcon[c.Status]
			if icon == "" {
				icon = "•"
			}
			lines = append(lines, fmt.Sprintf("%s `[%d]` *%s* — %s", icon, c.ID, c.Name, c.Status))
		}
		lines = append(lines, "\nUse `/pause-campaign <id>` or `/resume-campaign <id>`")
		tb.SendMessage(token, chatID, strings.Join(lines, "\n"))

	case "/resume-campaign":
		if len(fields) < 2 {
			tb.SendMessage(token, chatID, "Usage: `/resume-campaign <id>`")
			return
		}
		id, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Invalid campaign ID.")
			return
		}
		var c models.Campaign
		if err := tb.Store.DB.First(&c, id).Error; err != nil {
			tb.SendMessage(token, chatID, "❌ Campaign not found.")
			return
		}
		c.Status = "sending"
		tb.Store.DB.Save(&c)
		tb.SendMessage(token, chatID, fmt.Sprintf("▶️ Campaign `%d` *%s* resumed.", c.ID, c.Name))

	case "/warmup":
		domains, err := tb.Store.ListDomains()
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Could not fetch domains.")
			return
		}
		lines := []string{"🔥 *Warmup Status*\n"}
		for _, d := range domains {
			senders, _ := tb.Store.ListSendersByDomain(d.ID)
			for _, s := range senders {
				if s.WarmupPlan == "" {
					continue
				}
				rate := GetSenderRate(s)
				lines = append(lines, fmt.Sprintf("  • `%s` — plan: _%s_, day %d → %s/day",
					s.Email, s.WarmupPlan, s.WarmupDay, rate))
			}
		}
		if len(lines) == 1 {
			tb.SendMessage(token, chatID, "No senders on warmup.")
			return
		}
		tb.SendMessage(token, chatID, strings.Join(lines, "\n"))

	case "/disk":
		out, err := exec.Command("df", "-h", "/").CombinedOutput()
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Could not read disk info.")
			return
		}
		tb.SendMessage(token, chatID, "💾 *Disk Usage*\n```\n"+string(out)+"```")

	case "/mem":
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		out, _ := exec.Command("free", "-h").CombinedOutput()
		tb.SendMessage(token, chatID, fmt.Sprintf(
			"🖥 *Memory*\n```\n%s```\n_Go heap: %.1f MB_",
			string(out), float64(m.Alloc)/1024/1024,
		))

	// ── Destructive commands — require /confirm ────────────────────────────

	case "/flush":
		tb.requireConfirm(token, chatID, "Flush all deferred messages", func() {
			if _, err := kumoAPI("POST", "/api/admin/rebind", map[string]interface{}{
				"data": map[string]interface{}{"queue": nil},
			}); err != nil {
				// Fallback: restart (processes queue)
				exec.Command("systemctl", "try-restart", "kumomta").Run()
				tb.SendMessage(token, chatID, "✅ Flush triggered (service restarted to process deferred).")
				return
			}
			tb.SendMessage(token, chatID, "✅ Deferred messages flushed.")
		})

	case "/retry-all":
		tb.requireConfirm(token, chatID, "Force retry all deferred messages", func() {
			out, err := kumoAPI("POST", "/api/admin/rebind", map[string]interface{}{
				"data":           map[string]interface{}{},
				"trigger_rebind": true,
			})
			if err != nil {
				tb.SendMessage(token, chatID, "❌ Failed: "+err.Error())
				return
			}
			tb.SendMessage(token, chatID, "✅ Retry all triggered.\n```\n"+truncate(string(out), 300)+"```")
		})

	case "/drop-bounced":
		tb.requireConfirm(token, chatID, "Drop all bounced messages from queue", func() {
			msgs, err := GetQueueMessages(5000)
			if err != nil {
				tb.SendMessage(token, chatID, "❌ Could not read queue.")
				return
			}
			dropped := 0
			for _, m := range msgs {
				if strings.EqualFold(m.Status, "bounced") || strings.EqualFold(m.Status, "failed") {
					if err := DeleteQueueMessage(m.ID); err == nil {
						dropped++
					}
				}
			}
			tb.SendMessage(token, chatID, fmt.Sprintf("🗑 Dropped `%d` bounced messages.", dropped))
		})

	case "/pause-campaign":
		if len(fields) < 2 {
			tb.SendMessage(token, chatID, "Usage: `/pause-campaign <id>`")
			return
		}
		id, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			tb.SendMessage(token, chatID, "❌ Invalid campaign ID.")
			return
		}
		tb.requireConfirm(token, chatID, fmt.Sprintf("Pause campaign %d", id), func() {
			var c models.Campaign
			if err := tb.Store.DB.First(&c, id).Error; err != nil {
				tb.SendMessage(token, chatID, "❌ Campaign not found.")
				return
			}
			c.Status = "paused"
			tb.Store.DB.Save(&c)
			tb.SendMessage(token, chatID, fmt.Sprintf("⏸ Campaign `%d` *%s* paused.", c.ID, c.Name))
		})

	case "/reload":
		tb.requireConfirm(token, chatID, "Reload KumoMTA config (systemctl reload kumomta)", func() {
			out, err := exec.Command("systemctl", "reload-or-restart", "kumomta").CombinedOutput()
			if err != nil {
				tb.SendMessage(token, chatID, "❌ Reload failed:\n```\n"+string(out)+"```")
				return
			}
			tb.SendMessage(token, chatID, "✅ KumoMTA config reloaded.")
		})

	case "/restart":
		tb.requireConfirm(token, chatID, "RESTART KumoMTA service (brief delivery interruption)", func() {
			out, err := exec.Command("systemctl", "restart", "kumomta").CombinedOutput()
			if err != nil {
				tb.SendMessage(token, chatID, "❌ Restart failed:\n```\n"+string(out)+"```")
				return
			}
			tb.SendMessage(token, chatID, "✅ KumoMTA restarted.")
		})

	default:
		tb.SendMessage(token, chatID, "❓ Unknown command. Send /help.")
	}
}

// requireConfirm sends a confirmation request and stores the pending action.
func (tb *TelegramBot) requireConfirm(token, chatID, description string, action func()) {
	tb.setPending(chatID, &pendingConfirm{
		description: description,
		execute:     action,
		expiresAt:   time.Now().Add(60 * time.Second),
	})
	tb.SendMessage(token, chatID, fmt.Sprintf(
		"⚠️ *Confirm action:*\n\n_%s_\n\nReply `/confirm` within 60s to proceed, or `/cancel` to abort.",
		description,
	))
}

// ─── Notifications (called by scheduler) ─────────────────────────────────────

// SendDigest sends the daily stats digest to Telegram.
func (tb *TelegramBot) SendDigest(stats map[string][]DayStats) error {
	settings, err := tb.Store.GetSettings()
	if err != nil || settings == nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
		return nil
	}

	var totalSent, totalDelivered, totalBounced, totalDeferred int64
	for _, days := range stats {
		for _, d := range days {
			totalSent += d.Sent
			totalDelivered += d.Delivered
			totalBounced += d.Bounced
			totalDeferred += d.Deferred
		}
	}
	rate := 0.0
	if totalSent > 0 {
		rate = float64(totalDelivered) / float64(totalSent) * 100
	}
	indicator := "✅"
	if rate < 80 {
		indicator = "🚨"
	} else if rate < 95 {
		indicator = "⚠️"
	}

	msg := fmt.Sprintf(
		"📊 *Daily Digest — %s*\n\n"+
			"📤 Sent: `%d`\n"+
			"✅ Delivered: `%d`\n"+
			"❌ Bounced: `%d`\n"+
			"⏳ Deferred: `%d`\n"+
			"%s Rate: `%.1f%%`\n\n"+
			"_KumoOps — %s_",
		time.Now().Format("02 Jan 2006"),
		totalSent, totalDelivered, totalBounced, totalDeferred, indicator, rate,
		settings.MainHostname,
	)
	// Broadcast to ALL configured chat IDs (fixes the comma-separated bug)
	tb.broadcastAll(settings.TelegramBotToken, settings.TelegramChatID, label(settings, msg))
	return nil
}

// SendDiscordDigest sends the daily stats digest to Discord.
func (tb *TelegramBot) SendDiscordDigest(stats map[string][]DayStats) error {
	settings, err := tb.Store.GetSettings()
	if err != nil || settings == nil || !settings.DiscordEnabled || settings.DiscordWebhookURL == "" {
		return nil
	}

	var totalSent, totalDelivered, totalBounced, totalDeferred int64
	for _, days := range stats {
		for _, d := range days {
			totalSent += d.Sent
			totalDelivered += d.Delivered
			totalBounced += d.Bounced
			totalDeferred += d.Deferred
		}
	}
	rate := 0.0
	if totalSent > 0 {
		rate = float64(totalDelivered) / float64(totalSent) * 100
	}

	color := 3066993 // green
	if rate < 80 {
		color = 15158332 // red
	} else if rate < 95 {
		color = 16776960 // yellow
	}

	type discordField struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Inline bool   `json:"inline"`
	}
	type discordFooter struct {
		Text string `json:"text"`
	}
	type discordEmbed struct {
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Color       int            `json:"color"`
		Fields      []discordField `json:"fields"`
		Footer      discordFooter  `json:"footer"`
		Timestamp   string         `json:"timestamp"`
	}
	embed := discordEmbed{
		Title:       "📊 Daily Digest",
		Description: func() string {
			if settings.ServerLabel != "" {
				return fmt.Sprintf("[%s] Stats for %s", settings.ServerLabel, time.Now().Format("02 Jan 2006"))
			}
			return fmt.Sprintf("Stats for %s", time.Now().Format("02 Jan 2006"))
		}(),
		Color:       color,
		Fields: []discordField{
			{Name: "📤 Sent", Value: fmt.Sprintf("`%d`", totalSent), Inline: true},
			{Name: "✅ Delivered", Value: fmt.Sprintf("`%d`", totalDelivered), Inline: true},
			{Name: "❌ Bounced", Value: fmt.Sprintf("`%d`", totalBounced), Inline: true},
			{Name: "⏳ Deferred", Value: fmt.Sprintf("`%d`", totalDeferred), Inline: true},
			{Name: "📈 Delivery Rate", Value: fmt.Sprintf("`%.1f%%`", rate), Inline: true},
		},
		Footer:    discordFooter{Text: "KumoOps — " + settings.MainHostname},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"username": "KumoMTA",
		"embeds":   []interface{}{embed},
	})
	resp, err := http.Post(settings.DiscordWebhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// SendAlert broadcasts a Telegram alert to all configured chat IDs.
func (tb *TelegramBot) SendAlert(title, body string) error {
	settings, err := tb.Store.GetSettings()
	if err != nil || settings == nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
		return nil
	}
	msg := fmt.Sprintf("🚨 *%s*\n\n%s", title, body)
	tb.broadcastAll(settings.TelegramBotToken, settings.TelegramChatID, label(settings, msg))
	return nil
}

// ─── Proactive push alerts ────────────────────────────────────────────────────

// CheckAndAlertBounceSpike fires an alert if bounce rate spikes above the threshold.
// Call this from the hourly scheduler.
func (tb *TelegramBot) CheckAndAlertBounceSpike() {
	settings, err := tb.Store.GetSettings()
	if err != nil || settings == nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
		return
	}

	summary, err := tb.Store.GetDeliveryEventSummary(1)
	if err != nil {
		return
	}

	sent := summary["delivered"] + summary["bounce"] + summary["deferred"]
	if sent == 0 {
		return
	}
	bounceRate := float64(summary["bounce"]) / float64(sent) * 100
	threshold := settings.BounceAlertPct
	if threshold == 0 {
		threshold = 10.0
	}
	if bounceRate >= threshold {
		msg := fmt.Sprintf(
			"🚨 *Bounce Spike Alert*\n\nBounce rate hit `%.1f%%` (threshold: %.1f%%)\n\n"+
				"Sent: %d | Bounced: %d | Deferred: %d\n\nUse /bounces for details.",
			bounceRate, threshold, sent, summary["bounce"], summary["deferred"],
		)
		tb.broadcastAll(settings.TelegramBotToken, settings.TelegramChatID, label(settings, msg))
	}
}

// CheckAndAlertQueueBackpressure fires an alert if the queue exceeds a threshold.
// Call this from the hourly scheduler.
func (tb *TelegramBot) CheckAndAlertQueueBackpressure(threshold int) {
	settings, err := tb.Store.GetSettings()
	if err != nil || settings == nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
		return
	}

	stats, err := GetQueueStats()
	if err != nil || stats == nil || stats.Total <= threshold {
		return
	}
	msg := fmt.Sprintf(
		"⚠️ *Queue Backpressure*\n\n`%d` messages queued (threshold: %d)\n\nQueued: %d | Deferred: %d\n\nUse /queue for breakdown.",
		stats.Total, threshold, stats.Queued, stats.Deferred,
	)
	tb.broadcastAll(settings.TelegramBotToken, settings.TelegramChatID, label(settings, msg))
}

// CheckAndAlertNewBlacklisting sends an alert when a new blacklisting is detected.
// Call this after a reputation check.
func (tb *TelegramBot) CheckAndAlertNewBlacklisting(newlyListed []models.ReputationCheck) {
	if len(newlyListed) == 0 {
		return
	}
	settings, err := tb.Store.GetSettings()
	if err != nil || settings == nil || !settings.TelegramEnabled || settings.TelegramBotToken == "" {
		return
	}
	lines := make([]string, 0, len(newlyListed))
	for _, r := range newlyListed {
		lines = append(lines, fmt.Sprintf("  🔴 `%s` → _%s_", r.Target, r.ListedOn))
	}
	msg := fmt.Sprintf("🚨 *New Blacklisting Detected!*\n\n%s\n\nUse /check to refresh.", strings.Join(lines, "\n"))
	tb.broadcastAll(settings.TelegramBotToken, settings.TelegramChatID, label(settings, msg))
	// Mirror to Discord
	if settings.DiscordEnabled && settings.DiscordWebhookURL != "" {
		body := strings.Join(lines, "\n")
		if settings.ServerLabel != "" {
			body = fmt.Sprintf("**[%s]**\n\n%s", settings.ServerLabel, body)
		}
		SendDiscordAlert(settings.DiscordWebhookURL, "🚨 New Blacklisting Detected", body, 15158332)
	}
}

// ─── Discord helpers ──────────────────────────────────────────────────────────

// SendDiscordAlert posts an alert embed to a Discord webhook.
func SendDiscordAlert(webhookURL, title, body string, color int) error {
	if webhookURL == "" {
		return nil
	}
	type embed struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Color       int    `json:"color"`
		Timestamp   string `json:"timestamp"`
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"username": "KumoMTA",
		"embeds": []embed{{
			Title:       title,
			Description: body,
			Color:       color,
			Timestamp:   time.Now().Format(time.RFC3339),
		}},
	})
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

// TestTelegramConnection sends a test message.
func TestTelegramConnection(token, chatID string) error {
	bot := &TelegramBot{}
	return bot.SendMessage(token, chatID,
		"✅ *KumoOps connected!*\n\nYour Telegram bot is working.\n\nSend /help for all available commands.",
	)
}

// TestDiscordConnection sends a test embed to the Discord webhook.
func TestDiscordConnection(webhookURL string) error {
	if !strings.HasPrefix(webhookURL, "http") {
		return fmt.Errorf("invalid URL")
	}
	return SendDiscordAlert(webhookURL, "✅ Connection Test", "KumoOps is connected to this Discord channel.", 3066993)
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "…" + s[len(s)-max:]
}
