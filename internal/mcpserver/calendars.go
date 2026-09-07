package mcpserver

import (
	"context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pdfowler/fruit-forwarder/internal/model"
)

type CalendarStore interface {
	CalendarsEnabled() bool
	Calendars(context.Context) ([]model.Calendar, error)
	CalendarEvents(context.Context, string, string, string) ([]model.Event, error)
}

type CalendarsOutput struct {
	Calendars []model.Calendar `json:"calendars"`
}
type EventsInput struct {
	CalendarID string `json:"calendar_id" jsonschema:"Exact allowlisted event calendar identifier."`
	Start      string `json:"start" jsonschema:"Inclusive window start as RFC3339 with timezone."`
	End        string `json:"end" jsonschema:"Exclusive window end as RFC3339; positive window at most 366 days."`
}
type EventsOutput struct {
	Events []model.Event `json:"events"`
}

func addCalendarTools(server *mcp.Server, store CalendarStore) {
	mcp.AddTool(server, &mcp.Tool{Name: "calendar_lists", Description: "List configured event calendars. Does not discover other calendars or read event data."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, CalendarsOutput, error) {
			calendars, err := store.Calendars(ctx)
			return nil, CalendarsOutput{Calendars: calendars}, err
		})
	mcp.AddTool(server, &mcp.Tool{Name: "calendar_events", Description: "Read occurrences from one allowlisted calendar in a bounded time window. All-day end dates are exclusive. No event writes are supported."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input EventsInput) (*mcp.CallToolResult, EventsOutput, error) {
			events, err := store.CalendarEvents(ctx, input.CalendarID, input.Start, input.End)
			return nil, EventsOutput{Events: events}, err
		})
}
