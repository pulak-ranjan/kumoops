# Changelog

All notable changes to KumoOps will be documented in this file.
Licensed under the [GNU AGPLv3 License](../LICENSE).

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

> Planned improvements and upcoming features will be listed here before each release.

---

## [0.2.0] - 2026-03-09

### Phase 2 — Deliverability Intelligence Engine

#### FBL + VERP + DSN Engine
- **VERP encode/decode** — HMAC-SHA256 variable envelope return path encoding for per-recipient bounce attribution (`internal/core/verp.go`)
- **DSN parser + bounce classifier** — RFC 3464 DSN parser with automatic bounce categorisation (hard / soft / spam / transient) (`internal/core/dsn.go`)
- **FBL parser + service** — RFC 5965 ARF/FBL parser, FBL complaint stats aggregation by ISP, auto-suppression on complaint (`internal/core/fbl.go`)
- **Inbound mail watcher** — Maildir watcher (60-second polling) that processes incoming DSN and FBL emails (`internal/core/inbound.go`)
- **FBL REST API** — 10 endpoints at `/api/fbl/*`: list records, stats per ISP, bulk delete, suppression trigger, manual parse (`internal/api/fbl.go`)
- **FBL Dashboard** — 4-tab React page: complaint records, ISP breakdown, VERP config, DSN bounce log (`web/src/pages/FBLPage.jsx`)
- **New models**: `FBLRecord`, `BounceClassification`, `VERPConfig`

#### ISP Intelligence
- **Google Postmaster Tools integration** — OAuth2 (pure stdlib RS256 JWT) → pulls domain reputation, IP reputation, spam rate, DKIM pass rate, SPF pass rate per domain (`internal/core/isp_intel.go`, `internal/core/google_jwt.go`)
- **Microsoft SNDS integration** — fetches IP reputation and complaint data from Microsoft Smart Network Data Services
- **Local metrics aggregation** — computes per-ISP send/bounce/complaint stats from local DB when external APIs are unavailable
- **ISP Intel REST API** — 9 endpoints at `/api/isp-intel/*` for snapshots, history, refresh trigger (`internal/api/isp_intel.go`)
- **ISP Intelligence page** — reputation scores, 7-day trend sparklines, per-ISP panel with tabbed data (`web/src/pages/ISPIntelPage.jsx`)
- **New model**: `ISPSnapshot`

#### Adaptive Throttling
- **Auto-tighten / auto-relax** — reads latest ISP reputation snapshot every 5 minutes; if domain or IP reputation degrades, reduces per-ISP send rate; if reputation recovers, gradually increases back (`internal/core/adaptive_throttle.go`)
- **Throttle audit log** — every adjustment (direction, ISP, old rate, new rate, reason) stored to `ThrottleAdjustmentLog`
- **Throttle REST API** — `/api/throttle/logs`, `/api/throttle/status` endpoints

#### Anomaly Detection & Self-Healing
- **Anomaly detector** — 5-minute cycle checks: bounce spike (>10% in 30min), complaint spike (>0.5%), queue buildup (>5000 msgs), delivery rate drop (>30% in 1h) (`internal/core/anomaly.go`)
- **Self-healer** — auto-applies fixes: pause sending campaign, tighten throttle rules, flush deferred queue
- **Alert integration** — anomaly events fire `WebhookService.SendAlert()` → Slack/Discord + Telegram
- **Anomaly REST API** — `/api/anomalies/` list, stats, resolve, suppress endpoints (`internal/api/isp_intel.go`)
- **Anomaly Dashboard** — severity rings, timeline, active vs resolved tabs, per-type breakdown (`web/src/pages/AnomalyPage.jsx`)
- **New models**: `ThrottleAdjustmentLog`, `AnomalyEvent`

---

### Phase 3 — Advanced Sending Features

#### Inbox Placement Testing
- **Seed mailbox management** — CRUD for IMAP seed mailboxes (Gmail, Outlook, Yahoo, etc.) with encrypted password storage
- **Pure-stdlib IMAP client** — no external dependency; connects to IMAP, searches for test emails by Message-ID, reports inbox / spam / missing placement
- **Placement REST API** — `/api/placement/*`: seed mailboxes CRUD, trigger test, list results, per-test detail
- **Inbox Placement page** — seed mailbox table, run-test modal, per-mailbox placement badges (inbox/spam/missing), historical test list (`web/src/pages/InboxPlacementPage.jsx`)

#### A/B Testing Engine
- **Variant management** — create up to 4 variants per campaign with custom subject / body / from-name and split percentage
- **Automated winner selection** — scheduler checks open rate and click rate every 5 minutes; auto-selects winner when statistically significant; sends remaining traffic to winner
- **A/B REST API** — `/api/campaigns/{id}/variants/` CRUD, winner-select endpoint, stats per variant
- **A/B Testing page** — per-campaign expandable panels, variant stat bars, trophy-highlight winner, manual override (`web/src/pages/ABTestPage.jsx`)
- **New model**: `CampaignVariant`

#### Send-Time Optimization
- **7×24 engagement heatmap** — aggregates `opened_at` and `clicked_at` timestamps from all delivery events; builds hour-of-week matrix
- **Recommendations engine** — ranks top-5 time slots by engagement rate; labels best slot per day
- **Send-Time REST API** — `/api/send-time/heatmap`, `/api/send-time/recommendations`
- **Send-Time page** — colour-intensity grid (7 days × 24 hours), top-5 recommendation cards with score badges (`web/src/pages/SendTimePage.jsx`)

---

### Phase 4 — Infrastructure & Integration

#### SMTP Relay Management
- **Relay hub config** — configure KumoOps to function as an authenticated SMTP relay for external apps (MailWizz, Mautic, custom code)
- **Allowed relay IPs** — whitelist IPs that can relay without credentials; block others with 550
- **Relay stats** — messages relayed, connection count, per-sender breakdown
- **Relay REST API** — `/api/relay/*`: status, settings, allowed IPs CRUD, connection log
- **SMTP Relay page** — live status card, settings form, allowed-IP table, how-to guide (`web/src/pages/RelayPage.jsx`)

#### HTTP Sending API (Mailgun-Compatible)
- **`POST /api/v1/messages`** — Mailgun-compatible sending endpoint; authenticated via `Authorization: kumo_xxx` header (API key with `send` scope)
- **Request format** — `to`, `from_email`, `from_name`, `subject`, `html`, `text`, `reply_to`, `cc`, `bcc`, custom headers
- **Response** — queue ID, status, estimated delivery
- **API key auth middleware** — validates `kumo_xxx` format keys, checks `send` scope, updates `last_used` timestamp

#### Multi-Node Cluster
- **Remote node registration** — store remote KumoOps URLs with API token (uses `cluster`-scoped API key generated on the remote node)
- **Health ping** — periodic health check per registered node; displays online / offline / degraded status
- **Aggregate metrics** — pulls delivery stats from all nodes and displays unified totals
- **Config push** — push current KumoMTA config to all or selected remote nodes with per-node result display
- **Cluster REST API** — `/api/cluster/*`: list nodes, add node, delete node, health check, metrics aggregate, config push
- **Cluster page** — node health table with latency, aggregate metric cards, config-push panel with results (`web/src/pages/ClusterPage.jsx`)

#### API Keys — Scoped Access
- **Scopes system** — five scopes: `send` (HTTP API), `relay` (SMTP relay auth), `verify` (email verification endpoints), `cluster` (remote node auth), `read` (stats/queue read-only)
- **Key generation** — cryptographically random 48-character hex with `kumo_` prefix
- **Last-used tracking** — every API call updates `last_used` timestamp on the key
- **Upgraded UI** — table layout with key prefix, scope badges, created date, last-used `ago()` display; three collapsible how-to panels (HTTP API, Multi-VPS Cluster, External Integrations)

---

### Phase 5 — AI Intelligence Layer

#### Multi-Provider AI Support
- **8 providers supported**: OpenAI (GPT-4o-mini), Anthropic Claude (Claude 3.5 Haiku), Google Gemini (Gemini 2.0 Flash), Groq (Llama 3.3 70B), Mistral (Mistral Small), Together AI (Llama 3.2 11B), DeepSeek (DeepSeek Chat), Ollama (any local model)
- **Anthropic Claude native format** — separate `system` field, `x-api-key` + `anthropic-version: 2023-06-01` headers, response parsed from `content[].text` (`sendToAnthropic` function)
- **OpenAI-compatible unified path** — all other 7 providers (including Ollama) share one HTTP client via config map `{url, model}`
- **Ollama support** — self-hosted on VPS, zero cost, zero API key; configure base URL (`http://localhost:11434`) and model name in Settings
- **Settings redesign** — visual 8-card provider grid with click-to-select; Ollama panel shows base URL + model + inline bash setup guide

#### Deliverability Advisor
- **Data aggregation** — pulls ISP snapshots, anomaly events, FBL complaint stats, bounce classifications, throttle adjustment logs, and 30-day email stats
- **AI analysis** — sends structured context to configured AI provider → receives `SCORE:|TREND:|ISSUES:|ANALYSIS:` structured response
- **Score ring** — SVG score ring (0–100) with colour coding (green/yellow/red)
- **Trend indicator** — improving / stable / declining with arrow icon
- **Issues panel** — ranked list with severity badges (critical/warning/info) and suggested actions
- **Full Markdown analysis** — rendered AI narrative visible below the summary panel
- **Endpoint**: `GET /api/ai/deliverability-advisor`

#### Content Analyzer
- **Input** — subject line, HTML body, sender domain
- **Spam scoring** — AI-assessed spam risk on 0–10 scale (lower = safer)
- **Deliverability scoring** — deliverability likelihood 0–100
- **Issues + suggestions** — bulleted list of specific problems and how to fix them
- **Endpoint**: `POST /api/ai/analyze-content`

#### Subject Line Generator
- **Inputs** — topic, target audience, tone, goal (open rate / click rate / conversions), count (1–10)
- **AI output** — N variants, each with: text, style tag (curiosity/urgency/benefit/social-proof/personal/question), emoji version, reasoning notes
- **Copy-to-clipboard** per variant
- **Endpoint**: `POST /api/ai/subject-lines`

#### Pre-Send Campaign Score (Local, No AI)
- **7 weighted checks** — subject line quality (10pts), body quality (10pts), sender reputation (15pts), recipient list quality (15pts), complaint rate (20pts), active anomalies (15pts), unsubscribe config (15pts)
- **Grade** — A (90+), B (80+), C (70+), D (60+), F (<60)
- **Blockers list** — explicit list of issues that must be fixed before sending
- **Endpoint**: `GET /api/campaigns/{id}/send-score`

#### AI Advisor Page
- 4-tab page: Deliverability Advisor, Content Analyzer, Subject Line Generator, Pre-Send Score
- Score ring SVG component, TrendIcon, MarkdownView renderer, SeverityBadge, StyleBadge components
- Campaign selector dropdown for Pre-Send Score tab

---

### Infrastructure & Bug Fixes

- **`KUMO_APP_SECRET` env var** — AES-256 GCM encryption for AI API keys and SMTP passwords stored in DB
- **`OllamaBaseURL` / `OllamaModel`** fields added to `AppSettings` model
- **Scheduler goroutines** — adaptive throttle and anomaly detector run every 5 minutes; ISP intel refresh runs every hour; A/B testing winner check runs every 5 minutes
- **Navigation updated** — Layout.jsx now has named nav groups: Sending, Deliverability, Analytics, AI Intelligence, Campaigns, Infrastructure, System
- **Router updated** — App.jsx registers 6 new page routes: `/fbl`, `/isp-intel`, `/anomalies`, `/inbox-placement`, `/send-time`, `/ab-testing`, `/relay`, `/cluster`, `/ai-advisor`

---

## [0.0.1] - 2026-03-03

### Initial Release

This is the first versioned release of KumoMTA UI — a full-featured control panel for KumoMTA built with React and Go.

#### Authentication & Security
- Admin registration with bcrypt password hashing
- JWT-based session management with per-session revocation
- Two-Factor Authentication (TOTP) — Google Authenticator / Authy compatible
- Security audit scanner (file permissions, exposed ports)
- Hourly blacklist monitoring against Spamhaus and Barracuda RBLs
- Audit logging for all Create/Update/Delete admin actions

#### Sending Infrastructure
- Domain management with CRUD operations
- Sender management per domain with DKIM auto-generation (RSA 2048-bit)
- IP inventory with auto-detection and CIDR bulk import
- IP Pool grouping and assignment
- IP Warmup scheduling and progress tracking
- Traffic Shaping rules per ISP (Gmail, Microsoft, Yahoo)
- Queue inspection, flush, and per-message deletion

#### Deliverability
- Envelope-From separation for per-sender bounce isolation
- Bounce account management (system user creation) with analytics
- Suppression list management — add, remove, import (CSV/TXT), bulk delete, export
- Alert rules engine with Slack/Discord webhook delivery
- Email Auth tools — BIMI, MTA-STS, SPF/DKIM/DMARC live checker

#### Config & Automation
- KumoMTA config generator (`init.lua`, `sources.toml`, `listener_domains.toml`, etc.)
- ISP Traffic Shaping with built-in rate limits for Gmail, Microsoft, Yahoo
- SMTP authentication with correct 3-parameter `smtp_server_auth_plain`
- DKIM signing with `List-Unsubscribe-Post` header support
- Header scrubbing (removes `User-Agent`, `X-Mailer`, fingerprinting headers)
- Webhook integration (Slack, Discord) with daily reports and bounce alerts

#### Bots
- Telegram bot (long-poll, no public URL needed) with 16 commands
- Discord bot (interactions endpoint) with native slash commands and confirm/cancel buttons

#### UI
- Responsive layout with collapsible sidebar and mobile hamburger menu
- Dark / Light mode with system preference synchronisation
- Card-based design with Lucide icon set
- Real-time terminal log viewer for KumoMTA, Dovecot, Fail2Ban
- AI Assistant floating chat panel for log analysis and operational insights

#### API
- Full REST API for all management operations
- Bearer token authentication on all protected routes
- CORS enforcement and localhost-only API binding

---

## Future Roadmap

### [0.3.0] — Planned
- OpenAPI / Swagger spec for all endpoints
- Multi-user roles (read-only, operator, admin)
- Slack bot integration
- Docker / docker-compose setup

### [1.0.0] — Planned
- Stable API with guaranteed backwards compatibility
- Full test suite with CI/CD pipeline
- Official Docker image
- Prometheus metrics endpoint (`/metrics`)
