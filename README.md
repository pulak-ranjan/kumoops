# KumoOps

![Version](https://img.shields.io/badge/version-0.2.0-blue.svg)
![License](https://img.shields.io/badge/license-AGPLv3-green.svg)
![Status](https://img.shields.io/badge/status-active-purple.svg)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)
![React](https://img.shields.io/badge/React-18-61DAFB.svg)

**KumoOps** is a full-featured operations platform for [KumoMTA](https://kumomta.com). It combines a polished web dashboard, an AI-powered log analyser, multi-provider AI intelligence, and full two-way bots for **Telegram** and **Discord** into a single self-hosted binary — no cloud dependency, runs entirely on your VPS.

> Formerly "KumoMTA UI" — grown from a simple control panel into a complete MTA operations suite.

---

## What's Inside

| Layer | Stack |
|---|---|
| Backend | Go 1.24+, chi router, GORM, SQLite |
| Frontend | React 18, Vite 5, Tailwind CSS 3 |
| Bots | Telegram (long-poll) + Discord (interactions endpoint) |
| AI | 8 providers: OpenAI, Anthropic Claude, Gemini, Groq, Mistral, Together AI, DeepSeek, Ollama (local) |

---

## Features

### Dashboard & Monitoring
- **Dashboard** — live CPU, RAM, domain/sender count, service health (KumoMTA / Dovecot / Fail2Ban), open ports
- **Statistics** — per-domain and per-sender delivery charts (sent, delivered, bounced, deferred, rate) with time-range selector
- **Delivery Log** — full event log with domain/type/date filters and CSV export
- **Bounce Analytics** — breakdown by ISP, bounce category, and trend over time
- **AI Log Analysis** — floating AI assistant chat: spots errors, explains log entries, suggests fixes, executes safe tools (`status`, `queue`, `logs_kumo`, `block_ip`, `dig`, etc.)

### Email Infrastructure Management
- **Domains** — add sending domains, verify SPF / DKIM / DMARC DNS records
- **DKIM** — generate, rotate, and publish DKIM keys per selector
- **DMARC** — policy builder and aggregate report viewer
- **Auth Tools** — end-to-end email authentication tester (BIMI, MTA-STS, SPF/DKIM/DMARC live checker)
- **IP Inventory** — manage dedicated IPs with pool assignment
- **IP Pools** — group IPs by purpose (transactional, bulk, warmup)
- **IP Warmup** — automated warmup schedules with daily volume ramps and progress tracking
- **Traffic Shaping** — per-domain/per-IP throttling, connection limits, retry policies
- **Reputation** — DNSBL blacklist checks (Spamhaus, SORBS, Barracuda, etc.) with alerting on new listings
- **Config Generator** — GUI-based KumoMTA `.lua` config builder

### Deliverability Intelligence (Phase 2)
- **FBL + Complaint Management** — RFC 5965 ARF/FBL parser; complaint stats, ISP breakdown, suppression auto-apply
- **DSN / Bounce Classification** — RFC 3464 DSN parser; bounce categorisation (hard/soft/spam) with per-ISP analytics
- **VERP Engine** — HMAC-SHA256 VERP encode/decode for per-recipient bounce isolation
- **ISP Intelligence** — Google Postmaster Tools integration (domain reputation, IP reputation, spam rate, DKIM rate, FBL rate); Microsoft SNDS integration; 7-day trend sparklines per ISP
- **Adaptive Throttling** — auto-tightens per-ISP sending rates when reputation drops; auto-relaxes when reputation recovers; every 5-minute cycle with full audit log
- **Anomaly Detection & Self-Healing** — detects bounce spikes, complaint spikes, queue buildup, delivery rate drops; applies auto-fixes (pause campaign, tighten throttle); fires alerts

### Campaigns
- Create campaigns and send to contact lists
- Real-time send progress, pause / resume mid-flight
- Per-campaign delivery stats
- Contact list import (CSV / JSON)
- **A/B Testing** — define subject/body variants with split percentages per campaign; automated winner selection (by open rate or click rate) every 5 minutes; manual winner selection with one click
- **Send-Time Optimization** — 7×24 engagement heatmap from opened/clicked timestamps; top-5 time-slot recommendations per campaign type

### Advanced Sending (Phase 3/4)
- **Inbox Placement Testing** — manage seed mailboxes (IMAP); send test emails and check inbox / spam / missing placement across providers; per-test history with per-mailbox results
- **SMTP Relay Management** — configure KumoOps as a relay hub; manage allowed relay IPs, connection limits, rate limits; view active relay connections
- **HTTP Sending API** — Mailgun-compatible `POST /api/v1/messages` endpoint; use any API key with `send` scope; returns queue ID and status; dead simple integration for MailWizz, Mautic, Python, etc.
- **Multi-Node Cluster** — register remote KumoOps nodes; health check per node; aggregate metrics dashboard; push config to all nodes with one click

### AI Intelligence Layer (Phase 5)
- **Deliverability Advisor** — aggregates ISP reputation, anomaly events, FBL complaints, bounce classifications, and throttle adjustments → sends to AI → returns structured score (0–100), trend, ranked issues, and detailed analysis in Markdown
- **Content Analyzer** — paste HTML + subject line → AI scores spam risk (0–10) and deliverability (0–100); lists specific issues and actionable suggestions
- **Subject Line Generator** — provide topic, audience, tone, and goal → AI generates N variants with style tags (curiosity / urgency / benefit / social-proof / personal / question), emoji version, and reasoning notes
- **Pre-Send Campaign Score** — local computed check (no AI cost): validates subject length, body/HTML ratio, sender reputation, recipient list quality, active anomalies, unsubscribe config → A–F grade with per-factor breakdown and blocker list

### API Keys
- Generate scoped API keys (`kumo_xxxxxx` format)
- Scopes: `send`, `relay`, `verify`, `cluster`, `read`
- Full table with key prefix, scopes, created date, last-used timestamp
- One-time key reveal modal (shown only at creation)
- Multi-VPS cluster setup guide built into the page

### Alerting & Notifications
- Configurable triggers: bounce-rate spike, blacklist hit, queue depth, service down, anomaly detected
- Delivery channels: Telegram, Discord webhook, email
- Per-domain alert thresholds

### Queue Management
- Browse queued messages with search and domain filter
- Retry individual messages or flush all deferred at once
- Drop bounced / failed messages in bulk

### System
- **System Tools** — start / stop / restart / reload KumoMTA, view systemd journal
- **Live Logs** — real-time log stream in the browser (WebSocket)
- **Security** — Fail2Ban integration, login audit log, IP block/allow list
- **API Keys** — scoped tokens for external API access and multi-VPS cluster auth
- **Webhooks** — outgoing webhooks for delivery events
- **Remote Servers** — manage multiple KumoMTA instances from one panel
- **2FA** — TOTP two-factor authentication (Google Authenticator, Authy, etc.)

---

## AI Providers

KumoOps supports 8 AI providers. Set your preferred provider in **Settings → AI Configuration**.

| Provider | Model | Type | Notes |
|---|---|---|---|
| **OpenAI** | GPT-4o-mini | ☁️ Cloud | Best general quality |
| **Anthropic** | Claude 3.5 Haiku | ☁️ Cloud | Great at structured analysis |
| **Google Gemini** | Gemini 2.0 Flash | ☁️ Cloud | Fast, generous free tier |
| **Groq** | Llama 3.3 70B | ☁️ Cloud | Very fast, generous free tier |
| **Mistral** | Mistral Small | ☁️ Cloud | European-hosted, privacy-friendly |
| **Together AI** | Llama 3.2 11B | ☁️ Cloud | Open model, affordable |
| **DeepSeek** | DeepSeek Chat | ☁️ Cloud | Excellent reasoning, low cost |
| **Ollama** | Any local model | 🖥️ Local | **FREE** — runs on your VPS, no API key |

### Ollama Setup (local, free)

```bash
# Install Ollama on your VPS
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.2       # 2GB — fast
ollama pull mistral        # 4GB — better quality
ollama pull llama3.1:8b   # 5GB — excellent

# Ollama runs at http://localhost:11434 — set this as Base URL in Settings
```

---

## Bot Commands

Both Telegram and Discord support the full command set.

- **Discord** uses native slash commands with autocomplete. Destructive commands show **Confirm ✅ / Cancel ❌ buttons**.
- **Telegram** uses `/command` text messages. Destructive commands ask you to type `/confirm` or `/cancel`.

| Command | Description |
|---|---|
| `/help` | List all available commands |
| `/stats` | Today's delivery stats — sent, delivered, bounced, deferred, rate |
| `/queue` | Queue depth broken down by destination domain |
| `/bounces` | Bounce summary for the last 24 hours |
| `/tail [n]` | Last N KumoMTA log lines (default 20, max 50) |
| `/reputation` | Latest DNSBL blacklist check results |
| `/check` | Run a fresh DNSBL scan across all IPs and domains |
| `/campaigns` | List the last 10 campaigns with status |
| `/pause-campaign <id>` | Pause a running campaign |
| `/resume-campaign <id>` | Resume a paused campaign |
| `/warmup` | Warmup status per sender (plan, current day, daily volume) |
| `/disk` | Disk usage |
| `/mem` | Memory and CPU overview |
| `/flush` | ⚠️ Flush all deferred messages |
| `/retry-all` | ⚠️ Force retry all deferred messages |
| `/drop-bounced` | ⚠️ Drop all bounced/failed messages from the queue |
| `/reload` | ⚠️ Reload KumoMTA config without downtime |
| `/restart` | ⚠️ Restart the KumoMTA service |

> ⚠️ Destructive commands require confirmation before executing.

---

## Setup

### Requirements

- Linux VPS (Rocky Linux 9 recommended)
- [KumoMTA](https://docs.kumomta.com) installed and running
- Go 1.24+ *(build only)*
- Node.js 20+ *(build only)*

### Quick Install (Rocky Linux 9)

```bash
# 1. Update your system
sudo dnf update -y

# 2. Install Git
sudo dnf install -y git

# 3. Clone the repository
sudo mkdir -p /opt/kumoops
sudo git clone https://github.com/pulak-ranjan/kumoops.git /opt/kumoops
cd /opt/kumoops

# 4. Run the installer
sudo bash scripts/install-kumoops-rocky9.sh
```

> The installer sets up KumoMTA, builds the backend and frontend, configures systemd, firewall, Nginx, and optionally provisions a Let's Encrypt SSL certificate.

### Build from source

```bash
git clone https://github.com/pulak-ranjan/kumoops
cd kumoops

# 1 — Build frontend (output lands in web/dist, embedded into the binary)
cd web && npm install && npm run build && cd ..

# 2 — Build the binary
go build -o kumoops ./cmd/server/main.go

# 3 — Run database migrations (creates kumoops.db on first run)
./kumoops migrate
```

### Run

```bash
./kumoops
# → listening on :9000
```

**Environment variables:**

| Variable | Default | Description |
|---|---|---|
| `PORT` | `9000` | HTTP listen port |
| `DB_PATH` | `./kumoops.db` | SQLite database path |
| `JWT_SECRET` | auto-generated | Override JWT signing secret |
| `KUMOMTA_API` | `http://127.0.0.1:8000` | KumoMTA HTTP API base URL |
| `KUMO_APP_SECRET` | auto-generated | AES-256 encryption key for stored secrets (AI keys, SMTP passwords) |

### systemd service

```ini
# /etc/systemd/system/kumoops.service
[Unit]
Description=KumoOps
After=network.target kumomta.service

[Service]
ExecStart=/opt/kumoops/kumoops
WorkingDirectory=/opt/kumoops
Restart=always
RestartSec=5
Environment=PORT=9000
Environment=KUMO_APP_SECRET=your-32-char-random-secret-here

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kumoops
```

### First login

Open `http://your-server:9000` — you will be prompted to create the admin account on first visit. Enable 2FA from the Settings page afterwards.

---

## Bot Configuration

### Telegram

1. Message [@BotFather](https://t.me/BotFather) → `/newbot` → copy the **Bot Token**
2. Get your **Chat ID**: send any message to your bot, then visit `https://api.telegram.org/bot<TOKEN>/getUpdates` and note the `chat.id`
3. Open **Settings → Telegram Bot**, enter the token and chat ID, enable
4. The bot starts polling immediately — no public URL needed

### Discord

Discord requires a public HTTPS endpoint for interactions:

1. Go to [discord.com/developers/applications](https://discord.com/developers/applications) → **New Application**
2. **Bot** tab → **Reset Token** → copy the **Bot Token**
3. **General Information** tab → copy the **Application ID** and **Public Key**
4. In **Settings → Discord Bot**, fill in all three fields and toggle **Enable Discord Bot**
5. **Save Settings**, then set the **Interactions Endpoint URL** in the Discord portal:
   ```
   https://your-domain/api/discord/interactions
   ```
   *(Discord will verify the endpoint on save — the server must be publicly reachable over HTTPS)*
6. Click **Register Slash Commands** in Settings once — all commands appear in Discord immediately

---

## Multi-VPS Cluster Setup

KumoOps supports managing multiple KumoMTA servers from a single primary dashboard.

```
VPS-1 (Primary)          VPS-2 (Secondary)
┌─────────────────┐      ┌──────────────────┐
│   KumoOps UI    │ ──── │   KumoOps        │
│   (main panel)  │      │   (API only)     │
│   :9000         │      │   :9000          │
└─────────────────┘      └──────────────────┘
```

**Steps:**
1. On **VPS-2** → Settings → API Keys → **Create Key** → select `cluster` scope → copy `kumo_xxx`
2. On **VPS-1** → Remote Servers → **Add Server** → set URL to `https://vps2.yourdomain.com` and paste the `kumo_xxx` key as API Token
3. VPS-1 now shows VPS-2 health, metrics, and can push config to it

---

## HTTP Sending API (Mailgun-Compatible)

KumoOps exposes a Mailgun-compatible sending API. Any app that supports Mailgun can connect directly.

```bash
# Send an email via the HTTP API
curl -X POST https://your-server/api/v1/messages \
  -H "Authorization: kumo_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "recipient@example.com",
    "from_email": "sender@yourdomain.com",
    "from_name": "My App",
    "subject": "Hello from KumoOps",
    "html": "<p>Hello world!</p>",
    "text": "Hello world!"
  }'
```

Generate an API key with `send` scope from **Settings → API Keys**.

---

## Architecture

```mermaid
flowchart TD
    subgraph KumoOps [KumoOps Platform]
        React["React / Vite\n(embedded)"]
        API["Go HTTP API\n(chi router)"]
        DB[("SQLite DB\n(GORM)")]

        API --> React
        API --> DB

        BotT["Telegram Bot\n(long-poll)"]
        BotD["Discord Bot\n(interactions)"]
        Alert["Alert Checker\n(background)"]
        Sched["Scheduler\n5min / 1h / 24h ticks"]

        API --> BotT
        API --> BotD
        API --> Alert
        API --> Sched
    end

    MTA["KumoMTA HTTP API\n(queue / delivery / metrics)"]
    AI["AI Providers\n(OpenAI, Claude, Gemini,\nGroq, Mistral, Ollama...)"]
    Google["Google Postmaster\n(ISP Reputation)"]
    SNDS["Microsoft SNDS\n(IP Reputation)"]

    API -.-> MTA
    API -.-> AI
    Sched -.-> Google
    Sched -.-> SNDS
    BotT -.-> MTA
    BotD -.-> MTA
    Alert -.-> MTA
```

**Key packages:**

| Path | Purpose |
|---|---|
| `cmd/server` | Entry point — starts HTTP server + background goroutines |
| `internal/api/` | HTTP handlers — one file per domain (~40 files) |
| `internal/core/` | Business logic — stats, queue, bots, alerting, DKIM, DMARC, FBL, DSN, VERP, ISP intel, anomaly, adaptive throttle, config gen, campaigns |
| `internal/models/` | GORM model definitions (25+ tables) |
| `internal/store/` | Database layer — queries and CRUD |
| `web/src/` | React frontend (Vite 5, Tailwind CSS 3) |

---

## REST API

All endpoints require `Authorization: Bearer <token>` (token returned at login), **except:**

| Endpoint | Auth |
|---|---|
| `POST /api/auth/register` | Public (first-run only) |
| `POST /api/auth/login` | Public |
| `POST /api/auth/verify-2fa` | Public |
| `POST /api/discord/interactions` | Ed25519 signature (Discord) |
| `POST /api/v1/messages` | API Key (`send` scope) |

See [docs/API.md](docs/API.md) for the full endpoint reference.

---

## Themes

Switch between **Light**, **System**, and **Dark** from the sidebar footer. The preference persists in `localStorage`.

---

## Development

```bash
# Backend with hot reload (requires github.com/air-verse/air)
air

# Frontend dev server with HMR (proxies /api/* to :9000)
cd web && npm run dev
```

---

## Roadmap

- [ ] OpenAPI / Swagger docs
- [ ] Multi-user roles (read-only, operator, admin)
- [ ] Per-domain delivery reports (PDF export)
- [ ] Slack bot integration
- [ ] Docker / docker-compose setup
- [ ] Prometheus metrics endpoint
- [ ] Webhooks for FBL/complaint events
- [ ] Scheduled config apply (time-based deployment)
- [ ] Bulk DKIM rotation across all domains

---

## License

GNU AGPLv3 — see [LICENSE](LICENSE).

---

## Credits

Built on top of [KumoMTA](https://kumomta.com) — a next-generation MTA written in Rust with a Lua policy scripting layer, designed for high-volume, high-deliverability email sending.
