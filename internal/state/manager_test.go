package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReturnsDefaultWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	manager := NewManager(statePath)
	state, err := manager.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if state.WasOnCall {
		t.Fatalf("expected default state to be off-call")
	}
	if state.LastAdvanceNotificationSent != nil {
		t.Fatalf("expected LastAdvanceNotificationSent to be nil")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state", "state.json")

	manager := NewManager(statePath)

	lastNotification := time.Now().UTC().Add(-3 * time.Hour).Round(time.Second)
	original := &State{
		WasOnCall:                   true,
		LastAdvanceNotificationSent: &lastNotification,
	}

	if err := manager.Save(original); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// sanity check: file exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected state file to be created: %v", err)
	}

	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if !loaded.WasOnCall {
		t.Fatalf("expected WasOnCall to persist")
	}

	if loaded.LastAdvanceNotificationSent == nil {
		t.Fatalf("expected LastAdvanceNotificationSent to persist")
	}

	if !loaded.LastAdvanceNotificationSent.Equal(lastNotification) {
		t.Fatalf("expected timestamp %v, got %v", lastNotification, loaded.LastAdvanceNotificationSent)
	}
}

func TestTransitionDetectors(t *testing.T) {
	manager := NewManager("/tmp/unused")

	previous := &State{WasOnCall: false}
	if !manager.HasTransitionToOnCall(previous, true) {
		t.Fatalf("expected transition to on-call")
	}

	previous.WasOnCall = true
	if !manager.HasTransitionToOffCall(previous, false) {
		t.Fatalf("expected transition to off-call")
	}

	if manager.HasTransitionToOnCall(previous, true) {
		t.Fatalf("unexpected transition to on-call")
	}
}

func TestShouldSendAdvanceNotificationWithinWindow(t *testing.T) {
	manager := NewManager("/tmp/unused")
	state := &State{WasOnCall: false}

	shiftStart := time.Now().UTC().Add(30 * time.Minute)
	advance := time.Hour

	if !manager.ShouldSendAdvanceNotification(state, shiftStart, advance) {
		t.Fatalf("expected advance notification to be sent within window")
	}
}

func TestShouldSendAdvanceNotificationOutsideWindow(t *testing.T) {
	manager := NewManager("/tmp/unused")
	state := &State{WasOnCall: false}

	shiftStart := time.Now().UTC().Add(3 * time.Hour)
	advance := 2 * time.Hour

	if manager.ShouldSendAdvanceNotification(state, shiftStart, advance) {
		t.Fatalf("expected advance notification to be skipped outside window")
	}
}

func TestShouldSendAdvanceNotificationSkippedWhenAlreadySent(t *testing.T) {
	manager := NewManager("/tmp/unused")
	sent := time.Now().UTC().Add(-time.Hour)
	state := &State{LastAdvanceNotificationSent: &sent}

	shiftStart := time.Now().UTC().Add(30 * time.Minute)
	advance := time.Hour

	if manager.ShouldSendAdvanceNotification(state, shiftStart, advance) {
		t.Fatalf("expected advance notification to be skipped when already sent recently")
	}
}

func TestRecordAdvanceNotificationSent(t *testing.T) {
	manager := NewManager("/tmp/unused")
	state := &State{}

	before := time.Now().UTC()
	manager.RecordAdvanceNotificationSent(state)
	after := time.Now().UTC()

	if state.LastAdvanceNotificationSent == nil {
		t.Fatalf("expected timestamp to be recorded")
	}

	recorded := *state.LastAdvanceNotificationSent
	if recorded.Before(before) || recorded.After(after) {
		t.Fatalf("expected timestamp between %v and %v, got %v", before, after, recorded)
	}
}
