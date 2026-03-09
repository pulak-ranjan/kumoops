package core

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
)

// DSNResult holds all fields extracted from a parsed DSN (RFC 3464) message.
type DSNResult struct {
	// Per-recipient fields from message/delivery-status
	FinalRecipient string
	Action         string // "failed", "delayed", "delivered", "relayed", "expanded"
	EnhancedStatus string // e.g. "5.1.1"
	DiagnosticCode string // full SMTP response
	// Envelope fields from the outer message
	OriginalSender string
	// Classification
	Category string // "hard", "soft", "block", "quota", "dns", "tls", "auth", "unknown"
	IsHard   bool
}

// ParseDSN parses an RFC 3464 delivery status notification from raw email bytes.
// Returns nil if the message is not a DSN.
func ParseDSN(raw []byte) (*DSNResult, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}

	ct := msg.Header.Get("Content-Type")
	if ct == "" {
		return nil, nil
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return nil, nil
	}
	if !strings.EqualFold(mediaType, "multipart/report") {
		return nil, nil
	}
	reportType := strings.ToLower(params["report-type"])
	if reportType != "delivery-status" {
		return nil, nil // Not a DSN — might be FBL, handled elsewhere
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, nil
	}

	result := &DSNResult{
		OriginalSender: msg.Header.Get("Return-Path"),
	}
	// Fallback: try From if no Return-Path
	if result.OriginalSender == "" {
		result.OriginalSender = msg.Header.Get("From")
	}

	// Walk MIME parts to find message/delivery-status
	mr := multipart.NewReader(msg.Body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		partCT := part.Header.Get("Content-Type")
		partMedia, _, _ := mime.ParseMediaType(partCT)
		if !strings.EqualFold(partMedia, "message/delivery-status") {
			io.Copy(io.Discard, part) // consume part
			continue
		}
		// Parse the delivery-status body (RFC 3464 header format)
		parseDSNStatusPart(part, result)
		break
	}

	if result.FinalRecipient == "" && result.EnhancedStatus == "" {
		return nil, nil // No useful DSN data found
	}

	result.Category, result.IsHard = classifyDSN(result.EnhancedStatus, result.Action, result.DiagnosticCode)
	return result, nil
}

// parseDSNStatusPart reads the message/delivery-status MIME part and fills result.
func parseDSNStatusPart(r io.Reader, result *DSNResult) {
	body, err := io.ReadAll(r)
	if err != nil {
		return
	}

	// The delivery-status body consists of one or more RFC 822-style header blocks
	// separated by blank lines. First block = per-message, subsequent = per-recipient.
	// We want the first per-recipient block (second block).
	blocks := strings.Split(string(body), "\n\n")
	if len(blocks) < 2 {
		// Might be just one block — try parsing it anyway
		blocks = append(blocks, "")
	}

	for _, block := range blocks[1:] { // Skip per-message block
		if strings.TrimSpace(block) == "" {
			continue
		}
		headers := parseRFC822Headers(block)
		if v := headers.Get("Final-Recipient"); v != "" {
			// Format: "rfc822; user@example.com"
			result.FinalRecipient = extractAddrFromDSNField(v)
		}
		if v := headers.Get("Action"); v != "" {
			result.Action = strings.ToLower(strings.TrimSpace(v))
		}
		if v := headers.Get("Status"); v != "" {
			result.EnhancedStatus = strings.TrimSpace(strings.Split(v, " ")[0])
		}
		if v := headers.Get("Diagnostic-Code"); v != "" {
			result.DiagnosticCode = extractDSNDiagnostic(v)
		}
		if result.FinalRecipient != "" {
			break // Only need first recipient block
		}
	}
}

// classifyDSN maps an enhanced status code + action to a human-readable category.
//
// Categories:
//   - hard    → permanent failure, suppress the recipient
//   - soft    → temporary failure, retry according to schedule
//   - block   → policy/reputation block (5.7.x, 5.3.x)
//   - quota   → mailbox full / over quota (4.2.2, 5.2.2)
//   - dns     → DNS-related failure (5.1.2, 5.4.4)
//   - tls     → TLS negotiation failure
//   - auth    → authentication/authorization failure (5.7.x auth-related)
//   - unknown → unclassified
func classifyDSN(status, action, diagnostic string) (category string, isHard bool) {
	status = strings.TrimSpace(status)
	action = strings.ToLower(strings.TrimSpace(action))
	diagLower := strings.ToLower(diagnostic)

	if status == "" {
		// Fall back to diagnostic heuristics
		return classifyByDiagnostic(diagLower, action)
	}

	parts := strings.SplitN(status, ".", 3)
	if len(parts) < 3 {
		return classifyByDiagnostic(diagLower, action)
	}

	class := parts[0] // "4" or "5"
	subject := parts[1]
	detail := parts[2]

	switch {
	// TLS failures
	case strings.Contains(diagLower, "tls") || strings.Contains(diagLower, "starttls") ||
		strings.Contains(diagLower, "ssl") || strings.Contains(diagLower, "certificate"):
		return "tls", class == "5"

	// Quota / mailbox full
	case (subject == "2" && detail == "2") || (subject == "2" && detail == "1"):
		if class == "4" {
			return "quota", false
		}
		return "hard", true // 5.2.1 = mailbox disabled

	// DNS / routing
	case subject == "1" && detail == "2":
		return "dns", class == "5"
	case subject == "4" && detail == "4":
		return "dns", class == "5"

	// Unknown user / invalid address
	case subject == "1" && (detail == "1" || detail == "0"):
		return "hard", true

	// Policy / reputation blocks
	case subject == "7":
		if class == "5" {
			// Distinguish auth failures from spam blocks
			if strings.Contains(diagLower, "auth") || strings.Contains(diagLower, "dkim") ||
				strings.Contains(diagLower, "spf") || strings.Contains(diagLower, "dmarc") {
				return "auth", true
			}
			return "block", true
		}
		return "block", false // 4.7.x = temp block / rate limit

	// Rate limiting / too many connections
	case subject == "4" && class == "4":
		return "soft", false

	// Permanent failures
	case class == "5":
		return "hard", true

	// Temporary failures
	case class == "4" && action == "delayed":
		return "soft", false

	default:
		return classifyByDiagnostic(diagLower, action)
	}
}

func classifyByDiagnostic(diagLower, action string) (string, bool) {
	switch {
	case strings.Contains(diagLower, "user unknown") || strings.Contains(diagLower, "no such user") ||
		strings.Contains(diagLower, "does not exist") || strings.Contains(diagLower, "invalid address"):
		return "hard", true
	case strings.Contains(diagLower, "spam") || strings.Contains(diagLower, "blocked") ||
		strings.Contains(diagLower, "blacklist") || strings.Contains(diagLower, "policy"):
		return "block", true
	case strings.Contains(diagLower, "quota") || strings.Contains(diagLower, "full") ||
		strings.Contains(diagLower, "over the limit"):
		return "quota", false
	case strings.Contains(diagLower, "tls") || strings.Contains(diagLower, "ssl"):
		return "tls", false
	case strings.Contains(diagLower, "dns") || strings.Contains(diagLower, "host not found") ||
		strings.Contains(diagLower, "name or service not known"):
		return "dns", false
	case action == "delayed":
		return "soft", false
	default:
		return "unknown", false
	}
}

// DSNToClassification converts a DSNResult into a BounceClassification model
// ready to be saved to the database.
func DSNToClassification(dsn *DSNResult, receivedAt time.Time, senderID uint, provider string, sourceFile string, autoSuppressed bool) *models.BounceClassification {
	return &models.BounceClassification{
		ReceivedAt:     receivedAt,
		Category:       dsn.Category,
		IsHard:         dsn.IsHard,
		FinalRecipient: dsn.FinalRecipient,
		EnhancedStatus: dsn.EnhancedStatus,
		Action:         dsn.Action,
		DiagnosticCode: dsn.DiagnosticCode,
		OriginalSender: dsn.OriginalSender,
		SenderID:       senderID,
		Domain:         extractDomain(dsn.FinalRecipient),
		Provider:       provider,
		AutoSuppressed: autoSuppressed,
		SourceFile:     sourceFile,
	}
}

// --- Internal helpers ---

// parseRFC822Headers parses a block of RFC 822-style header text into textproto.MIMEHeader.
func parseRFC822Headers(block string) textproto.MIMEHeader {
	h := make(textproto.MIMEHeader)
	// Unfold multi-line headers
	block = strings.ReplaceAll(block, "\r\n", "\n")
	// Join continuation lines
	lines := strings.Split(block, "\n")
	var currentKey, currentVal string
	flush := func() {
		if currentKey != "" {
			h.Add(currentKey, strings.TrimSpace(currentVal))
		}
	}
	for _, line := range lines {
		if line == "" {
			flush()
			currentKey, currentVal = "", ""
			continue
		}
		// Continuation line (starts with whitespace)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			currentVal += " " + strings.TrimSpace(line)
			continue
		}
		flush()
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			currentKey, currentVal = "", ""
			continue
		}
		currentKey = strings.TrimSpace(line[:idx])
		currentVal = strings.TrimSpace(line[idx+1:])
	}
	flush()
	return h
}

// extractAddrFromDSNField extracts the email address from a DSN field value
// like "rfc822; user@example.com" or "rfc822;user@example.com".
func extractAddrFromDSNField(v string) string {
	if idx := strings.Index(v, ";"); idx >= 0 {
		v = v[idx+1:]
	}
	v = strings.TrimSpace(v)
	// Remove angle brackets if present
	v = strings.Trim(v, "<>")
	return strings.ToLower(strings.TrimSpace(v))
}

// extractDSNDiagnostic strips the "smtp; " or "X-Postfix; " prefix from Diagnostic-Code.
func extractDSNDiagnostic(v string) string {
	if idx := strings.Index(v, ";"); idx >= 0 {
		return strings.TrimSpace(v[idx+1:])
	}
	return strings.TrimSpace(v)
}

// ExtractDomain is the exported wrapper around extractDomain (defined in stats.go).
func ExtractDomain(email string) string { return extractDomain(email) }

// DetectProvider maps a recipient domain to a well-known ISP name.
func DetectProvider(domain string) string {
	domain = strings.ToLower(domain)
	switch {
	case strings.HasSuffix(domain, "gmail.com") || strings.HasSuffix(domain, "googlemail.com"):
		return "Gmail"
	case strings.HasSuffix(domain, "yahoo.com") || strings.HasSuffix(domain, "yahoo.co.uk") ||
		strings.HasSuffix(domain, "ymail.com") || strings.HasSuffix(domain, "yahoo.fr"):
		return "Yahoo"
	case strings.HasSuffix(domain, "hotmail.com") || strings.HasSuffix(domain, "outlook.com") ||
		strings.HasSuffix(domain, "live.com") || strings.HasSuffix(domain, "msn.com"):
		return "Outlook"
	case strings.HasSuffix(domain, "aol.com") || strings.HasSuffix(domain, "aim.com"):
		return "AOL"
	case strings.HasSuffix(domain, "icloud.com") || strings.HasSuffix(domain, "me.com") ||
		strings.HasSuffix(domain, "mac.com"):
		return "Apple"
	case strings.HasSuffix(domain, "protonmail.com") || strings.HasSuffix(domain, "proton.me"):
		return "ProtonMail"
	default:
		return "Other"
	}
}
