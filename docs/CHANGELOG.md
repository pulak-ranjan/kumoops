# Changelog

All notable changes to KumoMTA UI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

> Planned improvements and upcoming features will be listed here before each release.

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

#### UI
- Responsive layout with collapsible sidebar and mobile hamburger menu
- Dark / Light mode with system preference synchronization
- Card-based design with Lucide icon set
- Real-time terminal log viewer for KumoMTA, Dovecot, Fail2Ban
- AI Assistant panel for log analysis and health insights

#### API
- Full REST API for all management operations
- Bearer token authentication on all protected routes
- CORS enforcement and localhost-only API binding

---

## Future Roadmap

> The following items are tracked as potential improvements for future releases.

### [0.1.0] — Planned
- Per-provider delivery stats (sent, bounced, deferred per ISP)
- Expanded DMARC reporting with aggregate view
- Multi-user admin support with role-based access

### [0.2.0] — Planned
- Real-time queue WebSocket streaming
- Scheduled config apply (time-based deployment)
- Bulk DKIM rotation across all domains

### [1.0.0] — Planned
- Stable API with guaranteed backwards compatibility
- Full test suite with CI/CD pipeline
- Official Docker image and Helm chart
