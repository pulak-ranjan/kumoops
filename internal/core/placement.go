package core

// Inbox Placement Testing Engine
//
// Sends a test email through KumoMTA to all active seed mailboxes,
// then polls each mailbox via IMAP to determine if mail landed in inbox or spam.
// Results are stored in PlacementTest + PlacementResult records.

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

// PlacementService runs inbox placement tests.
type PlacementService struct {
	st *store.Store
}

func NewPlacementService(st *store.Store) *PlacementService {
	return &PlacementService{st: st}
}

// RunTest executes the given PlacementTest (must already be persisted with Status="pending").
// Updates PlacementTest status and creates PlacementResult records.
func (svc *PlacementService) RunTest(test *models.PlacementTest) {
	test.Status = "running"
	test.StartedAt = time.Now()
	svc.st.DB.Save(test)

	// Get active seed mailboxes
	var mailboxes []models.SeedMailbox
	if err := svc.st.DB.Where("is_active = ?", true).Find(&mailboxes).Error; err != nil || len(mailboxes) == 0 {
		test.Status = "failed"
		svc.st.DB.Save(test)
		return
	}

	// Get sender
	var sender models.Sender
	if err := svc.st.DB.Preload("Domain").First(&sender, test.SenderID).Error; err != nil {
		test.Status = "failed"
		svc.st.DB.Save(test)
		return
	}

	// Generate unique Message-ID for tracking
	msgID := fmt.Sprintf("<%d.placement.%d@kumoops>", test.ID, time.Now().UnixNano())

	// Collect recipient emails
	var toAddrs []string
	for _, mb := range mailboxes {
		toAddrs = append(toAddrs, mb.Email)
	}

	// Send the test email via local KumoMTA SMTP
	if err := svc.sendTestEmail(sender, test, msgID, toAddrs); err != nil {
		log.Printf("[Placement] send error for test %d: %v", test.ID, err)
		test.Status = "failed"
		svc.st.DB.Save(test)
		return
	}

	log.Printf("[Placement] Test %d sent (msg-id=%s), waiting 90s before checking...", test.ID, msgID)

	// Wait for email to be delivered
	time.Sleep(90 * time.Second)

	// Check each mailbox
	var results []models.PlacementResult
	for _, mb := range mailboxes {
		result := svc.checkMailbox(test.ID, &mb, msgID)
		results = append(results, result)
		svc.st.DB.Create(&result)
	}

	// Compute aggregate rates
	var inbox, spam, missing int
	for _, r := range results {
		switch r.Placement {
		case "inbox":
			inbox++
		case "spam":
			spam++
		default:
			missing++
		}
	}
	total := len(results)
	if total > 0 {
		test.InboxRate = float64(inbox) / float64(total) * 100
		test.SpamRate = float64(spam) / float64(total) * 100
		test.MissingRate = float64(missing) / float64(total) * 100
	}
	now := time.Now()
	test.Status = "completed"
	test.CompletedAt = &now
	svc.st.DB.Save(test)
	log.Printf("[Placement] Test %d complete: inbox=%.0f%% spam=%.0f%% missing=%.0f%%",
		test.ID, test.InboxRate, test.SpamRate, test.MissingRate)
}

// sendTestEmail sends the placement test email to all seed addresses via local SMTP.
func (svc *PlacementService) sendTestEmail(sender models.Sender, test *models.PlacementTest, msgID string, toAddrs []string) error {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:25", 10*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP connect: %w", err)
	}
	client, err := smtp.NewClient(conn, "localhost")
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP handshake: %w", err)
	}
	defer client.Quit()

	if err := client.Mail(sender.Email); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	for _, to := range toAddrs {
		_ = client.Rcpt(to)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: [PlacementTest] %s\r\nMessage-ID: %s\r\n"+
			"X-Mailer: KumoOps-PlacementTester/1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n",
		sender.Email,
		strings.Join(toAddrs, ", "),
		test.Subject,
		msgID,
	)
	body := test.HTMLBody
	if body == "" {
		body = "<p>This is an inbox placement test email from KumoOps.</p>"
	}
	_, err = fmt.Fprint(wc, headers+body)
	if err != nil {
		wc.Close()
		return err
	}
	return wc.Close()
}

// checkMailbox connects to the seed mailbox via IMAP and looks for the test email.
// Returns a PlacementResult with the placement and folder.
func (svc *PlacementService) checkMailbox(testID uint, mb *models.SeedMailbox, msgID string) models.PlacementResult {
	result := models.PlacementResult{
		TestID:        testID,
		SeedMailboxID: mb.ID,
		ISP:           mb.ISP,
		Email:         mb.Email,
		MessageID:     msgID,
		CheckedAt:     time.Now(),
		Placement:     "missing",
	}

	pwd, err := Decrypt(mb.Password)
	if err != nil {
		result.Placement = "error"
		result.ErrorMsg = "decrypt password: " + err.Error()
		return result
	}

	port := mb.IMAPPort
	if port == 0 {
		port = 993
	}

	addr := fmt.Sprintf("%s:%d", mb.IMAPHost, port)
	tlsCfg := &tls.Config{ServerName: mb.IMAPHost}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsCfg)
	if err != nil {
		result.Placement = "error"
		result.ErrorMsg = "IMAP connect: " + err.Error()
		return result
	}
	defer conn.Close()

	// Run simple IMAP session manually (no external library — pure Go)
	placement, folder, recvAt, iErr := imapSearch(conn, mb.Username, pwd, msgID)
	if iErr != nil {
		result.Placement = "error"
		result.ErrorMsg = iErr.Error()
		return result
	}
	result.Placement = placement
	result.InboxFolder = folder
	result.ReceivedAt = recvAt
	return result
}

// imapSearch performs a minimal IMAP LOGIN + SEARCH across common folders to find the email by Message-ID.
// Returns (placement, folder, receivedAt, error).
// placement: "inbox", "spam", or "missing"
func imapSearch(conn net.Conn, username, password, msgID string) (string, string, *time.Time, error) {
	buf := make([]byte, 65536)

	read := func() (string, error) {
		n, err := conn.Read(buf)
		if err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	}
	send := func(line string) error {
		_, err := fmt.Fprintf(conn, "%s\r\n", line)
		return err
	}

	// Read server greeting
	if _, err := read(); err != nil {
		return "", "", nil, fmt.Errorf("greeting: %w", err)
	}

	// LOGIN
	send("A001 LOGIN " + username + " " + password)
	resp, err := read()
	if err != nil || !strings.Contains(resp, "A001 OK") {
		return "", "", nil, fmt.Errorf("LOGIN failed")
	}

	// Folders to check: Inbox → spam
	inboxFolders := []string{"INBOX"}
	spamFolders := []string{"[Gmail]/Spam", "Junk", "Spam", "Bulk Mail", "Junk E-mail"}

	searchFolder := func(folder string) (bool, *time.Time) {
		tag := "B001"
		send(fmt.Sprintf("%s SELECT \"%s\"", tag, folder))
		resp, err := read()
		if err != nil || (!strings.Contains(resp, tag+" OK") && !strings.Contains(resp, "OK [READ")) {
			return false, nil
		}

		// SEARCH by Message-ID header
		cleanID := strings.Trim(msgID, "<>")
		send(fmt.Sprintf("C001 SEARCH HEADER Message-ID \"%s\"", cleanID))
		sresp, _ := read()
		if !strings.Contains(sresp, "* SEARCH") {
			return false, nil
		}
		// Parse sequence number
		parts := strings.Fields(sresp)
		// * SEARCH <nums>
		if len(parts) < 3 {
			return false, nil
		}
		// Got a hit
		now := time.Now()
		return true, &now
	}

	// Check inbox
	for _, f := range inboxFolders {
		if found, at := searchFolder(f); found {
			send("Z001 LOGOUT")
			return "inbox", f, at, nil
		}
	}
	// Check spam
	for _, f := range spamFolders {
		if found, at := searchFolder(f); found {
			send("Z001 LOGOUT")
			return "spam", f, at, nil
		}
	}

	send("Z001 LOGOUT")
	return "missing", "", nil, nil
}
