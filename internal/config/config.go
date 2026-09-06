package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type List struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Config struct {
	MCPReadOnly        bool   `json:"mcp_read_only"`
	BridgeID           string `json:"bridge_id"`
	HomeAssistantURL   string `json:"home_assistant_url"`
	PollInterval       string `json:"poll_interval"`
	KeychainService    string `json:"keychain_service"`
	KeychainAccount    string `json:"keychain_account"`
	StatePath          string `json:"state_path"`
	EventKitHelper     string `json:"eventkit_helper_path,omitempty"`
	CompletedRetention string `json:"completed_retention"`
	Lists              []List `json:"lists"`
	Calendars          []List `json:"calendars,omitempty"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "icloud-reminders-bridge", "config.json")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.BridgeID) == "" {
		return errors.New("bridge_id is required")
	}
	if c.HomeAssistantURL != "" {
		u, err := url.Parse(c.HomeAssistantURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("home_assistant_url must be an absolute HTTPS URL without credentials, query, or fragment")
		}
	}
	if c.KeychainService == "" {
		c.KeychainService = "com.example.icloud-reminders-bridge"
	}
	if c.KeychainAccount == "" {
		c.KeychainAccount = c.BridgeID
	}
	if c.PollInterval == "" {
		c.PollInterval = "30s"
	}
	interval, err := time.ParseDuration(c.PollInterval)
	if err != nil || interval < 10*time.Second {
		return errors.New("poll_interval must be a duration of at least 10s")
	}
	if c.CompletedRetention == "" {
		c.CompletedRetention = "720h"
	}
	retention, err := time.ParseDuration(c.CompletedRetention)
	if err != nil || retention < 0 {
		return errors.New("completed_retention must be a non-negative duration")
	}
	if c.StatePath == "" {
		home, _ := os.UserHomeDir()
		c.StatePath = filepath.Join(home, "Library", "Application Support", "icloud-reminders-bridge", "state.json")
	}
	seenIDs := make(map[string]struct{}, len(c.Lists))
	calendarIDs := make(map[string]bool)
	if len(c.Calendars) > 100 {
		return errors.New("at most 100 calendars may be configured")
	}
	for _, calendar := range c.Calendars {
		if strings.TrimSpace(calendar.ID) == "" || strings.TrimSpace(calendar.Name) == "" || calendarIDs[calendar.ID] {
			return errors.New("calendars require unique non-empty IDs and non-empty names")
		}
		calendarIDs[calendar.ID] = true
	}
	seenNames := make(map[string]string, len(c.Lists))
	for _, list := range c.Lists {
		if strings.TrimSpace(list.ID) == "" || strings.TrimSpace(list.Name) == "" {
			return errors.New("each list requires id and name")
		}
		if _, exists := seenIDs[list.ID]; exists {
			return fmt.Errorf("duplicate list id %q", list.ID)
		}
		seenIDs[list.ID] = struct{}{}
		folded := strings.ToLower(list.Name)
		if otherID, exists := seenNames[folded]; exists && otherID != list.ID {
			return fmt.Errorf("duplicate list name %q is unsafe for EventKit creates", list.Name)
		}
		seenNames[folded] = list.ID
	}
	return nil
}

// ValidateSync checks requirements that do not apply to discovery or local MCP.
func (c *Config) ValidateSync() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.HomeAssistantURL == "" {
		return errors.New("home_assistant_url is required for Home Assistant sync")
	}
	u, _ := url.Parse(c.HomeAssistantURL)
	if strings.HasSuffix(u.Hostname(), ".invalid") {
		return errors.New("replace the example home_assistant_url before syncing")
	}
	if len(c.Lists) == 0 {
		return errors.New("configure at least one allowlisted reminder list before syncing")
	}
	return nil
}

func (c *Config) Interval() time.Duration {
	d, _ := time.ParseDuration(c.PollInterval)
	return d
}

func (c *Config) CompletedRetentionDuration() time.Duration {
	d, _ := time.ParseDuration(c.CompletedRetention)
	return d
}

func (c *Config) AllowedIDs() map[string]List {
	result := make(map[string]List, len(c.Lists))
	for _, list := range c.Lists {
		result[list.ID] = list
	}
	return result
}

func DefaultBinDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "icloud-reminders-bridge", "bin")
}

func DefaultEventKitHelperPath() string {
	return filepath.Join(DefaultBinDir(), "icloud-reminders-eventkit")
}

func (c *Config) EventKitPath() string {
	if c.EventKitHelper != "" {
		return c.EventKitHelper
	}
	return DefaultEventKitHelperPath()
}
