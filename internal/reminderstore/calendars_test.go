package reminderstore

import (
	"context"
	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
	"testing"
)

type calendarRunner struct {
	calls  []request
	output response
}

func (r *calendarRunner) Run(_ context.Context, req request) (response, error) {
	r.calls = append(r.calls, req)
	return r.output, nil
}

func TestCalendarScopeAndWindow(t *testing.T) {
	for _, tc := range []struct{ name, id, start, end string }{
		{"outside", "other", "2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z"},
		{"bad_start", "allowed", "bad", "2026-02-01T00:00:00Z"},
		{"backwards", "allowed", "2026-03-01T00:00:00Z", "2026-02-01T00:00:00Z"},
		{"too_large", "allowed", "2026-01-01T00:00:00Z", "2028-02-01T00:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &calendarRunner{}
			store := NewWithRunner(&config.Config{Calendars: []config.List{{ID: "allowed", Name: "Events"}}}, runner)
			if _, err := store.CalendarEvents(context.Background(), tc.id, tc.start, tc.end); err == nil {
				t.Fatal("accepted invalid query")
			}
			if len(runner.calls) != 0 {
				t.Fatal("invalid query invoked EventKit")
			}
		})
	}
}

func TestCalendarResponseValidation(t *testing.T) {
	event := model.Event{UID: "occurrence", Start: "2026-01-03", End: "2026-01-04", AllDay: true}
	for _, tc := range []struct {
		name      string
		calendars []model.Calendar
		valid     bool
	}{
		{"valid_all_day", []model.Calendar{{ID: "allowed", Events: []model.Event{event}}}, true},
		{"empty", []model.Calendar{{ID: "allowed"}}, true},
		{"missing_calendar", nil, false},
		{"wrong_calendar", []model.Calendar{{ID: "outside"}}, false},
		{"duplicate_event", []model.Calendar{{ID: "allowed", Events: []model.Event{event, event}}}, false},
		{"invalid_dates", []model.Calendar{{ID: "allowed", Events: []model.Event{{UID: "bad", Start: "bad", End: "bad"}}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &calendarRunner{output: response{Calendars: tc.calendars}}
			store := NewWithRunner(&config.Config{Calendars: []config.List{{ID: "allowed", Name: "Events"}}}, runner)
			_, err := store.CalendarEvents(context.Background(), "allowed", "2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z")
			if (err == nil) != tc.valid {
				t.Fatalf("error = %v, valid = %v", err, tc.valid)
			}
			if len(runner.calls) != 1 || len(runner.calls[0].CalendarIDs) != 1 || runner.calls[0].CalendarIDs[0] != "allowed" {
				t.Fatal("helper query not scoped")
			}
		})
	}
}
