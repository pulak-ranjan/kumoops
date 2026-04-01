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

	AIProvider string `json:"ai_provider"` // "openai", "anthropic", "gemini", "groq", "mistral", "together", "deepseek", "ollama"
	AIAPIKey   string `json:"ai_api_key"`  // encrypted or blank
	// Ollama (local / self-hosted)
	OllamaBaseURL string `json:"ollama_base_url"` // e.g. "http://localhost:11434"
	OllamaModel   string `json:"ollama_model"`    // e.g. "llama3.2", "mistral", "phi3"

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

	// ISP Intelligence — Google Postmaster Tools
	GooglePostmasterEnabled     bool   `json:"google_postmaster_enabled"`
	GooglePostmasterCredentials string `json:"google_postmaster_credentials"` // encrypted service account JSON

	// ISP Intelligence — Microsoft SNDS
	MicrosoftSNDSKey string `json:"microsoft_snds_key"` // encrypted SNDS API key

	// Adaptive Throttling
	AdaptiveThrottlingEnabled bool `json:"adaptive_throttling_enabled"`

	// Anomaly Detection
	AnomalyDetectionEnabled bool `json:"anomaly_detection_enabled"`

	// Tracking overrides
	TrackingBaseURL string `json:"tracking_base_url"` // explicit base URL for open/click pixels; falls back to MainHostname

	// SMTP Relay (KumoOps listens on port 587, auth via API keys, injects to KumoMTA)
	SMTPRelayEnabled bool   `json:"smtp_relay_enabled"`
	SMTPRelayPort    int    `json:"smtp_relay_port"` // default 587
	SMTPRelayHost    string `json:"smtp_relay_host"` // binding address, default "0.0.0.0"
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
	TextBody    string    `json:"text_body"`     // Plain-text alternative
	SenderID    uint      `json:"sender_id"`     // From which Sender identity
	Sender      Sender    `json:"-" gorm:"foreignKey:SenderID"`

	Status      string    `json:"status"`        // "draft", "scheduled", "sending", "completed", "failed", "paused"
	ScheduledAt *time.Time `json:"scheduled_at"` // Nullable

	TotalSent   int       `json:"total_sent"`
	TotalFailed int       `json:"total_failed"`
	TotalOpens  int       `json:"total_opens"`
	TotalClicks int       `json:"total_clicks"`
	TotalUnsubs int       `json:"total_unsubs"`

	// A/B Testing
	IsABTest          bool   `json:"is_ab_test"`
	ABWinAfterHours   int    `json:"ab_win_after_hours"`    // 0 = manual selection
	ABWinMetric       string `json:"ab_win_metric"`         // "open_rate" or "click_rate"
	ABWinnerVariantID *uint  `json:"ab_winner_variant_id"`  // nil until winner decided

	CreatedAt   time.Time `json:"created_at"`
	Recipients  []CampaignRecipient `json:"recipients,omitempty" gorm:"foreignKey:CampaignID"`
	Variants    []CampaignVariant   `json:"variants,omitempty"   gorm:"foreignKey:CampaignID"`
}

// CampaignVariant is one variant in an A/B test campaign
type CampaignVariant struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	CampaignID uint       `gorm:"index" json:"campaign_id"`
	Name       string     `json:"name"`       // "A", "B", "C"
	Subject    string     `json:"subject"`    // variant subject line
	HTMLBody   string     `json:"html_body"`  // variant body (empty = use Campaign.Body)
	SplitPct   float64    `json:"split_pct"`  // fraction 0.0–1.0 of recipients for this variant
	SentCount  int64      `json:"sent_count"`
	OpenCount  int64      `json:"open_count"`
	ClickCount int64      `json:"click_count"`
	IsWinner   bool       `json:"is_winner"`
	WinnerAt   *time.Time `json:"winner_at"`
	CreatedAt  time.Time  `json:"created_at"`
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

	// A/B Testing
	VariantID  *uint `json:"variant_id"` // which variant was sent (nil = no A/B test)

	// Tracking
	OpenedAt  *time.Time `json:"opened_at"`
	ClickedAt *time.Time `json:"clicked_at"`

	// Unsubscribe
	UnsubToken      string     `gorm:"index" json:"unsub_token"`
	UnsubscribedAt  *time.Time `json:"unsubscribed_at"`
}

// SeedMailbox is an IMAP mailbox used for inbox placement testing
type SeedMailbox struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	ISP      string    `json:"isp"`       // "Gmail", "Yahoo", "Outlook", "Apple", "Other"
	Email    string    `gorm:"uniqueIndex" json:"email"`
	IMAPHost string    `json:"imap_host"` // e.g. "imap.gmail.com"
	IMAPPort int       `json:"imap_port"` // e.g. 993
	Username string    `json:"username"`  // usually same as email
	Password string    `json:"password"`  // encrypted via Encrypt()
	IsActive bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// PlacementTest is a triggered inbox placement test run
type PlacementTest struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `json:"name"`
	Subject     string     `json:"subject"`
	SenderID    uint       `json:"sender_id"`
	HTMLBody    string     `json:"html_body"`
	Status      string     `json:"status"` // "pending", "running", "completed", "failed"
	InboxRate   float64    `json:"inbox_rate"`   // % landed in inbox
	SpamRate    float64    `json:"spam_rate"`    // % in spam/junk
	MissingRate float64    `json:"missing_rate"` // % not found within timeout
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	Results     []PlacementResult `json:"results,omitempty" gorm:"foreignKey:TestID"`
}

// PlacementResult is the per-seed-mailbox result of a placement test
type PlacementResult struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	TestID        uint       `gorm:"index" json:"test_id"`
	SeedMailboxID uint       `json:"seed_mailbox_id"`
	ISP           string     `json:"isp"`
	Email         string     `json:"email"`
	Placement     string     `json:"placement"`    // "inbox", "spam", "missing", "error"
	InboxFolder   string     `json:"inbox_folder"` // actual IMAP folder found in
	MessageID     string     `json:"message_id"`   // Message-ID header for matching
	ReceivedAt    *time.Time `json:"received_at"`
	CheckedAt     time.Time  `json:"checked_at"`
	ErrorMsg      string     `json:"error_msg,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
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

// DeliveryEvent stores per-recipient events parsed from KumoMTA logs.
type DeliveryEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
	EventType   string    `gorm:"index" json:"event_type"` // Bounce, TransientFailure, Delivery, Expiration, OOB
	Sender      string    `gorm:"index" json:"sender"`
	Recipient   string    `gorm:"index" json:"recipient"`
	Domain      string    `gorm:"index" json:"domain"`          // extracted from recipient
	ErrorCode   int       `json:"error_code"`                    // SMTP code e.g. 550, 421, 250
	ErrorMsg    string    `json:"error_msg"`                     // full SMTP response message
	Provider    string    `gorm:"index" json:"provider"`         // Gmail, Outlook, Yahoo, etc.
	Site        string    `json:"site"`                          // MX hostname
	Queue       string    `json:"queue"`                         // queue name
	EgressPool  string    `json:"egress_pool"`                   // IP pool used
	EgressSrc   string    `json:"egress_source"`                 // source IP within pool
	NumAttempts int       `json:"num_attempts"`                  // retry count
	BounceClass string   `json:"bounce_classification"`          // KumoMTA bounce classifier
}

// FBLRecord stores a parsed Feedback Loop (ARF/RFC 5965) complaint received
// via a registered feedback loop mailbox. Each record represents one complaint
// from a recipient who reported the message as spam.
type FBLRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ReceivedAt     time.Time `gorm:"index" json:"received_at"`
	FeedbackType   string    `gorm:"index" json:"feedback_type"`    // "abuse", "fraud", "virus", "unsubscribe", "other"
	ReportingMTA   string    `json:"reporting_mta"`                 // ISP that sent the report (e.g. "yahoo.com")
	SourceIP       string    `gorm:"index" json:"source_ip"`        // Sending IP that triggered the complaint
	OriginalSender string    `gorm:"index" json:"original_sender"`  // Envelope-from of the complained message
	OriginalRcptTo string    `gorm:"index" json:"original_rcpt_to"` // Original recipient who complained
	ArrivalDate    time.Time `json:"arrival_date"`                   // When the original message arrived at ISP
	MessageID      string    `json:"message_id"`                    // Message-ID of the complained message
	SenderID       uint      `gorm:"index" json:"sender_id"`        // Matched Sender record, 0 if unknown
	SenderEmail    string    `gorm:"index" json:"sender_email"`     // Resolved sender email
	Domain         string    `gorm:"index" json:"domain"`           // Sending domain
	AutoSuppressed bool      `json:"auto_suppressed"`               // Was recipient auto-suppressed?
	RawHeaders     string    `json:"raw_headers"`                   // Raw feedback-report MIME part headers
}

// BounceClassification stores a fully classified bounce event with DSN details.
// This enriches DeliveryEvent data with RFC 3464 parsed fields and a
// human-readable classification for use in dashboards and suppression logic.
type BounceClassification struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ReceivedAt      time.Time `gorm:"index" json:"received_at"`
	// Classification
	Category        string    `gorm:"index" json:"category"`         // "hard", "soft", "block", "quota", "dns", "tls", "auth", "unknown"
	IsHard          bool      `gorm:"index" json:"is_hard"`          // True = permanent, suppress recipient
	// RFC 3464 fields
	FinalRecipient  string    `gorm:"index" json:"final_recipient"`  // The actual failed recipient
	EnhancedStatus  string    `json:"enhanced_status"`               // e.g. "5.1.1"
	Action          string    `json:"action"`                        // "failed", "delayed", "delivered"
	DiagnosticCode  string    `json:"diagnostic_code"`               // Full ISP error message
	// Source
	OriginalSender  string    `gorm:"index" json:"original_sender"`  // Envelope-from / Return-Path
	SenderID        uint      `gorm:"index" json:"sender_id"`        // Matched Sender record
	Domain          string    `gorm:"index" json:"domain"`           // Recipient domain
	Provider        string    `gorm:"index" json:"provider"`         // "Gmail", "Outlook", etc.
	// VERP
	VERPDecoded     bool      `json:"verp_decoded"`                  // Was recipient recovered via VERP?
	// Processing
	AutoSuppressed  bool      `json:"auto_suppressed"`               // Was recipient auto-suppressed?
	SourceFile      string    `json:"source_file"`                   // Maildir filename for audit
}

// ISPSnapshot stores a point-in-time snapshot of per-ISP delivery intelligence.
// Populated from local delivery metrics, Google Postmaster Tools, and Microsoft SNDS.
type ISPSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ISP       string    `gorm:"index" json:"isp"`        // "Gmail", "Yahoo", "Outlook", "AOL", "Other"
	Domain    string    `gorm:"index" json:"domain"`     // Sending domain this snapshot is for
	CapturedAt time.Time `gorm:"index" json:"captured_at"`

	// Local metrics (derived from DeliveryEvent + EmailStats)
	TotalSent      int64   `json:"total_sent"`
	TotalDelivered int64   `json:"total_delivered"`
	TotalBounced   int64   `json:"total_bounced"`
	TotalDeferred  int64   `json:"total_deferred"`
	AcceptanceRate float64 `json:"acceptance_rate"` // delivered / (delivered + bounced)
	BounceRate     float64 `json:"bounce_rate"`     // bounced / total
	DeferralRate   float64 `json:"deferral_rate"`   // deferred / total
	ComplaintRate  float64 `json:"complaint_rate"`  // fbl complaints / sent

	// Google Postmaster Tools (populated when OAuth configured)
	GPTDomainReputation string  `json:"gpt_domain_reputation"` // "HIGH","MEDIUM","LOW","BAD",""
	GPTIPReputation     string  `json:"gpt_ip_reputation"`
	GPTSpamRate         float64 `json:"gpt_spam_rate"` // 0.0–1.0 fraction
	GPTDeliveryErrors   int64   `json:"gpt_delivery_errors"`
	GPTEnabled          bool    `json:"gpt_enabled"`

	// Microsoft SNDS (populated when API key configured)
	SNDSFilterResult string  `json:"snds_filter_result"` // "GREEN","YELLOW","RED",""
	SNDSTrapRate     float64 `json:"snds_trap_rate"`
	SNDSEnabled      bool    `json:"snds_enabled"`

	// Overall health score 0–100 (computed)
	HealthScore int `json:"health_score"`
}

// ThrottleAdjustmentLog records every auto-adjustment made by the Adaptive Throttling Engine.
type ThrottleAdjustmentLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	ISP       string    `gorm:"index" json:"isp"`          // "Gmail", "Yahoo", etc.
	RuleID    uint      `gorm:"index" json:"rule_id"`      // TrafficShapingRule.ID
	RuleName  string    `json:"rule_name"`
	// Before
	OldRate       string `json:"old_rate"`        // e.g. "100/h"
	OldConns      int    `json:"old_conns"`
	// After
	NewRate       string `json:"new_rate"`
	NewConns      int    `json:"new_conns"`
	// Reason
	Direction     string  `json:"direction"`      // "up", "down", "reset"
	Reason        string  `json:"reason"`         // human-readable explanation
	DeferralRate  float64 `json:"deferral_rate"`  // metric that triggered adjustment
	AcceptRate    float64 `json:"accept_rate"`
}

// AnomalyEvent records a detected anomaly and its self-healing action.
type AnomalyEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DetectedAt  time.Time `gorm:"index" json:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at"` // nil = still active
	Type        string    `gorm:"index" json:"type"`     // "bounce_spike","complaint_spike","queue_backlog","acceptance_drop","blacklist"
	Severity    string    `json:"severity"`              // "warning","critical"
	ISP         string    `gorm:"index" json:"isp"`
	SenderEmail string    `gorm:"index" json:"sender_email"`
	Domain      string    `gorm:"index" json:"domain"`
	// Metrics at detection
	MetricValue float64   `json:"metric_value"`
	Threshold   float64   `json:"threshold"`
	BaselineVal float64   `json:"baseline_val"`
	// Action taken
	ActionTaken string    `json:"action_taken"` // "sender_paused","rate_reduced","alert_sent","none"
	AutoHealed  bool      `json:"auto_healed"`
	Notes       string    `json:"notes"`
}

// VERPConfig stores VERP (Variable Envelope Return Path) configuration per domain.
// When enabled, KumoMTA is configured to use a VERP-encoded return-path so
// incoming bounces can be attributed to specific recipients and senders.
type VERPConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DomainID     uint      `gorm:"uniqueIndex" json:"domain_id"` // One config per domain
	Domain       string    `json:"domain"`
	BounceDomain string    `json:"bounce_domain"` // e.g. "bounces.example.com"
	IsEnabled    bool      `json:"is_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
