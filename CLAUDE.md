# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A lightweight Go service that monitors PagerDuty on-call rotations and sends notifications when shifts start. The service polls the PagerDuty API, detects on-call transitions, and sends notifications through pluggable backends (webhook, ntfy).

## Build & Development Commands

### Local Development

```bash
# Install dependencies
go mod download

# Build the application
go build -o notifier ./cmd/notifier

# Run locally (requires environment variables to be set)
go run ./cmd/notifier
```

### Docker

```bash
# Build Docker image
docker build -t pagerduty-oncall-notifier .

# Run with Docker Compose (recommended)
docker-compose up -d

# View logs
docker-compose logs -f

# Stop service
docker-compose down
```

### Testing

No test files currently exist in this codebase. When adding tests, follow these patterns:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/pagerduty
```

## Architecture

### Core Components

1. **PagerDuty Client** (`internal/pagerduty/client.go`)
   - Wraps the official PagerDuty Go SDK
   - `IsOnCall()`: Checks current on-call status for a specific user/schedule
   - `GetUpcomingShift()`: Fetches the next upcoming shift (7-day lookahead)

2. **State Manager** (`internal/state/manager.go`)
   - Persists on-call state to JSON file to prevent duplicate notifications
   - Tracks: `WasOnCall` (boolean) and `LastAdvanceNotificationSent` (timestamp)
   - Detects transitions from not-on-call → on-call
   - Implements advance notification logic with 24-hour deduplication window

3. **Notification System** (`internal/notifier/`)
   - Interface-based design (`Notifier` interface)
   - Two implementations: `WebhookNotifier` and `NtfyNotifier`
   - Supports two event types: `EventShiftStarted` and `EventUpcomingShift`
   - NtfyNotifier includes birth/will messages for service lifecycle tracking

4. **Configuration** (`internal/config/config.go`)
   - All configuration via environment variables
   - Validates required variables at startup
   - Backend-specific validation (webhook URL or ntfy server/topic)

5. **Main Loop** (`cmd/notifier/main.go`)
   - Polls PagerDuty API at configurable intervals (default: 5 minutes)
   - Checks for shift transitions and sends notifications
   - Graceful shutdown with signal handling (SIGTERM, SIGINT)
   - Sends ntfy will message on shutdown

### Data Flow

1. Main loop polls PagerDuty API every `CHECK_INTERVAL` seconds
2. Client fetches current on-call status via PagerDuty SDK
3. If advance notifications enabled, client fetches upcoming shifts
4. State Manager compares current state with previous state from disk
5. On transition detection or advance window match, notifier is invoked
6. State is persisted to disk after each check

### Notification Backends

**Webhook Backend:**
- Sends JSON POST with `message`, `timestamp`, `event` fields
- Events: `oncall_shift_started` or `oncall_shift_upcoming`

**Ntfy Backend:**
- Posts to `{NTFY_SERVER_URL}/{NTFY_TOPIC}` with custom headers
- Different priorities: "urgent" for shift started, "default" for upcoming
- Optional authentication via Bearer token
- Includes birth message on startup and will message on shutdown

### State Persistence

- JSON file stored at `STATE_FILE_PATH` (default: `/data/state.json`)
- Structure: `{"was_on_call": bool, "last_advance_notification_sent": "RFC3339 timestamp"}`
- Advance notifications deduplicated within 24-hour windows

## Environment Variables

### Required for All Configurations

- `PD_API_TOKEN`: PagerDuty REST API v2 token
- `PD_SCHEDULE_ID`: Schedule/rotation ID to monitor
- `PD_USER_ID`: User ID to track
- `NOTIFICATION_BACKEND`: Either "webhook" or "ntfy"

### Backend-Specific (Webhook)

- `NOTIFICATION_WEBHOOK_URL`: Webhook URL for POST requests

### Backend-Specific (Ntfy)

- `NTFY_SERVER_URL`: Base URL of ntfy server
- `NTFY_TOPIC`: Topic name
- `NTFY_API_KEY`: (Optional) API key for authentication

### Optional

- `CHECK_INTERVAL`: Polling interval in seconds (default: 300)
- `ADVANCE_NOTIFICATION_TIME`: Time before shift for advance notification (e.g., "2h", "30m") - disabled if not set
- `STATE_FILE_PATH`: Path to state file (default: "/data/state.json")

## Adding New Notification Backends

1. Create new file in `internal/notifier/` (e.g., `slack.go`)
2. Implement the `Notifier` interface:
   ```go
   type Notifier interface {
       Notify(message string) error
       NotifyWithEvent(event NotificationEvent, shiftStartTime time.Time) error
   }
   ```
3. Add new backend constant to `internal/config/config.go`
4. Update `Load()` in config to validate new backend-specific env vars
5. Update `createNotifier()` in `cmd/notifier/main.go` to instantiate your backend

## Docker & Deployment

- Multi-stage build: Go 1.25.6 builder → Alpine runtime
- Runs as non-root user (uid/gid 1000)
- Volume mount `/data` for state persistence
- GitHub Actions workflow builds and pushes to `ghcr.io` on releases
- Images tagged with both release version and `latest`

## Module Information

- Module path: `github.com/a7d-corp/pagerduty-oncall-notifier`
- Go version: 1.25.6
- Primary dependency: `github.com/PagerDuty/go-pagerduty` v1.8.0
