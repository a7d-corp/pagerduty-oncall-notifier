package notifier

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type ntfyRequestCapture struct {
	path    string
	body    string
	headers http.Header
}

func TestNtfyNotifierSendsShiftStartedEvent(t *testing.T) {
	t.Parallel()

	captures := make(chan ntfyRequestCapture, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		captures <- ntfyRequestCapture{
			path:    r.URL.Path,
			body:    string(payload),
			headers: r.Header.Clone(),
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNtfyNotifier(server.URL, "alerts", "secret-key")
	notifier.client = server.Client()

	shiftStart := time.Now().UTC()
	if err := notifier.NotifyWithEvent(EventShiftStarted, shiftStart); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case capture := <-captures:
		if capture.path != "/alerts" {
			t.Fatalf("unexpected path: %s", capture.path)
		}
		expectedMessage := "🚨 Your PagerDuty on-call shift has started!"
		if capture.body != expectedMessage {
			t.Fatalf("unexpected body: %q", capture.body)
		}
		if got := capture.headers.Get("Title"); got != "PagerDuty On-Call Shift Started" {
			t.Fatalf("unexpected Title header: %s", got)
		}
		if got := capture.headers.Get("Priority"); got != "urgent" {
			t.Fatalf("unexpected Priority header: %s", got)
		}
		if got := capture.headers.Get("Tags"); got != "rotating_light,alarm_clock" {
			t.Fatalf("unexpected Tags header: %s", got)
		}
		if got := capture.headers.Get("Authorization"); got != "Bearer secret-key" {
			t.Fatalf("unexpected Authorization header: %s", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("did not receive request")
	}
}

func TestNtfyNotifierPropagatesHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewNtfyNotifier(server.URL, "alerts", "")
	notifier.client = server.Client()

	err := notifier.NotifyWithEvent(EventShiftStarted, time.Now().UTC())
	if err == nil {
		t.Fatalf("expected error when server returns non-2xx status")
	}
}
