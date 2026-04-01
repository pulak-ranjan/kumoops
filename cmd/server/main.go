package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pulak-ranjan/kumoops/internal/api"
	"github.com/pulak-ranjan/kumoops/internal/core"
	"github.com/pulak-ranjan/kumoops/internal/store"
)

func main() {
	// Check security configuration first
	if _, err := core.GetEncryptionKey(); err != nil {
		log.Fatalf("Security configuration error: %v. Please set KUMO_APP_SECRET environment variable.", err)
	}

	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		dbDir = "/var/lib/kumoops"
	}
	dbPath := dbDir + "/panel.db"

	// Ensure DB directory exists
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Printf("Warning: failed to create db directory: %v", err)
	}

	// Initialize Store
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}

	// Initialize Core Services
	ws := core.NewWebhookService(st)
	srv := api.NewServer(st, ws)

	// Start Telegram bot polling (runs forever, reconnects on failure)
	tgBot := core.NewTelegramBot(st)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Telegram] Bot polling crashed: %v — restarting in 10s", r)
				time.Sleep(10 * time.Second)
				go tgBot.StartPolling()
			}
		}()
		tgBot.StartPolling()
	}()

	// Start Inbound Mail Processor (FBL + DSN from Maildir)
	inbound := core.NewInboundProcessor(st)
	inbound.Start(60) // Scan every 60 seconds

	// Start ISP Intelligence engine (refresh every hour)
	ispIntel := core.NewISPIntelService(st)

	// Start Adaptive Throttler + Anomaly Detector + A/B Test winner checker (5 min via scheduler)
	adaptiveThrottle := core.NewAdaptiveThrottler(st)
	anomalyDetector := core.NewAnomalyDetector(st, ws)
	abTestSvc := core.NewABTestService(st)

	// Start Background Scheduler
	go startScheduler(ws, tgBot, ispIntel, adaptiveThrottle, anomalyDetector, abTestSvc)

	// Start HTTP Server
	addr := "127.0.0.1:9000"
	log.Printf("Kumo UI backend listening on %s\n", addr)
	if err := http.ListenAndServe(addr, srv.Router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func startScheduler(
	ws *core.WebhookService,
	tgBot *core.TelegramBot,
	ispIntel *core.ISPIntelService,
	adaptiveThrottle *core.AdaptiveThrottler,
	anomalyDetector *core.AnomalyDetector,
	abTestSvc *core.ABTestService,
) {
	log.Println("Starting background scheduler...")

	// Run startup checks immediately (non-blocking)
	go func() {
		log.Println("[Scheduler] Running initial startup checks...")

		// 1. Warmup (catch up on missing cycles)
		if err := core.ProcessDailyWarmup(ws.Store); err != nil {
			log.Printf("Warmup startup check error: %v", err)
		}

		// 2. Campaigns (Resume interrupted jobs)
		cs := core.NewCampaignService(ws.Store)
		if err := cs.ResumeInterruptedCampaigns(); err != nil {
			log.Printf("Campaign resumption error: %v", err)
		}

		// 3. Security & Compliance
		ws.RunSecurityAudit()
		ws.CheckBlacklists(false) // Silent unless issues found

		// 4. Stats & Alerts
		ws.CheckBounceRates()

		// 5. Backup (if missing/stale)
		if err := core.EnsureRecentBackup(); err != nil {
			log.Printf("Backup startup check error: %v", err)
		}

		// 6. Initial ISP intelligence snapshot
		go ispIntel.RefreshAll()
	}()

	dailyTicker := time.NewTicker(24 * time.Hour)
	hourlyTicker := time.NewTicker(1 * time.Hour)
	warmupTicker := time.NewTicker(5 * time.Minute) // Check every 5 mins

	// Initialize campaign service for scheduler usage
	cs := core.NewCampaignService(ws.Store)

	for {
		select {
		case <-warmupTicker.C:
			// 1. Warmup Progression
			if err := core.ProcessDailyWarmup(ws.Store); err != nil {
				log.Printf("Warmup error: %v", err)
			}

			// 2. Check Scheduled Campaigns (every 5 mins)
			if err := cs.StartScheduledCampaigns(); err != nil {
				log.Printf("Scheduled campaign error: %v", err)
			}

			// 3. Adaptive Throttle cycle (adjusts per-ISP shaping rules)
			go adaptiveThrottle.Run()

			// 4. Anomaly Detection + Self-Healing cycle
			go anomalyDetector.Run()

			// 5. A/B Test winner auto-selection
			go abTestSvc.Run()

		case <-dailyTicker.C:
			log.Println("[Scheduler] Running daily tasks...")

			// 1. Daily Summary (Slack/Discord webhook + Telegram + Discord)
			if stats, err := core.GetAllDomainsStats(1); err == nil {
				ws.SendDailySummary(stats)
				tgBot.SendDigest(stats)
				tgBot.SendDiscordDigest(stats)
			}

			// 2. Security Audit
			ws.RunSecurityAudit()

			// 3. Auto Backup
			if err := core.BackupConfig(); err != nil {
				log.Printf("Backup failed: %v", err)
			} else {
				log.Println("Configuration backed up.")
			}

		case <-hourlyTicker.C:
			log.Println("[Scheduler] Running hourly tasks...")
			ws.CheckBlacklists(false) // Silent check
			ws.CheckBounceRates()

			// Telegram proactive push alerts
			tgBot.CheckAndAlertBounceSpike()
			tgBot.CheckAndAlertQueueBackpressure(5000)

			// ISP Intelligence refresh (hourly)
			go ispIntel.RefreshAll()
		}
	}
}
