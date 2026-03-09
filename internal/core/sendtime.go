package core

// Send-Time Optimization Engine
//
// Analyzes past engagement (opens, clicks) by hour-of-day and day-of-week
// to recommend optimal send windows per ISP / domain.
// Data source: campaign_recipients.opened_at + clicked_at fields.

import (
	"time"

	"github.com/pulak-ranjan/kumoops/internal/store"
)

// HourlyEngagement holds aggregated engagement counts for one hour slot.
type HourlyEngagement struct {
	Hour       int     `json:"hour"`        // 0-23 UTC
	DayOfWeek  int     `json:"day_of_week"` // 0=Sun … 6=Sat
	Opens      int64   `json:"opens"`
	Clicks     int64   `json:"clicks"`
	Total      int64   `json:"total"`       // opens + clicks
	EngageRate float64 `json:"engage_rate"` // (opens+clicks) / total_sent in that slot
}

// SendTimeRecommendation is the top-N best send windows.
type SendTimeRecommendation struct {
	Hour      int     `json:"hour"`
	DayOfWeek int     `json:"day_of_week"`
	DayName   string  `json:"day_name"`
	Score     float64 `json:"score"` // 0-100
	Label     string  `json:"label"` // "Best", "Good", "Average"
}

// Heatmap is the full 7×24 engagement grid returned to the frontend.
type Heatmap struct {
	Cells           []HourlyEngagement       `json:"cells"`
	Recommendations []SendTimeRecommendation `json:"recommendations"`
	DataDays        int                      `json:"data_days"` // how many days of history used
}

var dayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// GetSendTimeHeatmap aggregates engagement data for a domain over the last N days.
// If domain is empty, aggregates across all domains.
func GetSendTimeHeatmap(st *store.Store, domain string, days int) (*Heatmap, error) {
	if days <= 0 {
		days = 90
	}
	since := time.Now().AddDate(0, 0, -days)

	type row struct {
		HourSlot  int
		DaySlot   int
		OpenCount int64
		ClickCount int64
		SentCount int64
	}

	// Opens heatmap
	openQuery := `
		SELECT
			CAST(strftime('%H', opened_at) AS INTEGER) as hour_slot,
			CAST(strftime('%w', opened_at) AS INTEGER) as day_slot,
			COUNT(*) as open_count
		FROM campaign_recipients
		WHERE opened_at IS NOT NULL AND opened_at >= ?`
	if domain != "" {
		openQuery += ` AND campaign_id IN (SELECT id FROM campaigns WHERE sender_id IN (SELECT id FROM senders WHERE domain_id IN (SELECT id FROM domains WHERE name = ?)))`
	}
	openQuery += ` GROUP BY hour_slot, day_slot`

	var openRows []struct {
		HourSlot  int
		DaySlot   int
		OpenCount int64
	}
	qOpen := st.DB.Raw(openQuery, since)
	if domain != "" {
		qOpen = st.DB.Raw(openQuery, since, domain)
	}
	qOpen.Scan(&openRows)

	// Clicks heatmap
	clickQuery := `
		SELECT
			CAST(strftime('%H', clicked_at) AS INTEGER) as hour_slot,
			CAST(strftime('%w', clicked_at) AS INTEGER) as day_slot,
			COUNT(*) as open_count
		FROM campaign_recipients
		WHERE clicked_at IS NOT NULL AND clicked_at >= ?`
	if domain != "" {
		clickQuery += ` AND campaign_id IN (SELECT id FROM campaigns WHERE sender_id IN (SELECT id FROM senders WHERE domain_id IN (SELECT id FROM domains WHERE name = ?)))`
	}
	clickQuery += ` GROUP BY hour_slot, day_slot`

	var clickRows []struct {
		HourSlot  int
		DaySlot   int
		OpenCount int64
	}
	qClick := st.DB.Raw(clickQuery, since)
	if domain != "" {
		qClick = st.DB.Raw(clickQuery, since, domain)
	}
	qClick.Scan(&clickRows)

	// Build 7×24 grid
	grid := make(map[[2]int]*HourlyEngagement)
	for d := 0; d < 7; d++ {
		for h := 0; h < 24; h++ {
			grid[[2]int{d, h}] = &HourlyEngagement{Hour: h, DayOfWeek: d}
		}
	}
	for _, r := range openRows {
		if r.HourSlot >= 0 && r.HourSlot < 24 && r.DaySlot >= 0 && r.DaySlot < 7 {
			grid[[2]int{r.DaySlot, r.HourSlot}].Opens += r.OpenCount
			grid[[2]int{r.DaySlot, r.HourSlot}].Total += r.OpenCount
		}
	}
	for _, r := range clickRows {
		if r.HourSlot >= 0 && r.HourSlot < 24 && r.DaySlot >= 0 && r.DaySlot < 7 {
			grid[[2]int{r.DaySlot, r.HourSlot}].Clicks += r.OpenCount
			grid[[2]int{r.DaySlot, r.HourSlot}].Total += r.OpenCount
		}
	}

	// Find max for normalisation
	var maxTotal int64
	for _, cell := range grid {
		if cell.Total > maxTotal {
			maxTotal = cell.Total
		}
	}

	// Build flat slice
	var cells []HourlyEngagement
	for _, cell := range grid {
		if maxTotal > 0 {
			cell.EngageRate = float64(cell.Total) / float64(maxTotal) * 100
		}
		cells = append(cells, *cell)
	}

	// Top-5 recommendations
	type scored struct {
		day, hour int
		score     float64
	}
	var ranked []scored
	for _, cell := range cells {
		ranked = append(ranked, scored{cell.DayOfWeek, cell.Hour, cell.EngageRate})
	}
	// Simple sort — bubble top-5
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[i].score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	var recs []SendTimeRecommendation
	for i, r := range ranked {
		if i >= 5 {
			break
		}
		label := "Average"
		if i == 0 {
			label = "Best"
		} else if i < 3 {
			label = "Good"
		}
		recs = append(recs, SendTimeRecommendation{
			Hour:      r.hour,
			DayOfWeek: r.day,
			DayName:   dayNames[r.day],
			Score:     r.score,
			Label:     label,
		})
	}

	return &Heatmap{
		Cells:           cells,
		Recommendations: recs,
		DataDays:        days,
	}, nil
}
