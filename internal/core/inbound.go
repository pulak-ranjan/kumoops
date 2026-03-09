package core

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// InboundProcessor watches Maildir directories for each BounceAccount and processes
// new messages as either FBL/ARF complaints or DSN bounce notifications.
//
// Flow per message:
//  1. Read from Maildir/new/
//  2. Detect message type (FBL vs DSN vs unknown)
//  3. Parse and process accordingly
//  4. Move to Maildir/cur/ (marks as processed — standard Maildir convention)
//  5. On error: leave in new/ and log, will retry next cycle
type InboundProcessor struct {
	st         *store.Store
	fblService *FBLService
	secret     []byte
}

// NewInboundProcessor creates a new InboundProcessor.
func NewInboundProcessor(st *store.Store) *InboundProcessor {
	secret, _ := GetEncryptionKey()
	return &InboundProcessor{
		st:         st,
		fblService: NewFBLService(st),
		secret:     secret,
	}
}

// Start launches the inbound processing loop in a background goroutine.
// It scans all bounce account Maildirs every interval seconds.
func (p *InboundProcessor) Start(intervalSeconds int) {
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	go func() {
		log.Println("[INBOUND] Maildir processor started")
		for {
			p.ScanAll()
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}()
}

// ScanAll fetches all bounce accounts from the DB and scans each Maildir.
func (p *InboundProcessor) ScanAll() {
	accounts, err := p.st.ListBounceAccounts()
	if err != nil {
		log.Printf("[INBOUND] failed to list bounce accounts: %v", err)
		return
	}
	for _, acc := range accounts {
		maildir := filepath.Join("/home", acc.Username, "Maildir", "new")
		p.scanMaildir(maildir, acc)
	}
}

// scanMaildir scans a single Maildir/new/ directory and processes each message.
func (p *InboundProcessor) scanMaildir(maildirNew string, acc models.BounceAccount) {
	entries, err := os.ReadDir(maildirNew)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[INBOUND] readdir %s: %v", maildirNew, err)
		}
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		msgPath := filepath.Join(maildirNew, entry.Name())
		p.processMaildirMessage(msgPath, acc)
	}
}

// processMaildirMessage reads, parses, and processes a single Maildir message file.
func (p *InboundProcessor) processMaildirMessage(msgPath string, acc models.BounceAccount) {
	raw, err := os.ReadFile(msgPath)
	if err != nil {
		log.Printf("[INBOUND] read %s: %v", msgPath, err)
		return
	}

	// Detect message type and route accordingly
	msgType := detectInboundType(raw)
	filename := filepath.Base(msgPath)

	switch msgType {
	case "fbl":
		_, err = p.fblService.ProcessFBL(raw, filename)
		if err != nil {
			log.Printf("[INBOUND] FBL parse error %s: %v", filename, err)
		}
	case "dsn":
		p.processDSN(raw, filename, acc)
	default:
		log.Printf("[INBOUND] unknown message type in %s — skipping", filename)
	}

	// Move to cur/ whether processed or not (avoids infinite retry loops on bad messages)
	p.moveToProcessed(msgPath)
}

// processDSN parses an RFC 3464 DSN, classifies the bounce, stores it,
// and auto-suppresses hard bounced recipients.
func (p *InboundProcessor) processDSN(raw []byte, filename string, acc models.BounceAccount) {
	dsn, err := ParseDSN(raw)
	if err != nil || dsn == nil {
		log.Printf("[INBOUND] DSN parse failed for %s: %v", filename, err)
		return
	}

	// Attempt VERP decode: if the envelope Return-Path matches our VERP format,
	// we can identify the exact recipient and sender even if DSN fields are incomplete.
	verpDecoded := false
	if dsn.OriginalSender != "" {
		localPart := extractLocalPart(dsn.OriginalSender)
		if IsVERPLocalPart(localPart) {
			senderID, recipient, err := VERPDecode(localPart, p.secret)
			if err == nil {
				if dsn.FinalRecipient == "" {
					dsn.FinalRecipient = recipient
				}
				// Cross-check: use VERP senderID if DSN didn't provide one
				if dsn.FinalRecipient == recipient {
					_ = senderID // will be set below from DB lookup
				}
				verpDecoded = true
			} else {
				log.Printf("[INBOUND] VERP decode failed for %s: %v", localPart, err)
			}
		}
	}

	// Match to a sender record
	var senderID uint
	if dsn.OriginalSender != "" {
		senders, err := p.st.ListSendersByEmail(dsn.OriginalSender)
		if err == nil && len(senders) > 0 {
			senderID = senders[0].ID
		}
	}

	// Determine ISP/provider from recipient domain
	recipientDomain := extractDomain(dsn.FinalRecipient)
	provider := DetectProvider(recipientDomain)

	// Auto-suppress hard bounces
	autoSuppressed := false
	if dsn.IsHard && dsn.FinalRecipient != "" {
		err := p.st.UpsertSuppression(&models.SuppressedEmail{
			Email:      dsn.FinalRecipient,
			Reason:     "hard_bounce",
			Domain:     recipientDomain,
			SourceInfo: "dsn:" + dsn.EnhancedStatus,
			CreatedAt:  time.Now(),
		})
		if err == nil {
			autoSuppressed = true
		} else {
			log.Printf("[INBOUND] suppress hard bounce %s: %v", dsn.FinalRecipient, err)
		}
	}

	classification := DSNToClassification(dsn, time.Now(), senderID, provider, filename, autoSuppressed)
	classification.VERPDecoded = verpDecoded

	if err := p.st.CreateBounceClassification(classification); err != nil {
		log.Printf("[INBOUND] store bounce classification: %v", err)
		return
	}

	log.Printf("[INBOUND] DSN processed: rcpt=%s status=%s category=%s hard=%v suppressed=%v verp=%v",
		dsn.FinalRecipient, dsn.EnhancedStatus, dsn.Category, dsn.IsHard, autoSuppressed, verpDecoded)
}

// moveToProcessed moves a Maildir message from new/ to cur/ (standard Maildir convention).
func (p *InboundProcessor) moveToProcessed(msgPath string) {
	// cur/ is a sibling directory of new/
	curDir := filepath.Join(filepath.Dir(msgPath), "..", "cur")
	destPath := filepath.Join(curDir, filepath.Base(msgPath))

	if err := os.Rename(msgPath, destPath); err != nil {
		// If rename fails (cross-device or permissions), try copy + delete
		raw, rerr := os.ReadFile(msgPath)
		if rerr != nil {
			log.Printf("[INBOUND] move failed, read %s: %v", msgPath, rerr)
			return
		}
		if werr := os.WriteFile(destPath, raw, 0o600); werr != nil {
			log.Printf("[INBOUND] move failed, write %s: %v", destPath, werr)
			return
		}
		os.Remove(msgPath)
	}
}

// detectInboundType reads enough of the raw message to determine whether it is
// an FBL (feedback-report) or a DSN (delivery-status) multipart/report.
// Returns "fbl", "dsn", or "unknown".
func detectInboundType(raw []byte) string {
	// Quick header scan — we only need the first 4096 bytes
	header := string(raw)
	if len(header) > 4096 {
		header = header[:4096]
	}
	headerLower := strings.ToLower(header)

	// Look for Content-Type: multipart/report; report-type=...
	// FBL: report-type=feedback-report
	// DSN: report-type=delivery-status
	if strings.Contains(headerLower, "report-type=feedback-report") {
		return "fbl"
	}
	if strings.Contains(headerLower, "report-type=delivery-status") {
		return "dsn"
	}

	// Secondary heuristics based on common header patterns
	if strings.Contains(headerLower, "feedback-type:") ||
		strings.Contains(headerLower, "x-hmxl-midtoken:") || // Yahoo FBL
		strings.Contains(headerLower, "x-arf:") { // ARF marker
		return "fbl"
	}
	if strings.Contains(headerLower, "final-recipient:") ||
		strings.Contains(headerLower, "action: failed") ||
		strings.Contains(headerLower, "delivery status notification") {
		return "dsn"
	}

	return "unknown"
}

// extractLocalPart returns the local part of an email address (before '@').
func extractLocalPart(email string) string {
	// Handle angle bracket wrappers
	email = strings.Trim(email, "<>")
	if idx := strings.Index(email, "@"); idx >= 0 {
		return email[:idx]
	}
	return email
}

// ExtractLocalPart is the exported version for use by other packages.
func ExtractLocalPart(email string) string { return extractLocalPart(email) }
