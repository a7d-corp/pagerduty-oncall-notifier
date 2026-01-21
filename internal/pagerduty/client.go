package pagerduty

import (
	"context"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// Client wraps the PagerDuty API client
type Client struct {
	client   *pagerduty.Client
	scheduleID string
	userID     string
}

// NewClient creates a new PagerDuty client
func NewClient(apiToken, scheduleID, userID string) *Client {
	client := pagerduty.NewClient(apiToken)
	return &Client{
		client:     client,
		scheduleID: scheduleID,
		userID:     userID,
	}
}

// IsOnCall checks if the configured user is currently on-call for the configured schedule
func (c *Client) IsOnCall(ctx context.Context) (bool, error) {
	opts := pagerduty.ListOnCallOptions{
		ScheduleIDs: []string{c.scheduleID},
	}

	response, err := c.client.ListOnCallsWithContext(ctx, opts)
	if err != nil {
		return false, fmt.Errorf("failed to fetch on-call status: %w", err)
	}

	// Check if any of the on-call entries match our user ID
	for _, oncall := range response.OnCalls {
		if oncall.User.ID == c.userID {
			return true, nil
		}
	}

	return false, nil
}

// UpcomingShift represents information about an upcoming on-call shift
type UpcomingShift struct {
	StartTime time.Time
	EndTime   time.Time
}

// GetUpcomingShift returns the next upcoming shift for the configured user
// Returns nil if no upcoming shift is found
func (c *Client) GetUpcomingShift(ctx context.Context) (*UpcomingShift, error) {
	// Get current time and look ahead for upcoming shifts
	now := time.Now().UTC()
	future := now.AddDate(0, 0, 7)
	opts := pagerduty.ListOnCallOptions{
		ScheduleIDs: []string{c.scheduleID},
		Since:       now.Format(time.RFC3339),
		Until:       future.Format(time.RFC3339),
	}

	response, err := c.client.ListOnCallsWithContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming shifts: %w", err)
	}

	// Find the next shift for our user
	var nextShift *UpcomingShift
	for _, oncall := range response.OnCalls {
		// Parse the start time from string
		startTime, err := time.Parse(time.RFC3339, oncall.Start)
		if err != nil {
			continue // Skip this entry if we can't parse the time
		}

		if oncall.User.ID == c.userID && startTime.After(now) {
			// Parse the end time from string
			endTime, err := time.Parse(time.RFC3339, oncall.End)
			if err != nil {
				continue // Skip this entry if we can't parse the time
			}

			shift := &UpcomingShift{
				StartTime: startTime,
				EndTime:   endTime,
			}
			// Keep the earliest upcoming shift
			if nextShift == nil || shift.StartTime.Before(nextShift.StartTime) {
				nextShift = shift
			}
		}
	}

	return nextShift, nil
}
