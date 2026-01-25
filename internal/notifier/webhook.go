package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookNotifier sends notifications via HTTP webhook
type WebhookNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(webhookURL string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Notify sends a simple notification message
func (w *WebhookNotifier) Notify(message string) error {
	return w.NotifyWithEvent(EventShiftStarted, time.Now().UTC())
}

// NotifyWithEvent sends a notification with event-specific formatting
func (w *WebhookNotifier) NotifyWithEvent(event NotificationEvent, shiftStartTime time.Time) error {
	var message string
	var eventType string

	switch event {
	case EventShiftStarted:
		message = "🚨 Your PagerDuty on-call shift has started!"
		eventType = "oncall_shift_started"
	case EventUpcomingShift:
		duration := time.Until(shiftStartTime)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60

		if hours > 0 {
			if minutes > 0 {
				message = fmt.Sprintf("⏰ Your PagerDuty on-call shift starts in %d hours and %d minutes!", hours, minutes)
			} else {
				message = fmt.Sprintf("⏰ Your PagerDuty on-call shift starts in %d hours!", hours)
			}
		} else if minutes > 0 {
			message = fmt.Sprintf("⏰ Your PagerDuty on-call shift starts in %d minutes!", minutes)
		} else {
			message = "⏰ Your PagerDuty on-call shift starts soon!"
		}
		eventType = "oncall_shift_upcoming"
	case EventShiftEnded:
		message = "✅ Your PagerDuty on-call shift has ended. Enjoy the downtime!"
		eventType = "oncall_shift_ended"
	default:
		message = "Unknown notification event"
		eventType = "unknown"
	}

	payload := map[string]interface{}{
		"message":   message,
		"timestamp": shiftStartTime.Format(time.RFC3339),
		"event":     eventType,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	resp, err := w.client.Post(w.webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
