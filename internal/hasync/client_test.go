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
	"time"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
	"github.com/pdfowler/icloud-reminders-bridge/internal/state"
)

type failingTransport struct{}

type calendarSyncStore struct {
	fakeReminderStore
	called bool
}

func (s *calendarSyncStore) CalendarEvents(_ context.Context, id, start, end string) ([]model.Event, error) {
	s.called = true
	if id != "events" {
		return nil, errors.New("wrong scope")
	}
	a, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, err
	}
	b, err := time.Parse(time.RFC3339, end)
	if err != nil || b.Sub(a) != 120*24*time.Hour {
		return nil, errors.New("wrong window")
	}
	return []model.Event{}, nil
}

type failingCalendarStore struct{ fakeReminderStore }

func (*failingCalendarStore) CalendarEvents(context.Context, string, string, string) ([]model.Event, error) {
	return nil, errors.New("calendar permission unavailable")
}

func TestCalendarSnapshotSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload model.Snapshot
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload.Calendars) != 1 || payload.Calendars[0].ID != "events" || payload.Calendars[0].WindowStart == "" || payload.Calendars[0].WindowEnd == "" {
			t.Errorf("invalid calendar snapshot: %+v", payload.Calendars)
		}
		_ = json.NewEncoder(w).Encode(model.SyncResponse{Version: model.ProtocolVersion})
	}))
	defer server.Close()
	store := &calendarSyncStore{fakeReminderStore: fakeReminderStore{items: []model.List{}}}
	cfg := &config.Config{BridgeID: "mac", HomeAssistantURL: server.URL, Calendars: []config.List{{ID: "events", Name: "Events"}}}
	client := New(cfg, "synthetic", store, &state.State{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client.httpClient = server.Client()
	if _, err := client.SyncOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !store.called {
		t.Fatal("calendar adapter not called")
	}
}

func TestCalendarFailureDoesNotBlockReminderPublication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload model.Snapshot
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if len(payload.Lists) != 1 || payload.Calendars != nil {
			t.Errorf("calendar failure should omit calendars but retain reminders: %+v", payload)
		}
		_ = json.NewEncoder(w).Encode(model.SyncResponse{Version: model.ProtocolVersion})
	}))
	defer server.Close()
	store := &failingCalendarStore{fakeReminderStore: fakeReminderStore{
		items: []model.List{{ID: "list", Name: "Tasks"}},
	}}
	cfg := &config.Config{
		BridgeID: "mac", HomeAssistantURL: server.URL, PollInterval: "30s",
		Calendars: []config.List{{ID: "events", Name: "Events"}},
	}
	client := New(cfg, "synthetic", store, &state.State{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client.httpClient = server.Client()
	if _, err := client.SyncOnce(context.Background()); err != nil {
		t.Fatalf("reminder sync failed when calendar read failed: %v", err)
	}
}

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
	const token = "synthetic-webhook-token-for-unit-tests"
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

func TestSyncOnceStopsOnUncertainCommandOutcome(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(model.SyncResponse{
			Version:  model.ProtocolVersion,
			Commands: []model.Command{{ID: "cmd-uncertain", Action: "complete", ListID: "list", Item: model.Item{UID: "item"}}},
		})
	}))
	defer server.Close()
	cfg := &config.Config{BridgeID: "mac", HomeAssistantURL: server.URL, PollInterval: "30s", StatePath: filepath.Join(t.TempDir(), "state.json")}
	store := &failingApplyStore{}
	client := New(cfg, "token", store, &state.State{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	client.httpClient = server.Client()
	if _, err := client.SyncOnce(context.Background()); err == nil || !strings.Contains(err.Error(), "uncertain") {
		t.Fatalf("expected uncertain outcome, got %v", err)
	}
	loaded, err := state.Load(cfg.StatePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.InFlight == nil || loaded.InFlight.ID != "cmd-uncertain" {
		t.Fatalf("in-flight command was not persisted: %#v", loaded.InFlight)
	}
	if store.calls != 1 {
		t.Fatalf("apply calls = %d, want one", store.calls)
	}
}

type failingApplyStore struct{ calls int }

func (*failingApplyStore) Snapshot(context.Context) ([]model.List, error) { return nil, nil }
func (s *failingApplyStore) ApplyCommand(context.Context, model.Command) error {
	s.calls++
	return errors.New("helper timed out")
}

func TestPostSnapshotRejectsUnboundedOrUnsupportedCommands(t *testing.T) {
	for _, command := range []model.Command{
		{ID: "", Action: "complete", ListID: "list", Item: model.Item{UID: "item"}},
		{ID: "cmd", Action: "delete", ListID: "list", Item: model.Item{UID: "item"}},
		{ID: "cmd", Action: "complete", ListID: "list"},
	} {
		t.Run(command.Action+command.ID, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(model.SyncResponse{Version: model.ProtocolVersion, Commands: []model.Command{command}})
			}))
			defer server.Close()
			cfg := &config.Config{BridgeID: "mac", HomeAssistantURL: server.URL, PollInterval: "30s"}
			client := New(cfg, "token", &fakeReminderStore{}, &state.State{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			client.httpClient = server.Client()
			if _, err := client.SyncOnce(context.Background()); err == nil {
				t.Fatal("accepted invalid command")
			}
		})
	}
}
