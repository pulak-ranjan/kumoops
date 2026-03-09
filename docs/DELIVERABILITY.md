# Deliverability Guide

How KumoOps maximizes email deliverability through envelope separation, ISP traffic shaping, bounce management, FBL complaint handling, ISP intelligence, inbox placement testing, and AI-powered analysis.

---

## Table of Contents

1. [Envelope-From Separation](#envelope-from-separation)
2. [ISP Traffic Shaping](#isp-traffic-shaping)
3. [Adaptive Throttling](#adaptive-throttling)
4. [Bounce Management & DSN Classification](#bounce-management--dsn-classification)
5. [FBL / Complaint Management](#fbl--complaint-management)
6. [VERP Engine](#verp-engine)
7. [ISP Intelligence](#isp-intelligence)
8. [Anomaly Detection & Self-Healing](#anomaly-detection--self-healing)
9. [Inbox Placement Testing](#inbox-placement-testing)
10. [DKIM Signing](#dkim-signing)
11. [Header Scrubbing](#header-scrubbing)
12. [IP Warmup](#ip-warmup)
13. [AI Intelligence Layer](#ai-intelligence-layer)
14. [Connection to External Senders](#connection-to-external-senders)
15. [Settings Reference](#settings-reference)
16. [Troubleshooting](#troubleshooting)

---

## Envelope-From Separation

### What It Is

Every email has two "from" addresses:
- **Header-From** (`From:` header): What the recipient sees in their email client
- **Envelope-From** (`MAIL FROM`): Used by receiving servers for SPF checks and bounce routing

KumoOps automatically separates these to isolate bounce handling per sender identity.

### How It Works

```
Header-From:   newsletter@domain.com    (visible to recipient)
Envelope-From: newsletter@newsletter.domain.com  (used for SPF + bounces)
EHLO Domain:   newsletter.domain.com    (server identification)
```

### Why This Matters

| Benefit | Explanation |
|---------|-------------|
| **Bounce isolation** | Bounces from `newsletter` don't affect `transactional` sender reputation |
| **Per-sender SPF** | Each sender subdomain has its own SPF record |
| **Reputation separation** | ISPs track reputation per subdomain |
| **DMARC compliance** | With `aspf=r` (relaxed), `newsletter.domain.com` aligns with `domain.com` |

---

## ISP Traffic Shaping

### Default Limits

| ISP | Messages/Hour | Connections | Msgs/Connection | Conn Rate |
|-----|---------------|-------------|-----------------|-----------|
| **Gmail** | 50/h | 3 | 20 | 5/min |
| **Microsoft** (Outlook, Hotmail, Live) | 50/h | 2 | 10 | 3/min |
| **Yahoo/AOL** (yahoodns.net) | 100/h | 3 | 20 | 5/min |
| **All others** | No ISP limit | 5 | 50 | 10/min |

### How Matching Works

Limits are matched against the **MX hostname** (site_name), not the recipient domain:
- `user@gmail.com` → MX `gmail-smtp-in.l.google.com` → matches `google.com`
- `user@outlook.com` → MX `*.protection.outlook.com` → matches `outlook.com`
- `user@yahoo.com` → MX `*.yahoodns.net` → matches `yahoodns.net`

---

## Adaptive Throttling

### What It Is

KumoOps automatically adjusts per-ISP sending rates based on real-time reputation data. No manual intervention required.

### How It Works

Every 5 minutes the adaptive throttler:

1. Reads the latest ISP reputation snapshot (from Google Postmaster / SNDS / local metrics)
2. Evaluates domain reputation, IP reputation, and spam rate per ISP
3. If reputation is **degrading** → reduces the per-ISP send rate (tightens throttle)
4. If reputation has **recovered** → gradually increases the rate back toward the configured limit
5. Logs every change to the `ThrottleAdjustmentLog` for full audit history

### Viewing Throttle Changes

Go to **Deliverability → Adaptive Throttling** to see:
- Current effective rate per ISP vs. configured limit
- Full history of auto-adjustments with direction, reason, and timestamps
- Manually override a rate if needed

### Thresholds

| Reputation Level | Action |
|---|---|
| Domain reputation: **HIGH** | No change needed |
| Domain reputation: **MEDIUM** | Tighten rate by 25% |
| Domain reputation: **LOW** | Tighten rate by 50% |
| Domain reputation: **BAD** | Tighten rate by 75%, trigger alert |
| Spam rate > 0.1% | Tighten rate by 25% |
| Spam rate > 0.3% | Tighten rate by 50%, trigger alert |

---

## Bounce Management & DSN Classification

### How DSN Processing Works

1. Bounced emails arrive at the bounce address (configured via VERP or Envelope-From)
2. The Maildir watcher polls every 60 seconds for new bounce emails
3. Each DSN is parsed per RFC 3464: final-recipient, action, status, diagnostic-code
4. Bounces are classified by category and stored in `BounceClassification`

### Bounce Categories

| Category | Status Code Range | Action |
|---|---|---|
| **Hard bounce** | 5.1.x (user unknown, domain invalid) | Auto-suppress email address |
| **Soft bounce** | 4.x.x (temporary failure) | Retry with exponential backoff |
| **Spam block** | 5.7.x (policy rejection) | Reduce volume, check reputation |
| **Quota** | 4.2.2 (mailbox full) | Retry after 24h |
| **Content** | 5.6.x (content rejected) | Review email content |

### Viewing Bounce Data

**Deliverability → FBL & Bounces → DSN Bounces tab:**
- Per-category breakdown with counts and percentages
- Per-ISP breakdown (Gmail bounces vs. Microsoft bounces vs. Yahoo bounces)
- Timeline: bounce rate per day for the last 30 days
- Filter by category, ISP, date range

### Retry Configuration

| Setting | Default | Meaning |
|---|---|---|
| `retry_interval` | 5 minutes | Wait between retry attempts |
| `max_age` | 3 days | Stop retrying after this period |

---

## FBL / Complaint Management

### What Is an FBL?

A Feedback Loop (FBL) is a service provided by major ISPs (Yahoo, AOL, Comcast, etc.) where they forward a copy of each email that a recipient marks as spam to the sender. The format follows RFC 5965 (ARF — Abuse Reporting Format).

### How KumoOps Handles FBL

1. FBL emails arrive at your abuse mailbox (e.g., `abuse@yourdomain.com`)
2. The Maildir watcher detects the new email within 60 seconds
3. The ARF parser extracts: original recipient, reporting ISP, complaint type, original message headers
4. A `FBLRecord` is created in the database
5. **Auto-suppression** — the complained-about email address is added to the suppression list
6. Stats aggregated per ISP for trend tracking

### Setting Up FBL

1. Register for FBL programs at each ISP:
   - **Yahoo FBL:** [mail.yahoo.com/neo/b/feedback-loop-request](https://help.yahoo.com/kb/SLN3438.html)
   - **AOL FBL:** Register at postmaster.aol.com
   - **Comcast FBL:** Contact postmaster@comcast.net
   - **Microsoft (JMRP/SNDS):** Register at [sendersupport.olc.protection.outlook.com/snds](https://sendersupport.olc.protection.outlook.com/snds/)

2. Set the abuse mailbox as an IMAP source in **Deliverability → FBL → VERP Config tab**

3. KumoOps will automatically process all incoming FBL reports

### Viewing Complaint Data

**Deliverability → FBL & Bounces → Complaint Records tab:**
- List of all FBL records with ISP, complaint type, received date, and original recipient
- Filter by ISP, date range
- Bulk delete resolved records

**Deliverability → FBL & Bounces → ISP Breakdown tab:**
- Complaint count and rate per ISP
- Trend: complaints this week vs. last week
- Alert threshold: >0.08% complaint rate from Yahoo requires action; >0.3% from Gmail triggers auto-throttle

---

## VERP Engine

### What Is VERP?

Variable Envelope Return Path (VERP) encodes the original recipient's email address into the bounce address, so when a bounce arrives you know exactly which recipient caused it — even through forwarding chains.

### How KumoOps Implements VERP

The Envelope-From is encoded as:
```
bounce+{encoded_recipient}@bounce.yourdomain.com
```

Where `{encoded_recipient}` is an HMAC-SHA256 signed encoding of the original recipient address. This prevents spoofing of bounce reports.

**Example:**
```
Original recipient: user@example.com
Envelope-From:      bounce+hmac_abc123def456@bounce.yourdomain.com
```

When a bounce arrives at `bounce+hmac_abc123def456@bounce.yourdomain.com`, KumoOps decodes it and hard-suppresses `user@example.com`.

### Configuration

Go to **Deliverability → FBL & Bounces → VERP Config tab:**
- **HMAC Secret** — 32+ character random string (change this before going live)
- **Bounce Domain** — subdomain that receives encoded bounces (e.g., `bounce.yourdomain.com`)
- **Enabled** — toggle VERP encoding on/off

### DNS Setup for VERP

```dns
# MX record for bounce subdomain
bounce.yourdomain.com.  IN  MX  10 mail.yourdomain.com.

# SPF for bounce subdomain
bounce.yourdomain.com.  IN  TXT "v=spf1 ip4:YOUR_SERVER_IP -all"
```

---

## ISP Intelligence

### What It Is

KumoOps pulls real-time reputation data from major ISP monitoring platforms and displays it in a unified dashboard.

### Data Sources

| Source | Data | How to Enable |
|---|---|---|
| **Google Postmaster Tools** | Domain reputation (HIGH/MEDIUM/LOW/BAD), IP reputation, spam rate, DKIM rate, SPF rate, FBL rate | Connect Google Workspace account in Settings → ISP Intelligence |
| **Microsoft SNDS** | IP reputation, complaint rate per IP block | Enter SNDS API key in Settings → ISP Intelligence |
| **Local metrics** | Per-ISP send/bounce/complaint rates computed from KumoOps delivery log | Always available, no config needed |

### Setting Up Google Postmaster

1. Go to [Google Postmaster Tools](https://postmaster.google.com/) and add your sending domain
2. Verify domain ownership via DNS TXT record
3. In KumoOps **Settings → ISP Intelligence**, enter your Google Service Account JSON (for OAuth2 API access)
4. KumoOps uses pure-stdlib RS256 JWT — no Google client library required

### Reputation Levels (Google)

| Level | Meaning | Action |
|---|---|---|
| **HIGH** | ✅ Good reputation | Maintain current practices |
| **MEDIUM** | ⚠️ Moderate reputation | Review bounce rate, reduce volume 20% |
| **LOW** | 🔴 Low reputation | Pause bulk sending, focus on engaged users |
| **BAD** | 🚨 Very low reputation | Immediately pause all sending to Gmail, contact Google |

### Viewing ISP Intelligence

**Deliverability → ISP Intelligence:**
- Score cards per ISP (domain reputation, IP reputation, spam rate, FBL rate)
- 7-day sparkline trend per metric
- Color-coded alerts when any metric crosses a threshold
- Refresh button for on-demand data pull

---

## Anomaly Detection & Self-Healing

### Detected Anomalies

The anomaly detector runs every 5 minutes and checks for:

| Anomaly Type | Condition | Severity |
|---|---|---|
| **Bounce spike** | Bounce rate increases >10% in the last 30 minutes | Warning |
| **Complaint spike** | Complaint rate exceeds 0.5% | Critical |
| **Queue buildup** | Queue depth exceeds 5,000 messages | Warning |
| **Delivery rate drop** | Delivery success rate drops >30% in 1 hour | Critical |
| **Service down** | KumoMTA process not running | Critical |

### Auto-Remediation

When a critical anomaly is detected, KumoOps can automatically:

| Anomaly | Auto-Fix |
|---|---|
| Complaint spike | Pause the sending campaign responsible |
| Bounce spike | Tighten throttle to affected ISP by 50% |
| Queue buildup | Flush deferred messages, alert via Telegram/Discord |
| Delivery drop | Tighten throttle to affected ISP, alert |

### Alert Channels

Every anomaly event fires `WebhookService.SendAlert()` which notifies:
- **Telegram bot** (if configured)
- **Discord webhook** (if configured)
- **Email** (if email alerts are configured)

### Viewing Anomalies

**Deliverability → Anomaly Monitor:**
- Active anomalies with severity ring and time-since-detected
- Resolved anomalies history
- Per-type count breakdown (last 24h)
- Manual resolve button per anomaly
- Suppress button (mutes a specific anomaly type for N hours)

---

## Inbox Placement Testing

### What It Is

Inbox placement testing sends a real test email to seed mailboxes at different providers and checks whether it lands in **Inbox**, **Spam**, or goes **Missing**.

### Setup

1. **Create seed mailboxes** at each major provider:
   - Gmail — create a separate Gmail account for testing
   - Outlook — create a separate Outlook.com account
   - Yahoo — create a separate Yahoo account
   - Any other providers you care about

2. Enable IMAP access for each mailbox:
   - Gmail: Settings → See all settings → Forwarding and POP/IMAP → Enable IMAP
   - Enable "App Passwords" if 2FA is enabled

3. In **Deliverability → Inbox Placement → Seed Mailboxes tab:**
   - Add each mailbox with: email address, IMAP host, IMAP port (993), username, password, provider label

### Running a Placement Test

1. Go to **Deliverability → Inbox Placement → Placement Tests tab**
2. Click **Run New Test**
3. Enter: subject, HTML body, sender email
4. KumoOps sends the test email to all active seed mailboxes
5. After 2–5 minutes, click **Check Results**
6. Results show per-mailbox: Inbox ✅ / Spam ⚠️ / Missing ❌

### Interpreting Results

| Result | Meaning | Action |
|---|---|---|
| **Inbox** at all providers | Great deliverability | No action needed |
| **Spam** at Gmail only | Gmail-specific issue | Check Google Postmaster reputation, review content |
| **Spam** at multiple providers | Widespread content or reputation issue | Review content with AI Content Analyzer, check IP reputation |
| **Missing** | Email not delivered or delayed >10 min | Check queue for deferrals, verify DNS records |

---

## DKIM Signing

### How It Works

1. **Auto-Setup** generates RSA-2048 DKIM keys per sender
2. Keys stored at `/opt/kumomta/etc/dkim/{domain}/{localpart}.key`
3. Selector = sender's localpart (e.g., `newsletter._domainkey.domain.com`)
4. Config matches `From:` header email to the correct key

### Signed Headers

```
From, To, Subject, Date, Message-ID, List-Unsubscribe, List-Unsubscribe-Post
```

> `List-Unsubscribe-Post` is required for Gmail one-click unsubscribe compliance.

### Verification

```bash
swaks --to test@gmail.com --from newsletter@domain.com \
  --server mta.domain.com:587 \
  --auth LOGIN --auth-user newsletter@domain.com --auth-password "pass" \
  --tls

# Check received email headers for:
# dkim=pass header.d=domain.com header.s=newsletter
```

---

## Header Scrubbing

KumoOps automatically removes headers that reveal internal infrastructure:

| Removed Header | Reason |
|---|---|
| `User-Agent` | Reveals sending software |
| `X-Mailer` | Reveals sending software |
| `X-Originating-IP` | Exposes internal IPs |
| `X-Report-Abuse` | Internal header |
| `X-Campaign`, `X-Tenant`, `X-KumoMTA` | Internal routing headers |

A clean `Received` header is injected that mimics standard Postfix format.

---

## IP Warmup

### Warmup Plans

| Plan | Day 1 | Day 7 | Day 14 | Day 30 | Best For |
|------|-------|-------|--------|--------|----------|
| Conservative | 50/hr | 200/hr | 500/hr | 2000/hr | Most senders |
| Moderate | 100/hr | 500/hr | 1000/hr | 5000/hr | Established brands |
| Aggressive | 500/hr | 2000/hr | 5000/hr | 10000/hr | Transactional only |

### Warmup Best Practices

1. **Start with engaged users** — send to most active subscribers first
2. **Monitor bounce rates** — keep hard bounces below 2%
3. **Watch for deferrals** — 421/450 means you're sending too fast
4. **Check ISP Intelligence** — monitor Google Postmaster reputation daily
5. **Maintain consistency** — similar volumes daily, avoid spikes

---

## AI Intelligence Layer

### Deliverability Advisor

The AI Deliverability Advisor aggregates all available signals — ISP reputation, anomaly events, FBL complaints, bounce classifications, throttle adjustments — and asks the AI to produce a structured analysis.

**Go to AI Intelligence → Deliverability Advisor:**
- **Score** — 0–100 overall deliverability health
- **Trend** — improving / stable / declining
- **Top Issues** — ranked list with severity (critical/warning/info) and recommended action
- **AI Analysis** — full Markdown narrative explaining what's happening and why

**When to use:** After any major sending event, when you see ISP reputation drop, or weekly for routine hygiene checks.

### Content Analyzer

Paste your email subject and HTML body → the AI scores it for spam risk and deliverability.

**Go to AI Intelligence → Content Analyzer:**
- **Spam Score** — 0 (clean) to 10 (very spammy)
- **Deliverability Score** — 0–100 estimate of inbox placement
- **Issues** — specific words, structures, or patterns that trigger spam filters
- **Suggestions** — actionable improvements

**Common spam triggers caught:**
- Excessive capitalization in subject ("HUGE SALE TODAY!!!")
- Too many exclamation marks
- Spam trigger words ("FREE", "CLICK HERE", "ACT NOW")
- Image-heavy emails with little text
- Missing unsubscribe link
- Excessive number of links
- Deceptive subject lines

### Subject Line Generator

**Go to AI Intelligence → Subject Line Generator:**
1. Enter topic, target audience, tone (professional/friendly/urgent/casual), and goal (open rate / clicks / conversions)
2. Set how many variants you want (1–10)
3. Click **Generate**

Each variant includes:
- The subject line text
- Style tag (curiosity / urgency / benefit / social-proof / personal / question)
- Emoji-enhanced version
- Reasoning notes from the AI

### Pre-Send Campaign Score

Before sending any campaign, run the Pre-Send Score check:

**Go to AI Intelligence → Pre-Send Score:**
1. Select your campaign from the dropdown
2. View grade (A–F) and per-factor breakdown

| Factor | Max Points | What's Checked |
|---|---|---|
| Subject Line | 10 | Length (30–60 chars), no spam words, personalisation |
| Body Quality | 10 | Text-to-HTML ratio, link count, unsubscribe present |
| Sender Reputation | 15 | DKIM configured, SPF aligned, DMARC in place |
| Recipient List | 15 | List age, suppression applied, hard bounce rate |
| Complaint Rate | 20 | Historical complaint rate for this sender |
| Active Anomalies | 15 | Any open critical anomalies affecting this sender |
| Unsubscribe Config | 15 | One-click unsubscribe header configured |

**Grades:**
- A (90–100): Ready to send
- B (80–89): Good, minor improvements possible
- C (70–79): Fix issues before large-scale sending
- D (60–69): Multiple problems, needs attention
- F (<60): Do not send — fix blockers first

---

## Connection to External Senders

### HTTP API (Mailgun-Compatible)

The simplest way to connect any app to KumoOps:

```bash
# Python example
import requests
requests.post('https://your-server/api/v1/messages',
  headers={'Authorization': 'kumo_your_send_key'},
  json={
    'to': 'user@example.com',
    'from_email': 'sender@yourdomain.com',
    'subject': 'Hello',
    'html': '<p>Hello world!</p>'
  })
```

Generate a key with `send` scope at **Settings → API Keys**.

### SMTP (Traditional)

| Setting | Value |
|---|---|
| **SMTP Host** | `mta.yourdomain.com` |
| **Port** | `587` (STARTTLS) or `465` (Implicit TLS) |
| **Encryption** | TLS/STARTTLS |
| **Auth** | LOGIN or PLAIN |
| **Username** | Full sender email (`newsletter@yourdomain.com`) |
| **Password** | SMTP password set in the panel |

---

## Settings Reference

### Panel Settings

| Setting | Description |
|---|---|
| **Main Hostname** | FQDN of your MTA server |
| **Main Server IP** | Primary sending IP |
| **Relay IPs** | IPs allowed to relay without auth |
| **TLS Certificate Path** | Path to SSL certificate |
| **TLS Private Key Path** | Path to SSL private key |
| **AI Provider** | OpenAI / Anthropic / Gemini / Groq / Mistral / Together / DeepSeek / Ollama |
| **AI API Key** | API key for chosen provider (encrypted at rest) |
| **Ollama Base URL** | Ollama server URL (default: `http://localhost:11434`) |
| **Ollama Model** | Local model name (e.g., `llama3.2`, `mistral`) |

### Generated Config Files

| File | Purpose |
|---|---|
| `init.lua` | Main KumoMTA policy — listeners, auth, hooks, routing |
| `auth.toml` | SMTP credentials |
| `sources.toml` | Egress sources with IP and EHLO domain per sender |
| `queues.toml` | Queue config per tenant (retry, rate limits) |
| `listener_domains.toml` | Accepted domains for relay |
| `dkim_data.toml` | DKIM keys, selectors, signing policies |

---

## Troubleshooting

### "535 Authentication failed"
- Verify SMTP username is the full email (`sender@domain.com`)
- Verify the password matches what's set in the panel
- Ensure TLS is configured — AUTH is not offered without TLS
- Check KumoMTA logs: `journalctl -u kumomta -f`

### "550 Relaying not permitted"
- Ensure the sender is authenticated (use port 587 with TLS + credentials)
- Check that the domain or subdomain exists in `listener_domains.toml`
- Verify auth succeeded first

### "SPF softfail/fail"
- Verify the sender subdomain has an SPF record
- The envelope-from subdomain is what's checked for SPF, not the header-from domain
- Run: `dig +short newsletter.domain.com TXT`

### "DKIM signature invalid"
- Ensure DKIM DNS record matches the public key generated by the panel
- Check the selector: `newsletter._domainkey.domain.com`
- Verify key file: `ls /opt/kumomta/etc/dkim/domain.com/newsletter.key`

### High spam rate at Gmail
1. Check Google Postmaster → Domain Reputation (should be HIGH)
2. Review recent campaigns with AI Content Analyzer
3. Check your FBL complaint count (target < 0.08%)
4. Pause bulk sends; switch to engaged-users-only for 2 weeks

### Messages stuck in queue
- Check queue in the panel (Queue page)
- Look for deferral reasons: `journalctl -u kumomta | grep "4[0-9][0-9]"`
- If ISP is throttling, traffic shaping handles retries automatically
- Use **Flush Queue** to force immediate retry of deferred messages

### Inbox Placement test shows "Spam"
1. Run AI Content Analyzer on your test email
2. Check your IP reputation at [SenderScore](https://senderscore.org/) and Google Postmaster
3. Verify no blacklist issues via the panel's RBL monitoring
4. Check DMARC alignment: `dig _dmarc.domain.com TXT`
5. Ensure `List-Unsubscribe` header is present and valid

### AI Advisor shows score < 50
1. Review the ranked issues list — start with **Critical** severity items
2. Check ISP Intelligence for any LOW/BAD reputation domains
3. Look at active anomalies — resolve them before sending
4. Run Pre-Send Score on your next campaign before sending
