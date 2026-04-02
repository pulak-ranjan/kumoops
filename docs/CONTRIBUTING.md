# Contributing to KumoOps

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to KumoOps.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Node.js 18+ and npm
- SQLite3
- Git

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/pulak-ranjan/f-kumo.git
   cd f-kumo
   ```

2. **Install Go dependencies**
   ```bash
   go mod tidy
   ```

3. **Install frontend dependencies**
   ```bash
   cd web
   npm install
   cd ..
   ```

4. **Run the backend (development)**
   ```bash
   export DB_DIR=./data
   go run ./cmd/server
   ```

5. **Run the frontend (development)**
   ```bash
   cd web
   npm run dev
   ```

6. **Access the panel**
   - Frontend: http://localhost:5173
   - API: http://localhost:9000/api

## Project Structure

```
f-kumo/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/             # HTTP handlers and routing
│   │   ├── authtools.go     # BIMI, MTA-STS, auth check endpoints
│   │   ├── alerts.go        # Alert rules and event endpoints
│   │   ├── bounce_analytics.go
│   │   ├── ippools.go       # IP Pool management
│   │   ├── suppression.go   # Suppression list management
│   │   ├── shaping.go       # Traffic shaping rules
│   │   └── ...
│   ├── core/            # Business logic (config generation, DKIM, etc.)
│   ├── models/          # Database models
│   └── store/           # Database operations
├── scripts/             # Installation and utility scripts
├── web/                 # React frontend
│   ├── src/
│   │   ├── pages/       # Page components
│   │   │   ├── Dashboard.jsx
│   │   │   ├── Domains.jsx
│   │   │   ├── IPsPage.jsx
│   │   │   ├── IPPoolPage.jsx
│   │   │   ├── WarmupPage.jsx
│   │   │   ├── TrafficShapingPage.jsx
│   │   │   ├── QueuePage.jsx
│   │   │   ├── BouncePage.jsx
│   │   │   ├── BounceAnalyticsPage.jsx
│   │   │   ├── SuppressionPage.jsx
│   │   │   ├── AlertsPage.jsx
│   │   │   ├── EmailAuthPage.jsx
│   │   │   ├── DKIMPage.jsx
│   │   │   ├── DMARCPage.jsx
│   │   │   ├── LogsPage.jsx
│   │   │   ├── SecurityPage.jsx
│   │   │   ├── WebhooksPage.jsx
│   │   │   ├── ConfigPage.jsx
│   │   │   ├── Settings.jsx
│   │   │   ├── StatsPage.jsx
│   │   │   ├── ToolsPage.jsx
│   │   │   └── APIKeysPage.jsx
│   │   ├── components/  # Shared components (Layout, ThemeProvider, etc.)
│   │   ├── api.js       # API client (all fetch calls go here)
│   │   └── AuthContext.jsx
│   └── ...
└── go.mod
```

## Code Style

### Go

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors explicitly
- Use the existing patterns in the codebase

### React / JavaScript

- Use functional components with hooks
- Follow the existing Tailwind CSS patterns
- Keep components focused and reasonably sized
- Use the `api.js` module for all API calls — do not call `fetch` directly in page components unless adding a new route to `api.js` is disproportionate

## Making Changes

### Branch Naming

- `feature/description` — New features
- `fix/description` — Bug fixes
- `docs/description` — Documentation updates
- `refactor/description` — Code refactoring

### Commit Messages

Use clear, descriptive commit messages:

```
feat: add per-provider delivery stats
fix: token expiry check timezone issue
docs: update installation instructions
refactor: extract DNS helpers to separate module
```

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Test thoroughly (see checklist below)
4. Update documentation if needed
5. Submit a pull request with a clear description

## Testing

### Manual Testing Checklist

Before submitting a PR, please verify:

- [ ] Backend compiles without errors (`go build ./cmd/server`)
- [ ] Frontend builds without errors (`cd web && npm run build`)
- [ ] New features work as expected
- [ ] Existing functionality is not broken
- [ ] API responses are correct
- [ ] UI displays correctly in both dark and light mode
- [ ] UI displays correctly on mobile (responsive)

### Running the Full Stack

```bash
# Terminal 1: Backend
export DB_DIR=./data
go run ./cmd/server

# Terminal 2: Frontend
cd web
npm run dev
```

## Reporting Issues

When reporting issues, please include:

- Description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, Node version)
- Relevant logs or error messages

## Security Issues

If you discover a security vulnerability, please **do not** open a public issue. Contact the maintainer directly.

## Questions?

Feel free to open an issue for questions or discussions about the project.
