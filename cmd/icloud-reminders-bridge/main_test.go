package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
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
	}{
		{"read_only", true, false},
		{"writable", false, false},
		{"helper_unavailable", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := filepath.Join(t.TempDir(), "synthetic-helper")
			script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '{\"lists\":[{\"id\":\"allowed\",\"name\":\"Tasks\",\"read_only\":false,\"items\":[{\"uid\":\"one\",\"summary\":\"Synthetic task\",\"status\":\"needs_action\"}]}]}'\n"
			if tc.helperFails {
				script = "#!/bin/sh\ncat >/dev/null\nexit 1\n"
			}
			if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(t.TempDir(), "config.json")
			config, err := json.Marshal(map[string]any{
				"bridge_id": "test", "mcp_read_only": tc.readOnly,
				"eventkit_helper_path": helper,
				"lists":                []map[string]string{{"id": "allowed", "name": "Tasks"}},
			})
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
			if len(catalog.Tools) != wantTools {
				t.Fatalf("tool count = %d, want %d", len(catalog.Tools), wantTools)
			}
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
