# PagerDuty On-Call Notifier

A lightweight Go-based service that monitors PagerDuty on-call rotations and sends notifications when your shift starts. The notification system is modular to support multiple platforms (currently webhook and ntfy, easily extensible).

## Features

- Polls PagerDuty API at configurable intervals (default: 5 minutes)
- Detects when your on-call shift starts (transition from not-on-call to on-call)
- **NEW**: Configurable advance notifications before your shift starts (e.g., notify 2 hours in advance)
- Supports multiple notification backends: webhook, ntfy (self-hosted), and Pushover
- Optional authentication for ntfy servers (API key)
- Persists state to avoid duplicate notifications
- Runs in a Docker container with minimal resource usage
- Graceful shutdown handling

## Architecture

The application consists of:

- **PagerDuty Client**: Fetches on-call status for a specific schedule
- **State Manager**: Tracks previous on-call status and detects transitions
- **Notification System**: Modular interface supporting webhook and ntfy backends
- **Main Loop**: Polls PagerDuty API and orchestrates notifications

## Prerequisites

- Docker and Docker Compose (for containerized deployment)
- PagerDuty API token (REST API v2)
- PagerDuty Schedule ID (the rotation you want to monitor)
- PagerDuty User ID (your user ID)
- Notification backend: Either a webhook URL or a self-hosted ntfy server

## Configuration

### Environment Variables

#### PagerDuty Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PD_API_TOKEN` | Yes | - | PagerDuty REST API v2 token |
| `PD_SCHEDULE_ID` | Yes | - | Schedule/rotation ID to monitor |
| `PD_USER_ID` | Yes | - | Your PagerDuty user ID |
| `CHECK_INTERVAL` | No | `300` | Polling interval in seconds (default: 5 minutes) |
| `ADVANCE_NOTIFICATION_TIME` | No | - | Time before shift to send advance notification (e.g., "2h", "30m", "1h30m"). Disabled if not set |
| `STATE_FILE_PATH` | No | `/data/state.json` | Path to state persistence file |

#### Notification Backend Selection

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NOTIFICATION_BACKEND` | Yes | - | Backend to use: `webhook`, `ntfy`, or `pushover` |

#### Webhook Backend (when `NOTIFICATION_BACKEND=webhook`)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NOTIFICATION_WEBHOOK_URL` | Yes | - | Webhook URL for notifications |

#### Ntfy Backend (when `NOTIFICATION_BACKEND=ntfy`)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NTFY_SERVER_URL` | Yes | - | Base URL of your self-hosted ntfy server (e.g., `https://ntfy.example.com`) |
| `NTFY_TOPIC` | Yes | - | Topic name to publish to |
| `NTFY_API_KEY` | No | - | API key for authentication (optional, if server requires auth) |

#### Pushover Backend (when `NOTIFICATION_BACKEND=pushover`)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PUSHOVER_APP_TOKEN` | Yes | - | Application/API token from your Pushover account |
| `PUSHOVER_USER_KEY` | Yes | - | Pushover user or group key that should receive notifications |
| `PUSHOVER_DEVICE` | No | - | Optional device name to target a single device |
| `PUSHOVER_SOUND` | No | - | Optional sound override (e.g., `siren`, `magic`) |

### Finding Your PagerDuty IDs

1. **API Token**:
   - Go to PagerDuty → Configuration → API Access Keys
   - Create a new REST API v2 key
   - Copy the token

2. **Schedule ID**:
   - Navigate to your schedule in PagerDuty
   - The Schedule ID is in the URL: `https://your-domain.pagerduty.com/schedules#SCHEDULE_ID`
   - Or use the API: `GET /schedules` and find your schedule

3. **User ID**:
   - Go to your PagerDuty profile
   - The User ID is in the URL: `https://your-domain.pagerduty.com/users#USER_ID`
   - Or use the API: `GET /users` and find your user

## Usage

### Using Docker Compose (Recommended)

#### Webhook Backend

1. Create a `.env` file in the project root:

```bash
PD_API_TOKEN=your_pagerduty_api_token
PD_SCHEDULE_ID=your_schedule_id
PD_USER_ID=your_user_id
NOTIFICATION_BACKEND=webhook
NOTIFICATION_WEBHOOK_URL=https://your-webhook-url.com/notify
CHECK_INTERVAL=300
ADVANCE_NOTIFICATION_TIME=2h
```

#### Ntfy Backend

1. Create a `.env` file in the project root:

```bash
PD_API_TOKEN=your_pagerduty_api_token
PD_SCHEDULE_ID=your_schedule_id
PD_USER_ID=your_user_id
NOTIFICATION_BACKEND=ntfy
NTFY_SERVER_URL=https://ntfy.example.com
NTFY_TOPIC=your-topic-name
NTFY_API_KEY=your_api_key_optional
CHECK_INTERVAL=300
ADVANCE_NOTIFICATION_TIME=2h
```

**Note**: `NTFY_API_KEY` is optional. Only include it if your ntfy server requires authentication.

#### Pushover Backend

```bash
PD_API_TOKEN=your_pagerduty_api_token
PD_SCHEDULE_ID=your_schedule_id
PD_USER_ID=your_user_id
NOTIFICATION_BACKEND=pushover
PUSHOVER_APP_TOKEN=your_pushover_app_token
PUSHOVER_USER_KEY=your_pushover_user_key
PUSHOVER_DEVICE=optional_device_name
PUSHOVER_SOUND=optional_sound
CHECK_INTERVAL=300
ADVANCE_NOTIFICATION_TIME=2h
```

2. Build and run:

```bash
docker-compose up -d
```

3. View logs:

```bash
docker-compose logs -f
```

4. Stop the service:

```bash
docker-compose down
```

### Using Docker Directly

1. Build the image:

```bash
docker build -t pagerduty-oncall-notifier .
```

2. Run the container with webhook backend:

```bash
docker run -d \
  --name pagerduty-oncall-notifier \
  --restart unless-stopped \
  -e PD_API_TOKEN=your_token \
  -e PD_SCHEDULE_ID=your_schedule_id \
  -e PD_USER_ID=your_user_id \
  -e NOTIFICATION_BACKEND=webhook \
  -e NOTIFICATION_WEBHOOK_URL=https://your-webhook-url.com/notify \
  -e CHECK_INTERVAL=300 \
  -e ADVANCE_NOTIFICATION_TIME=2h \
  -v $(pwd)/data:/data \
  pagerduty-oncall-notifier
```

3. Or run with ntfy backend:

```bash
docker run -d \
  --name pagerduty-oncall-notifier \
  --restart unless-stopped \
  -e PD_API_TOKEN=your_token \
  -e PD_SCHEDULE_ID=your_schedule_id \
  -e PD_USER_ID=your_user_id \
  -e NOTIFICATION_BACKEND=ntfy \
  -e NTFY_SERVER_URL=https://ntfy.example.com \
  -e NTFY_TOPIC=your-topic-name \
  -e NTFY_API_KEY=your_api_key_optional \
  -e CHECK_INTERVAL=300 \
  -e ADVANCE_NOTIFICATION_TIME=2h \
  -v $(pwd)/data:/data \
  pagerduty-oncall-notifier
```

4. Or run with Pushover backend:

```bash
docker run -d \
  --name pagerduty-oncall-notifier \
  --restart unless-stopped \
  -e PD_API_TOKEN=your_token \
  -e PD_SCHEDULE_ID=your_schedule_id \
  -e PD_USER_ID=your_user_id \
  -e NOTIFICATION_BACKEND=pushover \
  -e PUSHOVER_APP_TOKEN=your_pushover_app_token \
  -e PUSHOVER_USER_KEY=your_pushover_user_key \
  -e PUSHOVER_DEVICE=optional_device_name \
  -e PUSHOVER_SOUND=optional_sound \
  -e CHECK_INTERVAL=300 \
  -e ADVANCE_NOTIFICATION_TIME=2h \
  -v $(pwd)/data:/data \
  pagerduty-oncall-notifier
```

### Running Locally (Development)

1. Install dependencies:

```bash
go mod download
```

2. Set environment variables (webhook backend):

```bash
export PD_API_TOKEN=your_token
export PD_SCHEDULE_ID=your_schedule_id
export PD_USER_ID=your_user_id
export NOTIFICATION_BACKEND=webhook
export NOTIFICATION_WEBHOOK_URL=https://your-webhook-url.com/notify
export CHECK_INTERVAL=300
export ADVANCE_NOTIFICATION_TIME=2h
export STATE_FILE_PATH=./data/state.json
```

Or set environment variables (ntfy backend):

```bash
export PD_API_TOKEN=your_token
export PD_SCHEDULE_ID=your_schedule_id
export PD_USER_ID=your_user_id
export NOTIFICATION_BACKEND=ntfy
export NTFY_SERVER_URL=https://ntfy.example.com
export NTFY_TOPIC=your-topic-name
export NTFY_API_KEY=your_api_key_optional
export CHECK_INTERVAL=300
export ADVANCE_NOTIFICATION_TIME=2h
export STATE_FILE_PATH=./data/state.json
```

Or set environment variables (Pushover backend):

```bash
export PD_API_TOKEN=your_token
export PD_SCHEDULE_ID=your_schedule_id
export PD_USER_ID=your_user_id
export NOTIFICATION_BACKEND=pushover
export PUSHOVER_APP_TOKEN=your_pushover_app_token
export PUSHOVER_USER_KEY=your_pushover_user_key
export PUSHOVER_DEVICE=optional_device_name
export PUSHOVER_SOUND=optional_sound
export CHECK_INTERVAL=300
export ADVANCE_NOTIFICATION_TIME=2h
export STATE_FILE_PATH=./data/state.json
```

3. Run the application:

```bash
go run ./cmd/notifier
```

## Notification Formats

### Webhook Backend

When your shift starts, the webhook receives a POST request with the following JSON payload:

```json
{
  "message": "🚨 Your PagerDuty on-call shift has started!",
  "timestamp": "2024-01-15T10:30:00Z",
  "event": "oncall_shift_started"
}
```

#### Advance Notification

When advance notification is enabled and your upcoming shift is within the configured time window, an additional webhook is sent with:

```json
{
  "message": "⏰ Your PagerDuty on-call shift starts in 2 hours!",
  "timestamp": "2024-01-15T10:30:00Z",
  "event": "oncall_shift_upcoming"
}
```

### Ntfy Backend

When your shift starts, the ntfy server receives a POST request to `{NTFY_SERVER_URL}/{NTFY_TOPIC}` with:

- **Message Body**: `🚨 Your PagerDuty on-call shift has started!`
- **Headers**:
  - `Title`: "PagerDuty On-Call Shift Started"
  - `Priority`: "urgent"
  - `Tags`: "rotating_light,alarm_clock"
  - `Authorization`: `Bearer {NTFY_API_KEY}` (if API key is provided)

#### Advance Notification

When advance notification is enabled and your upcoming shift is within the configured time window, an additional notification is sent with:

- **Message Body**: `⏰ Your PagerDuty on-call shift starts in X hours/minutes!` (time calculated dynamically)
- **Headers**:
  - `Title`: "PagerDuty On-Call Shift Upcoming"
  - `Priority`: "default"
  - `Tags`: "alarm_clock,clock1"
  - `Authorization`: `Bearer {NTFY_API_KEY}` (if API key is provided)

The notification will appear on any device subscribed to the topic. For more information about ntfy, see the [ntfy documentation](https://docs.ntfy.sh/).

#### Ntfy Authentication

If your self-hosted ntfy server requires authentication, you can provide an API key via the `NTFY_API_KEY` environment variable. The notifier will include this as a Bearer token in the `Authorization` header. For details on setting up access tokens, see the [ntfy authentication documentation](https://docs.ntfy.sh/publish/#access-tokens).

### Pushover Backend

When your shift starts, the notifier issues a POST to the [Pushover message API](https://pushover.net/api) with the following form data:

- `token`: Your application token (`PUSHOVER_APP_TOKEN`)
- `user`: Your user or group key (`PUSHOVER_USER_KEY`)
- `title`: "PagerDuty On-Call Shift Started"
- `message`: `🚨 Your PagerDuty on-call shift has started!`
- `priority`: `1` (high priority)
- `timestamp`: Current UTC timestamp
- `sound` and `device` if you configured overrides

Advance notifications send the same API request with:

- `title`: "PagerDuty On-Call Shift Upcoming"
- Dynamic `message` describing the time remaining
- `priority`: `0` (normal priority)

Errors returned by the API (non-2xx statuses) are logged, and the response body is included to aid debugging.

## State Persistence

The application persists its state to a JSON file (default: `/data/state.json` in the container). This ensures:

- No duplicate notifications if the container restarts
- Accurate detection of shift transitions
- State survives container restarts

The state file contains:

```json
{
  "was_on_call": false,
  "last_advance_notification_sent": "2024-01-15T08:30:00Z"
}
```

The `last_advance_notification_sent` field tracks when the last advance notification was sent to prevent duplicate notifications for the same shift.

## Extending Notification Backends

The notification system is modular. To add a new notification backend:

1. Implement the `Notifier` interface in `internal/notifier/`:

```go
type Notifier interface {
    Notify(message string) error
}
```

2. Create a new file (e.g., `internal/notifier/slack.go`) with your implementation

3. Update the main application to use your new notifier

Example implementations could include:
- Slack webhook
- Discord webhook
- Email (SMTP)
- Telegram bot
- Matrix room message

## Development

### Project Structure

```
pagerduty-oncall-notifier/
├── cmd/
│   └── notifier/
│       └── main.go          # Main application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration loading
│   ├── pagerduty/
│   │   └── client.go         # PagerDuty API client
│   ├── state/
│   │   └── manager.go        # State persistence
│   └── notifier/
│       ├── notifier.go       # Notification interface
│       └── webhook.go        # Webhook implementation
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### Building

```bash
go build -o notifier ./cmd/notifier
```

### Testing

The application logs all operations. Check logs to verify:
- Configuration loading
- PagerDuty API calls
- State transitions
- Notification sending

## Troubleshooting

### No notifications received

1. Verify all environment variables are set correctly
2. Check that your User ID matches the on-call user
3. Verify the Schedule ID is correct
4. Check webhook URL is accessible
5. Review container logs: `docker-compose logs`

### Duplicate notifications

- The state file should prevent this. If you're seeing duplicates:
  - Check that the state file is being persisted (volume mount)
  - Verify file permissions allow writing

### API errors

- Verify your API token has the correct permissions
- Check that the Schedule ID and User ID are valid
- Ensure network connectivity to PagerDuty API

## License

This project is provided as-is for personal use.
