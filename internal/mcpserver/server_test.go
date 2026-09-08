package mcpserver

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pdfowler/fruit-forwarder/internal/model"
)

type protocolStore struct{ writes atomic.Int32 }

type calendarProtocolStore struct{ protocolStore }

func (*calendarProtocolStore) CalendarsEnabled() bool { return true }
func (*calendarProtocolStore) Calendars(context.Context) ([]model.Calendar, error) {
	return []model.Calendar{{ID: "events", Name: "Events"}}, nil
}
func (*calendarProtocolStore) CalendarEvents(context.Context, string, string, string) ([]model.Event, error) {
	return []model.Event{{UID: "event", Start: "2026-01-01", End: "2026-01-02", AllDay: true}}, nil
}

func TestCalendarToolsAvailableInReadOnlyMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := newServer(&calendarProtocolStore{}, true).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "calendar-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	catalog, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tools) != 4 {
		t.Fatalf("expected four read tools, got %d", len(catalog.Tools))
	}
	for name, args := range map[string]map[string]any{
		"calendar_lists":  {},
		"calendar_events": {"calendar_id": "events", "start": "2026-01-01T00:00:00Z", "end": "2026-02-01T00:00:00Z"},
	} {
		result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || result.IsError {
			t.Fatalf("%s: %v, %v", name, result, err)
		}
	}
}

func (*protocolStore) Lists(context.Context) ([]model.List, error) {
	return []model.List{{ID: "allowed", Name: "Tasks"}}, nil
}
func (*protocolStore) Snapshot(context.Context) ([]model.List, error) {
	return []model.List{{ID: "allowed", Name: "Tasks", Items: []model.Item{
		{UID: "active", Summary: "Active", Status: "needs_action"},
		{UID: "done", Summary: "Done", Status: "completed"},
	}}}, nil
}
func (s *protocolStore) Create(_ context.Context, _ string, item model.Item) (*model.Item, error) {
	s.writes.Add(1)
	item.UID = "created"
	return &item, nil
}
func (s *protocolStore) Update(_ context.Context, _ string, item model.Item) (*model.Item, error) {
	s.writes.Add(1)
	return &item, nil
}
func (s *protocolStore) SetCompleted(_ context.Context, _ string, uid string, completed bool) (*model.Item, error) {
	s.writes.Add(1)
	status := "needs_action"
	if completed {
		status = "completed"
	}
	return &model.Item{UID: uid, Summary: "Task", Status: status}, nil
}

func TestProtocolAccessPolicy(t *testing.T) {
	for _, readOnly := range []bool{true, false} {
		name := "writable"
		if readOnly {
			name = "read_only"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			store := &protocolStore{}
			st, ct := mcp.NewInMemoryTransports()
			ss, err := newServer(store, readOnly).Connect(ctx, st, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ss.Close()
			client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
			cs, err := client.Connect(ctx, ct, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer cs.Close()
			catalog, err := cs.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := 6
			if readOnly {
				want = 2
			}
			if len(catalog.Tools) != want {
				t.Fatalf("got %d tools, want %d", len(catalog.Tools), want)
			}
			result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "reminders_list", Arguments: map[string]any{"list_id": "allowed", "status": "needs_action"}})
			if err != nil || result.IsError {
				t.Fatalf("list: %v, %v", result, err)
			}
			data, err := json.Marshal(result.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			var output ItemsOutput
			if err := json.Unmarshal(data, &output); err != nil {
				t.Fatal(err)
			}
			if len(output.Items) != 1 || output.Items[0].UID != "active" {
				t.Fatalf("unexpected filtered items: %+v", output)
			}
			page, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "reminders_list", Arguments: map[string]any{
				"list_id": "allowed", "limit": 1,
			}})
			if err != nil || page.IsError {
				t.Fatalf("paginated list: %v, %v", page, err)
			}
			var firstPage ItemsOutput
			data, err = json.Marshal(page.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &firstPage); err != nil {
				t.Fatal(err)
			}
			if len(firstPage.Items) != 1 || firstPage.Total != 2 || firstPage.NextOffset == nil || *firstPage.NextOffset != 1 {
				t.Fatalf("unexpected first page: %+v", firstPage)
			}
			invalid, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "reminders_list", Arguments: map[string]any{
				"list_id": "allowed", "limit": maxPageSize + 1,
			}})
			if err == nil && !invalid.IsError {
				t.Fatal("oversized page unexpectedly succeeded")
			}
			for _, args := range []map[string]any{
				{"list_id": "outside"},
				{"list_id": "allowed", "status": "invalid"},
			} {
				res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "reminders_list", Arguments: args})
				if err == nil && !res.IsError {
					t.Fatal("invalid read unexpectedly succeeded")
				}
			}
			for _, tool := range []string{"reminders_create", "reminders_update", "reminders_complete", "reminders_reopen"} {
				args := map[string]any{"list_id": "allowed"}
				if tool != "reminders_create" {
					args["uid"] = "active"
				}
				if tool == "reminders_create" || tool == "reminders_update" {
					args["summary"] = "Task"
				}
				if tool == "reminders_update" {
					args["status"] = "needs_action"
				}
				res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
				if readOnly {
					if err == nil && !res.IsError {
						t.Fatalf("read-only accepted %s", tool)
					}
				} else if err != nil || res.IsError {
					t.Fatalf("%s: %v, %v", tool, res, err)
				}
			}
			wantWrites := int32(4)
			if readOnly {
				wantWrites = 0
			}
			if store.writes.Load() != wantWrites {
				t.Fatalf("writes = %d, want %d", store.writes.Load(), wantWrites)
			}
		})
	}
}
