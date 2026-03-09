package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// ─── Deliverability Advisor ───────────────────────────────────────────────────

type AdvisorIssue struct {
	Severity string `json:"severity"` // critical / warning / info
	Domain   string `json:"domain,omitempty"`
	ISP      string `json:"isp,omitempty"`
	Issue    string `json:"issue"`
	Action   string `json:"action"`
}

type AdvisorReport struct {
	Score       int            `json:"score"`
	Trend       string         `json:"trend"` // improving / stable / declining
	GeneratedAt time.Time      `json:"generated_at"`
	Issues      []AdvisorIssue `json:"issues"`
	Analysis    string         `json:"analysis"`
	DataSummary map[string]int `json:"data_summary"`
}

// GET /api/ai/deliverability-advisor
func (s *Server) handleDeliverabilityAdvisor(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI not configured. Add an API key in Settings."})
		return
	}
	aiKey, err := core.Decrypt(settings.AIAPIKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to decrypt AI key"})
		return
	}

	since7d := time.Now().Add(-7 * 24 * time.Hour)

	// Gather all deliverability signals
	anomalies, _ := s.Store.ListAnomalyEvents(since7d, 20)
	activeAnomalies, _ := s.Store.ListActiveAnomalyEvents()
	throttleLogs, _ := s.Store.ListThrottleAdjustmentLogs("", since7d, 15)
	fblStats, _ := s.Store.GetFBLStats(since7d)
	bounceSummary, _ := s.Store.GetBounceClassificationSummary(since7d)
	domains, _ := s.Store.ListDomains()
	emailStats, _ := s.Store.GetEmailStatsAll(7)

	// Build context string
	var sb strings.Builder
	sb.WriteString("=== DELIVERABILITY SIGNAL REPORT ===\n\n")

	// Domains
	sb.WriteString(fmt.Sprintf("DOMAINS CONFIGURED: %d\n", len(domains)))
	for _, d := range domains {
		sb.WriteString(fmt.Sprintf("  - %s\n", d.Name))
	}
	sb.WriteString("\n")

	// Email Stats (7d) — fields: Sent, Delivered, Bounced
	sb.WriteString("EMAIL STATS (LAST 7 DAYS):\n")
	totalSent, totalDelivered, totalBounced := int64(0), int64(0), int64(0)
	for _, es := range emailStats {
		totalSent += es.Sent
		totalDelivered += es.Delivered
		totalBounced += es.Bounced
	}
	sb.WriteString(fmt.Sprintf("  Total Sent: %d | Delivered: %d | Bounced: %d\n", totalSent, totalDelivered, totalBounced))
	if totalSent > 0 {
		sb.WriteString(fmt.Sprintf("  Bounce Rate: %.2f%%\n", float64(totalBounced)/float64(totalSent)*100))
	}
	sb.WriteString("\n")

	// Active Anomalies — fields: Type, Severity, Domain, Notes
	sb.WriteString(fmt.Sprintf("ACTIVE ANOMALIES: %d\n", len(activeAnomalies)))
	for _, a := range activeAnomalies {
		sb.WriteString(fmt.Sprintf("  [%s] %s — %s: %s\n", strings.ToUpper(a.Severity), a.Domain, a.Type, a.Notes))
	}
	if len(activeAnomalies) == 0 {
		sb.WriteString("  None\n")
	}
	sb.WriteString("\n")

	// All anomaly events (7d) — fields: DetectedAt, Domain, Type, AutoHealed, Notes
	sb.WriteString(fmt.Sprintf("ANOMALY EVENTS (7d): %d\n", len(anomalies)))
	for _, a := range anomalies {
		healed := "manual"
		if a.AutoHealed {
			healed = "auto-healed"
		}
		sb.WriteString(fmt.Sprintf("  %s | %s | %s (%s) | %s\n",
			a.DetectedAt.Format("01-02 15:04"), a.Domain, a.Type, healed, a.Notes))
	}
	sb.WriteString("\n")

	// FBL / Complaint stats — fields: Domain, TotalComplaints
	sb.WriteString("FBL COMPLAINT STATS (7d):\n")
	if len(fblStats) == 0 {
		sb.WriteString("  No complaints recorded.\n")
	}
	for _, f := range fblStats {
		sb.WriteString(fmt.Sprintf("  %s: %d complaints (abuse: %d, unsub: %d)\n",
			f.Domain, f.TotalComplaints, f.AbuseComplaints, f.UnsubComplaints))
	}
	sb.WriteString("\n")

	// Bounce classification — fields: Category, Count
	sb.WriteString("BOUNCE CLASSIFICATION SUMMARY (7d):\n")
	for _, b := range bounceSummary {
		sb.WriteString(fmt.Sprintf("  Category: %s | Count: %d\n", b.Category, b.Count))
	}
	if len(bounceSummary) == 0 {
		sb.WriteString("  No bounces recorded.\n")
	}
	sb.WriteString("\n")

	// Throttle adjustments — fields: ISP, RuleName, OldRate, NewRate, Direction, Reason
	sb.WriteString(fmt.Sprintf("THROTTLE ADJUSTMENTS (7d): %d\n", len(throttleLogs)))
	for _, t := range throttleLogs {
		sb.WriteString(fmt.Sprintf("  ISP: %s | Rule: %s | %s→%s (%s) | %s\n",
			t.ISP, t.RuleName, t.OldRate, t.NewRate, t.Direction, t.Reason))
	}
	sb.WriteString("\n")

	prompt := fmt.Sprintf(`You are an expert email deliverability engineer. Analyze the following MTA deliverability data and provide a comprehensive health report.

%s

Provide your analysis in this exact format:

SCORE: [0-100 integer]
TREND: [improving|stable|declining]

ISSUES:
[List each issue as: SEVERITY|DOMAIN|ISP|ISSUE DESCRIPTION|RECOMMENDED ACTION]
Use severity: critical, warning, or info
Use "all" for domain/ISP if not specific

ANALYSIS:
[Write a detailed markdown analysis with sections: ## Overall Health, ## Top Risks, ## What's Working, ## Priority Actions]

Be specific, data-driven, and actionable. Focus on what needs immediate attention.`, sb.String())

	messages := []ChatMessage{
		{Role: "system", Content: "You are an expert email deliverability engineer providing MTA health analysis."},
		{Role: "user", Content: prompt},
	}

	rawReply, err := s.sendToAI(settings, aiKey, messages)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "AI request failed: " + err.Error()})
		return
	}

	// Parse structured response
	report := parseAdvisorReport(rawReply)
	report.GeneratedAt = time.Now()
	report.DataSummary = map[string]int{
		"domains":           len(domains),
		"active_anomalies":  len(activeAnomalies),
		"anomaly_events_7d": len(anomalies),
		"fbl_sources":       len(fblStats),
		"bounce_categories": len(bounceSummary),
		"throttle_changes":  len(throttleLogs),
	}

	writeJSON(w, http.StatusOK, report)
}

func parseAdvisorReport(raw string) AdvisorReport {
	report := AdvisorReport{Score: 70, Trend: "stable"}

	lines := strings.Split(raw, "\n")
	var analysisLines []string
	inAnalysis := false
	inIssues := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "SCORE:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(trimmed, "SCORE:"))
			if v, err := strconv.Atoi(scoreStr); err == nil {
				report.Score = v
			}
			continue
		}
		if strings.HasPrefix(trimmed, "TREND:") {
			report.Trend = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "TREND:")))
			continue
		}
		if trimmed == "ISSUES:" {
			inIssues = true
			inAnalysis = false
			continue
		}
		if trimmed == "ANALYSIS:" {
			inAnalysis = true
			inIssues = false
			continue
		}
		if inIssues && strings.Contains(trimmed, "|") {
			parts := strings.SplitN(trimmed, "|", 5)
			if len(parts) >= 5 {
				report.Issues = append(report.Issues, AdvisorIssue{
					Severity: strings.ToLower(strings.TrimSpace(parts[0])),
					Domain:   strings.TrimSpace(parts[1]),
					ISP:      strings.TrimSpace(parts[2]),
					Issue:    strings.TrimSpace(parts[3]),
					Action:   strings.TrimSpace(parts[4]),
				})
			}
			continue
		}
		if inAnalysis {
			analysisLines = append(analysisLines, line)
		}
	}

	if len(analysisLines) > 0 {
		report.Analysis = strings.TrimSpace(strings.Join(analysisLines, "\n"))
	} else {
		report.Analysis = raw // fallback: return full response
	}

	return report
}

// ─── Content Analyzer ─────────────────────────────────────────────────────────

type ContentAnalysisRequest struct {
	Subject    string `json:"subject"`
	HTMLBody   string `json:"html_body"`
	SenderDomain string `json:"sender_domain"`
}

type ContentAnalysisResult struct {
	SpamScore          float64  `json:"spam_score"`          // 0–10 (lower = better)
	DeliverabilityScore int     `json:"deliverability_score"` // 0–100
	Issues             []string `json:"issues"`
	Suggestions        []string `json:"suggestions"`
	Analysis           string   `json:"analysis"`
}

// POST /api/ai/analyze-content
func (s *Server) handleAnalyzeContent(w http.ResponseWriter, r *http.Request) {
	var req ContentAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI not configured"})
		return
	}
	aiKey, _ := core.Decrypt(settings.AIAPIKey)

	// Truncate HTML to avoid token limits
	htmlPreview := req.HTMLBody
	if len(htmlPreview) > 3000 {
		htmlPreview = htmlPreview[:3000] + "\n... [truncated]"
	}

	prompt := fmt.Sprintf(`Analyze this email for spam triggers and deliverability issues.

SUBJECT: %s
SENDER DOMAIN: %s
HTML BODY (preview):
%s

Respond in this exact format:

SPAM_SCORE: [0.0-10.0]
DELIVERABILITY_SCORE: [0-100]

ISSUES:
- [specific issue 1]
- [specific issue 2]

SUGGESTIONS:
- [specific fix 1]
- [specific fix 2]

ANALYSIS:
[Detailed markdown analysis with sections: ## Subject Line, ## Content Quality, ## HTML Structure, ## Spam Triggers, ## Overall Assessment]`,
		req.Subject, req.SenderDomain, htmlPreview)

	messages := []ChatMessage{
		{Role: "system", Content: "You are an expert email deliverability analyst. Evaluate email content for spam triggers, inbox placement, and best practices."},
		{Role: "user", Content: prompt},
	}

	rawReply, err := s.sendToAI(settings, aiKey, messages)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	result := parseContentAnalysis(rawReply)
	writeJSON(w, http.StatusOK, result)
}

func parseContentAnalysis(raw string) ContentAnalysisResult {
	result := ContentAnalysisResult{SpamScore: 5.0, DeliverabilityScore: 70}
	lines := strings.Split(raw, "\n")
	var analysisLines []string
	inIssues, inSuggestions, inAnalysis := false, false, false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SPAM_SCORE:") {
			re := regexp.MustCompile(`[\d.]+`)
			if m := re.FindString(strings.TrimPrefix(trimmed, "SPAM_SCORE:")); m != "" {
				if v, err := strconv.ParseFloat(m, 64); err == nil {
					result.SpamScore = v
				}
			}
			inIssues, inSuggestions, inAnalysis = false, false, false
			continue
		}
		if strings.HasPrefix(trimmed, "DELIVERABILITY_SCORE:") {
			re := regexp.MustCompile(`\d+`)
			if m := re.FindString(strings.TrimPrefix(trimmed, "DELIVERABILITY_SCORE:")); m != "" {
				if v, err := strconv.Atoi(m); err == nil {
					result.DeliverabilityScore = v
				}
			}
			inIssues, inSuggestions, inAnalysis = false, false, false
			continue
		}
		if trimmed == "ISSUES:" {
			inIssues, inSuggestions, inAnalysis = true, false, false
			continue
		}
		if trimmed == "SUGGESTIONS:" {
			inIssues, inSuggestions, inAnalysis = false, true, false
			continue
		}
		if trimmed == "ANALYSIS:" {
			inIssues, inSuggestions, inAnalysis = false, false, true
			continue
		}
		if inIssues && strings.HasPrefix(trimmed, "- ") {
			result.Issues = append(result.Issues, strings.TrimPrefix(trimmed, "- "))
			continue
		}
		if inSuggestions && strings.HasPrefix(trimmed, "- ") {
			result.Suggestions = append(result.Suggestions, strings.TrimPrefix(trimmed, "- "))
			continue
		}
		if inAnalysis {
			analysisLines = append(analysisLines, line)
		}
	}

	if len(analysisLines) > 0 {
		result.Analysis = strings.TrimSpace(strings.Join(analysisLines, "\n"))
	} else {
		result.Analysis = raw
	}
	return result
}

// ─── Subject Line Generator ──────────────────────────────────────────────────

type SubjectLineRequest struct {
	Topic       string `json:"topic"`
	Audience    string `json:"audience"`
	Tone        string `json:"tone"`   // professional, friendly, urgent, playful
	Goal        string `json:"goal"`   // open_rate, click_rate, awareness
	Count       int    `json:"count"`  // number of variants (default 5)
}

type SubjectVariant struct {
	Text     string `json:"text"`
	Style    string `json:"style"`
	Notes    string `json:"notes"`
	EmojiVer string `json:"emoji_ver"`
}

type SubjectLineResponse struct {
	Variants []SubjectVariant `json:"variants"`
}

// POST /api/ai/subject-lines
func (s *Server) handleGenerateSubjectLines(w http.ResponseWriter, r *http.Request) {
	var req SubjectLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Count <= 0 || req.Count > 10 {
		req.Count = 5
	}
	if req.Tone == "" {
		req.Tone = "professional"
	}
	if req.Goal == "" {
		req.Goal = "open_rate"
	}

	settings, err := s.Store.GetSettings()
	if err != nil || settings == nil || settings.AIAPIKey == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "AI not configured"})
		return
	}
	aiKey, _ := core.Decrypt(settings.AIAPIKey)

	prompt := fmt.Sprintf(`Generate %d email subject line variants for:

Topic: %s
Audience: %s
Tone: %s
Optimization Goal: %s

For each variant, respond in this format:
VARIANT:
TEXT: [the subject line]
STYLE: [curiosity|urgency|benefit|social-proof|personal|question]
NOTES: [brief explanation of why this works]
EMOJI: [emoji version of the subject line]
---

Generate exactly %d variants. Be creative, specific, and avoid spam trigger words.`,
		req.Count, req.Topic, req.Audience, req.Tone, req.Goal, req.Count)

	messages := []ChatMessage{
		{Role: "system", Content: "You are an expert email copywriter specializing in high-open-rate subject lines. Never use spam trigger words."},
		{Role: "user", Content: prompt},
	}

	rawReply, err := s.sendToAI(settings, aiKey, messages)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := parseSubjectLines(rawReply)
	writeJSON(w, http.StatusOK, resp)
}

func parseSubjectLines(raw string) SubjectLineResponse {
	var result SubjectLineResponse
	blocks := strings.Split(raw, "---")
	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		var sv SubjectVariant
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "TEXT:") {
				sv.Text = strings.TrimSpace(strings.TrimPrefix(trimmed, "TEXT:"))
			} else if strings.HasPrefix(trimmed, "STYLE:") {
				sv.Style = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "STYLE:")))
			} else if strings.HasPrefix(trimmed, "NOTES:") {
				sv.Notes = strings.TrimSpace(strings.TrimPrefix(trimmed, "NOTES:"))
			} else if strings.HasPrefix(trimmed, "EMOJI:") {
				sv.EmojiVer = strings.TrimSpace(strings.TrimPrefix(trimmed, "EMOJI:"))
			}
		}
		if sv.Text != "" {
			result.Variants = append(result.Variants, sv)
		}
	}
	return result
}

// ─── Campaign Pre-Send Score ──────────────────────────────────────────────────

type SendScoreCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Points  int    `json:"points"`
	MaxPts  int    `json:"max_pts"`
	Details string `json:"details"`
}

type SendScoreResult struct {
	Score       int              `json:"score"`
	Grade       string           `json:"grade"`
	ReadyToSend bool             `json:"ready_to_send"`
	Checks      []SendScoreCheck `json:"checks"`
	Blockers    []string         `json:"blockers"`
}

// GET /api/campaigns/{id}/send-score
func (s *Server) handleCampaignSendScore(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	campaignID, _ := strconv.ParseUint(idStr, 10, 64)

	var campaign models.Campaign
	if err := s.Store.DB.First(&campaign, campaignID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "campaign not found"})
		return
	}

	since7d := time.Now().Add(-7 * 24 * time.Hour)
	var checks []SendScoreCheck
	var blockers []string
	totalScore := 0
	totalMax := 0

	// ── Check 1: Subject line (10 pts)
	{
		c := SendScoreCheck{Name: "Subject Line", MaxPts: 10}
		subj := strings.TrimSpace(campaign.Subject)
		if len(subj) == 0 {
			c.Details = "Missing subject line."
			blockers = append(blockers, "Campaign has no subject line")
		} else if len(subj) > 70 {
			c.Points = 5
			c.Passed = true
			c.Details = fmt.Sprintf("Subject is %d chars — recommended max is 70.", len(subj))
		} else {
			c.Points = 10
			c.Passed = true
			c.Details = fmt.Sprintf("Subject line looks good (%d chars).", len(subj))
		}
		checks = append(checks, c)
	}

	// ── Check 2: HTML body (10 pts)
	{
		c := SendScoreCheck{Name: "Email Body", MaxPts: 10}
		body := strings.TrimSpace(campaign.Body)
		if len(body) == 0 {
			c.Details = "Email body is empty."
			blockers = append(blockers, "Email body is empty")
		} else {
			c.Points = 10
			c.Passed = true
			c.Details = fmt.Sprintf("Body present (%d chars).", len(body))
		}
		checks = append(checks, c)
	}

	// ── Check 3: Sender configured (15 pts)
	{
		c := SendScoreCheck{Name: "Sender Address", MaxPts: 15}
		if campaign.SenderID == 0 {
			c.Details = "No sender assigned to this campaign."
			blockers = append(blockers, "No sender assigned")
		} else {
			sender, err := s.Store.GetSenderByID(campaign.SenderID)
			if err != nil {
				c.Details = "Sender not found."
			} else {
				c.Points = 15
				c.Passed = true
				c.Details = fmt.Sprintf("Sender: %s", sender.Email)
			}
		}
		checks = append(checks, c)
	}

	// ── Check 4: Has recipients (15 pts)
	{
		c := SendScoreCheck{Name: "Recipients", MaxPts: 15}
		var recipCount int64
		s.Store.DB.Model(&models.CampaignRecipient{}).Where("campaign_id = ?", campaignID).Count(&recipCount)
		if recipCount == 0 {
			c.Details = "No recipients imported."
			blockers = append(blockers, "No recipients imported")
		} else {
			c.Points = 15
			c.Passed = true
			c.Details = fmt.Sprintf("%d recipients ready.", recipCount)
		}
		checks = append(checks, c)
	}

	// ── Check 5: Domain health (FBL complaint rate) (20 pts)
	{
		c := SendScoreCheck{Name: "Domain Complaint Rate (7d)", MaxPts: 20}
		var domain string
		if campaign.SenderID > 0 {
			sender, _ := s.Store.GetSenderByID(campaign.SenderID)
			if sender != nil {
				d, _ := s.Store.GetDomainByID(sender.DomainID)
				if d != nil {
					domain = d.Name
				}
			}
		}
		if domain != "" {
			count, _ := s.Store.CountFBLRecordsSince(domain, since7d)
			stats, _ := s.Store.GetEmailStatsByDomain(domain, 7)
			var totalSent7d int64
			for _, st := range stats {
				totalSent7d += st.Sent
			}
			if totalSent7d > 100 {
				rate := float64(count) / float64(totalSent7d)
				if rate > 0.003 {
					c.Points = 0
					c.Details = fmt.Sprintf("Complaint rate %.3f%% exceeds 0.3%% threshold — HIGH RISK", rate*100)
					blockers = append(blockers, fmt.Sprintf("Complaint rate %.3f%% is dangerously high", rate*100))
				} else if rate > 0.001 {
					c.Points = 10
					c.Passed = true
					c.Details = fmt.Sprintf("Complaint rate %.3f%% — approaching threshold (>0.1%%)", rate*100)
				} else {
					c.Points = 20
					c.Passed = true
					c.Details = fmt.Sprintf("Complaint rate %.4f%% — healthy", rate*100)
				}
			} else {
				c.Points = 15
				c.Passed = true
				c.Details = "Insufficient data — proceed with caution."
			}
		} else {
			c.Points = 10
			c.Passed = true
			c.Details = "Could not determine domain — check sender assignment."
		}
		checks = append(checks, c)
	}

	// ── Check 6: Active anomalies (15 pts)
	{
		c := SendScoreCheck{Name: "Active Anomalies", MaxPts: 15}
		activeAnoms, _ := s.Store.ListActiveAnomalyEvents()
		criticalCount := 0
		for _, a := range activeAnoms {
			if a.Severity == "critical" {
				criticalCount++
			}
		}
		if criticalCount > 0 {
			c.Points = 0
			c.Details = fmt.Sprintf("%d critical anomaly/anomalies active — sending may be impacted.", criticalCount)
			blockers = append(blockers, fmt.Sprintf("%d critical anomaly events active", criticalCount))
		} else if len(activeAnoms) > 0 {
			c.Points = 8
			c.Passed = true
			c.Details = fmt.Sprintf("%d warning-level anomalies active — monitor closely.", len(activeAnoms))
		} else {
			c.Points = 15
			c.Passed = true
			c.Details = "No active anomalies detected."
		}
		checks = append(checks, c)
	}

	// ── Check 7: Unsubscribe mechanism (15 pts)
	{
		c := SendScoreCheck{Name: "Unsubscribe Mechanism", MaxPts: 15}
		settings, _ := s.Store.GetSettings()
		hostname := ""
		if settings != nil {
			hostname = settings.TrackingBaseURL
			if hostname == "" {
				hostname = settings.MainHostname
			}
		}
		if hostname != "" {
			c.Points = 15
			c.Passed = true
			c.Details = fmt.Sprintf("Tracking host configured: %s. List-Unsubscribe headers will be injected.", hostname)
		} else {
			c.Points = 5
			c.Passed = true
			c.Details = "No tracking host set in Settings — List-Unsubscribe headers may be missing."
		}
		checks = append(checks, c)
	}

	// Compute totals
	for _, c := range checks {
		totalScore += c.Points
		totalMax += c.MaxPts
	}
	finalScore := 0
	if totalMax > 0 {
		finalScore = (totalScore * 100) / totalMax
	}

	grade := "F"
	switch {
	case finalScore >= 90:
		grade = "A"
	case finalScore >= 80:
		grade = "B"
	case finalScore >= 70:
		grade = "C"
	case finalScore >= 60:
		grade = "D"
	}

	writeJSON(w, http.StatusOK, SendScoreResult{
		Score:       finalScore,
		Grade:       grade,
		ReadyToSend: len(blockers) == 0 && finalScore >= 70,
		Checks:      checks,
		Blockers:    blockers,
	})
}
