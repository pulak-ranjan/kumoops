package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// VERPEncode generates a VERP-encoded return-path local part.
//
// Format: bounces+{senderID}.{hmac8}.{b64recipient}
//   - senderID:    database ID of the Sender record
//   - hmac8:       first 11 chars of base64url-encoded HMAC-SHA256 of the payload
//   - b64recipient: base64url-encoded (no padding) recipient email address
//
// Example: bounces+42.dX93kQzA1bE.dXNlckBleGFtcGxlLmNvbQ@bounces.example.com
func VERPEncode(bounceDomain string, senderID uint, recipientEmail string, secret []byte) string {
	b64rcpt := base64.RawURLEncoding.EncodeToString([]byte(recipientEmail))
	payload := fmt.Sprintf("%d.%s", senderID, b64rcpt)
	mac := verpHMAC(payload, secret)
	return fmt.Sprintf("bounces+%s.%s@%s", payload, mac, bounceDomain)
}

// VERPDecode parses a VERP-encoded local part (everything before '@') and
// recovers the original senderID and recipient email.
// Returns an error if the format is wrong or the HMAC does not match.
func VERPDecode(localPart string, secret []byte) (senderID uint, recipientEmail string, err error) {
	if !strings.HasPrefix(localPart, "bounces+") {
		return 0, "", fmt.Errorf("not a VERP address: missing bounces+ prefix")
	}
	inner := strings.TrimPrefix(localPart, "bounces+")

	// Expected parts: senderID . hmac8 . b64recipient
	// Split into exactly 3 parts from the left
	parts := strings.SplitN(inner, ".", 3)
	if len(parts) != 3 {
		return 0, "", fmt.Errorf("invalid VERP format: expected 3 dot-separated parts, got %d", len(parts))
	}

	rawID, rawMAC, rawB64 := parts[0], parts[1], parts[2]

	// Parse sender ID
	id64, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid VERP sender ID: %w", err)
	}

	// Verify HMAC
	payload := fmt.Sprintf("%s.%s", rawID, rawB64)
	expectedMAC := verpHMAC(payload, secret)
	if !hmac.Equal([]byte(rawMAC), []byte(expectedMAC)) {
		return 0, "", fmt.Errorf("VERP HMAC mismatch: address may be forged or secret changed")
	}

	// Decode recipient
	emailBytes, err := base64.RawURLEncoding.DecodeString(rawB64)
	if err != nil {
		return 0, "", fmt.Errorf("failed to decode VERP recipient: %w", err)
	}

	return uint(id64), string(emailBytes), nil
}

// VERPAddress returns a fully-qualified VERP return-path address.
// Convenience wrapper around VERPEncode.
func VERPAddress(bounceDomain string, senderID uint, recipientEmail string, secret []byte) string {
	return VERPEncode(bounceDomain, senderID, recipientEmail, secret)
}

// IsVERPLocalPart returns true if the local part looks like a VERP-encoded address.
func IsVERPLocalPart(localPart string) bool {
	return strings.HasPrefix(localPart, "bounces+")
}

// verpHMAC computes an 11-character base64url-encoded HMAC-SHA256 truncated tag.
// 11 chars of base64url gives ~66 bits of security — sufficient for VERP authentication.
func verpHMAC(data string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	full := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if len(full) > 11 {
		return full[:11]
	}
	return full
}
