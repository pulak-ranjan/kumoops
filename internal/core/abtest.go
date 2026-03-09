package core

// A/B Test Engine
//
// Manages variant assignment, winner auto-selection, and stat aggregation
// for A/B test campaigns. Winner selection runs every 5 minutes via scheduler.

import (
	"fmt"
	"log"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// ABTestService checks campaigns for auto-winner selection.
type ABTestService struct {
	st *store.Store
}

func NewABTestService(st *store.Store) *ABTestService {
	return &ABTestService{st: st}
}

// Run checks all active A/B test campaigns for auto-winner conditions.
// Called every 5 minutes from scheduler.
func (svc *ABTestService) Run() {
	var campaigns []models.Campaign
	if err := svc.st.DB.
		Where("is_ab_test = ? AND status IN ('sending','completed') AND ab_winner_variant_id IS NULL AND ab_win_after_hours > 0", true).
		Find(&campaigns).Error; err != nil {
		return
	}
	for _, c := range campaigns {
		svc.checkWinner(&c)
	}
}

// checkWinner evaluates whether it's time to pick a winner for a campaign.
func (svc *ABTestService) checkWinner(c *models.Campaign) {
	if c.ABWinAfterHours <= 0 {
		return
	}
	deadline := c.CreatedAt.Add(time.Duration(c.ABWinAfterHours) * time.Hour)
	if time.Now().Before(deadline) {
		return // not time yet
	}

	var variants []models.CampaignVariant
	if err := svc.st.DB.Where("campaign_id = ?", c.ID).Find(&variants).Error; err != nil || len(variants) == 0 {
		return
	}

	// Refresh per-variant open/click counts from CampaignRecipient table
	for i := range variants {
		svc.refreshVariantStats(&variants[i])
	}

	winner := pickWinner(variants, c.ABWinMetric)
	if winner == nil {
		return
	}

	now := time.Now()
	winner.IsWinner = true
	winner.WinnerAt = &now
	svc.st.DB.Save(winner)

	wid := winner.ID
	c.ABWinnerVariantID = &wid
	svc.st.DB.Save(c)

	log.Printf("[ABTest] Campaign %d — winner is variant %s (ID=%d) by %s",
		c.ID, winner.Name, winner.ID, c.ABWinMetric)
}

// refreshVariantStats recomputes open/click counts for a variant from CampaignRecipient.
func (svc *ABTestService) refreshVariantStats(v *models.CampaignVariant) {
	type row struct {
		Cnt int64
	}
	var opens, clicks row
	svc.st.DB.Raw(
		`SELECT COUNT(*) as cnt FROM campaign_recipients
		 WHERE campaign_id = ? AND variant_id = ? AND opened_at IS NOT NULL`, v.CampaignID, v.ID).
		Scan(&opens)
	svc.st.DB.Raw(
		`SELECT COUNT(*) as cnt FROM campaign_recipients
		 WHERE campaign_id = ? AND variant_id = ? AND clicked_at IS NOT NULL`, v.CampaignID, v.ID).
		Scan(&clicks)

	v.OpenCount = opens.Cnt
	v.ClickCount = clicks.Cnt
	svc.st.DB.Save(v)
}

// pickWinner selects the variant with the highest open or click rate.
func pickWinner(variants []models.CampaignVariant, metric string) *models.CampaignVariant {
	var best *models.CampaignVariant
	bestRate := -1.0

	for i := range variants {
		v := &variants[i]
		if v.SentCount == 0 {
			continue
		}
		var rate float64
		switch metric {
		case "click_rate":
			rate = float64(v.ClickCount) / float64(v.SentCount)
		default: // "open_rate"
			rate = float64(v.OpenCount) / float64(v.SentCount)
		}
		if rate > bestRate {
			bestRate = rate
			best = v
		}
	}
	return best
}

// AssignVariant picks a variant for a recipient based on their position in the send queue.
// recipientIndex is 0-based position among all pending recipients.
// Returns nil if the campaign is not an A/B test.
func AssignVariant(variants []models.CampaignVariant, recipientIndex int, totalRecipients int) *models.CampaignVariant {
	if len(variants) == 0 || totalRecipients == 0 {
		return nil
	}
	// Build cumulative split boundaries
	var boundaries []struct {
		variant *models.CampaignVariant
		cutoff  float64
	}
	cumulative := 0.0
	for i := range variants {
		cumulative += variants[i].SplitPct
		boundaries = append(boundaries, struct {
			variant *models.CampaignVariant
			cutoff  float64
		}{&variants[i], cumulative})
	}
	// Determine bucket for this recipient
	position := float64(recipientIndex) / float64(totalRecipients)
	for _, b := range boundaries {
		if position < b.cutoff {
			return b.variant
		}
	}
	return &variants[len(variants)-1]
}

// VariantSubject returns the subject to use: variant.Subject if set, otherwise campaign.Subject.
func VariantSubject(variant *models.CampaignVariant, campaign *models.Campaign) string {
	if variant != nil && variant.Subject != "" {
		return variant.Subject
	}
	return campaign.Subject
}

// VariantBody returns the HTML body to use: variant.HTMLBody if set, otherwise campaign.Body.
func VariantBody(variant *models.CampaignVariant, campaign *models.Campaign) string {
	if variant != nil && variant.HTMLBody != "" {
		return variant.HTMLBody
	}
	return campaign.Body
}

// ABTestSummary holds aggregated A/B test metrics for API responses.
type ABTestSummary struct {
	CampaignID uint                `json:"campaign_id"`
	Metric     string              `json:"metric"`
	WinnerID   *uint               `json:"winner_id"`
	Variants   []VariantMetrics    `json:"variants"`
}

type VariantMetrics struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	Subject    string  `json:"subject"`
	SentCount  int64   `json:"sent_count"`
	OpenCount  int64   `json:"open_count"`
	ClickCount int64   `json:"click_count"`
	OpenRate   float64 `json:"open_rate"`
	ClickRate  float64 `json:"click_rate"`
	IsWinner   bool    `json:"is_winner"`
}

// GetABTestSummary builds an ABTestSummary for a campaign.
func GetABTestSummary(st *store.Store, campaignID uint) (*ABTestSummary, error) {
	var c models.Campaign
	if err := st.DB.First(&c, campaignID).Error; err != nil {
		return nil, fmt.Errorf("campaign not found")
	}
	var variants []models.CampaignVariant
	if err := st.DB.Where("campaign_id = ?", campaignID).Find(&variants).Error; err != nil {
		return nil, err
	}

	svc := &ABTestService{st: st}
	var metrics []VariantMetrics
	for i := range variants {
		svc.refreshVariantStats(&variants[i])
		v := variants[i]
		var or_, cr float64
		if v.SentCount > 0 {
			or_ = float64(v.OpenCount) / float64(v.SentCount) * 100
			cr = float64(v.ClickCount) / float64(v.SentCount) * 100
		}
		metrics = append(metrics, VariantMetrics{
			ID:         v.ID,
			Name:       v.Name,
			Subject:    v.Subject,
			SentCount:  v.SentCount,
			OpenCount:  v.OpenCount,
			ClickCount: v.ClickCount,
			OpenRate:   or_,
			ClickRate:  cr,
			IsWinner:   v.IsWinner,
		})
	}
	return &ABTestSummary{
		CampaignID: campaignID,
		Metric:     c.ABWinMetric,
		WinnerID:   c.ABWinnerVariantID,
		Variants:   metrics,
	}, nil
}
