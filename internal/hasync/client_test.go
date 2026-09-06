package hasync

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
	"github.com/pdfowler/icloud-reminders-bridge/internal/state"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection unavailable")
}

func TestTransportFailureDoesNotExposeWebhookToken(t *testing.T) {
	const token = "test-secret-must-never-appear-in-logs"
	client := New(&config.Config{BridgeID: "mac", HomeAssistantURL: "https://home.example"}, token,
		&fakeReminderStore{}, &state.State{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client.httpClient.Transport = failingTransport{}
	_, err := client.SyncOnce(context.Background())
	if err == nil || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "/api/webhook/") {
		t.Fatalf("transport failure was missing or exposed a credential: %v", err)
	}
	if !strings.Contains(err.Error(), "connection unavailable") {
		t.Fatalf("lost useful failure reason: %v", err)
	}
}

type fakeReminderStore struct {
	commands []model.Command
	items    []model.List
}

func (s *fakeReminderStore) Snapshot(context.Context) ([]model.List, error) {
	return s.items, nil
}

func (s *fakeReminderStore) ApplyCommand(_ context.Context, command model.Command) error {
	s.commands = append(s.commands, command)
	return nil
}

func TestSyncOnceUsesScopedWebhookAndAcknowledgesCommands(t *testing.T) {
	t.Parallel()
	const token = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/api/webhook/"+token {
			t.Errorf("unexpected webhook path %q", r.URL.Path)
		}
		var snapshot model.Snapshot
		if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
			t.Error(err)
		}
		if snapshot.BridgeID != "mac" || len(snapshot.Lists) != 1 {
			t.Errorf("unexpected snapshot: %#v", snapshot)
		}
		_ = json.NewEncoder(w).Encode(model.SyncResponse{
			Version:  model.ProtocolVersion,
			Commands: []model.Command{{ID: "cmd-1", Action: "complete", ListID: "list-1", Item: model.Item{UID: "item-1"}}},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		BridgeID: "mac", HomeAssistantURL: server.URL, PollInterval: "30s",
		StatePath: filepath.Join(t.TempDir(), "state.json"),
	}
	store := &fakeReminderStore{items: []model.List{{ID: "list-1", Name: "Example List"}}}
	bridgeState := &state.State{}
	client := New(cfg, token, store, bridgeState, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client.httpClient = server.Client()
	applied, err := client.SyncOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 || len(store.commands) != 1 || requestCount != 2 {
		t.Fatalf("applied=%d commands=%d requests=%d", applied, len(store.commands), requestCount)
	}
	if !bridgeState.Has("cmd-1") {
		t.Fatal("command acknowledgement was not persisted")
	}
}

func TestSyncOnceDoesNotReapplyPersistedCommand(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(model.SyncResponse{
			Version:  model.ProtocolVersion,
			Commands: []model.Command{{ID: "cmd-1", Action: "complete"}},
		})
	}))
	defer server.Close()
	cfg := &config.Config{BridgeID: "mac", HomeAssistantURL: server.URL, PollInterval: "30s", StatePath: filepath.Join(t.TempDir(), "state.json")}
	store := &fakeReminderStore{}
	bridgeState := &state.State{AppliedCommandIDs: []string{"cmd-1"}}
	client := New(cfg, "token", store, bridgeState, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client.httpClient = server.Client()
	applied, err := client.SyncOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 || len(store.commands) != 0 {
		t.Fatalf("persisted command was reapplied")
	}
}
