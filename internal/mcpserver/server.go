package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
)

type ReminderStore interface {
	Lists(context.Context) ([]model.List, error)
	Snapshot(context.Context) ([]model.List, error)
	Create(context.Context, string, model.Item) (*model.Item, error)
	Update(context.Context, string, model.Item) (*model.Item, error)
	SetCompleted(context.Context, string, string, bool) (*model.Item, error)
}

type Server struct {
	store ReminderStore
}

type EmptyInput struct{}

type ListsOutput struct {
	Lists []model.List `json:"lists" jsonschema:"Only the reminder lists explicitly allowlisted in bridge configuration."`
}

type ListInput struct {
	ListID string `json:"list_id" jsonschema:"Exact allowlisted EventKit reminder list identifier."`
	Status string `json:"status,omitempty" jsonschema:"Optional status filter: needs_action or completed."`
}

type ItemsOutput struct {
	Items []model.Item `json:"items"`
}

type CreateInput struct {
	ListID      string `json:"list_id" jsonschema:"Exact allowlisted EventKit reminder list identifier."`
	Summary     string `json:"summary" jsonschema:"Reminder title."`
	Description string `json:"description,omitempty" jsonschema:"Optional plain-text notes."`
	Due         string `json:"due,omitempty" jsonschema:"Optional RFC3339 due date and time."`
}

type UpdateInput struct {
	ListID      string `json:"list_id"`
	UID         string `json:"uid"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Due         string `json:"due,omitempty" jsonschema:"RFC3339 due date/time, or empty to clear."`
	Status      string `json:"status" jsonschema:"needs_action or completed."`
}

type StatusInput struct {
	ListID string `json:"list_id"`
	UID    string `json:"uid"`
}

type ItemOutput struct {
	Item model.Item `json:"item"`
}

func Run(ctx context.Context, store ReminderStore, readOnly bool) error {
	return newServer(store, readOnly).Run(ctx, &mcp.StdioTransport{})
}

func newServer(store ReminderStore, readOnly bool) *mcp.Server {
	bridge := &Server{store: store}
	server := mcp.NewServer(&mcp.Implementation{Name: "icloud-reminders", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "reminder_lists", Description: "List the EventKit reminder lists explicitly allowlisted for this bridge."}, bridge.lists)
	mcp.AddTool(server, &mcp.Tool{Name: "reminders_list", Description: "List reminders in one allowlisted list. This never reads outside the configured list IDs."}, bridge.items)
	if calendars, ok := store.(CalendarStore); ok && calendars.CalendarsEnabled() {
		addCalendarTools(server, calendars)
	}
	if readOnly {
		return server
	}
	mcp.AddTool(server, &mcp.Tool{Name: "reminders_create", Description: "Create a reminder in one allowlisted list."}, bridge.create)
	mcp.AddTool(server, &mcp.Tool{Name: "reminders_update", Description: "Update title, notes, due date, and status for an allowlisted reminder. This cannot delete reminders."}, bridge.update)
	mcp.AddTool(server, &mcp.Tool{Name: "reminders_complete", Description: "Mark an allowlisted reminder complete."}, bridge.complete)
	mcp.AddTool(server, &mcp.Tool{Name: "reminders_reopen", Description: "Mark an allowlisted reminder incomplete."}, bridge.reopen)
	return server
}

func (s *Server) lists(ctx context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, ListsOutput, error) {
	lists, err := s.store.Lists(ctx)
	for i := range lists {
		lists[i].Items = nil
	}
	return nil, ListsOutput{Lists: lists}, err
}

func (s *Server) items(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ItemsOutput, error) {
	if input.Status != "" && input.Status != "needs_action" && input.Status != "completed" {
		return nil, ItemsOutput{}, errors.New("status must be needs_action or completed")
	}
	lists, err := s.store.Snapshot(ctx)
	if err != nil {
		return nil, ItemsOutput{}, err
	}
	for _, list := range lists {
		if list.ID != input.ListID {
			continue
		}
		items := make([]model.Item, 0, len(list.Items))
		for _, item := range list.Items {
			if input.Status == "" || item.Status == input.Status {
				items = append(items, item)
			}
		}
		return nil, ItemsOutput{Items: items}, nil
	}
	return nil, ItemsOutput{}, fmt.Errorf("list %q is outside the allowlist", input.ListID)
}

func (s *Server) create(ctx context.Context, _ *mcp.CallToolRequest, input CreateInput) (*mcp.CallToolResult, ItemOutput, error) {
	item, err := s.store.Create(ctx, input.ListID, model.Item{
		Summary: strings.TrimSpace(input.Summary), Description: input.Description, Due: input.Due, Status: "needs_action",
	})
	if err != nil {
		return nil, ItemOutput{}, err
	}
	return nil, ItemOutput{Item: *item}, nil
}

func (s *Server) update(ctx context.Context, _ *mcp.CallToolRequest, input UpdateInput) (*mcp.CallToolResult, ItemOutput, error) {
	if input.Status != "needs_action" && input.Status != "completed" {
		return nil, ItemOutput{}, errors.New("status must be needs_action or completed")
	}
	item, err := s.store.Update(ctx, input.ListID, model.Item{
		UID: input.UID, Summary: strings.TrimSpace(input.Summary), Description: input.Description, Due: input.Due, Status: input.Status,
	})
	if err != nil {
		return nil, ItemOutput{}, err
	}
	return nil, ItemOutput{Item: *item}, nil
}

func (s *Server) complete(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, ItemOutput, error) {
	item, err := s.store.SetCompleted(ctx, input.ListID, input.UID, true)
	if err != nil {
		return nil, ItemOutput{}, err
	}
	return nil, ItemOutput{Item: *item}, nil
}

func (s *Server) reopen(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, ItemOutput, error) {
	item, err := s.store.SetCompleted(ctx, input.ListID, input.UID, false)
	if err != nil {
		return nil, ItemOutput{}, err
	}
	return nil, ItemOutput{Item: *item}, nil
}
