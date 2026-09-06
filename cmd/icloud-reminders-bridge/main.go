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
	"syscall"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/hasync"
	"github.com/pdfowler/icloud-reminders-bridge/internal/keychain"
	"github.com/pdfowler/icloud-reminders-bridge/internal/mcpserver"
	"github.com/pdfowler/icloud-reminders-bridge/internal/reminderstore"
	"github.com/pdfowler/icloud-reminders-bridge/internal/state"
)

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
	if command == "discover-calendars" {
		calendars, err := reminderstore.DiscoverCalendars(context.Background(), config.DefaultEventKitHelperPath())
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(calendars)
	}
	if command == "discover" {
		return discover()
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	configPath := flags.String("config", config.DefaultPath(), "path to bridge config")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
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
		return mcpserver.Run(ctx, store, cfg.MCPReadOnly)
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
		logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
		client := hasync.New(cfg, token, store, bridgeState, logger)
		if command == "sync-once" {
			_, err = client.SyncOnce(ctx)
			return err
		}
		return client.Run(ctx)
	default:
		return usageError()
	}
}

func discover() error {
	lists, err := reminderstore.Discover(context.Background(), config.DefaultEventKitHelperPath())
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(lists)
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
	return errors.New("usage: icloud-reminders-bridge <discover|discover-calendars|pair|check-config|sync-once|serve|mcp> [--config path]")
}
