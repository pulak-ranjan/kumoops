package store

import (
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
)

// ─────────────────────────────────────────────
// FBL Records
// ─────────────────────────────────────────────

func (s *Store) CreateFBLRecord(r *models.FBLRecord) error {
	return s.DB.Create(r).Error
}

// ListFBLRecords returns FBL complaint records ordered newest-first.
// Optional filters: domain, feedbackType, senderEmail, since.
func (s *Store) ListFBLRecords(domain, feedbackType, senderEmail string, since time.Time, limit int) ([]models.FBLRecord, error) {
	q := s.DB.Model(&models.FBLRecord{}).Order("received_at DESC")
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	if feedbackType != "" {
		q = q.Where("feedback_type = ?", feedbackType)
	}
	if senderEmail != "" {
		q = q.Where("sender_email = ?", senderEmail)
	}
	if !since.IsZero() {
		q = q.Where("received_at >= ?", since)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	records := make([]models.FBLRecord, 0)
	return records, q.Find(&records).Error
}

// CountFBLRecordsSince returns the number of FBL complaints received since a given time.
func (s *Store) CountFBLRecordsSince(domain string, since time.Time) (int64, error) {
	q := s.DB.Model(&models.FBLRecord{}).Where("received_at >= ?", since)
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	var count int64
	return count, q.Count(&count).Error
}

// FBLComplaintStats holds aggregated complaint statistics per domain/sender.
type FBLComplaintStats struct {
	Domain          string  `json:"domain"`
	SenderEmail     string  `json:"sender_email"`
	TotalComplaints int64   `json:"total_complaints"`
	AbuseComplaints int64   `json:"abuse_complaints"`
	UnsubComplaints int64   `json:"unsub_complaints"`
	LastSeen        string  `json:"last_seen"`
	AutoSuppressed  int64   `json:"auto_suppressed"`
}

// GetFBLStats returns per-domain complaint stats for a given time window.
func (s *Store) GetFBLStats(since time.Time) ([]FBLComplaintStats, error) {
	type row struct {
		Domain          string
		SenderEmail     string
		TotalComplaints int64
		AbuseComplaints int64
		UnsubComplaints int64
		LastSeen        string
		AutoSuppressed  int64
	}
	var rows []row
	err := s.DB.Raw(`
		SELECT
			domain,
			sender_email,
			COUNT(*) as total_complaints,
			SUM(CASE WHEN feedback_type = 'abuse' THEN 1 ELSE 0 END) as abuse_complaints,
			SUM(CASE WHEN feedback_type = 'unsubscribe' THEN 1 ELSE 0 END) as unsub_complaints,
			MAX(received_at) as last_seen,
			SUM(CASE WHEN auto_suppressed = 1 THEN 1 ELSE 0 END) as auto_suppressed
		FROM fbl_records
		WHERE received_at >= ?
		GROUP BY domain, sender_email
		ORDER BY total_complaints DESC
	`, since).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]FBLComplaintStats, 0, len(rows))
	for _, r := range rows {
		result = append(result, FBLComplaintStats{
			Domain:          r.Domain,
			SenderEmail:     r.SenderEmail,
			TotalComplaints: r.TotalComplaints,
			AbuseComplaints: r.AbuseComplaints,
			UnsubComplaints: r.UnsubComplaints,
			LastSeen:        r.LastSeen,
			AutoSuppressed:  r.AutoSuppressed,
		})
	}
	return result, nil
}

// DeleteFBLRecord deletes a single FBL record by ID.
func (s *Store) DeleteFBLRecord(id uint) error {
	return s.DB.Delete(&models.FBLRecord{}, id).Error
}

// ─────────────────────────────────────────────
// Bounce Classifications (DSN)
// ─────────────────────────────────────────────

func (s *Store) CreateBounceClassification(bc *models.BounceClassification) error {
	return s.DB.Create(bc).Error
}

// ListBounceClassifications returns classified bounce records with optional filters.
func (s *Store) ListBounceClassifications(domain, category string, since time.Time, limit int) ([]models.BounceClassification, error) {
	q := s.DB.Model(&models.BounceClassification{}).Order("received_at DESC")
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if !since.IsZero() {
		q = q.Where("received_at >= ?", since)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	records := make([]models.BounceClassification, 0)
	return records, q.Find(&records).Error
}

// BounceClassificationSummary holds aggregate counts by category.
type BounceClassificationSummary struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// GetBounceClassificationSummary returns per-category counts for a time window.
func (s *Store) GetBounceClassificationSummary(since time.Time) ([]BounceClassificationSummary, error) {
	rows := make([]BounceClassificationSummary, 0)
	err := s.DB.Raw(`
		SELECT category, COUNT(*) as count
		FROM bounce_classifications
		WHERE received_at >= ?
		GROUP BY category
		ORDER BY count DESC
	`, since).Scan(&rows).Error
	return rows, err
}

// ─────────────────────────────────────────────
// VERP Config
// ─────────────────────────────────────────────

func (s *Store) GetVERPConfig(domainID uint) (*models.VERPConfig, error) {
	var cfg models.VERPConfig
	err := s.DB.Where("domain_id = ?", domainID).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Store) UpsertVERPConfig(cfg *models.VERPConfig) error {
	if cfg.ID == 0 {
		return s.DB.Create(cfg).Error
	}
	return s.DB.Save(cfg).Error
}

func (s *Store) ListVERPConfigs() ([]models.VERPConfig, error) {
	configs := make([]models.VERPConfig, 0)
	return configs, s.DB.Find(&configs).Error
}

// ─────────────────────────────────────────────
// Sender lookup helpers (used by FBL + DSN)
// ─────────────────────────────────────────────

// ListSendersByEmail returns Sender records whose Email field matches exactly.
func (s *Store) ListSendersByEmail(email string) ([]models.Sender, error) {
	senders := make([]models.Sender, 0)
	err := s.DB.Where("email = ?", email).Find(&senders).Error
	return senders, err
}

// ─────────────────────────────────────────────
// Suppression upsert (used by FBL + DSN)
// ─────────────────────────────────────────────

// UpsertSuppression inserts a SuppressedEmail or ignores it if already present.
func (s *Store) UpsertSuppression(se *models.SuppressedEmail) error {
	// Use INSERT OR IGNORE via raw SQL to avoid GORM unique constraint error
	return s.DB.Exec(`
		INSERT OR IGNORE INTO suppressed_emails (email, reason, domain, source_info, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, se.Email, se.Reason, se.Domain, se.SourceInfo, se.CreatedAt).Error
}
