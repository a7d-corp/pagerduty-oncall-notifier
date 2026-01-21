package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// NotificationBackend represents the type of notification backend
type NotificationBackend string

const (
	BackendWebhook NotificationBackend = "webhook"
	BackendNtfy    NotificationBackend = "ntfy"
)

// Config holds all configuration for the application
type Config struct {
	PagerDutyAPIToken      string
	PagerDutyScheduleID    string
	PagerDutyUserID        string
	CheckInterval          time.Duration
	AdvanceNotificationTime time.Duration
	NotificationBackend   NotificationBackend
	NotificationWebhookURL string
	NtfyServerURL         string
	NtfyTopic             string
	NtfyAPIKey            string
	StateFilePath         string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Required: PagerDuty API Token
	cfg.PagerDutyAPIToken = os.Getenv("PD_API_TOKEN")
	if cfg.PagerDutyAPIToken == "" {
		return nil, fmt.Errorf("PD_API_TOKEN environment variable is required")
	}

	// Required: PagerDuty Schedule ID
	cfg.PagerDutyScheduleID = os.Getenv("PD_SCHEDULE_ID")
	if cfg.PagerDutyScheduleID == "" {
		return nil, fmt.Errorf("PD_SCHEDULE_ID environment variable is required")
	}

	// Required: PagerDuty User ID
	cfg.PagerDutyUserID = os.Getenv("PD_USER_ID")
	if cfg.PagerDutyUserID == "" {
		return nil, fmt.Errorf("PD_USER_ID environment variable is required")
	}

	// Required: Notification Backend
	backendStr := os.Getenv("NOTIFICATION_BACKEND")
	if backendStr == "" {
		return nil, fmt.Errorf("NOTIFICATION_BACKEND environment variable is required (must be 'webhook' or 'ntfy')")
	}
	cfg.NotificationBackend = NotificationBackend(backendStr)
	if cfg.NotificationBackend != BackendWebhook && cfg.NotificationBackend != BackendNtfy {
		return nil, fmt.Errorf("NOTIFICATION_BACKEND must be 'webhook' or 'ntfy', got: %s", backendStr)
	}

	// Backend-specific configuration
	switch cfg.NotificationBackend {
	case BackendWebhook:
		cfg.NotificationWebhookURL = os.Getenv("NOTIFICATION_WEBHOOK_URL")
		if cfg.NotificationWebhookURL == "" {
			return nil, fmt.Errorf("NOTIFICATION_WEBHOOK_URL environment variable is required when using webhook backend")
		}
	case BackendNtfy:
		cfg.NtfyServerURL = os.Getenv("NTFY_SERVER_URL")
		if cfg.NtfyServerURL == "" {
			return nil, fmt.Errorf("NTFY_SERVER_URL environment variable is required when using ntfy backend")
		}
		cfg.NtfyTopic = os.Getenv("NTFY_TOPIC")
		if cfg.NtfyTopic == "" {
			return nil, fmt.Errorf("NTFY_TOPIC environment variable is required when using ntfy backend")
		}
		// API key is optional for ntfy
		cfg.NtfyAPIKey = os.Getenv("NTFY_API_KEY")
	}

	// Optional: Check Interval (default: 300 seconds / 5 minutes)
	checkIntervalStr := os.Getenv("CHECK_INTERVAL")
	if checkIntervalStr == "" {
		cfg.CheckInterval = 5 * time.Minute
	} else {
		interval, err := strconv.Atoi(checkIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("CHECK_INTERVAL must be a valid integer: %w", err)
		}
		if interval <= 0 {
			return nil, fmt.Errorf("CHECK_INTERVAL must be greater than 0")
		}
		cfg.CheckInterval = time.Duration(interval) * time.Second
	}

	// Optional: Advance Notification Time (default: disabled/0 if not set)
	advanceTimeStr := os.Getenv("ADVANCE_NOTIFICATION_TIME")
	if advanceTimeStr != "" {
		advanceTime, err := time.ParseDuration(advanceTimeStr)
		if err != nil {
			return nil, fmt.Errorf("ADVANCE_NOTIFICATION_TIME must be a valid duration (e.g., '2h', '30m', '1h30m'): %w", err)
		}
		if advanceTime <= 0 {
			return nil, fmt.Errorf("ADVANCE_NOTIFICATION_TIME must be greater than 0")
		}
		cfg.AdvanceNotificationTime = advanceTime
		log.Printf("Advance notification time: %v", advanceTime)
	}

	// Optional: State File Path (default: /data/state.json)
	cfg.StateFilePath = os.Getenv("STATE_FILE_PATH")
	if cfg.StateFilePath == "" {
		cfg.StateFilePath = "/data/state.json"
	}

	return cfg, nil
}
