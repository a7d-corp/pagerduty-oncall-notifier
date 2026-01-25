package notifier

import (
	"time"
)

// NotificationEvent represents different types of notifications
type NotificationEvent string

const (
	EventShiftStarted  NotificationEvent = "shift_started"
	EventUpcomingShift NotificationEvent = "upcoming_shift"
	EventShiftEnded    NotificationEvent = "shift_ended"
)

// Notifier defines the interface for notification backends
type Notifier interface {
	Notify(message string) error
	NotifyWithEvent(event NotificationEvent, shiftStartTime time.Time) error
}
