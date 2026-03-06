package models

import "time"

// Represents global application settings.
type AppSettings struct {
	ID uint `gorm:"primaryKey" json:"id"`

	MainHostname string `json:"main_hostname"`
	MainServerIP string `json:"main_server_ip"`
	MailWizzIP   string `json:"mailwizz_ip"` // optional relay IP

	// NEW: Listener Binding (e.g., "127.0.0.1:25" or "0.0.0.0:25")
	SMTPListenAddr string `json:"smtp_listen_addr"`

	// TLS Certificate paths for SMTP AUTH
	TLSCertPath string `json:"tls_cert_path"` // e.g., /etc/ssl/certs/mail.crt
	TLSKeyPath  string `json:"tls_key_path"`  // e.g., /etc/ssl/private/mail.key

	AIProvider string `json:"ai_provider"` // "openai", "deepseek"
	AIAPIKey   string `json:"ai_api_key"`  // encrypted or blank

	// Webhook Settings
	WebhookURL     string  `json:"webhook_url"`
	WebhookEnabled bool    `json:"webhook_enabled"`
	BounceAlertPct float64 `json:"bounce_alert_pct"`

	// List Cleaner Proxy
	ProxyURL string `json:"proxy_url"` // SOCKS5 or HTTP proxy for fallback

	// CORS
	AllowedOrigins string `json:"allowed_origins"` // Comma-separated list of origins

	// WhatsApp Configuration
	WhatsAppPhoneNumberID string `json:"whatsapp_phone_number_id"`
	WhatsAppAccessToken   string `json:"whatsapp_access_token"` // Should be encrypted
	WhatsAppVerifyToken   string `json:"whatsapp_verify_token"`

	// Telegram Bot
	TelegramBotToken   string `json:"telegram_bot_token"`   // from @BotFather
	TelegramChatID     string `json:"telegram_chat_id"`     // comma-separated chat/group IDs allowed to use bot
	TelegramEnabled    bool   `json:"telegram_enabled"`
	TelegramDigestHour int    `json:"telegram_digest_hour"` // 0-23, default 8

	// Discord Webhook (outgoing alerts/digest)
	DiscordWebhookURL string `json:"discord_webhook_url"`
	DiscordEnabled    bool   `json:"discord_enabled"`

	// Discord Bot (interactive slash commands — requires a Discord Application)
	DiscordBotToken       string `json:"discord_bot_token"`       // Bot token from Discord developer portal
	DiscordApplicationID  string `json:"discord_application_id"`  // Application / Client ID
	DiscordPublicKey      string `json:"discord_public_key"`      // Ed25519 public key for verifying interactions
	DiscordBotEnabled     bool   `json:"discord_bot_enabled"`

	// Server identity (used to label bot messages when managing multiple VPS)
	ServerLabel string `json:"server_label"` // e.g. "NYC-01", "LA-02"
}

// A domain managed by the system
type Domain struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex" json:"name"`

	MailHost   string `json:"mail_host"`
	BounceHost string `json:"bounce_host"`

	// DMARC Settings
	DMARCPolicy     string `json:"dmarc_policy"`     // none, quarantine, reject
	DMARCRua        string `json:"dmarc_rua"`        // Aggregate report email
	DMARCRuf        string `json:"dmarc_ruf"`        // Forensic report email
	DMARCPercentage int    `json:"dmarc_percentage"` // 0-100

	Senders []Sender `gorm:"constraint:OnDelete:CASCADE" json:"senders"`
}

// AdminUser represents a panel admin account
type AdminUser struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Email        string `gorm:"uniqueIndex" json:"email"`
	PasswordHash string `json:"-"` // Never return hash

	// 2FA Support
	TwoFactorSecret  string `json:"-"`
	TwoFactorEnabled bool   `json:"has_2fa"`

	// User Preferences
	Theme string `json:"theme"`
}

// Auth Sessions for multi-device support
type AuthSession struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AdminID   uint      `gorm:"index" json:"admin_id"`
	Token     string    `gorm:"uniqueIndex" json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	DeviceIP  string    `json:"device_ip"`
	UserAgent string    `json:"user_agent"`
}

// BounceAccount represents a system user for handling bounced emails
type BounceAccount struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex" json:"username"`
	PasswordHash string `json:"-"`
	Domain       string `json:"domain"`
	Notes        string `json:"notes"`
}

// A sender identity associated with a domain
type Sender struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	DomainID uint `gorm:"index" json:"domain_id"`
	Domain   Domain `json:"-" gorm:"foreignKey:DomainID"` // Relation for Warmup Logic

	LocalPart    string `json:"local_part"`
	Email        string `json:"email"`
	IP           string `json:"ip"` // specific IP for this sender
	SMTPPassword string `json:"smtp_password,omitempty"`
	
	BounceUsername string `json:"bounce_username"`

	// Warmup State (NEW)
	WarmupEnabled    bool      `json:"warmup_enabled"`
	WarmupPlan       string    `json:"warmup_plan"`        // "conservative", "standard"
	WarmupDay        int       `json:"warmup_day"`         // 1, 2, 3...
	WarmupLastUpdate time.Time `json:"warmup_last_update"` // Last time we bumped the day

	// Virtual field for DKIM check (computed at runtime)
	HasDKIM bool `gorm:"-" json:"has_dkim"` 
}

// Inventory of IPs available on the server
type SystemIP struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Value     string    `gorm:"uniqueIndex" json:"value"` // IPv4 address
	Netmask   string    `json:"netmask"`                  // e.g. /24
	Interface string    `json:"interface"`                // e.g. eth0 (optional)
	CreatedAt time.Time `json:"created_at"`

	// Virtual Field: True if this IP is actually configured on the OS
	IsActive  bool      `gorm:"-" json:"is_active"` 
}

// EmailStats stores aggregated sending statistics
type EmailStats struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"index" json:"domain"`
	Date      time.Time `gorm:"index" json:"date"` // Date only (no time)
	Sent      int64     `json:"sent"`
	Delivered int64     `json:"delivered"`
	Bounced   int64     `json:"bounced"`
	Deferred  int64     `json:"deferred"`
	UpdatedAt time.Time `json:"updated_at"`
}

// QueueMessage represents a message in the mail queue (RESTORED)
type QueueMessage struct {
	ID          string    `json:"id"`
	Sender      string    `json:"sender"`
	Recipient   string    `json:"recipient"`
	Subject     string    `json:"subject"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"` // queued, deferred, etc.
	Attempts    int       `json:"attempts"`
	LastAttempt time.Time `json:"last_attempt"`
	NextRetry   time.Time `json:"next_retry"`
	ErrorMsg    string    `json:"error_msg"`
}

// WebhookLog stores webhook delivery history
type WebhookLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	EventType string    `json:"event_type"` // bounce_alert, daily_summary
	Payload   string    `json:"payload"`    // JSON payload sent
	Status    int       `json:"status"`     // HTTP status code
	Response  string    `json:"response"`   // Response body
	CreatedAt time.Time `json:"created_at"`
}

// APIKey for external applications (NEW)
type APIKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`                   // e.g. "EmailVerifier App"
	Key       string    `gorm:"uniqueIndex" json:"key"` // The secret token
	Scopes    string    `json:"scopes"`                 // e.g. "verify,relay"
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

// ChatLog stores AI conversation history (NEW)
type ChatLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Role      string    `json:"role"` // "user", "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ContactList represents a managed list of contacts
type ContactList struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Contacts  []Contact `json:"contacts,omitempty" gorm:"foreignKey:ListID"`
}

// Contact represents a single person/lead
type Contact struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ListID    uint      `gorm:"index" json:"list_id"`
	Email     string    `gorm:"index" json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`

	// Validation Status
	IsValid   bool      `json:"is_valid"`
	RiskScore int       `json:"risk_score"` // 0-100 (0=safe)
	VerifyLog string    `json:"verify_log"` // "MX ok, SMTP failed"

	// Engagement (AI Superlead data)
	Score     int       `json:"score"`      // Lead Score
	TotalOpens int      `json:"total_opens"`
	TotalClicks int     `json:"total_clicks"`

	CreatedAt time.Time `json:"created_at"`
}

// Campaign represents a bulk email job
type Campaign struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`          // HTML Content
	SenderID    uint      `json:"sender_id"`     // From which Sender identity
	Sender      Sender    `json:"-" gorm:"foreignKey:SenderID"`

	Status      string    `json:"status"`        // "draft", "scheduled", "sending", "completed", "failed"
	ScheduledAt *time.Time `json:"scheduled_at"` // Nullable

	TotalSent   int       `json:"total_sent"`
	TotalFailed int       `json:"total_failed"`
	TotalOpens  int       `json:"total_opens"`
	TotalClicks int       `json:"total_clicks"`

	CreatedAt   time.Time `json:"created_at"`
	Recipients  []CampaignRecipient `json:"recipients,omitempty" gorm:"foreignKey:CampaignID"`
}

// CampaignRecipient tracks individual status in a campaign
type CampaignRecipient struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CampaignID uint      `gorm:"index" json:"campaign_id"`
	Email      string    `gorm:"index" json:"email"`
	ContactID  uint      `gorm:"index" json:"contact_id"` // Optional link to persistent contact

	Status     string    `json:"status"` // "pending", "sent", "failed"
	Error      string    `json:"error,omitempty"`
	SentAt     time.Time `json:"sent_at,omitempty"`

	// Tracking
	OpenedAt   *time.Time `json:"opened_at"`
	ClickedAt  *time.Time `json:"clicked_at"`
}

// AutomationWorkflow represents a visual automation flow
type AutomationWorkflow struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Trigger   string    `json:"trigger"` // e.g. "contact_added", "email_opened"
	StepsJSON string    `json:"steps"`   // JSON array of steps (React Flow format)
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// SuppressedEmail is a global suppression list entry - never send to these addresses
type SuppressedEmail struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Email      string    `gorm:"uniqueIndex" json:"email"`
	Reason     string    `json:"reason"`      // "hard_bounce", "spam_complaint", "manual", "unsubscribe"
	Domain     string    `gorm:"index" json:"domain"` // extracted domain for filtering
	SourceInfo string    `json:"source_info"` // e.g. "campaign_id:42", "api", "import"
	CreatedAt  time.Time `json:"created_at"`
}

// AlertRule defines a threshold-based monitoring rule that fires notifications
type AlertRule struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`        // "bounce_rate","delivery_rate","queue_depth","blacklist"
	Threshold   float64    `json:"threshold"`   // e.g. 5.0 for 5%
	Operator    string     `json:"operator"`    // "gt" or "lt"
	Channel     string     `json:"channel"`     // "slack","webhook","email"
	Destination string     `json:"destination"` // Slack URL, email, or webhook URL
	IsEnabled   bool       `json:"is_enabled"`
	LastFired   *time.Time `json:"last_fired"`
	CooldownMin int        `json:"cooldown_min"` // Minutes before same alert fires again
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AlertEvent is a historical log of fired alerts
type AlertEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RuleID    uint      `gorm:"index" json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Message   string    `json:"message"`
	Value     float64   `json:"value"`     // Actual metric value that triggered
	Threshold float64   `json:"threshold"` // Threshold at time of firing
	Channel   string    `json:"channel"`
	Status    string    `json:"status"` // "sent", "failed"
	Error     string    `json:"error,omitempty"`
	FiredAt   time.Time `json:"fired_at"`
}

// BIMIRecord stores per-domain BIMI (Brand Indicators for Message Identification) config
type BIMIRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex" json:"domain"`
	LogoURL   string    `json:"logo_url"` // HTTPS URL to SVG logo
	VMCURL    string    `json:"vmc_url"`  // Verified Mark Certificate URL (optional)
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MTASTSPolicy stores per-domain MTA-STS policy configuration
type MTASTSPolicy struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex" json:"domain"`
	Mode      string    `json:"mode"`    // "none", "testing", "enforce"
	MaxAge    int       `json:"max_age"` // seconds (e.g. 86400)
	MXHosts   string    `json:"mx_hosts"` // newline-separated MX hostnames
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WarmupLog records warmup events (advancement, pauses, plan changes)
type WarmupLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SenderID  uint      `gorm:"index" json:"sender_id"`
	Email     string    `json:"email"`
	Event     string    `json:"event"`   // "advanced","paused","resumed","plan_changed","completed"
	OldDay    int       `json:"old_day"`
	NewDay    int       `json:"new_day"`
	OldRate   string    `json:"old_rate"`
	NewRate   string    `json:"new_rate"`
	Reason    string    `json:"reason"` // optional pause reason
	CreatedAt time.Time `json:"created_at"`
}

// WhatsAppMessage represents a message sent via WhatsApp Cloud API
type WhatsAppMessage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ContactID   uint      `gorm:"index" json:"contact_id"`
	ToNumber    string    `json:"to_number"`
	Body        string    `json:"body"`
	Status      string    `json:"status"` // sent, delivered, read
	MessageSID  string    `json:"message_sid"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReputationCheck stores the result of a single blacklist/reputation check
// for an IP address or sending domain.
type ReputationCheck struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Target     string    `gorm:"index" json:"target"`      // IP or domain name
	TargetType string    `json:"target_type"`              // "ip" or "domain"
	Status     string    `json:"status"`                   // "clean", "listed", "error"
	ListedOn   string    `json:"listed_on"`                // comma-separated RBL names where listed
	CheckedAt  time.Time `gorm:"index" json:"checked_at"`
}

// RemoteServer stores connection details for a remote f-kumo instance.
type RemoteServer struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex" json:"name"` // e.g. "VPS-2 (Frankfurt)"
	URL       string    `json:"url"`                     // e.g. "https://vps2.example.com:8080"
	APIToken  string    `json:"api_token"`               // Bearer token for the remote instance
	LastSeen  time.Time `json:"last_seen"`
	Status    string    `json:"status"`                  // "online", "offline", "unknown"
	CreatedAt time.Time `json:"created_at"`
}

// TrafficShapingRule defines per-ISP throttle limits stored in the DB
type TrafficShapingRule struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	Provider             string    `json:"provider"`               // Display name: "Gmail", "Yahoo", "Outlook", "Custom"
	Pattern              string    `json:"pattern"`                // MX hostname pattern: "google.com", "yahoodns.net"
	MaxMessageRate       string    `json:"max_message_rate"`       // e.g. "50/h", "100/min"
	MaxConnectionRate    string    `json:"max_connection_rate"`    // e.g. "5/min"
	MaxDeliveriesPerConn int       `json:"max_deliveries_per_conn"` // e.g. 20
	ConnectionLimit      int       `json:"connection_limit"`       // e.g. 3
	RetrySchedule        string    `json:"retry_schedule"`         // e.g. "5m,15m,30m,1h,4h"
	IsEnabled            bool      `json:"is_enabled"`
	Notes                string    `json:"notes"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// IPPool groups IPs for different sending purposes
type IPPool struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `gorm:"uniqueIndex" json:"name"` // e.g. "transactional"
	Description string        `json:"description"`
	Members     []IPPoolMember `gorm:"foreignKey:PoolID;constraint:OnDelete:CASCADE" json:"members"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// IPPoolMember associates an IP address with a pool
type IPPoolMember struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	PoolID  uint   `gorm:"index" json:"pool_id"`
	IPValue string `json:"ip_value"` // e.g. "192.168.1.10"
}

// DeliveryEvent stores per-recipient failure events parsed from KumoMTA logs.
// Only Bounce and TransientFailure events are stored (not every delivery).
type DeliveryEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
	EventType string    `gorm:"index" json:"event_type"` // "Bounce" or "TransientFailure"
	Sender    string    `gorm:"index" json:"sender"`
	Recipient string    `gorm:"index" json:"recipient"`
	Domain    string    `gorm:"index" json:"domain"`   // extracted from recipient
	ErrorCode int       `json:"error_code"`             // SMTP code e.g. 550, 421
	ErrorMsg  string    `json:"error_msg"`              // full SMTP response message
	Provider  string    `gorm:"index" json:"provider"`  // "Gmail", "Outlook", "Yahoo", etc.
}
