package core

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// ─── Discord interaction types ────────────────────────────────────────────────

const (
	discordTypePing             = 1
	discordTypeApplicationCmd   = 2
	discordTypeMessageComponent = 3 // button clicks

	discordRespPong           = 1
	discordRespChannelMessage = 4
	discordRespDeferredMsg    = 5 // acknowledge now, follow-up later

	discordFlagEphemeral = 1 << 6 // only visible to the invoker
)

type DiscordInteraction struct {
	ID            string                  `json:"id"`
	ApplicationID string                  `json:"application_id"`
	Type          int                     `json:"type"`
	Token         string                  `json:"token"` // for follow-up messages
	Data          *DiscordInteractionData `json:"data"`
	Member        *struct {
		User struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	} `json:"member"`
	User *struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

type DiscordInteractionData struct {
	Name        string                   `json:"name"`
	CustomID    string                   `json:"custom_id"` // for components
	Options     []DiscordInteractionOpt  `json:"options"`
}

type DiscordInteractionOpt struct {
	Name  string `json:"name"`
	Value interface{} `json:"value"`
}

type DiscordResponse struct {
	Type int                  `json:"type"`
	Data *DiscordResponseData `json:"data,omitempty"`
}

type DiscordResponseData struct {
	Content    string             `json:"content,omitempty"`
	Embeds     []DiscordBotEmbed  `json:"embeds,omitempty"`
	Components []DiscordComponent `json:"components,omitempty"`
	Flags      int                `json:"flags,omitempty"`
}

type DiscordBotEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

type DiscordComponent struct {
	Type       int                    `json:"type"`
	Components []DiscordComponentItem `json:"components,omitempty"`
}

type DiscordComponentItem struct {
	Type     int    `json:"type"`
	Label    string `json:"label"`
	Style    int    `json:"style"` // 1=primary,2=secondary,3=success,4=danger,5=link
	CustomID string `json:"custom_id"`
	Disabled bool   `json:"disabled,omitempty"`
}

// ─── DiscordBot ───────────────────────────────────────────────────────────────

type DiscordBot struct {
	Store    *store.Store
	mu       sync.Mutex
	confirms map[string]*discordPending // keyed by userID
}

type discordPending struct {
	action    func()
	label     string
	expiresAt time.Time
}

func NewDiscordBot(st *store.Store) *DiscordBot {
	db := &DiscordBot{
		Store:    st,
		confirms: make(map[string]*discordPending),
	}
	go db.expireConfirms()
	return db
}

func (db *DiscordBot) expireConfirms() {
	t := time.NewTicker(15 * time.Second)
	for range t.C {
		db.mu.Lock()
		now := time.Now()
		for uid, p := range db.confirms {
			if now.After(p.expiresAt) {
				delete(db.confirms, uid)
			}
		}
		db.mu.Unlock()
	}
}

func (db *DiscordBot) setPending(userID string, p *discordPending) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.confirms[userID] = p
}

func (db *DiscordBot) takePending(userID string) (*discordPending, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	p, ok := db.confirms[userID]
	if ok {
		delete(db.confirms, userID)
	}
	return p, ok
}

// ─── Signature verification ───────────────────────────────────────────────────

// VerifySignature checks the Ed25519 signature Discord sends on every interaction.
// Returns false if invalid — must respond 401 in that case.
func VerifyDiscordSignature(publicKeyHex, signature, timestamp string, body []byte) bool {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return false
	}
	msg := append([]byte(timestamp), body...)
	return ed25519.Verify(pubKeyBytes, msg, sigBytes)
}

// ─── Interaction handler ──────────────────────────────────────────────────────

// Handle processes a verified Discord interaction and returns the JSON response.
func (db *DiscordBot) Handle(body []byte) ([]byte, error) {
	var ix DiscordInteraction
	if err := json.Unmarshal(body, &ix); err != nil {
		return nil, err
	}

	// PING — Discord sends this to verify the endpoint
	if ix.Type == discordTypePing {
		return json.Marshal(DiscordResponse{Type: discordRespPong})
	}

	settings, err := db.Store.GetSettings()
	if err != nil || settings == nil {
		return db.errResp("Configuration error."), nil
	}

	// Resolve user ID (works in guilds and DMs)
	userID := ""
	username := "unknown"
	if ix.Member != nil {
		userID = ix.Member.User.ID
		username = ix.Member.User.Username
	} else if ix.User != nil {
		userID = ix.User.ID
		username = ix.User.Username
	}

	// Log the command
	cmdName := ""
	if ix.Data != nil {
		if ix.Type == discordTypeMessageComponent {
			cmdName = ix.Data.CustomID
		} else {
			cmdName = ix.Data.Name
		}
	}
	db.Store.SaveChatLog("discord-cmd", fmt.Sprintf("[%s] %s: %s", userID, username, cmdName))

	// ── Button component interactions ─────────────────────────────────────
	if ix.Type == discordTypeMessageComponent && ix.Data != nil {
		return db.handleComponent(ix, userID, settings)
	}

	// ── Slash command interactions ────────────────────────────────────────
	if ix.Type == discordTypeApplicationCmd && ix.Data != nil {
		return db.handleSlash(ix, userID, settings)
	}

	return db.errResp("Unknown interaction type."), nil
}

func (db *DiscordBot) handleComponent(ix DiscordInteraction, userID string, settings *models.AppSettings) ([]byte, error) {
	customID := ix.Data.CustomID

	if customID == "cancel" {
		db.takePending(userID)
		return db.textResp("✅ Cancelled.", true), nil
	}

	if strings.HasPrefix(customID, "confirm_") {
		p, ok := db.takePending(userID)
		if !ok || time.Now().After(p.expiresAt) {
			return db.textResp("⏱ Confirmation expired or not found.", true), nil
		}
		// Acknowledge immediately with deferred response, run action async
		go func() {
			p.action()
		}()
		return db.textResp(fmt.Sprintf("⚙️ Running: _%s_…", p.label), false), nil
	}

	return db.errResp("Unknown button."), nil
}

func (db *DiscordBot) handleSlash(ix DiscordInteraction, userID string, settings *models.AppSettings) ([]byte, error) {
	cmd := ix.Data.Name
	serverPfx := ""
	if settings.ServerLabel != "" {
		serverPfx = fmt.Sprintf("**[%s]** ", settings.ServerLabel)
	}

	switch cmd {

	case "help":
		return db.embedResp("🤖 KumoMTA Bot Commands", serverPfx+
			"**📊 Stats**\n`/stats` `/bounces` `/tail`\n\n"+
			"**📬 Queue**\n`/queue` `/flush` `/retry-all` `/drop-bounced`\n\n"+
			"**🌐 Reputation**\n`/reputation` `/check`\n\n"+
			"**🚀 Campaigns**\n`/campaigns` `/pause-campaign` `/resume-campaign`\n\n"+
			"**🔥 Warmup**\n`/warmup`\n\n"+
			"**⚙️ System**\n`/disk` `/mem` `/restart` `/reload`\n\n"+
			"Destructive commands show a Confirm/Cancel button.",
			3447003, false), nil

	case "stats":
		stats, err := GetAllDomainsStats(1)
		if err != nil {
			return db.errResp("Could not fetch stats."), nil
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
		color := 3066993
		if rate < 80 {
			color = 15158332
		} else if rate < 95 {
			color = 16776960
		}
		embed := DiscordBotEmbed{
			Title:       "📊 Today's Stats",
			Description: serverPfx + time.Now().Format("02 Jan 2006 15:04"),
			Color:       color,
			Fields: []DiscordEmbedField{
				{Name: "📤 Sent", Value: fmt.Sprintf("`%d`", sent), Inline: true},
				{Name: "✅ Delivered", Value: fmt.Sprintf("`%d`", delivered), Inline: true},
				{Name: "❌ Bounced", Value: fmt.Sprintf("`%d`", bounced), Inline: true},
				{Name: "⏳ Deferred", Value: fmt.Sprintf("`%d`", deferred), Inline: true},
				{Name: "📈 Rate", Value: fmt.Sprintf("`%.1f%%`", rate), Inline: true},
			},
			Footer:    &DiscordEmbedFooter{Text: "KumoMTA UI"},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		return db.embedFromStruct(embed, false), nil

	case "queue":
		msgs, err := GetQueueMessages(1000)
		if err != nil {
			return db.errResp("Could not read queue."), nil
		}
		domains := map[string]int{}
		for _, m := range msgs {
			domain := m.Recipient
			if i := strings.LastIndex(m.Recipient, "@"); i != -1 {
				domain = m.Recipient[i+1:]
			}
			domains[domain]++
		}
		lines := []string{fmt.Sprintf("%s**Queue — %d message(s)**\n", serverPfx, len(msgs))}
		for domain, count := range domains {
			lines = append(lines, fmt.Sprintf("• `%s`: %d", domain, count))
		}
		return db.textResp(strings.Join(lines, "\n"), false), nil

	case "bounces":
		events, _, err := db.Store.ListDeliveryEvents(1, 200, store.DeliveryEventFilter{
			EventType: "Bounce", Hours: 24,
		})
		if err != nil {
			return db.errResp("Could not fetch bounces."), nil
		}
		if len(events) == 0 {
			return db.textResp(serverPfx+"✅ No bounces in the last 24h!", false), nil
		}
		domains := map[string]int{}
		for _, e := range events {
			domains[e.Domain]++
		}
		lines := []string{fmt.Sprintf("%s📉 **Bounces (24h) — %d total**\n", serverPfx, len(events))}
		for domain, count := range domains {
			lines = append(lines, fmt.Sprintf("• `%s`: %d", domain, count))
		}
		return db.textResp(strings.Join(lines, "\n"), false), nil

	case "tail":
		n := 20
		if len(ix.Data.Options) > 0 {
			if v, ok := ix.Data.Options[0].Value.(float64); ok {
				n = int(v)
				if n < 1 {
					n = 1
				}
				if n > 50 {
					n = 50
				}
			}
		}
		out, err := exec.Command("journalctl", "-u", "kumomta", "-n", strconv.Itoa(n), "--no-pager", "--output=short").CombinedOutput()
		if err != nil {
			return db.errResp("Could not read logs."), nil
		}
		return db.textResp(serverPfx+"```\n"+truncate(string(out), 1900)+"\n```", false), nil

	case "reputation":
		rows, err := db.Store.GetLatestReputationChecks()
		if err != nil || len(rows) == 0 {
			return db.textResp(serverPfx+"⚠️ No reputation data. Use `/check` to run a scan.", false), nil
		}
		var listed, clean []string
		for _, r := range rows {
			if r.Status == "listed" {
				listed = append(listed, fmt.Sprintf("🔴 `%s` → *%s*", r.Target, r.ListedOn))
			} else {
				clean = append(clean, r.Target)
			}
		}
		color := 3066993
		desc := serverPfx + fmt.Sprintf("✅ All %d targets are clean!", len(clean))
		if len(listed) > 0 {
			color = 15158332
			desc = serverPfx + fmt.Sprintf("🚨 **%d/%d targets blacklisted**\n\n%s\n\n✅ Clean: %d",
				len(listed), len(rows), strings.Join(listed, "\n"), len(clean))
		}
		return db.embedResp("🌐 Reputation Check", desc, color, false), nil

	case "check":
		// Deferred — this takes a few seconds
		db.setPending(userID, &discordPending{
			label:     "reputation scan",
			expiresAt: time.Now().Add(5 * time.Minute),
			action: func() {
				results, err := CheckReputation(db.Store)
				if err != nil {
					db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, "❌ Scan failed: "+err.Error())
					return
				}
				listed := 0
				for _, r := range results {
					if r.Status == "listed" {
						listed++
					}
				}
				if listed == 0 {
					db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID,
						serverPfx+fmt.Sprintf("✅ Scan complete — all %d targets are clean!", len(results)))
				} else {
					db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID,
						serverPfx+fmt.Sprintf("🚨 **%d/%d targets blacklisted!** Use `/reputation` for details.", listed, len(results)))
				}
			},
		})
		p, _ := db.takePending(userID) // take it back, run immediately
		resp, _ := json.Marshal(DiscordResponse{Type: discordRespDeferredMsg})
		go p.action()
		return resp, nil

	case "campaigns":
		var campaigns []models.Campaign
		if err := db.Store.DB.Order("created_at desc").Limit(10).Find(&campaigns).Error; err != nil {
			return db.errResp("Could not fetch campaigns."), nil
		}
		if len(campaigns) == 0 {
			return db.textResp(serverPfx+"No campaigns found.", false), nil
		}
		icons := map[string]string{"sending": "▶️", "paused": "⏸", "completed": "✅", "failed": "❌", "draft": "📝", "scheduled": "⏰"}
		lines := []string{serverPfx + "**Campaigns (last 10)**\n"}
		for _, c := range campaigns {
			icon := icons[c.Status]
			if icon == "" {
				icon = "•"
			}
			lines = append(lines, fmt.Sprintf("%s `[%d]` **%s** — %s", icon, c.ID, c.Name, c.Status))
		}
		return db.textResp(strings.Join(lines, "\n"), false), nil

	case "resume-campaign":
		idVal := db.optFloat(ix.Data.Options, "id")
		var c models.Campaign
		if err := db.Store.DB.First(&c, uint(idVal)).Error; err != nil {
			return db.errResp("Campaign not found."), nil
		}
		c.Status = "sending"
		db.Store.DB.Save(&c)
		return db.textResp(serverPfx+fmt.Sprintf("▶️ Campaign `%d` **%s** resumed.", c.ID, c.Name), false), nil

	case "warmup":
		domains, err := db.Store.ListDomains()
		if err != nil {
			return db.errResp("Could not fetch domains."), nil
		}
		lines := []string{serverPfx + "**🔥 Warmup Status**\n"}
		for _, d := range domains {
			senders, _ := db.Store.ListSendersByDomain(d.ID)
			for _, s := range senders {
				if s.WarmupPlan == "" {
					continue
				}
				lines = append(lines, fmt.Sprintf("• `%s` — plan: *%s*, day %d → %s/day",
					s.Email, s.WarmupPlan, s.WarmupDay, GetSenderRate(s)))
			}
		}
		if len(lines) == 1 {
			return db.textResp(serverPfx+"No senders on warmup.", false), nil
		}
		return db.textResp(strings.Join(lines, "\n"), false), nil

	case "disk":
		out, _ := exec.Command("df", "-h", "/").CombinedOutput()
		return db.textResp(serverPfx+"**💾 Disk Usage**\n```\n"+string(out)+"```", false), nil

	case "mem":
		out, _ := exec.Command("free", "-h").CombinedOutput()
		return db.textResp(serverPfx+"**🖥 Memory**\n```\n"+string(out)+"```", false), nil

	// ── Destructive — require button confirmation ─────────────────────────

	case "flush":
		return db.confirmButtons(userID, "flush", "Flush all deferred messages",
			func() { db.runFlush(settings, ix, serverPfx) }), nil

	case "retry-all":
		return db.confirmButtons(userID, "retry_all", "Force retry all deferred messages",
			func() { db.runRetryAll(settings, ix, serverPfx) }), nil

	case "drop-bounced":
		return db.confirmButtons(userID, "drop_bounced", "Drop all bounced/failed messages from queue",
			func() { db.runDropBounced(settings, ix, serverPfx) }), nil

	case "pause-campaign":
		idVal := db.optFloat(ix.Data.Options, "id")
		return db.confirmButtons(userID, fmt.Sprintf("pause_campaign_%d", int(idVal)),
			fmt.Sprintf("Pause campaign %d", int(idVal)),
			func() { db.runPauseCampaign(settings, ix, int(idVal), serverPfx) }), nil

	case "reload":
		return db.confirmButtons(userID, "reload", "Reload KumoMTA config",
			func() { db.runReload(settings, ix, serverPfx) }), nil

	case "restart":
		return db.confirmButtons(userID, "restart", "RESTART KumoMTA service (brief interruption)",
			func() { db.runRestart(settings, ix, serverPfx) }), nil
	}

	return db.errResp("Unknown command."), nil
}

// ─── Destructive action runners ───────────────────────────────────────────────

func (db *DiscordBot) runFlush(settings *models.AppSettings, ix DiscordInteraction, pfx string) {
	if _, err := kumoAPI("POST", "/api/admin/rebind", map[string]interface{}{"data": map[string]interface{}{"queue": nil}}); err != nil {
		exec.Command("systemctl", "try-restart", "kumomta").Run()
		db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"✅ Flush triggered via service restart.")
		return
	}
	db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"✅ Deferred messages flushed.")
}

func (db *DiscordBot) runRetryAll(settings *models.AppSettings, ix DiscordInteraction, pfx string) {
	out, err := kumoAPI("POST", "/api/admin/rebind", map[string]interface{}{"trigger_rebind": true})
	if err != nil {
		db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"❌ Failed: "+err.Error())
		return
	}
	db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"✅ Retry all triggered.\n```\n"+truncate(string(out), 300)+"```")
}

func (db *DiscordBot) runDropBounced(settings *models.AppSettings, ix DiscordInteraction, pfx string) {
	msgs, err := GetQueueMessages(5000)
	if err != nil {
		db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"❌ Could not read queue.")
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
	db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+fmt.Sprintf("🗑 Dropped `%d` bounced messages.", dropped))
}

func (db *DiscordBot) runPauseCampaign(settings *models.AppSettings, ix DiscordInteraction, id int, pfx string) {
	var c models.Campaign
	if err := db.Store.DB.First(&c, id).Error; err != nil {
		db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"❌ Campaign not found.")
		return
	}
	c.Status = "paused"
	db.Store.DB.Save(&c)
	db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+fmt.Sprintf("⏸ Campaign `%d` **%s** paused.", c.ID, c.Name))
}

func (db *DiscordBot) runReload(settings *models.AppSettings, ix DiscordInteraction, pfx string) {
	out, err := exec.Command("systemctl", "reload-or-restart", "kumomta").CombinedOutput()
	if err != nil {
		db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"❌ Reload failed:\n```\n"+string(out)+"```")
		return
	}
	db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"✅ KumoMTA config reloaded.")
}

func (db *DiscordBot) runRestart(settings *models.AppSettings, ix DiscordInteraction, pfx string) {
	out, err := exec.Command("systemctl", "restart", "kumomta").CombinedOutput()
	if err != nil {
		db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"❌ Restart failed:\n```\n"+string(out)+"```")
		return
	}
	db.followUp(settings.DiscordBotToken, ix.Token, ix.ApplicationID, pfx+"✅ KumoMTA restarted.")
}

// ─── Follow-up messages (for deferred/async responses) ───────────────────────

// followUp sends a follow-up message to a deferred interaction.
func (db *DiscordBot) followUp(botToken, interactionToken, appID, content string) {
	if botToken == "" || interactionToken == "" || appID == "" {
		return
	}
	url := fmt.Sprintf("https://discord.com/api/v10/webhooks/%s/%s", appID, interactionToken)
	payload, _ := json.Marshal(map[string]string{"content": content})
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+botToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// ─── Slash command registration ───────────────────────────────────────────────

// RegisterSlashCommands registers (or updates) all slash commands with Discord.
// Call this once after configuring the bot, or on settings save.
func RegisterSlashCommands(appID, botToken string) error {
	commands := []map[string]interface{}{
		{"name": "help", "description": "Show all available KumoMTA bot commands"},
		{"name": "stats", "description": "Today's delivery stats"},
		{"name": "queue", "description": "Current queue depth, broken down by domain"},
		{"name": "bounces", "description": "Bounce summary for the last 24 hours"},
		{
			"name":        "tail",
			"description": "Last N KumoMTA log lines",
			"options": []map[string]interface{}{
				{"name": "lines", "description": "Number of lines (1–50)", "type": 4 /*INTEGER*/, "required": false},
			},
		},
		{"name": "reputation", "description": "Latest blacklist check results"},
		{"name": "check", "description": "Run a fresh DNSBL reputation scan"},
		{"name": "campaigns", "description": "List last 10 campaigns with status"},
		{
			"name":        "pause-campaign",
			"description": "Pause a running campaign",
			"options": []map[string]interface{}{
				{"name": "id", "description": "Campaign ID", "type": 4 /*INTEGER*/, "required": true},
			},
		},
		{
			"name":        "resume-campaign",
			"description": "Resume a paused campaign",
			"options": []map[string]interface{}{
				{"name": "id", "description": "Campaign ID", "type": 4 /*INTEGER*/, "required": true},
			},
		},
		{"name": "warmup", "description": "Warmup status per sender"},
		{"name": "disk", "description": "Disk usage"},
		{"name": "mem", "description": "Memory and CPU overview"},
		{"name": "flush", "description": "⚠️ Flush all deferred messages (requires confirmation)"},
		{"name": "retry-all", "description": "⚠️ Force retry all deferred messages (requires confirmation)"},
		{"name": "drop-bounced", "description": "⚠️ Drop all bounced messages from queue (requires confirmation)"},
		{"name": "reload", "description": "⚠️ Reload KumoMTA config (requires confirmation)"},
		{"name": "restart", "description": "⚠️ Restart KumoMTA service (requires confirmation)"},
	}

	payload, _ := json.Marshal(commands)
	url := fmt.Sprintf("https://discord.com/api/v10/applications/%s/commands", appID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+botToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord API returned %d when registering commands", resp.StatusCode)
	}
	return nil
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func (db *DiscordBot) textResp(content string, ephemeral bool) []byte {
	flags := 0
	if ephemeral {
		flags = discordFlagEphemeral
	}
	b, _ := json.Marshal(DiscordResponse{
		Type: discordRespChannelMessage,
		Data: &DiscordResponseData{Content: content, Flags: flags},
	})
	return b
}

func (db *DiscordBot) embedResp(title, desc string, color int, ephemeral bool) []byte {
	flags := 0
	if ephemeral {
		flags = discordFlagEphemeral
	}
	b, _ := json.Marshal(DiscordResponse{
		Type: discordRespChannelMessage,
		Data: &DiscordResponseData{
			Embeds: []DiscordBotEmbed{{Title: title, Description: desc, Color: color,
				Footer: &DiscordEmbedFooter{Text: "KumoMTA UI"}, Timestamp: time.Now().Format(time.RFC3339)}},
			Flags: flags,
		},
	})
	return b
}

func (db *DiscordBot) embedFromStruct(embed DiscordBotEmbed, ephemeral bool) []byte {
	flags := 0
	if ephemeral {
		flags = discordFlagEphemeral
	}
	b, _ := json.Marshal(DiscordResponse{
		Type: discordRespChannelMessage,
		Data: &DiscordResponseData{Embeds: []DiscordBotEmbed{embed}, Flags: flags},
	})
	return b
}

func (db *DiscordBot) errResp(msg string) []byte {
	return db.textResp("❌ "+msg, true)
}

func (db *DiscordBot) confirmButtons(userID, actionID, description string, action func()) []byte {
	db.setPending(userID, &discordPending{
		action:    action,
		label:     description,
		expiresAt: time.Now().Add(60 * time.Second),
	})
	b, _ := json.Marshal(DiscordResponse{
		Type: discordRespChannelMessage,
		Data: &DiscordResponseData{
			Content: fmt.Sprintf("⚠️ **Confirm action:**\n\n*%s*\n\nThis will expire in 60 seconds.", description),
			Flags:   discordFlagEphemeral, // only the user who typed it sees this
			Components: []DiscordComponent{{
				Type: 1, // ActionRow
				Components: []DiscordComponentItem{
					{Type: 2, Label: "✅ Confirm", Style: 3, CustomID: "confirm_" + actionID},
					{Type: 2, Label: "❌ Cancel", Style: 4, CustomID: "cancel"},
				},
			}},
		},
	})
	return b
}

func (db *DiscordBot) optFloat(opts []DiscordInteractionOpt, name string) float64 {
	for _, o := range opts {
		if o.Name == name {
			if v, ok := o.Value.(float64); ok {
				return v
			}
		}
	}
	return 0
}
