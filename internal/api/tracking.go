package api

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
	"gorm.io/gorm"
)

// Transparent 1x1 GIF
var pixelGIF, _ = base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")

type TrackingHandler struct {
	Store *store.Store
}

func NewTrackingHandler(st *store.Store) *TrackingHandler {
	return &TrackingHandler{Store: st}
}

// GET /api/track/open/{recipient_id}
func (h *TrackingHandler) HandleTrackOpen(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	if id > 0 {
		go h.recordOpen(uint(id), r)
	}

	// Return transparent pixel
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(pixelGIF)
}

// GET /api/track/click/{recipient_id}?url=...
func (h *TrackingHandler) HandleTrackClick(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)
	targetURL := r.URL.Query().Get("url")
	signature := r.URL.Query().Get("sig")

	// Security: Prevent Open Redirect using HMAC
	// We verify that the 'url' param was signed by our system.
	if targetURL == "" || !core.VerifyLinkSignature(targetURL, signature) {
		http.Error(w, "Invalid link signature", http.StatusForbidden)
		return
	}

	if id > 0 {
		go h.recordClick(uint(id))
	}

	http.Redirect(w, r, targetURL, http.StatusFound)
}

func (h *TrackingHandler) recordOpen(id uint, r *http.Request) {
	var recip models.CampaignRecipient
	if err := h.Store.DB.First(&recip, id).Error; err != nil {
		return
	}

	// Update Recipient
	now := time.Now()
	if recip.OpenedAt == nil {
		recip.OpenedAt = &now
		h.Store.DB.Save(&recip)

		// Increment Campaign Stats (atomic to prevent race conditions)
		h.Store.DB.Model(&models.Campaign{}).Where("id = ?", recip.CampaignID).
			UpdateColumn("total_opens", gorm.Expr("total_opens + 1"))

		// Update Contact Score (AI Superlead) — atomic
		if recip.ContactID > 0 {
			h.Store.DB.Model(&models.Contact{}).Where("id = ?", recip.ContactID).
				UpdateColumns(map[string]interface{}{
					"total_opens": gorm.Expr("total_opens + 1"),
					"score":       gorm.Expr("score + 1"),
				})
		}
	}
}

func (h *TrackingHandler) recordClick(id uint) {
	var recip models.CampaignRecipient
	if err := h.Store.DB.First(&recip, id).Error; err != nil {
		return
	}

	now := time.Now()
	if recip.ClickedAt == nil {
		recip.ClickedAt = &now
		h.Store.DB.Save(&recip)

		// Increment Campaign Stats (atomic to prevent race conditions)
		h.Store.DB.Model(&models.Campaign{}).Where("id = ?", recip.CampaignID).
			UpdateColumn("total_clicks", gorm.Expr("total_clicks + 1"))

		// Update Contact Score — atomic
		if recip.ContactID > 0 {
			h.Store.DB.Model(&models.Contact{}).Where("id = ?", recip.ContactID).
				UpdateColumns(map[string]interface{}{
					"total_clicks": gorm.Expr("total_clicks + 1"),
					"score":        gorm.Expr("score + 5"),
				})
		}
	}
}
