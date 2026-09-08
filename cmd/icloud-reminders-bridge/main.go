package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pdfowler/fruit-forwarder/internal/config"
	"github.com/pdfowler/fruit-forwarder/internal/hasync"
	"github.com/pdfowler/fruit-forwarder/internal/keychain"
	"github.com/pdfowler/fruit-forwarder/internal/mcpserver"
	"github.com/pdfowler/fruit-forwarder/internal/reminderstore"
	"github.com/pdfowler/fruit-forwarder/internal/state"
)

// version is injected by release builds; development builds report dev.
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usageError()
	}
	command := os.Args[1]
	if command == "version" {
		fmt.Println(version)
		return nil
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	configPath := flags.String("config", config.DefaultPath(), "path to bridge config")
	eventKitHelper := flags.String("eventkit-helper", "", "override the EventKit helper executable path")
	jsonOutput := flags.Bool("json", false, "format status or doctor output as JSON")
	commandID := flags.String("command-id", "", "in-flight command identifier for recover")
	resolution := flags.String("resolution", "", "recover resolution: applied or retry")
	queueEpoch := flags.String("queue-epoch", "", "Home Assistant queue epoch to accept after reviewing a restore")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if command == "discover" || command == "discover-calendars" {
		helperPath, err := discoveryHelperPath(*configPath, *eventKitHelper)
		if err != nil {
			return err
		}
		if command == "discover-calendars" {
			calendars, err := reminderstore.DiscoverCalendars(context.Background(), helperPath)
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(calendars)
		}
		return discover(helperPath)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *eventKitHelper != "" {
		cfg.EventKitHelper = *eventKitHelper
	}
	if command == "status" || command == "doctor" {
		return reportStatus(cfg, *configPath, command == "doctor", *jsonOutput)
	}
	if command == "recover" {
		return recoverCommand(cfg, *commandID, *resolution)
	}
	if command == "reset-queue-epoch" {
		return resetQueueEpoch(cfg, *queueEpoch)
	}
	if command == "pair" {
		return pair(cfg)
	}
	if command == "check-config" || command == "sync-once" || command == "serve" {
		if err := cfg.ValidateSync(); err != nil {
			return err
		}
	}
	if command == "check-config" {
		fmt.Println("Home Assistant sync configuration is valid (connectivity and permissions not tested).")
		return nil
	}
	store, err := reminderstore.New(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	switch command {
	case "mcp":
		lockedStore := reminderstore.NewLocked(store, cfg.StatePath+".lock")
		return mcpserver.RunWithVersion(ctx, lockedStore, cfg.MCPReadOnly, version)
	case "sync-once", "serve":
		if cfg.HomeAssistantURL == "" {
			return errors.New("home_assistant_url is required for Home Assistant sync")
		}
		token, err := keychain.Load(cfg.KeychainService, cfg.KeychainAccount)
		if err != nil {
			return err
		}
		bridgeState, err := state.Load(cfg.StatePath)
		if err != nil {
			return err
		}
		releaseLock, err := state.Acquire(cfg.StatePath + ".lock")
		if err != nil {
			return err
		}
		defer releaseLock()
		logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		client := hasync.NewWithVersion(cfg, token, store, bridgeState, logger, version)
		if command == "sync-once" {
			_, err = client.SyncOnce(ctx)
			return err
		}
		return client.Run(ctx)
	default:
		return usageError()
	}
}

type statusReport struct {
	Version                 string `json:"version"`
	ConfigPath              string `json:"config_path"`
	BridgeID                string `json:"bridge_id"`
	HomeAssistantEnabled    bool   `json:"home_assistant_enabled"`
	EventKitHelper          string `json:"eventkit_helper"`
	EventKitReady           bool   `json:"eventkit_ready"`
	EventKitRemindersAccess string `json:"eventkit_reminders_access,omitempty"`
	EventKitCalendarsAccess string `json:"eventkit_calendars_access,omitempty"`
	EventKitAccessError     string `json:"eventkit_access_error,omitempty"`
	KeychainReady           *bool  `json:"keychain_ready,omitempty"`
	StatePath               string `json:"state_path"`
	StateReady              bool   `json:"state_ready"`
	StateLockStatus         string `json:"state_lock_status"`
	QueueEpoch              string `json:"queue_epoch,omitempty"`
	InFlightCommandID       string `json:"in_flight_command_id,omitempty"`
}

func reportStatus(cfg *config.Config, configPath string, deep, jsonOutput bool) error {
	report := statusReport{
		Version:              version,
		ConfigPath:           configPath,
		BridgeID:             cfg.BridgeID,
		HomeAssistantEnabled: cfg.HomeAssistantURL != "",
		EventKitHelper:       cfg.EventKitPath(),
		EventKitReady:        reminderstore.ValidateHelperPath(cfg.EventKitPath()) == nil,
		StatePath:            cfg.StatePath,
		StateReady:           true,
		StateLockStatus:      "unknown",
	}
	bridgeState, stateErr := state.Load(cfg.StatePath)
	if stateErr != nil {
		report.StateReady = false
	} else {
		report.QueueEpoch = bridgeState.QueueEpoch
		if bridgeState.InFlight != nil {
			report.InFlightCommandID = bridgeState.InFlight.ID
		}
	}
	if lockStatus, lockErr := state.Probe(cfg.StatePath + ".lock"); lockErr != nil {
		report.StateLockStatus = "error"
	} else {
		report.StateLockStatus = lockStatus
	}
	if deep && cfg.HomeAssistantURL != "" {
		_, keychainErr := keychain.Load(cfg.KeychainService, cfg.KeychainAccount)
		ready := keychainErr == nil
		report.KeychainReady = &ready
	}
	if deep && (len(cfg.Lists) > 0 || len(cfg.Calendars) > 0) {
		access, accessErr := reminderstore.Authorization(context.Background(), cfg.EventKitPath())
		if accessErr != nil {
			report.EventKitAccessError = accessErr.Error()
		} else {
			report.EventKitRemindersAccess = access.Reminders
			report.EventKitCalendarsAccess = access.Calendars
		}
	}
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		fmt.Printf("Fruit Forwarder %s\n", report.Version)
		fmt.Printf("  configuration: %s\n", report.ConfigPath)
		fmt.Printf("  EventKit helper: %s (%s)\n", report.EventKitHelper, readiness(report.EventKitReady))
		if report.EventKitRemindersAccess != "" {
			fmt.Printf("  Reminders access: %s\n", report.EventKitRemindersAccess)
		}
		if report.EventKitCalendarsAccess != "" {
			fmt.Printf("  Calendar access: %s\n", report.EventKitCalendarsAccess)
		}
		if report.EventKitAccessError != "" {
			fmt.Printf("  EventKit access check: %s\n", report.EventKitAccessError)
		}
		fmt.Printf("  Home Assistant sync: %s\n", readiness(report.HomeAssistantEnabled))
		if report.KeychainReady != nil {
			fmt.Printf("  pairing Keychain item: %s\n", readiness(*report.KeychainReady))
		}
		fmt.Printf("  acknowledgement state: %s\n", readiness(report.StateReady))
		fmt.Printf("  bridge state lock: %s\n", report.StateLockStatus)
		if report.QueueEpoch != "" {
			fmt.Printf("  Home Assistant queue epoch: %s\n", report.QueueEpoch)
		}
		if report.InFlightCommandID != "" {
			fmt.Printf("  uncertain command: %s (run recover with an explicit resolution)\n", report.InFlightCommandID)
		}
	}
	if !report.EventKitReady {
		return errors.New("doctor: EventKit helper is missing, unsafe, or not executable")
	}
	if deep && report.KeychainReady != nil && !*report.KeychainReady {
		return errors.New("doctor: Home Assistant pairing token is not available in Keychain")
	}
	if deep && report.EventKitAccessError != "" {
		return fmt.Errorf("doctor: EventKit access check failed: %s", report.EventKitAccessError)
	}
	if deep && len(cfg.Lists) > 0 && report.EventKitRemindersAccess != "full_access" && report.EventKitRemindersAccess != "authorized" {
		return fmt.Errorf("doctor: Reminders access is %s; grant Full Access in System Settings", report.EventKitRemindersAccess)
	}
	if deep && len(cfg.Calendars) > 0 && report.EventKitCalendarsAccess != "full_access" && report.EventKitCalendarsAccess != "authorized" {
		return fmt.Errorf("doctor: Calendar access is %s; grant Full Access in System Settings", report.EventKitCalendarsAccess)
	}
	if !report.StateReady {
		return errors.New("doctor: acknowledgement state is unreadable")
	}
	if deep && report.StateLockStatus == "error" {
		return errors.New("doctor: bridge state lock cannot be probed safely")
	}
	if report.InFlightCommandID != "" {
		return fmt.Errorf("doctor: command %s has an uncertain outcome; resolve it before syncing", report.InFlightCommandID)
	}
	return nil
}

func recoverCommand(cfg *config.Config, commandID, resolution string) error {
	if commandID == "" {
		return errors.New("recover requires --command-id")
	}
	if resolution != "applied" && resolution != "retry" {
		return errors.New("recover requires --resolution applied or retry")
	}
	releaseLock, err := state.Acquire(cfg.StatePath + ".lock")
	if err != nil {
		return fmt.Errorf("acquire bridge state lock: %w", err)
	}
	defer releaseLock()
	bridgeState, err := state.Load(cfg.StatePath)
	if err != nil {
		return fmt.Errorf("load acknowledgement state: %w", err)
	}
	if bridgeState.InFlight == nil || bridgeState.InFlight.ID != commandID {
		return fmt.Errorf("no matching in-flight command %q", commandID)
	}
	bridgeState.InFlight = nil
	if resolution == "applied" {
		bridgeState.MarkApplied(commandID)
	}
	if err := bridgeState.Save(cfg.StatePath); err != nil {
		return fmt.Errorf("save acknowledgement state: %w", err)
	}
	if resolution == "applied" {
		fmt.Printf("Marked uncertain command %s as applied; it will not be retried.\n", commandID)
	} else {
		fmt.Printf("Cleared uncertain command %s; it may be retried on the next sync.\n", commandID)
	}
	return nil
}

func resetQueueEpoch(cfg *config.Config, queueEpoch string) error {
	if strings.TrimSpace(queueEpoch) == "" || len(queueEpoch) > 128 {
		return errors.New("reset-queue-epoch requires a non-empty queue epoch of at most 128 characters")
	}
	releaseLock, err := state.Acquire(cfg.StatePath + ".lock")
	if err != nil {
		return fmt.Errorf("acquire bridge state lock: %w", err)
	}
	defer releaseLock()
	bridgeState, err := state.Load(cfg.StatePath)
	if err != nil {
		return fmt.Errorf("load acknowledgement state: %w", err)
	}
	if bridgeState.InFlight != nil {
		return fmt.Errorf("cannot reset queue epoch while command %s has an uncertain outcome", bridgeState.InFlight.ID)
	}
	bridgeState.QueueEpoch = queueEpoch
	if err := bridgeState.Save(cfg.StatePath); err != nil {
		return fmt.Errorf("save queue epoch: %w", err)
	}
	fmt.Printf("Accepted Home Assistant queue epoch %s after explicit review.\n", queueEpoch)
	return nil
}

func readiness(value bool) string {
	if value {
		return "ready"
	}
	return "not ready"
}

func discover(helperPath string) error {
	lists, err := reminderstore.Discover(context.Background(), helperPath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(lists)
}

func discoveryHelperPath(configPath, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	cfg, err := config.Load(configPath)
	if err == nil {
		return cfg.EventKitPath(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.DefaultEventKitHelperPath(), nil
	}
	return "", fmt.Errorf("load config for discovery: %w", err)
}

func pair(cfg *config.Config) error {
	token, err := keychain.GenerateToken()
	if err != nil {
		return err
	}
	if err := keychain.Store(cfg.KeychainService, cfg.KeychainAccount, token); err != nil {
		return err
	}
	clipboard := exec.Command("/usr/bin/pbcopy")
	stdin, err := clipboard.StdinPipe()
	if err != nil {
		return err
	}
	if err := clipboard.Start(); err != nil {
		return err
	}
	if _, err := fmt.Fprint(stdin, token); err != nil {
		return err
	}
	if err := stdin.Close(); err != nil {
		return err
	}
	if err := clipboard.Wait(); err != nil {
		return err
	}
	fmt.Println("Pairing token stored in the login Keychain and copied to the clipboard.")
	fmt.Println("In Home Assistant, add iCloud Reminders Bridge and paste it into Pairing token.")
	return nil
}

func usageError() error {
	return errors.New("usage: icloud-reminders-bridge <version|discover|discover-calendars|pair|status|doctor|recover|reset-queue-epoch|check-config|sync-once|serve|mcp> [--config path] [--eventkit-helper path] [--json] [--command-id id] [--resolution applied|retry] [--queue-epoch id]")
}
