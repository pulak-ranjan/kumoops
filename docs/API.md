# KumoOps — API Reference

**Version:** 0.2.0
**Base URL:** `http://your-server:9000/api` (or `https://your-domain/api`)
**License:** GNU AGPLv3

---

## Table of Contents

- [Authentication](#-authentication)
- [Domains & Senders](#-domains--senders)
- [Bounce Accounts](#-bounce-accounts)
- [DNS & Authentication (DKIM/DMARC)](#%EF%B8%8F-dns--authentication)
- [FBL + DSN + Bounce Intelligence](#-fbl--dsn--bounce-intelligence)
- [ISP Intelligence](#-isp-intelligence)
- [Adaptive Throttling](#-adaptive-throttling)
- [Anomaly Detection](#-anomaly-detection)
- [Inbox Placement Testing](#-inbox-placement-testing)
- [Traffic Shaping](#-traffic-shaping)
- [Send-Time Optimization](#-send-time-optimization)
- [A/B Testing](#-ab-testing)
- [SMTP Relay](#-smtp-relay)
- [Multi-Node Cluster](#-multi-node-cluster)
- [API Keys](#-api-keys)
- [HTTP Sending API (Mailgun-Compatible)](#-http-sending-api-mailgun-compatible)
- [AI Chat](#-ai-chat)
- [AI Intelligence (Advisor)](#-ai-intelligence-advisor)
- [Webhooks & Automation](#-webhooks--automation)
- [System & Networking](#%EF%B8%8F-system--networking)
- [Configuration & Queue](#%EF%B8%8F-configuration--queue)
- [Logs](#-logs)
- [IP Pools](#-ip-pools)
- [Suppression List](#-suppression-list)
- [Alert Rules](#-alert-rules)
- [Email Auth Tools](#%EF%B8%8F-email-auth-tools)
- [Campaigns](#-campaigns)
- [Error Responses](#error-responses)

---

## 🔐 Authentication

All protected endpoints require a Bearer token in the `Authorization` header:
```
Authorization: Bearer <jwt_token>
```

The HTTP Sending API (`POST /api/v1/messages`) accepts API keys instead:
```
Authorization: kumo_your_api_key_here
```

### Public Endpoints

#### Register Admin
Create the first admin account (only allowed if no admin exists).
- **POST** `/auth/register`
- **Body:** `{ "email": "admin@example.com", "password": "StrongPassword1!" }`

#### Login
Authenticate and receive a session token.
- **POST** `/auth/login`
- **Body:** `{ "email": "admin@example.com", "password": "..." }`
- **Response (Success):** `{ "token": "...", "email": "..." }`
- **Response (2FA Required):** `{ "requires_2fa": true, "temp_token": "..." }`

#### Verify 2FA (Login Step 2)
Complete login if 2FA is enabled.
- **POST** `/auth/verify-2fa`
- **Header:** `X-Temp-Token: <temp_token_from_login>`
- **Body:** `{ "code": "123456" }`
- **Response:** `{ "token": "...", "email": "..." }`

---

### Protected Profile & Security

#### Get Current User
- **GET** `/auth/me`

#### Logout
Invalidate the current session token.
- **POST** `/auth/logout`

#### Setup 2FA
Initialize TOTP setup (returns QR code URI).
- **POST** `/auth/setup-2fa`
- **Body:** `{ "password": "current_password" }`
- **Response:** `{ "secret": "...", "uri": "otpauth://..." }`

#### Enable 2FA
- **POST** `/auth/enable-2fa`
- **Body:** `{ "code": "123456" }`

#### Disable 2FA
- **POST** `/auth/disable-2fa`
- **Body:** `{ "password": "...", "code": "123456" }`

#### Update Theme
- **POST** `/auth/theme`
- **Body:** `{ "theme": "dark" }` (or `light`, `system`)

#### List Sessions
- **GET** `/auth/sessions`

---

## 📧 Domains & Senders

#### List Domains
- **GET** `/domains`

#### Create Domain
- **POST** `/domains`
- **Body:** `{ "name": "example.com", "mail_host": "mail.example.com", "bounce_host": "bounce.example.com" }`

#### Get Domain
- **GET** `/domains/{id}`

#### Update Domain
- **PUT** `/domains/{id}`

#### Delete Domain
- **DELETE** `/domains/{id}`

#### List Senders
- **GET** `/domains/{domainID}/senders`

#### Create Sender
- **POST** `/domains/{domainID}/senders`
- **Body:** `{ "local_part": "newsletter", "ip": "1.2.3.4", "smtp_password": "..." }`

#### Update Sender
- **PUT** `/senders/{id}`

#### Delete Sender
- **DELETE** `/senders/{id}`

#### Auto-Setup Sender
Generate DKIM keys and create bounce account for a sender.
- **POST** `/domains/{domainID}/senders/{id}/setup`

---

## 📬 Bounce Accounts

#### List Bounce Accounts
- **GET** `/bounces`

#### Create Bounce Account
Creates DB entry AND Linux system user.
- **POST** `/bounces`
- **Body:** `{ "username": "b-news", "domain": "example.com", "password": "..." }`

#### Delete Bounce Account
- **DELETE** `/bounces/{id}`

#### Apply System State
Ensure all DB bounce accounts exist as Linux users.
- **POST** `/bounces/apply`

---

## 🛡️ DNS & Authentication

#### List DKIM Keys
- **GET** `/dkim/records`

#### Generate DKIM
- **POST** `/dkim/generate`
- **Body:** `{ "domain": "example.com", "local_part": "optional_selector" }`

#### Get DMARC Record
- **GET** `/dmarc/{domainID}`

#### Set DMARC Policy
- **POST** `/dmarc/{domainID}`
- **Body:** `{ "policy": "quarantine", "percentage": 100, "rua": "..." }`

#### Get All DNS Records
- **GET** `/dns/{domainID}`

---

## 📨 FBL + DSN + Bounce Intelligence

> Feedback Loop complaint management, DSN bounce classification, and VERP tracking.

#### List FBL Records
Paginated complaint records.
- **GET** `/fbl/records?page=1&limit=50&isp=gmail`

#### FBL Stats by ISP
Complaint counts and rates per ISP (Gmail, Yahoo, etc.).
- **GET** `/fbl/stats`

#### FBL Record Detail
- **GET** `/fbl/records/{id}`

#### Delete FBL Record
- **DELETE** `/fbl/records/{id}`

#### Bulk Delete FBL Records
- **POST** `/fbl/records/bulk-delete`
- **Body:** `{ "ids": [1, 2, 3] }`

#### Trigger Suppression
Apply suppressions for all emails in FBL complaint records.
- **POST** `/fbl/suppress`

#### Manual FBL Parse
Parse a raw FBL email body.
- **POST** `/fbl/parse`
- **Body:** `{ "raw": "From: ...\n\nReturn-Path: ..." }`

#### List DSN Bounce Classifications
Classified bounce records.
- **GET** `/fbl/bounces?category=hard&page=1&limit=50`

#### Bounce Summary
Aggregated bounce stats by category and ISP.
- **GET** `/fbl/bounces/summary`

#### VERP Config
Get or update VERP encode/decode settings.
- **GET** `/fbl/verp`
- **POST** `/fbl/verp` — Body: `{ "enabled": true, "hmac_secret": "...", "bounce_domain": "bounce.example.com" }`

---

## 📡 ISP Intelligence

> Google Postmaster Tools and Microsoft SNDS reputation data.

#### Latest ISP Snapshots
Current reputation snapshot for each configured ISP/domain.
- **GET** `/isp-intel/snapshots`

#### ISP Snapshot History
7-day trend for a specific domain.
- **GET** `/isp-intel/snapshots/{domain}?days=7`

#### Refresh ISP Data
Trigger an immediate pull from Google Postmaster and SNDS.
- **POST** `/isp-intel/refresh`

#### ISP Intel Summary
Aggregate view: worst/best reputation domains, average scores.
- **GET** `/isp-intel/summary`

---

## ⚡ Adaptive Throttling

#### Throttle Adjustment Log
All automatic rate changes made by the adaptive throttler.
- **GET** `/throttle/logs?limit=50`

#### Throttle Status
Current per-ISP send rate vs. configured limit.
- **GET** `/throttle/status`

---

## 🔍 Anomaly Detection

#### List Anomaly Events
Active and resolved anomaly events.
- **GET** `/anomalies?status=active&limit=50`

#### Anomaly Stats
Counts by severity and type for the last 24 hours.
- **GET** `/anomalies/stats`

#### Resolve Anomaly
Mark an anomaly as resolved.
- **POST** `/anomalies/{id}/resolve`

#### Suppress Anomaly Type
Stop alerting for a specific anomaly type temporarily.
- **POST** `/anomalies/{id}/suppress`
- **Body:** `{ "duration_hours": 24 }`

---

## 🧪 Inbox Placement Testing

#### List Seed Mailboxes
- **GET** `/placement/mailboxes`

#### Add Seed Mailbox
- **POST** `/placement/mailboxes`
- **Body:** `{ "name": "Gmail Seed", "email": "seed@gmail.com", "imap_host": "imap.gmail.com", "imap_port": 993, "username": "...", "password": "...", "provider": "gmail" }`

#### Update Seed Mailbox
- **PUT** `/placement/mailboxes/{id}`

#### Delete Seed Mailbox
- **DELETE** `/placement/mailboxes/{id}`

#### Run Placement Test
Sends a test email to all active seed mailboxes and checks placement.
- **POST** `/placement/tests`
- **Body:** `{ "subject": "Placement Test", "html": "<p>Test</p>", "from_email": "test@yourdomain.com", "campaign_id": 0 }`

#### List Placement Tests
- **GET** `/placement/tests?limit=20`

#### Get Placement Test Detail
Per-mailbox results for a specific test.
- **GET** `/placement/tests/{id}`

---

## 🔀 Traffic Shaping

#### List Shaping Rules
- **GET** `/shaping`

#### Create Shaping Rule
- **POST** `/shaping`
- **Body:** `{ "site_name": "gmail", "max_connection_rate": "50/h", "connection_limit": 3 }`

#### Update Shaping Rule
- **PUT** `/shaping/{id}`

#### Delete Shaping Rule
- **DELETE** `/shaping/{id}`

#### Seed Default Rules
Load built-in ISP presets (Gmail, Microsoft, Yahoo).
- **POST** `/shaping/seed`

---

## 📊 Send-Time Optimization

#### Engagement Heatmap
7×24 engagement matrix (hour-of-week × day-of-week) from delivery events.
- **GET** `/send-time/heatmap?campaign_id=0` (0 = all campaigns)

#### Time Slot Recommendations
Top-5 optimal send times with engagement score.
- **GET** `/send-time/recommendations?campaign_id=0`

---

## 🧬 A/B Testing

#### List Variants for Campaign
- **GET** `/campaigns/{id}/variants`

#### Create Variant
- **POST** `/campaigns/{id}/variants`
- **Body:** `{ "name": "Variant B", "subject": "Alt Subject", "split_pct": 20 }`

#### Update Variant
- **PUT** `/campaigns/{id}/variants/{variantID}`

#### Delete Variant
- **DELETE** `/campaigns/{id}/variants/{variantID}`

#### Select Winner
Manually designate the winning variant. Routes all remaining traffic to it.
- **POST** `/campaigns/{id}/variants/{variantID}/winner`

#### Variant Stats
Open rate, click rate, conversion stats per variant.
- **GET** `/campaigns/{id}/variants/stats`

---

## 📡 SMTP Relay

#### Relay Status
Current relay service status, connection count, message counters.
- **GET** `/relay/status`

#### Relay Settings
- **GET** `/relay/settings`
- **POST** `/relay/settings` — Body: `{ "enabled": true, "max_connections": 50, "rate_limit": "1000/h" }`

#### List Allowed Relay IPs
- **GET** `/relay/allowed-ips`

#### Add Allowed IP
- **POST** `/relay/allowed-ips`
- **Body:** `{ "ip": "10.0.0.5", "description": "MailWizz server" }`

#### Delete Allowed IP
- **DELETE** `/relay/allowed-ips/{id}`

#### Relay Connection Log
Recent relay connection attempts and results.
- **GET** `/relay/logs?limit=100`

---

## 🌐 Multi-Node Cluster

#### List Cluster Nodes
- **GET** `/cluster/nodes`

#### Add Cluster Node
- **POST** `/cluster/nodes`
- **Body:** `{ "name": "VPS-2", "url": "https://vps2.yourdomain.com", "api_token": "kumo_xxx" }`

#### Delete Cluster Node
- **DELETE** `/cluster/nodes/{id}`

#### Node Health Check
Ping a specific node and return latency + service status.
- **POST** `/cluster/nodes/{id}/ping`

#### All Nodes Health
Ping all nodes simultaneously.
- **GET** `/cluster/health`

#### Aggregate Metrics
Pull and sum delivery metrics from all online nodes.
- **GET** `/cluster/metrics`

#### Push Config to Nodes
Push current KumoMTA config to all or selected nodes.
- **POST** `/cluster/push-config`
- **Body:** `{ "node_ids": [1, 2] }` (omit to push to all)

---

## 🔑 API Keys

#### List API Keys
- **GET** `/keys`

#### Create API Key
- **POST** `/keys`
- **Body:** `{ "name": "MailWizz Production", "scopes": "send,relay" }`
- **Response:** `{ "id": 1, "name": "...", "key": "kumo_xxxx", "scopes": "send,relay", "created_at": "..." }`
  > ⚠️ The full key is only returned once at creation time. Store it securely.

**Available scopes:**
| Scope | Permission |
|---|---|
| `send` | POST /api/v1/messages — HTTP Sending API |
| `relay` | SMTP relay authentication |
| `verify` | Email verification endpoints |
| `cluster` | Multi-node cluster authentication (remote server token) |
| `read` | Read-only stats and queue access |

#### Delete API Key
- **DELETE** `/keys/{id}`

---

## 📤 HTTP Sending API (Mailgun-Compatible)

> Requires API key with `send` scope. Use `Authorization: kumo_xxx` header (no "Bearer" prefix).

#### Send Email
- **POST** `/v1/messages`
- **Header:** `Authorization: kumo_your_send_key`
- **Body:**
```json
{
  "to": "recipient@example.com",
  "from_email": "sender@yourdomain.com",
  "from_name": "My App",
  "subject": "Hello!",
  "html": "<p>Hello world!</p>",
  "text": "Hello world!",
  "reply_to": "noreply@yourdomain.com",
  "cc": "cc@example.com",
  "bcc": "bcc@example.com"
}
```
- **Response:** `{ "id": "queue_id_xxx", "status": "queued" }`

---

## 🤖 AI Chat

> Requires AI provider configured in Settings.

#### Get Chat History
Last 50 chat exchanges.
- **GET** `/ai/history`

#### Send Chat Message
Send a message to the AI assistant. The assistant may execute safe tools (`status`, `queue`, `logs_kumo`, `block_ip`, `dig`, etc.) and return formatted output.
- **POST** `/ai/chat`
- **Body:** `{ "new_msg": "What does my bounce rate look like?" }`
- **Response:** `{ "reply": "## Bounce Analysis\n\n..." }`

---

## 🧠 AI Intelligence (Advisor)

> Requires AI provider configured in Settings.

#### Deliverability Advisor
Aggregates ISP reputation, anomalies, FBL data, and bounces → AI-generated deliverability report.
- **GET** `/ai/deliverability-advisor`
- **Response:**
```json
{
  "score": 72,
  "trend": "declining",
  "issues": [
    { "severity": "critical", "title": "Gmail spam rate elevated", "action": "Pause Gmail-bound campaigns" }
  ],
  "analysis": "## Deliverability Report\n\n...",
  "generated_at": "2026-03-09T14:00:00Z"
}
```

#### Analyze Email Content
Score a subject + HTML body for spam risk and deliverability.
- **POST** `/ai/analyze-content`
- **Body:** `{ "subject": "Big Sale!", "html_body": "<p>Click here!</p>", "sender_domain": "example.com" }`
- **Response:**
```json
{
  "spam_score": 3.2,
  "deliverability_score": 78,
  "issues": ["Subject uses spam trigger word 'Sale'"],
  "suggestions": ["Add personalisation to subject line"],
  "analysis": "..."
}
```

#### Generate Subject Lines
Generate AI subject line variants.
- **POST** `/ai/subject-lines`
- **Body:** `{ "topic": "Summer sale", "audience": "Existing customers", "tone": "friendly", "goal": "open_rate", "count": 5 }`
- **Response:**
```json
{
  "variants": [
    {
      "text": "Your summer deal is waiting, {{first_name}}",
      "style": "personal",
      "emoji_version": "☀️ Your summer deal is waiting, {{first_name}}",
      "notes": "Personalisation drives opens"
    }
  ]
}
```

#### Pre-Send Campaign Score
Local computed readiness check for a campaign. No AI required.
- **GET** `/campaigns/{id}/send-score`
- **Response:**
```json
{
  "score": 83,
  "grade": "B",
  "checks": [
    { "name": "Subject Line", "score": 9, "max": 10, "status": "pass", "detail": "Good length and no spam words" },
    { "name": "Complaint Rate", "score": 16, "max": 20, "status": "warning", "detail": "Complaint rate at 0.12%" }
  ],
  "blockers": []
}
```

---

## 🔔 Webhooks & Automation

#### Get Webhook Settings
- **GET** `/webhooks/settings`

#### Update Webhook Settings
- **POST** `/webhooks/settings`
- **Body:** `{ "webhook_url": "...", "webhook_enabled": true, "bounce_alert_pct": 5.0 }`

#### Test Webhook
- **POST** `/webhooks/test`
- **Body:** `{ "webhook_url": "..." }`

#### Webhook Logs
- **GET** `/webhooks/logs`

#### Manual: Check Bounces
- **POST** `/webhooks/check-bounces`

---

## 🖥️ System & Networking

#### Dashboard Stats
CPU, RAM, Disk, and Service status.
- **GET** `/dashboard/stats`

#### List System IPs
- **GET** `/system/ips`

#### Add IP
- **POST** `/system/ips`
- **Body:** `{ "value": "1.2.3.4", "interface": "eth0" }`

#### Bulk Add IPs
- **POST** `/system/ips/bulk`
- **Body:** `{ "ips": ["1.2.3.4", "5.6.7.8"] }`

#### Add IPs by CIDR
- **POST** `/system/ips/cidr`
- **Body:** `{ "cidr": "192.168.1.0/24" }`

#### Auto-Detect IPs
- **POST** `/system/ips/detect`

#### Manual: Check Blacklists
- **POST** `/system/check-blacklist`

#### Manual: Security Audit
- **POST** `/system/check-security`

---

## ⚙️ Configuration & Queue

#### Preview Config
Generate KumoMTA config files in memory.
- **GET** `/config/preview`

#### Apply Config
Write configs to disk and restart KumoMTA service.
- **POST** `/config/apply`

#### View Queue
- **GET** `/queue?limit=100`

#### Queue Stats
- **GET** `/queue/stats`

#### Delete Message
- **DELETE** `/queue/{id}`

#### Flush Queue
Force retry of all deferred messages.
- **POST** `/queue/flush`

---

## 📝 Logs

#### Service Logs
- **GET** `/logs/kumomta?lines=100`
- **GET** `/logs/dovecot?lines=100`
- **GET** `/logs/fail2ban?lines=100`

---

## 📥 Import

#### Bulk Import (CSV)
- **POST** `/import/csv`
- **Form Data:** `file` (CSV with headers: `domain, localpart, ip, password`)

---

## 🧱 IP Pools

#### List Pools
- **GET** `/ippools`

#### Create Pool
- **POST** `/ippools`
- **Body:** `{ "name": "warmup-pool", "description": "..." }`

#### Update Pool
- **PUT** `/ippools/{id}`

#### Delete Pool
- **DELETE** `/ippools/{id}`

#### Add Member to Pool
- **POST** `/ippools/{id}/members`
- **Body:** `{ "ip": "1.2.3.4" }`

#### Remove Member from Pool
- **DELETE** `/ippools/{id}/members/{memberID}`

---

## 🚫 Suppression List

#### List Entries (paginated)
- **GET** `/suppression?page=1&limit=50&q=search`

#### Add Entry
- **POST** `/suppression`
- **Body:** `{ "email": "bad@example.com", "reason": "bounced" }`

#### Delete Entry
- **DELETE** `/suppression/{id}`

#### Check Email
- **GET** `/suppression/check?email=user@example.com`

#### Bulk Add
- **POST** `/suppression/bulk`
- **Body:** `{ "emails": ["a@x.com", "b@x.com"], "reason": "complaint" }`

#### Export List
- **GET** `/suppression/export`

#### Import List
- **POST** `/suppression/import`
- **Form Data:** `file` (plain text, one email per line)

#### Wipe All Entries
- **DELETE** `/suppression/wipe`

---

## 🔔 Alert Rules

#### List Rules
- **GET** `/alerts/rules`

#### Create Rule
- **POST** `/alerts/rules`
- **Body:** `{ "name": "High Bounce", "metric": "bounce_rate", "threshold": 5.0, "channel": "webhook" }`

#### Update Rule
- **PUT** `/alerts/rules/{id}`

#### Delete Rule
- **DELETE** `/alerts/rules/{id}`

#### Test Rule
- **POST** `/alerts/test/{id}`

#### Alert Event History
- **GET** `/alerts/events?limit=50`

---

## 🛡️ Email Auth Tools

#### Check Domain Auth
- **GET** `/authtools/check/{domain}`

#### BIMI Record
- **GET** `/authtools/bimi/{domain}`
- **POST** `/authtools/bimi/{domain}` — Body: `{ "vmc_url": "...", "logo_url": "..." }`

#### MTA-STS Policy
- **GET** `/authtools/mtasts/{domain}`
- **POST** `/authtools/mtasts/{domain}` — Body: `{ "mode": "enforce", "mx": ["mail.example.com"], "max_age": 86400 }`

---

## 📊 Bounce Analytics

#### Get Bounce Stats
- **GET** `/bounce-analytics?lines=500`

---

## 📋 Campaigns

#### List Campaigns
- **GET** `/campaigns`

#### Create Campaign
- **POST** `/campaigns`
- **Body:** `{ "name": "June Newsletter", "subject": "...", "html_body": "...", "from_email": "...", "list_id": 1 }`

#### Get Campaign
- **GET** `/campaigns/{id}`

#### Update Campaign
- **PUT** `/campaigns/{id}`

#### Delete Campaign
- **DELETE** `/campaigns/{id}`

#### Campaign Stats
Per-campaign delivery counters, open rate, click rate.
- **GET** `/campaigns/{id}/stats`

#### Pause Campaign
- **POST** `/campaigns/{id}/pause`

#### Resume Campaign
- **POST** `/campaigns/{id}/resume`

#### Pre-Send Score
Local readiness check. Returns grade A–F and per-factor breakdown.
- **GET** `/campaigns/{id}/send-score`

---

## Error Responses

All errors return JSON:
```json
{ "error": "human readable error message" }
```

Common status codes:
| Code | Meaning |
|---|---|
| `200` | Success |
| `201` | Created |
| `400` | Bad request / invalid JSON |
| `401` | Missing or invalid token |
| `403` | Insufficient scope (API key) |
| `404` | Resource not found |
| `500` | Internal server error |
