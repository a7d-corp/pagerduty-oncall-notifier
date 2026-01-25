package notifier

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

// PushoverNotifier sends notifications via the Pushover API
type PushoverNotifier struct {
	appToken string
	userKey  string
	device   string
	sound    string
	client   *http.Client
	apiURL   string
}

// NewPushoverNotifier creates a new Pushover notifier
func NewPushoverNotifier(appToken, userKey, device, sound string) *PushoverNotifier {
	return &PushoverNotifier{
		appToken: appToken,
		userKey:  userKey,
		device:   device,
		sound:    sound,
		client:   &http.Client{Timeout: 30 * time.Second},
		apiURL:   pushoverAPIURL,
	}
}

// Notify sends a simple notification message
func (p *PushoverNotifier) Notify(message string) error {
	return p.NotifyWithEvent(EventShiftStarted, time.Now().UTC())
}

// NotifyWithEvent sends a notification with event-specific formatting
func (p *PushoverNotifier) NotifyWithEvent(event NotificationEvent, shiftStartTime time.Time) error {
	var message, title string
	priority := "0"

	switch event {
	case EventShiftStarted:
		message = "🚨 Your PagerDuty on-call shift has started!"
		title = "PagerDuty On-Call Shift Started"
		priority = "1"
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
		priority = "0"
	case EventShiftEnded:
		message = "✅ Your PagerDuty on-call shift has ended. Enjoy the downtime!"
		title = "PagerDuty On-Call Shift Ended"
		priority = "0"
	default:
		message = "Unknown notification event"
		title = "PagerDuty Notification"
		priority = "0"
	}

	values := url.Values{}
	values.Set("token", p.appToken)
	values.Set("user", p.userKey)
	values.Set("message", message)
	values.Set("title", title)
	values.Set("priority", priority)
	values.Set("timestamp", fmt.Sprintf("%d", shiftStartTime.Unix()))

	if p.device != "" {
		values.Set("device", p.device)
	}
	if p.sound != "" {
		values.Set("sound", p.sound)
	}

	resp, err := p.client.PostForm(p.apiURL, values)
	if err != nil {
		return fmt.Errorf("failed to send pushover notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pushover returned non-2xx status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
