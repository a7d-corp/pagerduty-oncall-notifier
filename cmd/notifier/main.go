package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a7d-corp/pagerduty-oncall-notifier/internal/config"
	"github.com/a7d-corp/pagerduty-oncall-notifier/internal/notifier"
	"github.com/a7d-corp/pagerduty-oncall-notifier/internal/pagerduty"
	"github.com/a7d-corp/pagerduty-oncall-notifier/internal/state"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "PagerDuty On-Call Notifier\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  notifier [flags]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "\nKey environment variables:")
		fmt.Fprintln(flag.CommandLine.Output(), "  PD_API_TOKEN (required)        PagerDuty REST API token")
		fmt.Fprintln(flag.CommandLine.Output(), "  PD_SCHEDULE_ID (required)      PagerDuty schedule to monitor")
		fmt.Fprintln(flag.CommandLine.Output(), "  PD_USER_ID (required)          PagerDuty user expected to be on call")
		fmt.Fprintln(flag.CommandLine.Output(), "  NOTIFICATION_BACKEND           webhook | ntfy | pushover")
		fmt.Fprintln(flag.CommandLine.Output(), "  CHECK_INTERVAL                 poll interval in seconds (default 300)")
		fmt.Fprintln(flag.CommandLine.Output(), "  ADVANCE_NOTIFICATION_TIME      duration before shift for advance alerts")
		fmt.Fprintln(flag.CommandLine.Output(), "  STATE_FILE_PATH                path for persisted state (default /data/state.json)")
		fmt.Fprintln(flag.CommandLine.Output(), "\nSee README.md for full configuration details.")
	}

	help := flag.Bool("help", false, "Show help and exit")
	shortHelp := flag.Bool("h", false, "Show help and exit")
	flag.Parse()

	if *help || *shortHelp {
		flag.Usage()
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("PagerDuty On-Call Notifier starting...")
	log.Printf("Schedule ID: %s", cfg.PagerDutyScheduleID)
	log.Printf("User ID: %s", cfg.PagerDutyUserID)
	log.Printf("Check interval: %v", cfg.CheckInterval)
	log.Printf("Notification backend: %s", cfg.NotificationBackend)

	// Initialize components
	pdClient := pagerduty.NewClient(
		cfg.PagerDutyAPIToken,
		cfg.PagerDutyScheduleID,
		cfg.PagerDutyUserID,
	)

	stateManager := state.NewManager(cfg.StateFilePath)

	// Create notifier based on backend selection
	notifierInstance, err := createNotifier(cfg)
	if err != nil {
		log.Fatalf("Failed to create notifier: %v", err)
	}

	// Send birth message for ntfy notifier
	if ntfyNotifier, ok := notifierInstance.(*notifier.NtfyNotifier); ok {
		log.Println("Sending birth message...")
		if err := ntfyNotifier.SendBirthMessage(); err != nil {
			log.Printf("Failed to send birth message: %v", err)
			// Don't fail startup if birth message fails
		} else {
			log.Println("Birth message sent successfully")
		}
	}

	// Load initial state
	currentState, err := stateManager.Load()
	if err != nil {
		log.Fatalf("Failed to load state: %v", err)
	}
	log.Printf("Initial state: was_on_call=%v", currentState.WasOnCall)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start polling loop in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- runPollingLoop(ctx, pdClient, stateManager, notifierInstance, cfg.CheckInterval, cfg)
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
		// Send will message for ntfy notifier before shutdown
		if ntfyNotifier, ok := notifierInstance.(*notifier.NtfyNotifier); ok {
			log.Println("Sending will message...")
			if err := ntfyNotifier.SendWillMessage(); err != nil {
				log.Printf("Failed to send will message: %v", err)
			} else {
				log.Println("Will message sent successfully")
			}
		}
		cancel()
		<-done
	case err := <-done:
		if err != nil {
			log.Fatalf("Polling loop error: %v", err)
		}
		// Send will message for ntfy notifier on graceful shutdown
		if ntfyNotifier, ok := notifierInstance.(*notifier.NtfyNotifier); ok {
			log.Println("Sending will message...")
			if err := ntfyNotifier.SendWillMessage(); err != nil {
				log.Printf("Failed to send will message: %v", err)
			} else {
				log.Println("Will message sent successfully")
			}
		}
	}

	log.Println("Shutdown complete")
}

// createNotifier creates the appropriate notifier based on the configuration
func createNotifier(cfg *config.Config) (notifier.Notifier, error) {
	switch cfg.NotificationBackend {
	case config.BackendWebhook:
		log.Printf("Using webhook notifier: %s", cfg.NotificationWebhookURL)
		return notifier.NewWebhookNotifier(cfg.NotificationWebhookURL), nil
	case config.BackendNtfy:
		log.Printf("Using ntfy notifier: %s/%s", cfg.NtfyServerURL, cfg.NtfyTopic)
		if cfg.NtfyAPIKey != "" {
			log.Println("Ntfy authentication enabled")
		}
		return notifier.NewNtfyNotifier(cfg.NtfyServerURL, cfg.NtfyTopic, cfg.NtfyAPIKey), nil
	case config.BackendPushover:
		log.Println("Using Pushover notifier")
		if cfg.PushoverDevice != "" {
			log.Printf("Pushover device targeting enabled: %s", cfg.PushoverDevice)
		}
		if cfg.PushoverSound != "" {
			log.Printf("Pushover sound override: %s", cfg.PushoverSound)
		}
		return notifier.NewPushoverNotifier(cfg.PushoverAppToken, cfg.PushoverUserKey, cfg.PushoverDevice, cfg.PushoverSound), nil
	default:
		return nil, fmt.Errorf("unsupported notification backend: %s", cfg.NotificationBackend)
	}
}

func runPollingLoop(
	ctx context.Context,
	pdClient *pagerduty.Client,
	stateManager *state.Manager,
	n notifier.Notifier,
	interval time.Duration,
	cfg *config.Config,
) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Load initial state
	currentState, err := stateManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load initial state: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Check on-call status
			isOnCall, err := pdClient.IsOnCall(ctx)
			if err != nil {
				log.Printf("Error checking on-call status: %v", err)
				continue
			}

			log.Printf("On-call status: %v (previous: %v)", isOnCall, currentState.WasOnCall)

			// Check for upcoming shifts if advance notification is enabled
			if cfg.AdvanceNotificationTime > 0 {
				upcomingShift, err := pdClient.GetUpcomingShift(ctx)
				if err != nil {
					log.Printf("Error checking upcoming shifts: %v", err)
				} else if upcomingShift != nil {
					log.Printf("Upcoming shift found: starts at %v", upcomingShift.StartTime)

					// Check if we should send an advance notification
					if stateManager.ShouldSendAdvanceNotification(currentState, upcomingShift.StartTime, cfg.AdvanceNotificationTime) {
						log.Printf("Sending advance notification for shift starting at %v", upcomingShift.StartTime)

						event := notifier.EventUpcomingShift
						if err := n.NotifyWithEvent(event, upcomingShift.StartTime); err != nil {
							log.Printf("Failed to send advance notification: %v", err)
							// Continue even if notification fails
						} else {
							log.Println("Advance notification sent successfully")
							// Record that we sent the advance notification
							stateManager.RecordAdvanceNotificationSent(currentState)
						}
					} else {
						log.Printf("Advance notification not needed (already sent or not in window)")
					}
				} else {
					log.Printf("No upcoming shifts found")
				}
			}

			// Check for transition to on-call
			if stateManager.HasTransitionToOnCall(currentState, isOnCall) {
				log.Printf("Shift started! Sending notifier...")

				event := notifier.EventShiftStarted
				if err := n.NotifyWithEvent(event, time.Now().UTC()); err != nil {
					log.Printf("Failed to send shift started notification: %v", err)
					// Continue even if notification fails
				} else {
					log.Println("Shift started notification sent successfully")
				}
			}

			// Update state
			currentState.WasOnCall = isOnCall
			if err := stateManager.Save(currentState); err != nil {
				log.Printf("Failed to save state: %v", err)
				// Continue even if state save fails
			}
		}
	}
}
