package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pdfowler/fruit-forwarder/internal/config"
	"github.com/pdfowler/fruit-forwarder/internal/model"
	"github.com/pdfowler/fruit-forwarder/internal/state"
)

// This crosses the real CLI/stdio/helper-process boundary, using synthetic
// reminder data. It never launches EventKit or requests Apple permissions.
func TestStdioProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	binary := filepath.Join(dir, "bridge")
	build := exec.CommandContext(ctx, "go", "build", "-race", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build bridge: %v\n%s", err, output)
	}
	for _, tc := range []struct {
		name        string
		readOnly    bool
		helperFails bool
		calendar    bool
	}{
		{"read_only", true, false, false},
		{"writable", false, false, false},
		{"helper_unavailable", true, true, false},
		{"calendar_only", true, false, true},
		{"combined", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := filepath.Join(t.TempDir(), "synthetic-helper")
			script := "#!/bin/sh\npayload=$(cat)\nif printf '%s' \"$payload\" | grep -q '\"action\":\"calendar_snapshot\"'; then\n  printf '%s\\n' '{\"calendars\":[{\"id\":\"events\",\"name\":\"Events\",\"events\":[{\"uid\":\"event-1\",\"summary\":\"Synthetic event\",\"start\":\"2026-01-01T10:00:00Z\",\"end\":\"2026-01-01T11:00:00Z\",\"all_day\":false,\"time_zone\":\"UTC\"}]}]}'\n"
			script += "elif printf '%s' \"$payload\" | grep -q '\"action\":\"calendars\"'; then\n  printf '%s\\n' '{\"calendars\":[{\"id\":\"events\",\"name\":\"Events\",\"events\":[]}]}'\n"
			script += "else\n  printf '%s\\n' '{\"lists\":[{\"id\":\"allowed\",\"name\":\"Tasks\",\"read_only\":false,\"items\":[{\"uid\":\"one\",\"summary\":\"Synthetic task\",\"status\":\"needs_action\"}]}]}'\nfi\n"
			if tc.helperFails {
				script = "#!/bin/sh\ncat >/dev/null\nexit 1\n"
			}
			if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(t.TempDir(), "config.json")
			configValues := map[string]any{
				"bridge_id": "test", "mcp_read_only": tc.readOnly,
				"eventkit_helper_path": helper,
				"lists":                []map[string]string{{"id": "allowed", "name": "Tasks"}},
			}
			if tc.calendar {
				configValues["calendars"] = []map[string]string{{"id": "events", "name": "Events"}}
				if tc.name == "calendar_only" {
					configValues["lists"] = []map[string]string{}
				}
			}
			config, err := json.Marshal(configValues)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, config, 0600); err != nil {
				t.Fatal(err)
			}
			command := exec.CommandContext(ctx, binary, "mcp", "--config", configPath)
			client := mcp.NewClient(&mcp.Implementation{Name: "process-test", Version: "1"}, nil)
			session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			catalog, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			wantTools := 6
			if tc.readOnly {
				wantTools = 2
			}
			if tc.calendar {
				wantTools += 2
			}
			if len(catalog.Tools) != wantTools {
				t.Fatalf("tool count = %d, want %d", len(catalog.Tools), wantTools)
			}
			if tc.calendar {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "calendar_events", Arguments: map[string]any{
					"calendar_id": "events", "start": "2026-01-01T00:00:00Z", "end": "2026-01-02T00:00:00Z",
				}})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError != tc.helperFails {
					t.Fatalf("calendar tool error = %v, want %v", result.IsError, tc.helperFails)
				}
				if !tc.helperFails {
					data, err := json.Marshal(result.StructuredContent)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(data), "event-1") {
						t.Fatalf("unexpected calendar events: %s", data)
					}
				}
			} else {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "reminders_list", Arguments: map[string]any{"list_id": "allowed"}})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError != tc.helperFails {
					t.Fatalf("tool error = %v, want %v", result.IsError, tc.helperFails)
				}
				if !tc.helperFails {
					data, err := json.Marshal(result.StructuredContent)
					if err != nil {
						t.Fatal(err)
					}
					var output struct {
						Items []struct {
							UID string `json:"uid"`
						} `json:"items"`
					}
					if err := json.Unmarshal(data, &output); err != nil {
						t.Fatal(err)
					}
					if len(output.Items) != 1 || output.Items[0].UID != "one" {
						t.Fatalf("unexpected items: %s", data)
					}
				}
			}
			if tc.readOnly {
				res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "reminders_complete", Arguments: map[string]any{"list_id": "allowed", "uid": "one"}})
				if err == nil && !res.IsError {
					t.Fatal("read-only process allowed mutation")
				}
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDiscoveryUsesConfiguredHelperPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	helperPath := filepath.Join(dir, "configured-helper")
	configData := []byte(`{"bridge_id":"mac","eventkit_helper_path":"` + helperPath + `","lists":[]}`)
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := discoveryHelperPath(configPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != helperPath {
		t.Fatalf("helper path = %q, want %q", got, helperPath)
	}
	if got, err := discoveryHelperPath(filepath.Join(dir, "missing.json"), ""); err != nil || got != config.DefaultEventKitHelperPath() {
		t.Fatalf("missing config discovery = %q, %v", got, err)
	}
}

func TestRecoverCommandRequiresAndAppliesExplicitResolution(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := (&state.State{InFlight: &model.Command{
		ID: "command-1", Action: "create", ListID: "list-1",
		Item: model.Item{Summary: "Task"},
	}}).Save(path); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{StatePath: path}
	if err := recoverCommand(cfg, "command-1", "applied"); err != nil {
		t.Fatal(err)
	}
	loaded, err := state.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.InFlight != nil || !loaded.Has("command-1") {
		t.Fatalf("applied resolution did not update state: %#v", loaded)
	}
}

func TestRecoverCommandRejectsMissingResolution(t *testing.T) {
	cfg := &config.Config{StatePath: filepath.Join(t.TempDir(), "state.json")}
	if err := recoverCommand(cfg, "command-1", ""); err == nil {
		t.Fatal("recover accepted an implicit resolution")
	}
}

func TestResetQueueEpochRequiresExplicitSafeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := (&state.State{}).Save(path); err != nil {
		t.Fatal(err)
	}
	if err := resetQueueEpoch(&config.Config{StatePath: path}, "epoch-two"); err != nil {
		t.Fatal(err)
	}
	loaded, err := state.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.QueueEpoch != "epoch-two" {
		t.Fatalf("queue epoch = %q, want epoch-two", loaded.QueueEpoch)
	}
	if err := resetQueueEpoch(&config.Config{StatePath: path}, ""); err == nil {
		t.Fatal("empty queue epoch accepted")
	}
}

func TestResetQueueEpochRejectsUncertainCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := (&state.State{InFlight: &model.Command{
		ID: "command-uncertain", Action: "complete", ListID: "list", Item: model.Item{UID: "item"},
	}}).Save(path); err != nil {
		t.Fatal(err)
	}
	if err := resetQueueEpoch(&config.Config{StatePath: path}, "epoch-two"); err == nil || !strings.Contains(err.Error(), "uncertain") {
		t.Fatalf("uncertain state was reset: %v", err)
	}
}

func TestReportStatusAcceptsReadyConfiguration(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "eventkit-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		BridgeID:       "mac",
		EventKitHelper: helper,
		StatePath:      filepath.Join(dir, "state.json"),
	}
	if err := reportStatus(cfg, filepath.Join(dir, "config.json"), false, true); err != nil {
		t.Fatalf("ready status returned an error: %v", err)
	}
}

func TestReportStatusRejectsUnsafeHelper(t *testing.T) {
	cfg := &config.Config{
		BridgeID:       "mac",
		EventKitHelper: filepath.Join(t.TempDir(), "missing-helper"),
		StatePath:      filepath.Join(t.TempDir(), "state.json"),
	}
	if err := reportStatus(cfg, "config.json", false, true); err == nil || !strings.Contains(err.Error(), "helper") {
		t.Fatalf("missing helper was not rejected: %v", err)
	}
}

func TestReportStatusBlocksUncertainCommand(t *testing.T) {
	dir := t.TempDir()
	helper := filepath.Join(dir, "eventkit-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dir, "state.json")
	if err := (&state.State{InFlight: &model.Command{
		ID: "command-uncertain", Action: "complete", ListID: "list", Item: model.Item{UID: "item"},
	}}).Save(statePath); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{BridgeID: "mac", EventKitHelper: helper, StatePath: statePath}
	if err := reportStatus(cfg, "config.json", false, true); err == nil || !strings.Contains(err.Error(), "uncertain") {
		t.Fatalf("uncertain command was not surfaced: %v", err)
	}
}
