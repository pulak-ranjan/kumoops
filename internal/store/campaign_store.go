package store

import (
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
)

// ─────────────────────────────────────────────
// Campaign Variants (A/B Testing)
// ─────────────────────────────────────────────

func (s *Store) CreateCampaignVariant(v *models.CampaignVariant) error {
	return s.DB.Create(v).Error
}

func (s *Store) ListCampaignVariants(campaignID uint) ([]models.CampaignVariant, error) {
	vs := make([]models.CampaignVariant, 0)
	return vs, s.DB.Where("campaign_id = ?", campaignID).Order("name").Find(&vs).Error
}

func (s *Store) GetCampaignVariant(id uint) (*models.CampaignVariant, error) {
	var v models.CampaignVariant
	return &v, s.DB.First(&v, id).Error
}

func (s *Store) UpdateCampaignVariant(v *models.CampaignVariant) error {
	return s.DB.Save(v).Error
}

func (s *Store) DeleteCampaignVariant(id uint) error {
	return s.DB.Delete(&models.CampaignVariant{}, id).Error
}

func (s *Store) SetABWinner(campaignID, variantID uint) error {
	now := time.Now()
	// Mark variant as winner
	if err := s.DB.Model(&models.CampaignVariant{}).
		Where("id = ?", variantID).
		Updates(map[string]interface{}{"is_winner": true, "winner_at": now}).Error; err != nil {
		return err
	}
	// Update campaign
	return s.DB.Model(&models.Campaign{}).
		Where("id = ?", campaignID).
		Update("ab_winner_variant_id", variantID).Error
}

// ─────────────────────────────────────────────
// Seed Mailboxes (Inbox Placement)
// ─────────────────────────────────────────────

func (s *Store) CreateSeedMailbox(m *models.SeedMailbox) error {
	return s.DB.Create(m).Error
}

func (s *Store) ListSeedMailboxes() ([]models.SeedMailbox, error) {
	ms := make([]models.SeedMailbox, 0)
	return ms, s.DB.Order("isp, email").Find(&ms).Error
}

func (s *Store) GetSeedMailbox(id uint) (*models.SeedMailbox, error) {
	var m models.SeedMailbox
	return &m, s.DB.First(&m, id).Error
}

func (s *Store) UpdateSeedMailbox(m *models.SeedMailbox) error {
	return s.DB.Save(m).Error
}

func (s *Store) DeleteSeedMailbox(id uint) error {
	return s.DB.Delete(&models.SeedMailbox{}, id).Error
}

// ─────────────────────────────────────────────
// Placement Tests
// ─────────────────────────────────────────────

func (s *Store) CreatePlacementTest(t *models.PlacementTest) error {
	return s.DB.Create(t).Error
}

func (s *Store) ListPlacementTests(limit int) ([]models.PlacementTest, error) {
	ts := make([]models.PlacementTest, 0)
	q := s.DB.Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return ts, q.Find(&ts).Error
}

func (s *Store) GetPlacementTest(id uint) (*models.PlacementTest, error) {
	var t models.PlacementTest
	err := s.DB.Preload("Results").First(&t, id).Error
	return &t, err
}

func (s *Store) UpdatePlacementTest(t *models.PlacementTest) error {
	return s.DB.Save(t).Error
}

func (s *Store) ListPlacementResults(testID uint) ([]models.PlacementResult, error) {
	rs := make([]models.PlacementResult, 0)
	return rs, s.DB.Where("test_id = ?", testID).Order("isp").Find(&rs).Error
}
