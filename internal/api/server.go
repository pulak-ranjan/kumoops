package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/middleware/custom"
	"github.com/pulak-ranjan/kumoops/internal/models"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

type Server struct {
	Store      *store.Store
	WS         *core.WebhookService
	AC         *core.AlertChecker
	DiscordBot *core.DiscordBot
	Router     chi.Router
}

const adminContextKey contextKey = "admin"

type contextKey string

func NewServer(st *store.Store, ws *core.WebhookService) *Server {
	s := &Server{
		Store:      st,
		WS:         ws,
		DiscordBot: core.NewDiscordBot(st),
	}
	s.AC = core.NewAlertChecker(st)
	s.AC.Start()
	s.Router = s.routes()
	return s
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(custom.GeneralLimiter.Limit)

	// Dynamic CORS for Credentials support
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			// In development, allow localhost
			if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
				return true
			}

			settings, err := s.Store.GetSettings()
			if err != nil {
				return false
			}

			// If no origins configured, default to denying external access (safe default)
			if settings.AllowedOrigins == "" {
				return false
			}

			allowed := strings.Split(settings.AllowedOrigins, ",")
			for _, a := range allowed {
				if strings.TrimSpace(a) == origin {
					return true
				}
			}
			return false
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Temp-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Public Routes ---
	r.With(custom.AuthLimiter.Limit).Post("/api/auth/register", s.handleRegister)
	r.With(custom.AuthLimiter.Limit).Post("/api/auth/login", s.handleLogin)
	r.With(custom.AuthLimiter.Limit).Post("/api/auth/verify-2fa", s.handleVerify2FA)
	// Discord interactions endpoint — auth is done via Ed25519 signature, not session token
	r.Post("/api/discord/interactions", s.handleDiscordInteractions)

	// --- Protected Routes ---
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Auth & Profile
		r.Get("/api/auth/me", s.handleMe)
		r.Post("/api/auth/logout", s.handleLogout)
		r.Post("/api/auth/setup-2fa", s.handleSetup2FA)
		r.Post("/api/auth/enable-2fa", s.handleEnable2FA)
		r.Post("/api/auth/disable-2fa", s.handleDisable2FA)
		r.Post("/api/auth/theme", s.handleSetTheme)
		r.Get("/api/auth/sessions", s.handleListSessions)

		// Dashboard
		r.Get("/api/dashboard/stats", s.handleGetDashboardStats)

		// Settings
		r.Get("/api/settings", s.handleGetSettings)
		r.Post("/api/settings", s.handleSetSettings)

		// Domains
		r.Get("/api/domains", s.handleListDomains)
		r.Post("/api/domains", s.handleCreateDomain)
		r.Get("/api/domains/{id}", s.handleGetDomain)
		r.Put("/api/domains/{id}", s.handleUpdateDomain)
		r.Delete("/api/domains/{id}", s.handleDeleteDomain)

		// Senders
		r.Get("/api/domains/{domainID}/senders", s.handleListSenders)
		r.Post("/api/domains/{domainID}/senders", s.handleCreateSender)
		r.Get("/api/senders/{id}", s.handleGetSender)
		r.Put("/api/senders/{id}", s.handleUpdateSender)
		r.Delete("/api/senders/{id}", s.handleDeleteSender)
		r.Post("/api/domains/{domainID}/senders/{id}/setup", s.handleSetupSender)

		// Bounce Accounts
		r.Get("/api/bounces", s.handleListBounceAccounts)
		r.Post("/api/bounces", s.handleSaveBounceAccount)
		r.Delete("/api/bounces/{bounceID}", s.handleDeleteBounceAccount)
		r.Post("/api/bounces/apply", s.handleApplyBounceAccounts)
		r.Post("/api/bounces/create-required", s.handleCreateRequiredInboxes)
		r.Get("/api/bounces/{id}/messages", s.handleListMailboxMessages)
		r.Get("/api/bounces/{id}/messages/{msgid}", s.handleGetMailboxMessage)
		r.Get("/api/bounce-analytics", s.handleBounceAnalytics)

		// System IPs
		r.Get("/api/system/ips", s.handleListIPs)
		r.Post("/api/system/ips", s.handleAddIP)
		r.Post("/api/system/ips/configure", s.handleConfigureIP) // <--- NEW ROUTE
		r.Delete("/api/system/ips/{id}", s.handleDeleteIP)
		r.Post("/api/system/ips/bulk", s.handleBulkAddIPs)
		r.Post("/api/system/ips/cidr", s.handleAddIPsByCIDR)
		r.Post("/api/system/ips/detect", s.handleDetectIPs)

		// DKIM
		r.Get("/api/dkim/records", s.handleListDKIM)
		r.Post("/api/dkim/generate", s.handleGenerateDKIM)

		// DMARC & DNS
		r.Get("/api/dmarc/{domainID}", s.handleGetDMARC)
		r.Post("/api/dmarc/{domainID}", s.handleSetDMARC)
		r.Get("/api/dns/{domainID}", s.handleGetAllDNS)

		// Stats
		r.Get("/api/stats/domains", s.handleGetDomainStats)
		r.Get("/api/stats/domains/{domain}", s.handleGetSingleDomainStats)
		r.Get("/api/stats/summary", s.handleGetStatsSummary)
		r.Post("/api/stats/refresh", s.handleRefreshStats)
		r.Get("/api/stats/providers", s.handleGetProviderStats)
		r.Get("/api/stats/hourly", s.handleGetHourlyStats)

		// Delivery Log (per-recipient failure tracking)
		r.Get("/api/delivery-log", s.handleListDeliveryLog)
		r.Get("/api/delivery-log/summary", s.handleDeliveryLogSummary)
		r.Post("/api/delivery-log/refresh", s.handleRefreshDeliveryLog)

		// Queue
		r.Get("/api/queue", s.handleGetQueue)
		r.Get("/api/queue/stats", s.handleGetQueueStats)
		r.Delete("/api/queue/{id}", s.handleDeleteQueueMessage)
		r.Post("/api/queue/flush", s.handleFlushQueue)

		// Webhooks
		r.Get("/api/webhooks/settings", s.handleGetWebhookSettings)
		r.Post("/api/webhooks/settings", s.handleSetWebhookSettings)
		r.Post("/api/webhooks/test", s.handleTestWebhook)
		r.Get("/api/webhooks/logs", s.handleGetWebhookLogs)
		r.Post("/api/webhooks/check-bounces", s.handleCheckBounces)

		// Multi-VPS Remote Servers
		r.Get("/api/servers", s.handleListRemoteServers)
		r.Post("/api/servers", s.handleCreateRemoteServer)
		r.Delete("/api/servers/{id}", s.handleDeleteRemoteServer)
		r.Post("/api/servers/{id}/test", s.handleTestRemoteServer)
		r.Get("/api/servers/{id}/proxy", s.handleProxyRemoteServer)
		r.Post("/api/servers/{id}/proxy", s.handleProxyRemoteServer)
		r.Put("/api/servers/{id}/proxy", s.handleProxyRemoteServer)
		r.Delete("/api/servers/{id}/proxy", s.handleProxyRemoteServer)

		// Reputation Monitor
		r.Get("/api/reputation", s.handleGetReputation)
		r.Post("/api/reputation/check", s.handleRunReputationCheck)
		r.Get("/api/reputation/status", s.handleReputationStatus)
		r.Get("/api/reputation/delist-urls", s.handleDelistURLs)

		// System Tools & Actions (Guardian)
		r.Post("/api/system/check-blacklist", s.handleCheckBlacklist)
		r.Post("/api/system/check-security", s.handleCheckSecurity)
		r.Post("/api/system/action/block-ip", s.handleBlockIP)
		r.Post("/api/tools/send-test", s.handleSendTestEmail)
		r.Post("/api/system/run-command", s.handleRunCommand)

		// AI Chat & Analysis
		r.Post("/api/system/ai-analyze", s.handleAIAnalyze)
		r.Get("/api/ai/history", s.handleGetChatHistory)
		r.Post("/api/ai/chat", s.handleAIChat)

		// Warmup Routes
		r.Get("/api/warmup", s.handleGetWarmupList)
		r.Post("/api/warmup/{id}", s.handleUpdateWarmup)
		r.Post("/api/warmup/{id}/pause", s.handlePauseWarmup)
		r.Post("/api/warmup/{id}/resume", s.handleResumeWarmup)
		r.Get("/api/warmup/{id}/calendar", s.handleWarmupCalendar)
		r.Get("/api/warmup/{id}/logs", s.handleWarmupLogs)

		// API Keys Routes
		r.Get("/api/keys", s.handleListKeys)
		r.Post("/api/keys", s.handleCreateKey)
		r.Delete("/api/keys/{id}", s.handleDeleteKey)

		// Traffic Shaping Rules
		r.Get("/api/shaping", s.handleListShaping)
		r.Post("/api/shaping", s.handleCreateShaping)
		r.Put("/api/shaping/{id}", s.handleUpdateShaping)
		r.Delete("/api/shaping/{id}", s.handleDeleteShaping)
		r.Post("/api/shaping/seed", s.handleSeedShaping)

		// IP Pools
		r.Get("/api/ippools", s.handleListIPPools)
		r.Post("/api/ippools", s.handleCreateIPPool)
		r.Put("/api/ippools/{id}", s.handleUpdateIPPool)
		r.Delete("/api/ippools/{id}", s.handleDeleteIPPool)
		r.Post("/api/ippools/{id}/members", s.handleAddIPToPool)
		r.Delete("/api/ippools/{id}/members/{mid}", s.handleRemoveIPFromPool)

		// Suppression List
		r.Get("/api/suppression", s.handleListSuppression)
		r.Post("/api/suppression", s.handleAddSuppression)
		r.Delete("/api/suppression/{id}", s.handleRemoveSuppression)
		r.Post("/api/suppression/bulk", s.handleBulkSuppression)
		r.Get("/api/suppression/export", s.handleExportSuppression)
		r.Post("/api/suppression/import", s.handleImportSuppression)
		r.Get("/api/suppression/check", s.handleCheckSuppressed)

		// Alert Rules
		r.Get("/api/alerts/rules", s.handleListAlertRules)
		r.Post("/api/alerts/rules", s.handleCreateAlertRule)
		r.Put("/api/alerts/rules/{id}", s.handleUpdateAlertRule)
		r.Delete("/api/alerts/rules/{id}", s.handleDeleteAlertRule)
		r.Get("/api/alerts/events", s.handleListAlertEvents)
		r.Post("/api/alerts/test/{id}", s.handleTestAlert)

		// FBL / Feedback Loop Engine
		r.Get("/api/fbl", s.handleListFBLRecords)
		r.Get("/api/fbl/stats", s.handleGetFBLStats)
		r.Delete("/api/fbl/{id}", s.handleDeleteFBLRecord)
		r.Post("/api/fbl/upload", s.handleUploadFBL)
		r.Post("/api/fbl/upload-dsn", s.handleUploadDSN)
		// DSN Bounce Classifications
		r.Get("/api/fbl/bounces", s.handleListBounceClassifications)
		r.Get("/api/fbl/bounces/summary", s.handleGetBounceClassSummary)
		// VERP Configuration
		r.Get("/api/fbl/verp", s.handleListVERPConfigs)
		r.Get("/api/fbl/verp/{domainID}", s.handleGetVERPConfig)
		r.Post("/api/fbl/verp/{domainID}", s.handleSetVERPConfig)

		// ISP Intelligence
		r.Get("/api/isp-intel/snapshots", s.handleListISPSnapshots)
		r.Get("/api/isp-intel/snapshots/latest", s.handleGetLatestISPSnapshots)
		r.Post("/api/isp-intel/refresh", s.handleRefreshISPIntel)
		r.Get("/api/isp-intel/metrics", s.handleGetISPMetrics)

		// Adaptive Throttle
		r.Get("/api/throttle/logs", s.handleListThrottleLogs)
		r.Post("/api/throttle/run", s.handleRunAdaptiveThrottle)

		// Anomaly Detection
		r.Get("/api/anomalies", s.handleListAnomalyEvents)
		r.Get("/api/anomalies/active", s.handleListActiveAnomalies)
		r.Post("/api/anomalies/{id}/resolve", s.handleResolveAnomaly)

		// Inbox Placement Testing
		r.Get("/api/placement/mailboxes", s.handleListSeedMailboxes)
		r.Post("/api/placement/mailboxes", s.handleCreateSeedMailbox)
		r.Delete("/api/placement/mailboxes/{id}", s.handleDeleteSeedMailbox)
		r.Get("/api/placement/tests", s.handleListPlacementTests)
		r.Post("/api/placement/tests", s.handleCreatePlacementTest)
		r.Get("/api/placement/tests/{id}", s.handleGetPlacementTest)

		// A/B Testing
		r.Get("/api/campaigns/{id}/variants", s.handleListVariants)
		r.Post("/api/campaigns/{id}/variants", s.handleCreateVariant)
		r.Delete("/api/campaigns/{id}/variants/{vid}", s.handleDeleteVariant)
		r.Post("/api/campaigns/{id}/variants/{vid}/set-winner", s.handleSetABWinner)
		r.Get("/api/campaigns/{id}/ab-summary", s.handleGetABSummary)

		// Send-Time Optimization
		r.Get("/api/analytics/send-time", s.handleSendTimeHeatmap)

		// SMTP Relay Management
		r.Get("/api/relay/status", s.handleGetRelayStatus)
		r.Put("/api/relay/settings", s.handleUpdateRelaySettings)
		r.Post("/api/relay/apply", s.handleApplyRelayConfig)

		// Cluster / Multi-Node
		r.Get("/api/cluster/nodes", s.handleListClusterNodes)
		r.Post("/api/cluster/push-config", s.handleClusterPushConfig)
		r.Get("/api/cluster/metrics", s.handleClusterMetrics)

		// AI Intelligence Layer (Phase 5)
		r.Get("/api/ai/deliverability-advisor", s.handleDeliverabilityAdvisor)
		r.Post("/api/ai/analyze-content", s.handleAnalyzeContent)
		r.Post("/api/ai/subject-lines", s.handleGenerateSubjectLines)
		r.Get("/api/campaigns/{id}/send-score", s.handleCampaignSendScore)

		// Email Auth Tools (BIMI, MTA-STS, TLS-RPT)
		r.Get("/api/authtools/bimi/{domain}", s.handleGetBIMI)
		r.Post("/api/authtools/bimi/{domain}", s.handleSetBIMI)
		r.Get("/api/authtools/mtasts/{domain}", s.handleGetMTASTS)
		r.Post("/api/authtools/mtasts/{domain}", s.handleSetMTASTS)
		r.Get("/api/authtools/check/{domain}", s.handleCheckAuthTools)

		// Queue Intelligence (extended)
		r.Get("/api/queue/providers", s.handleQueueByProvider)
		r.Get("/api/queue/stuck", s.handleStuckMessages)

		// Log Intelligence
		r.Get("/api/logs/search", s.handleLogSearch)
		r.Get("/api/logs/patterns", s.handleLogPatterns)

		// Config
		r.Get("/api/config/preview", s.handlePreviewConfig)
		r.Post("/api/config/apply", s.handleApplyConfig)

		// Logs
		r.Get("/api/logs/kumomta", s.handleLogsKumo)
		r.Get("/api/logs/dovecot", s.handleLogsDovecot)
		r.Get("/api/logs/fail2ban", s.handleLogsFail2ban)
		r.Get("/api/logs/stream", s.handleLiveLogStream) // SSE live stream

		// Telegram / Discord notifications
		r.Post("/api/notify/test-telegram", s.handleTestTelegram)
		r.Post("/api/notify/test-discord", s.handleTestDiscord)
		r.Post("/api/discord/register-commands", s.handleDiscordRegisterCommands)

		// System Health
		r.Get("/api/system/health", s.handleSystemHealth)
		r.Get("/api/system/services", s.handleSystemServices)
		r.Get("/api/system/ports", s.handleSystemPorts)

		// Bulk Import
		r.Post("/api/import/csv", s.handleCSVImport)

		// Campaigns
		r.Route("/api/campaigns", NewCampaignHandler(s.Store).Routes)

		// Tracking (Public, no auth)
		// Note: TrackingHandler methods need to be wrapped or unprotected.
		// Since this group is protected, we must move tracking OUTSIDE or use skip logic.
	})

	// --- Tracking Routes (Unprotected) ---
	tracking := NewTrackingHandler(s.Store)
	r.Get("/api/track/open/{id}", tracking.HandleTrackOpen)
	r.Get("/api/track/click/{id}", tracking.HandleTrackClick)

	// --- Unsubscribe Routes (Unprotected, public-facing) ---
	r.Get("/unsubscribe/{token}", s.handleUnsubscribePage)
	r.Post("/unsubscribe/{token}", s.handleUnsubscribePost)

	// --- HTTP Sending API (API-key authenticated, separate auth) ---
	r.Post("/api/v1/messages", s.handleAPISendMessage)

	// --- Analytics (Protected) ---
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		analytics := NewAnalyticsHandler(s.Store)
		r.Get("/api/analytics/top-leads", analytics.GetTopLeads)
		r.Get("/api/analytics/campaign-summary", analytics.GetCampaignSummary)

		contacts := NewContactHandler(s.Store)
		r.With(custom.VerifyLimiter.Limit).Post("/api/contacts/verify", contacts.HandleVerifyEmail)
		r.Post("/api/lists/{id}/clean", contacts.HandleCleanList)

		// Automation & WhatsApp
		wa := NewWhatsAppHandler(s.Store)
		r.Post("/api/whatsapp/send", wa.HandleSend)
		r.Post("/api/whatsapp/webhook", wa.HandleWebhook)
	})

	return r
}

// ... (Rest of file same as before) ...
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SSE streams (EventSource) cannot set headers — accept token via ?token= query param
		token := r.URL.Query().Get("token")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
				return
			}
			token = strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
				return
			}
		}

		admin, err := s.Store.GetAdminBySessionToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), adminContextKey, admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getAdminFromContext(ctx context.Context) *models.AdminUser {
	if u, ok := ctx.Value(adminContextKey).(*models.AdminUser); ok {
		return u
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	s.Store.DeleteSession(token)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}
