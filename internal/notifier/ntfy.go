package notifier

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// NtfyNotifier sends notifications via ntfy.sh or self-hosted ntfy server
type NtfyNotifier struct {
	serverURL string
	topic     string
	apiKey    string
	client    *http.Client
}

// NewNtfyNotifier creates a new ntfy notifier
func NewNtfyNotifier(serverURL, topic, apiKey string) *NtfyNotifier {
	return &NtfyNotifier{
		serverURL: serverURL,
		topic:     topic,
		apiKey:    apiKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Notify sends a simple notification message
func (n *NtfyNotifier) Notify(message string) error {
	return n.NotifyWithEvent(EventShiftStarted, time.Now().UTC())
}

// NotifyWithEvent sends a notification with event-specific formatting
func (n *NtfyNotifier) NotifyWithEvent(event NotificationEvent, shiftStartTime time.Time) error {
	var message, title string
	var priority, tags string

	switch event {
	case EventShiftStarted:
		message = "🚨 Your PagerDuty on-call shift has started!"
		title = "PagerDuty On-Call Shift Started"
		priority = "urgent"
		tags = "rotating_light,alarm_clock"
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
		title = "PagerDuty On-Call Shift Upcoming"
		priority = "default"
		tags = "alarm_clock,clock1"
	case EventShiftEnded:
		message = "✅ Your PagerDuty on-call shift has ended. Enjoy the downtime!"
		title = "PagerDuty On-Call Shift Ended"
		priority = "default"
		tags = "white_check_mark,beach_with_umbrella"
	default:
		message = "Unknown notification event"
		title = "PagerDuty Notification"
		priority = "default"
		tags = "question"
	}

	url := fmt.Sprintf("%s/%s", n.serverURL, n.topic)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(message))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority)
	req.Header.Set("Tags", tags)

	// Add authentication if API key is provided
	if n.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", n.apiKey))
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ntfy notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// SendBirthMessage sends a birth message (used for ntfy lifecycle)
func (n *NtfyNotifier) SendBirthMessage() error {
	return n.sendLifecycleMessage("Birth message", "PagerDuty Notifier Started", "white_check_mark")
}

// SendWillMessage sends a will message (used for ntfy lifecycle)
func (n *NtfyNotifier) SendWillMessage() error {
	return n.sendLifecycleMessage("Will message", "PagerDuty Notifier Stopped", "x")
}

// sendLifecycleMessage sends a lifecycle message for ntfy
func (n *NtfyNotifier) sendLifecycleMessage(message, title, tags string) error {
	url := fmt.Sprintf("%s/%s", n.serverURL, n.topic)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(message))
	if err != nil {
		return fmt.Errorf("failed to create lifecycle request: %w", err)
	}

	req.Header.Set("Title", title)
	req.Header.Set("Priority", "low")
	req.Header.Set("Tags", tags)

	// Add authentication if API key is provided
	if n.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", n.apiKey))
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send lifecycle message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("lifecycle message returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
