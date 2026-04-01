package store

import (
	"errors"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulak-ranjan/kumoops/internal/models"
)

type Store struct {
	DB *gorm.DB
}

func NewStore(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.AppSettings{},
		&models.Domain{},
		&models.Sender{},
		&models.AdminUser{},
		&models.AuthSession{},
		&models.BounceAccount{},
		&models.SystemIP{},
		&models.EmailStats{},
		&models.WebhookLog{},
		&models.APIKey{},
		&models.ChatLog{},
		&models.ContactList{},        // NEW
		&models.Contact{},            // NEW
		&models.Campaign{},           // NEW
		&models.CampaignRecipient{},  // NEW
		&models.AutomationWorkflow{}, // NEW
		&models.WhatsAppMessage{},    // NEW
		&models.TrafficShapingRule{}, // NEW
		&models.IPPool{},             // NEW
		&models.IPPoolMember{},       // NEW
		&models.SuppressedEmail{},    // NEW
		&models.AlertRule{},          // NEW
		&models.AlertEvent{},         // NEW
		&models.BIMIRecord{},         // NEW
		&models.MTASTSPolicy{},       // NEW
		&models.WarmupLog{},          // NEW
		&models.DeliveryEvent{},      // NEW
		&models.ReputationCheck{},        // NEW
		&models.RemoteServer{},           // NEW
		&models.FBLRecord{},              // FBL engine
		&models.BounceClassification{},   // DSN bounce classifier
		&models.VERPConfig{},             // VERP per-domain config
		&models.ISPSnapshot{},            // ISP intelligence snapshots
		&models.ThrottleAdjustmentLog{},  // Adaptive throttle audit log
		&models.AnomalyEvent{},           // Anomaly detection events
		&models.CampaignVariant{},        // A/B test variants
		&models.SeedMailbox{},            // Inbox placement seed mailboxes
		&models.PlacementTest{},          // Inbox placement test runs
		&models.PlacementResult{},        // Per-mailbox placement results
	); err != nil {
		return nil, err
	}

	st := &Store{DB: db}

	// Seed default traffic shaping rules if table is empty
	if err := st.SeedDefaultShapingRules(); err != nil {
		log.Println("[STORE] warning: failed to seed shaping rules:", err)
	}

	return st, nil
}

func (s *Store) LogError(err error) {
	if err != nil {
		log.Println("[STORE ERROR]", err)
	}
}

var ErrNotFound = gorm.ErrRecordNotFound

// ... [Existing code for Sessions, Settings, Domains, etc. remains unchanged] ...

// ----------------------
// Auth Sessions (Multi-Device)
// ----------------------

func (s *Store) CreateSession(adminID uint, token string, ip string, userAgent string, duration time.Duration) error {
	// 1. Cleanup expired
	s.DB.Where("expires_at < ?", time.Now()).Delete(&models.AuthSession{})

	// 2. Enforce Max 3 Limit (Delete oldest if needed)
	var count int64
	s.DB.Model(&models.AuthSession{}).Where("admin_id = ?", adminID).Count(&count)
	if count >= 3 {
		var oldest models.AuthSession
		s.DB.Where("admin_id = ?", adminID).Order("created_at asc").First(&oldest)
		if oldest.ID != 0 {
			s.DB.Delete(&oldest)
		}
	}

	// 3. Create New
	sess := models.AuthSession{
		AdminID:   adminID,
		Token:     token,
		ExpiresAt: time.Now().Add(duration),
		DeviceIP:  ip,
		UserAgent: userAgent,
	}
	return s.DB.Create(&sess).Error
}

func (s *Store) GetAdminBySessionToken(token string) (*models.AdminUser, error) {
	var sess models.AuthSession
	err := s.DB.Where("token = ? AND expires_at > ?", token, time.Now()).First(&sess).Error
	if err != nil {
		return nil, err
	}

	var admin models.AdminUser
	err = s.DB.First(&admin, sess.AdminID).Error
	return &admin, err
}

func (s *Store) DeleteSession(token string) error {
	return s.DB.Where("token = ?", token).Delete(&models.AuthSession{}).Error
}

func (s *Store) ListSessionsByAdmin(adminID uint) ([]models.AuthSession, error) {
	sessions := make([]models.AuthSession, 0)
	err := s.DB.Where("admin_id = ? AND expires_at > ?", adminID, time.Now()).
		Order("created_at desc").Find(&sessions).Error
	return sessions, err
}

// ----------------------
// App Settings
// ----------------------

func (s *Store) GetSettings() (*models.AppSettings, error) {
	var st models.AppSettings
	err := s.DB.First(&st).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Store) UpsertSettings(st *models.AppSettings) error {
	if st.ID == 0 {
		return s.DB.Create(st).Error
	}
	return s.DB.Save(st).Error
}

// ----------------------
// Domains
// ----------------------

func (s *Store) ListDomains() ([]models.Domain, error) {
	domains := make([]models.Domain, 0)
	err := s.DB.Preload("Senders").Find(&domains).Error
	return domains, err
}

func (s *Store) GetDomainByID(id uint) (*models.Domain, error) {
	var d models.Domain
	err := s.DB.Preload("Senders").First(&d, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) GetDomainByName(name string) (*models.Domain, error) {
	var d models.Domain
	err := s.DB.Preload("Senders").Where("name = ?", name).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) CreateDomain(d *models.Domain) error {
	return s.DB.Create(d).Error
}

func (s *Store) UpdateDomain(d *models.Domain) error {
	return s.DB.Save(d).Error
}

func (s *Store) DeleteDomain(id uint) error {
	return s.DB.Delete(&models.Domain{}, id).Error
}

func (s *Store) CountDomains() (int64, error) {
	var c int64
	err := s.DB.Model(&models.Domain{}).Count(&c).Error
	return c, err
}

// ----------------------
// Senders
// ----------------------

func (s *Store) ListSendersByDomain(domainID uint) ([]models.Sender, error) {
	senders := make([]models.Sender, 0)
	err := s.DB.Where("domain_id = ?", domainID).Find(&senders).Error
	return senders, err
}

func (s *Store) GetSenderByID(id uint) (*models.Sender, error) {
	var snd models.Sender
	err := s.DB.First(&snd, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &snd, nil
}

func (s *Store) CreateSender(snd *models.Sender) error {
	return s.DB.Create(snd).Error
}

func (s *Store) UpdateSender(snd *models.Sender) error {
	return s.DB.Save(snd).Error
}

func (s *Store) DeleteSender(id uint) error {
	return s.DB.Delete(&models.Sender{}, id).Error
}

func (s *Store) CountSenders() (int64, error) {
	var c int64
	err := s.DB.Model(&models.Sender{}).Count(&c).Error
	return c, err
}

// ----------------------
// Admin Users
// ----------------------

func (s *Store) AdminCount() (int64, error) {
	var count int64
	err := s.DB.Model(&models.AdminUser{}).Count(&count).Error
	return count, err
}

func (s *Store) GetAdminByEmail(email string) (*models.AdminUser, error) {
	var u models.AdminUser
	err := s.DB.Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetAdminByID(id uint) (*models.AdminUser, error) {
	var u models.AdminUser
	err := s.DB.First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) CreateAdmin(u *models.AdminUser) error {
	return s.DB.Create(u).Error
}

func (s *Store) UpdateAdmin(u *models.AdminUser) error {
	return s.DB.Save(u).Error
}

// ----------------------
// Bounce Accounts
// ----------------------

func (s *Store) ListBounceAccounts() ([]models.BounceAccount, error) {
	list := make([]models.BounceAccount, 0)
	err := s.DB.Find(&list).Error
	return list, err
}

func (s *Store) GetBounceAccountByID(id uint) (*models.BounceAccount, error) {
	var b models.BounceAccount
	err := s.DB.First(&b, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) CreateBounceAccount(b *models.BounceAccount) error {
	return s.DB.Create(b).Error
}

func (s *Store) UpdateBounceAccount(b *models.BounceAccount) error {
	return s.DB.Save(b).Error
}

func (s *Store) DeleteBounceAccount(id uint) error {
	return s.DB.Delete(&models.BounceAccount{}, id).Error
}

func (s *Store) ListBounceAccountsByUsername(username string) ([]models.BounceAccount, error) {
	list := make([]models.BounceAccount, 0)
	err := s.DB.Where("username = ?", username).Find(&list).Error
	return list, err
}

// ----------------------
// System IPs
// ----------------------

func (s *Store) ListSystemIPs() ([]models.SystemIP, error) {
	list := make([]models.SystemIP, 0)
	err := s.DB.Find(&list).Error
	return list, err
}

func (s *Store) CreateSystemIP(ip *models.SystemIP) error {
	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(ip).Error
}

func (s *Store) CreateSystemIPs(ips []models.SystemIP) error {
	if len(ips) == 0 {
		return nil
	}
	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&ips).Error
}

func (s *Store) DeleteSystemIP(id uint) error {
	return s.DB.Delete(&models.SystemIP{}, id).Error
}

// ----------------------
// Email Stats
// ----------------------

func (s *Store) UpsertEmailStats(stats *models.EmailStats) error {
	var existing models.EmailStats
	date := stats.Date.Truncate(24 * time.Hour)

	err := s.DB.Where("domain = ? AND date = ?", stats.Domain, date).First(&existing).Error
	if err == nil {
		existing.Sent += stats.Sent
		existing.Delivered += stats.Delivered
		existing.Bounced += stats.Bounced
		existing.Deferred += stats.Deferred
		existing.UpdatedAt = time.Now()
		return s.DB.Save(&existing).Error
	}

	stats.Date = date
	stats.UpdatedAt = time.Now()
	return s.DB.Create(stats).Error
}

func (s *Store) GetEmailStatsByDomain(domain string, days int) ([]models.EmailStats, error) {
	stats := make([]models.EmailStats, 0)
	since := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	err := s.DB.Where("domain = ? AND date >= ?", domain, since).
		Order("date asc").Find(&stats).Error
	return stats, err
}

func (s *Store) GetEmailStatsAll(days int) ([]models.EmailStats, error) {
	stats := make([]models.EmailStats, 0)
	since := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	err := s.DB.Where("date >= ?", since).Order("date asc").Find(&stats).Error
	return stats, err
}


func (s *Store) GetTodayStats() ([]models.EmailStats, error) {
	stats := make([]models.EmailStats, 0)
	today := time.Now().Truncate(24 * time.Hour)
	err := s.DB.Where("date = ?", today).Find(&stats).Error
	return stats, err
}

func (s *Store) SetEmailStats(stats *models.EmailStats) error {
	date := stats.Date.Truncate(24 * time.Hour)
	s.DB.Where("domain = ? AND date = ?", stats.Domain, date).Delete(&models.EmailStats{})
	stats.Date = date
	stats.UpdatedAt = time.Now()
	return s.DB.Create(stats).Error
}

// ----------------------
// Webhook Logs
// ----------------------

func (s *Store) CreateWebhookLog(wl *models.WebhookLog) error {
	return s.DB.Create(wl).Error
}

func (s *Store) ListWebhookLogs(limit int) ([]models.WebhookLog, error) {
	logs := make([]models.WebhookLog, 0)
	err := s.DB.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// ----------------------
// Chat Logs (AI Memory)
// ----------------------

func (s *Store) SaveChatLog(role, content string) error {
	// Limit history to 500 messages to prevent DB bloat
	var count int64
	s.DB.Model(&models.ChatLog{}).Count(&count)
	if count > 500 {
		var oldest models.ChatLog
		s.DB.Order("created_at asc").First(&oldest)
		s.DB.Delete(&oldest)
	}

	log := &models.ChatLog{
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}
	return s.DB.Create(log).Error
}

func (s *Store) GetChatHistory(limit int) ([]models.ChatLog, error) {
	logs := make([]models.ChatLog, 0)
	err := s.DB.Order("created_at desc").Limit(limit).Find(&logs).Error
	// Reverse order for chat display
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
	return logs, err
}

// ----------------------
// Traffic Shaping Rules
// ----------------------

func (s *Store) ListShapingRules() ([]models.TrafficShapingRule, error) {
	rules := make([]models.TrafficShapingRule, 0)
	err := s.DB.Order("id asc").Find(&rules).Error
	return rules, err
}

func (s *Store) GetShapingRule(id uint) (*models.TrafficShapingRule, error) {
	var rule models.TrafficShapingRule
	err := s.DB.First(&rule, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *Store) CreateShapingRule(r *models.TrafficShapingRule) error {
	return s.DB.Create(r).Error
}

func (s *Store) UpdateShapingRule(r *models.TrafficShapingRule) error {
	return s.DB.Save(r).Error
}

func (s *Store) DeleteShapingRule(id uint) error {
	return s.DB.Delete(&models.TrafficShapingRule{}, id).Error
}

func (s *Store) SeedDefaultShapingRules() error {
	var count int64
	s.DB.Model(&models.TrafficShapingRule{}).Count(&count)
	if count > 0 {
		return nil
	}

	defaults := []models.TrafficShapingRule{
		{
			Provider:             "Gmail",
			Pattern:              "google.com",
			MaxMessageRate:       "50/h",
			MaxConnectionRate:    "5/min",
			MaxDeliveriesPerConn: 20,
			ConnectionLimit:      3,
			IsEnabled:            true,
		},
		{
			Provider:             "Gmail",
			Pattern:              "googlemail.com",
			MaxMessageRate:       "50/h",
			MaxConnectionRate:    "5/min",
			MaxDeliveriesPerConn: 20,
			ConnectionLimit:      3,
			IsEnabled:            true,
		},
		{
			Provider:             "Outlook",
			Pattern:              "outlook.com",
			MaxMessageRate:       "50/h",
			MaxConnectionRate:    "3/min",
			MaxDeliveriesPerConn: 10,
			ConnectionLimit:      2,
			IsEnabled:            true,
		},
		{
			Provider:             "Outlook",
			Pattern:              "hotmail.com",
			MaxMessageRate:       "50/h",
			MaxConnectionRate:    "3/min",
			MaxDeliveriesPerConn: 10,
			ConnectionLimit:      2,
			IsEnabled:            true,
		},
		{
			Provider:             "Outlook",
			Pattern:              "live.com",
			MaxMessageRate:       "50/h",
			MaxConnectionRate:    "3/min",
			MaxDeliveriesPerConn: 10,
			ConnectionLimit:      2,
			IsEnabled:            true,
		},
		{
			Provider:             "Outlook",
			Pattern:              "office365.com",
			MaxMessageRate:       "50/h",
			MaxConnectionRate:    "3/min",
			MaxDeliveriesPerConn: 10,
			ConnectionLimit:      2,
			IsEnabled:            true,
		},
		{
			Provider:             "Yahoo",
			Pattern:              "yahoodns.net",
			MaxMessageRate:       "100/h",
			MaxConnectionRate:    "5/min",
			MaxDeliveriesPerConn: 20,
			ConnectionLimit:      3,
			IsEnabled:            true,
		},
		{
			Provider:             "Yahoo",
			Pattern:              "yahoo.com",
			MaxMessageRate:       "100/h",
			MaxConnectionRate:    "5/min",
			MaxDeliveriesPerConn: 20,
			ConnectionLimit:      3,
			IsEnabled:            true,
		},
		{
			Provider:             "Yahoo",
			Pattern:              "aol.com",
			MaxMessageRate:       "100/h",
			MaxConnectionRate:    "5/min",
			MaxDeliveriesPerConn: 20,
			ConnectionLimit:      3,
			IsEnabled:            true,
		},
	}

	return s.DB.Create(&defaults).Error
}

// ----------------------
// IP Pools
// ----------------------

func (s *Store) ListIPPools() ([]models.IPPool, error) {
	pools := make([]models.IPPool, 0)
	err := s.DB.Preload("Members").Order("id asc").Find(&pools).Error
	return pools, err
}

func (s *Store) GetIPPool(id uint) (*models.IPPool, error) {
	var pool models.IPPool
	err := s.DB.Preload("Members").First(&pool, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (s *Store) CreateIPPool(p *models.IPPool) error {
	return s.DB.Create(p).Error
}

func (s *Store) UpdateIPPool(p *models.IPPool) error {
	return s.DB.Save(p).Error
}

func (s *Store) DeleteIPPool(id uint) error {
	return s.DB.Delete(&models.IPPool{}, id).Error
}

func (s *Store) AddIPToPool(poolID uint, ipValue string) error {
	member := &models.IPPoolMember{
		PoolID:  poolID,
		IPValue: ipValue,
	}
	return s.DB.Create(member).Error
}

func (s *Store) RemoveIPFromPool(memberID uint) error {
	return s.DB.Delete(&models.IPPoolMember{}, memberID).Error
}

// ----------------------
// Suppression List
// ----------------------

func (s *Store) ListSuppressed(page, pageSize int, search string) ([]models.SuppressedEmail, int64, error) {
	results := make([]models.SuppressedEmail, 0)
	var total int64
	q := s.DB.Model(&models.SuppressedEmail{})
	if search != "" {
		q = q.Where("email LIKE ? OR domain LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	q.Count(&total)
	offset := (page - 1) * pageSize
	err := q.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&results).Error
	return results, total, err
}

func (s *Store) IsSuppressed(email string) bool {
	var count int64
	s.DB.Model(&models.SuppressedEmail{}).Where("email = ?", email).Count(&count)
	return count > 0
}

func (s *Store) AddSuppression(email, reason, sourceInfo string) error {
	domain := ""
	if idx := len(email) - len(email); idx >= 0 {
		for i, c := range email {
			if c == '@' {
				domain = email[i+1:]
				break
			}
		}
	}
	entry := models.SuppressedEmail{
		Email:      email,
		Reason:     reason,
		Domain:     domain,
		SourceInfo: sourceInfo,
	}
	// Upsert: ignore duplicates
	return s.DB.Where("email = ?", email).FirstOrCreate(&entry).Error
}

func (s *Store) RemoveSuppression(id uint) error {
	return s.DB.Delete(&models.SuppressedEmail{}, id).Error
}

func (s *Store) BulkAddSuppression(emails []string, reason, source string) (int, error) {
	count := 0
	for _, email := range emails {
		if email == "" {
			continue
		}
		if err := s.AddSuppression(email, reason, source); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Store) CountSuppressed() (int64, error) {
	var c int64
	err := s.DB.Model(&models.SuppressedEmail{}).Count(&c).Error
	return c, err
}

// ----------------------
// Alert Rules
// ----------------------

func (s *Store) ListAlertRules() ([]models.AlertRule, error) {
	rules := make([]models.AlertRule, 0)
	err := s.DB.Order("id asc").Find(&rules).Error
	return rules, err
}

func (s *Store) GetAlertRule(id uint) (*models.AlertRule, error) {
	var rule models.AlertRule
	err := s.DB.First(&rule, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &rule, err
}

func (s *Store) CreateAlertRule(r *models.AlertRule) error {
	if r.CooldownMin == 0 {
		r.CooldownMin = 60
	}
	return s.DB.Create(r).Error
}

func (s *Store) UpdateAlertRule(r *models.AlertRule) error {
	return s.DB.Save(r).Error
}

func (s *Store) DeleteAlertRule(id uint) error {
	return s.DB.Delete(&models.AlertRule{}, id).Error
}

func (s *Store) UpdateAlertRuleLastFired(id uint, t time.Time) error {
	return s.DB.Model(&models.AlertRule{}).Where("id = ?", id).Update("last_fired", t).Error
}

func (s *Store) CreateAlertEvent(e *models.AlertEvent) error {
	if e.FiredAt.IsZero() {
		e.FiredAt = time.Now()
	}
	return s.DB.Create(e).Error
}

func (s *Store) ListAlertEvents(limit int) ([]models.AlertEvent, error) {
	events := make([]models.AlertEvent, 0)
	err := s.DB.Order("fired_at desc").Limit(limit).Find(&events).Error
	return events, err
}

// ----------------------
// BIMI Records
// ----------------------

func (s *Store) GetBIMI(domain string) (*models.BIMIRecord, error) {
	var b models.BIMIRecord
	err := s.DB.Where("domain = ?", domain).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &b, err
}

func (s *Store) UpsertBIMI(b *models.BIMIRecord) error {
	var existing models.BIMIRecord
	if err := s.DB.Where("domain = ?", b.Domain).First(&existing).Error; err == nil {
		b.ID = existing.ID
		b.CreatedAt = existing.CreatedAt
		return s.DB.Save(b).Error
	}
	return s.DB.Create(b).Error
}

// ----------------------
// MTA-STS Policies
// ----------------------

func (s *Store) GetMTASTS(domain string) (*models.MTASTSPolicy, error) {
	var p models.MTASTSPolicy
	err := s.DB.Where("domain = ?", domain).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (s *Store) UpsertMTASTS(p *models.MTASTSPolicy) error {
	var existing models.MTASTSPolicy
	if err := s.DB.Where("domain = ?", p.Domain).First(&existing).Error; err == nil {
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
		return s.DB.Save(p).Error
	}
	return s.DB.Create(p).Error
}

// ----------------------
// Warmup Logs
// ----------------------

func (s *Store) CreateWarmupLog(l *models.WarmupLog) error {
	return s.DB.Create(l).Error
}

func (s *Store) ListWarmupLogs(senderID uint, limit int) ([]models.WarmupLog, error) {
	logs := make([]models.WarmupLog, 0)
	q := s.DB.Order("created_at desc").Limit(limit)
	if senderID > 0 {
		q = q.Where("sender_id = ?", senderID)
	}
	err := q.Find(&logs).Error
	return logs, err
}

// ----------------------
// Delivery Events (Per-Recipient Failure Tracking)
// ----------------------

// DeliveryEventFilter holds query parameters for listing delivery events.
type DeliveryEventFilter struct {
	Recipient string
	Domain    string
	EventType string
	Hours     int
}

// BulkInsertDeliveryEvents clears events in the given window then re-inserts.
// This prevents duplicates on repeated parse runs.
func (s *Store) BulkInsertDeliveryEvents(events []models.DeliveryEvent, since time.Time) error {
	if len(events) == 0 {
		return nil
	}
	// Remove records in the window we're about to re-populate
	s.DB.Where("timestamp >= ?", since).Delete(&models.DeliveryEvent{})
	return s.DB.CreateInBatches(events, 500).Error
}

// ListDeliveryEvents returns a paginated, filtered list of delivery failure events.
func (s *Store) ListDeliveryEvents(page, limit int, f DeliveryEventFilter) ([]models.DeliveryEvent, int64, error) {
	results := make([]models.DeliveryEvent, 0)
	var total int64

	hours := 72
	if f.Hours > 0 && f.Hours <= 168 {
		hours = f.Hours
	}
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

	q := s.DB.Model(&models.DeliveryEvent{}).Where("timestamp >= ?", cutoff)
	if f.Recipient != "" {
		q = q.Where("recipient LIKE ?", "%"+f.Recipient+"%")
	}
	if f.Domain != "" {
		q = q.Where("domain LIKE ?", "%"+f.Domain+"%")
	}
	if f.EventType != "" {
		q = q.Where("event_type = ?", f.EventType)
	}

	q.Count(&total)
	offset := (page - 1) * limit
	err := q.Order("timestamp desc").Offset(offset).Limit(limit).Find(&results).Error
	return results, total, err
}

// GetDeliveryEventSummary returns counts by event type for the given hours window.
func (s *Store) GetDeliveryEventSummary(hours int) (map[string]int64, error) {
	if hours <= 0 {
		hours = 72
	}
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

	type row struct {
		EventType string
		Count     int64
	}
	var rows []row
	err := s.DB.Model(&models.DeliveryEvent{}).
		Select("event_type, count(*) as count").
		Where("timestamp >= ?", cutoff).
		Group("event_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := map[string]int64{"Bounce": 0, "TransientFailure": 0}
	for _, r := range rows {
		result[r.EventType] = r.Count
	}
	return result, nil
}

// PruneDeliveryEvents deletes events older than the given number of hours.
func (s *Store) PruneDeliveryEvents(hours int) error {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	return s.DB.Where("timestamp < ?", cutoff).Delete(&models.DeliveryEvent{}).Error
}

// ----------------------
// Reputation Checks
// ----------------------

// SaveReputationCheck upserts (by target) the latest reputation check result.
// Deletes ALL old rows for the target first, then inserts fresh — avoids stale duplicates.
func (s *Store) SaveReputationCheck(rc *models.ReputationCheck) error {
	rc.CheckedAt = time.Now()
	// Delete every old row for this target (clean slate)
	s.DB.Where("target = ?", rc.Target).Delete(&models.ReputationCheck{})
	// Insert the fresh result
	return s.DB.Create(rc).Error
}

// GetLatestReputationChecks returns the most recent check for every target (one row per target).
func (s *Store) GetLatestReputationChecks() ([]models.ReputationCheck, error) {
	rows := make([]models.ReputationCheck, 0)
	err := s.DB.Order("target_type, target").Find(&rows).Error
	return rows, err
}

// PurgeStaleReputationTargets removes reputation rows for targets that no longer exist
// in system IPs or configured domains.
func (s *Store) PurgeStaleReputationTargets(activeIPs, activeDomains []string) error {
	all := make(map[string]bool)
	for _, ip := range activeIPs {
		all[ip] = true
	}
	for _, d := range activeDomains {
		all[d] = true
	}
	var existing []models.ReputationCheck
	s.DB.Find(&existing)
	for _, rc := range existing {
		if !all[rc.Target] {
			s.DB.Delete(&rc)
		}
	}
	return nil
}

// GetReputationHistory returns the last N checks for a specific target.
func (s *Store) GetReputationHistory(target string, limit int) ([]models.ReputationCheck, error) {
	rows := make([]models.ReputationCheck, 0)
	err := s.DB.Where("target = ?", target).Order("checked_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

// ----------------------
// Remote Servers
// ----------------------

func (s *Store) ListRemoteServers() ([]models.RemoteServer, error) {
	servers := make([]models.RemoteServer, 0)
	err := s.DB.Order("name asc").Find(&servers).Error
	return servers, err
}

func (s *Store) GetRemoteServer(id uint) (*models.RemoteServer, error) {
	var srv models.RemoteServer
	err := s.DB.First(&srv, id).Error
	if err != nil {
		return nil, err
	}
	return &srv, nil
}

func (s *Store) CreateRemoteServer(srv *models.RemoteServer) error {
	srv.CreatedAt = time.Now()
	srv.Status = "unknown"
	return s.DB.Create(srv).Error
}

func (s *Store) UpdateRemoteServer(srv *models.RemoteServer) error {
	return s.DB.Save(srv).Error
}

func (s *Store) DeleteRemoteServer(id uint) error {
	return s.DB.Delete(&models.RemoteServer{}, id).Error
}
