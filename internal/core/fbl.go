package core

import (
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// FBLResult holds the parsed fields from an ARF/RFC 5965 feedback report.
type FBLResult struct {
	FeedbackType   string    // "abuse", "fraud", "virus", "unsubscribe", "other"
	ReportingMTA   string    // ISP/MTA that sent this FBL report
	OriginalSender string    // Envelope-from of the original complained message
	OriginalRcptTo string    // Original recipient who complained
	SourceIP       string    // IP that delivered the message to the ISP
	ArrivalDate    time.Time // When the original message arrived at ISP
	MessageID      string    // Message-ID of the original message
	RawHeaders     string    // Raw message/feedback-report headers
}

// ParseFBL parses an RFC 5965 Abuse Reporting Format (ARF) message from raw bytes.
// Returns nil if the message is not a valid FBL report.
func ParseFBL(raw []byte) (*FBLResult, error) {
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
	if reportType != "feedback-report" {
		return nil, nil
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, nil
	}

	result := &FBLResult{
		ReportingMTA: extractFBLDomain(msg.Header.Get("From")),
	}

	// Walk MIME parts to find message/feedback-report
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

		if strings.EqualFold(partMedia, "message/feedback-report") {
			parseFBLReportPart(part, result)
			break
		}
		io.Copy(io.Discard, part)
	}

	if result.FeedbackType == "" {
		return nil, nil // Not a valid ARF report
	}
	return result, nil
}

// parseFBLReportPart reads a message/feedback-report MIME part and fills result.
func parseFBLReportPart(r io.Reader, result *FBLResult) {
	body, err := io.ReadAll(r)
	if err != nil {
		return
	}
	result.RawHeaders = string(body)

	headers := parseRFC822Headers(string(body))

	if v := headers.Get("Feedback-Type"); v != "" {
		result.FeedbackType = strings.ToLower(strings.TrimSpace(v))
	}
	if v := headers.Get("Original-Rcpt-To"); v != "" {
		result.OriginalRcptTo = strings.ToLower(strings.TrimSpace(strings.Trim(v, "<>")))
	}
	if v := headers.Get("Original-Mail-From"); v != "" {
		result.OriginalSender = strings.ToLower(strings.TrimSpace(strings.Trim(v, "<>")))
	}
	if v := headers.Get("Source-Ip"); v != "" {
		result.SourceIP = strings.TrimSpace(v)
	} else if v := headers.Get("Source-IP"); v != "" {
		result.SourceIP = strings.TrimSpace(v)
	}
	if v := headers.Get("Arrival-Date"); v != "" {
		if t, err := mail.ParseDate(strings.TrimSpace(v)); err == nil {
			result.ArrivalDate = t
		}
	}
	if v := headers.Get("Original-Message-Id"); v != "" {
		result.MessageID = strings.TrimSpace(v)
	}
}

// FBLService handles processing of incoming FBL reports.
type FBLService struct {
	st     *store.Store
	secret []byte
}

// NewFBLService creates a new FBLService.
func NewFBLService(st *store.Store) *FBLService {
	secret, _ := GetEncryptionKey()
	return &FBLService{st: st, secret: secret}
}

// ProcessFBL parses raw FBL email bytes, stores the complaint record,
// and auto-suppresses the original recipient if applicable.
// Returns (record, nil) on success, or (nil, err) on failure.
func (f *FBLService) ProcessFBL(raw []byte, sourceFile string) (*models.FBLRecord, error) {
	fbl, err := ParseFBL(raw)
	if err != nil || fbl == nil {
		return nil, err
	}

	receivedAt := time.Now()

	// Try to match the original sender to a Sender record
	var senderID uint
	senderEmail := fbl.OriginalSender
	if senderEmail != "" {
		senders, err := f.st.ListSendersByEmail(senderEmail)
		if err == nil && len(senders) > 0 {
			senderID = senders[0].ID
		}
	}

	// Extract sending domain from original sender
	domain := extractDomain(fbl.OriginalSender)

	// Auto-suppress the complaining recipient
	autoSuppressed := false
	if fbl.OriginalRcptTo != "" {
		err := f.st.UpsertSuppression(&models.SuppressedEmail{
			Email:      fbl.OriginalRcptTo,
			Reason:     "spam_complaint",
			Domain:     extractDomain(fbl.OriginalRcptTo),
			SourceInfo: "fbl:" + fbl.ReportingMTA,
			CreatedAt:  receivedAt,
		})
		if err == nil {
			autoSuppressed = true
		} else {
			log.Printf("[FBL] failed to suppress %s: %v", fbl.OriginalRcptTo, err)
		}
	}

	arrivalDate := fbl.ArrivalDate
	if arrivalDate.IsZero() {
		arrivalDate = receivedAt
	}

	record := &models.FBLRecord{
		ReceivedAt:     receivedAt,
		FeedbackType:   fbl.FeedbackType,
		ReportingMTA:   fbl.ReportingMTA,
		SourceIP:       fbl.SourceIP,
		OriginalSender: fbl.OriginalSender,
		OriginalRcptTo: fbl.OriginalRcptTo,
		ArrivalDate:    arrivalDate,
		MessageID:      fbl.MessageID,
		SenderID:       senderID,
		SenderEmail:    senderEmail,
		Domain:         domain,
		AutoSuppressed: autoSuppressed,
		RawHeaders:     truncateFBL(fbl.RawHeaders, 2000),
	}

	if err := f.st.CreateFBLRecord(record); err != nil {
		return nil, err
	}

	log.Printf("[FBL] Complaint recorded: type=%s rcpt=%s from=%s isp=%s suppressed=%v",
		fbl.FeedbackType, fbl.OriginalRcptTo, fbl.OriginalSender, fbl.ReportingMTA, autoSuppressed)

	return record, nil
}

// ComplaintRateReport holds complaint rate statistics per sender/domain.
type ComplaintRateReport struct {
	SenderEmail    string  `json:"sender_email"`
	Domain         string  `json:"domain"`
	TotalSent      int64   `json:"total_sent"`
	TotalComplaints int64  `json:"total_complaints"`
	ComplaintRate  float64 `json:"complaint_rate"` // percentage
	ISPThreshold   float64 `json:"isp_threshold"`  // 0.08 for Gmail, 0.10 for Yahoo
	AboveThreshold bool    `json:"above_threshold"`
	Provider       string  `json:"provider"` // Which ISP the complaint rate is against
}

// ISP complaint rate thresholds (percentage)
var ispComplaintThresholds = map[string]float64{
	"Gmail":   0.08,
	"Yahoo":   0.10,
	"Outlook": 0.30,
	"AOL":     0.10,
}

// DefaultComplaintThreshold is used when ISP is unknown.
const DefaultComplaintThreshold = 0.10

// ComplaintThresholdFor returns the complaint rate threshold for a given ISP.
func ComplaintThresholdFor(isp string) float64 {
	if t, ok := ispComplaintThresholds[isp]; ok {
		return t
	}
	return DefaultComplaintThreshold
}

// --- Internal helpers ---

// extractFBLDomain extracts a domain from an email address string like "From: <fbl@yahoo.com>"
func extractFBLDomain(fromHeader string) string {
	// Try to parse as email address
	addr, err := mail.ParseAddress(fromHeader)
	if err == nil {
		return extractDomain(addr.Address)
	}
	// Fallback: find "@" and take the domain
	if idx := strings.LastIndex(fromHeader, "@"); idx >= 0 {
		end := strings.IndexAny(fromHeader[idx:], " \t\r\n>\"'")
		if end < 0 {
			end = len(fromHeader) - idx
		}
		return strings.ToLower(fromHeader[idx+1 : idx+end])
	}
	return fromHeader
}

// truncateFBL caps a string to maxLen bytes (for raw header storage).
// Named to avoid conflict with the truncate() function in telegram.go.
func truncateFBL(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
