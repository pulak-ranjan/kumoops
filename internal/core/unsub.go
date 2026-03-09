package core

// One-Click Unsubscribe Engine
//
// Implements RFC 8058 (List-Unsubscribe=One-Click) and RFC 2369.
// Generates per-recipient HMAC-signed unsub tokens.
// On unsubscribe: marks the recipient, increments campaign counter, auto-suppresses.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// GenerateUnsubToken creates a stable, HMAC-signed token for a campaign recipient.
// Token format (URL-safe base64): "campaignID.recipientID.hmac12"
func GenerateUnsubToken(campaignID, recipientID uint) string {
	key, _ := GetEncryptionKey()
	data := fmt.Sprintf("%d.%d", campaignID, recipientID)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:12]
	return fmt.Sprintf("%d.%d.%s", campaignID, recipientID, sig)
}

// VerifyUnsubToken validates a token and returns (campaignID, recipientID, ok).
func VerifyUnsubToken(token string) (campaignID uint, recipientID uint, ok bool) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return 0, 0, false
	}
	var cid, rid uint64
	if _, err := fmt.Sscanf(parts[0], "%d", &cid); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &rid); err != nil {
		return 0, 0, false
	}
	expected := GenerateUnsubToken(uint(cid), uint(rid))
	if !hmac.Equal([]byte(expected), []byte(token)) {
		return 0, 0, false
	}
	return uint(cid), uint(rid), true
}

// ProcessUnsubscribe handles a confirmed unsubscribe event.
// Marks the recipient, increments the campaign counter, and suppresses the address.
func ProcessUnsubscribe(st *store.Store, token string) (*models.CampaignRecipient, error) {
	campaignID, recipientID, ok := VerifyUnsubToken(token)
	if !ok {
		return nil, fmt.Errorf("invalid or tampered unsubscribe token")
	}

	var recip models.CampaignRecipient
	if err := st.DB.First(&recip, recipientID).Error; err != nil {
		return nil, fmt.Errorf("recipient not found")
	}
	if recip.CampaignID != campaignID {
		return nil, fmt.Errorf("token mismatch")
	}

	now := time.Now()
	if recip.UnsubscribedAt == nil {
		recip.UnsubscribedAt = &now
		st.DB.Save(&recip)

		// Increment campaign counter (raw SQL to avoid race condition)
		st.DB.Exec("UPDATE campaigns SET total_unsubs = total_unsubs + 1 WHERE id = ?", campaignID)

		// Suppress the address globally
		_ = st.AddSuppression(recip.Email, "unsubscribe",
			fmt.Sprintf("campaign_id:%d", campaignID))
	}
	return &recip, nil
}

// ListUnsubscribeHeaders returns the two List-Unsubscribe header values for injection into campaign email.
// baseURL is the public-facing URL of KumoOps (e.g. https://mail.example.com).
func ListUnsubscribeHeaders(baseURL, token string) (listUnsub, listUnsubPost string) {
	url := fmt.Sprintf("%s/unsubscribe/%s", baseURL, token)
	listUnsub = fmt.Sprintf("<%s>", url)
	listUnsubPost = "List-Unsubscribe=One-Click"
	return
}
