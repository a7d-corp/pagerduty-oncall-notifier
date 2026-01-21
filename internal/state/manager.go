package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the persisted on-call state
type State struct {
	WasOnCall                bool       `json:"was_on_call"`
	LastAdvanceNotificationSent *time.Time `json:"last_advance_notification_sent,omitempty"`
}

// Manager handles state persistence and transition detection
type Manager struct {
	filePath string
}

// NewManager creates a new state manager
func NewManager(filePath string) *Manager {
	return &Manager{
		filePath: filePath,
	}
}

// Load loads the state from disk, returning default state if file doesn't exist
func (m *Manager) Load() (*State, error) {
	// Check if file exists
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// Return default state (not on-call)
		return &State{WasOnCall: false}, nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Save persists the state to disk
func (m *Manager) Save(state *State) error {
	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// HasTransitionToOnCall checks if there was a transition from not-on-call to on-call
func (m *Manager) HasTransitionToOnCall(previousState *State, currentlyOnCall bool) bool {
	return !previousState.WasOnCall && currentlyOnCall
}

// ShouldSendAdvanceNotification checks if an advance notification should be sent
// Returns true if:
// - The shift starts within the advance notification window
// - No advance notification has been sent yet, or the last one was for a different shift
func (m *Manager) ShouldSendAdvanceNotification(state *State, shiftStartTime time.Time, advanceTime time.Duration) bool {
	if advanceTime <= 0 {
		return false
	}

	now := time.Now().UTC()
	timeUntilShift := shiftStartTime.Sub(now)

	// Check if shift is within the advance notification window
	if timeUntilShift <= 0 || timeUntilShift > advanceTime {
		return false
	}

	// Check if we've already sent an advance notification for this shift
	// We'll consider it a different shift if more than 24 hours have passed since the last notification
	if state.LastAdvanceNotificationSent != nil {
		timeSinceLastNotification := now.Sub(*state.LastAdvanceNotificationSent)
		if timeSinceLastNotification < 24*time.Hour {
			return false
		}
	}

	return true
}

// RecordAdvanceNotificationSent updates the state to record when an advance notification was sent
func (m *Manager) RecordAdvanceNotificationSent(state *State) {
	now := time.Now().UTC()
	state.LastAdvanceNotificationSent = &now
}
