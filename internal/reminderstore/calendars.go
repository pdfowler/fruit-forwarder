package reminderstore

import (
	"context"
	"errors"
	"time"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
)

func DiscoverCalendars(ctx context.Context, helperPath string) ([]model.Calendar, error) {
	if err := validateExecutable(helperPath); err != nil {
		return nil, err
	}
	out, err := (&commandRunner{helperPath: helperPath}).Run(ctx, request{Action: "calendars"})
	return out.Calendars, err
}

func (s *Store) CalendarsEnabled() bool { return len(s.calendars) > 0 }

// Calendars exposes configuration only: no account-wide discovery over MCP.
func (s *Store) Calendars(context.Context) ([]model.Calendar, error) {
	result := make([]model.Calendar, 0, len(s.calendars))
	for _, calendar := range s.calendars {
		result = append(result, model.Calendar{ID: calendar.ID, Name: calendar.Name, Events: []model.Event{}})
	}
	return result, nil
}

func (s *Store) CalendarEvents(ctx context.Context, id, start, end string) ([]model.Event, error) {
	allowed := false
	for _, calendar := range s.calendars {
		if calendar.ID == id {
			allowed = true
		}
	}
	if !allowed {
		return nil, errors.New("calendar is outside the allowlist")
	}
	from, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, errors.New("start must be RFC3339")
	}
	to, err := time.Parse(time.RFC3339, end)
	if err != nil || !to.After(from) || to.Sub(from) > 366*24*time.Hour {
		return nil, errors.New("end must be RFC3339 and the window must be positive and at most 366 days")
	}
	out, err := s.runner.Run(ctx, request{Action: "calendar_snapshot", CalendarIDs: []string{id}, Start: from.UTC().Format(time.RFC3339), End: to.UTC().Format(time.RFC3339)})
	if err != nil {
		return nil, err
	}
	if len(out.Calendars) != 1 || out.Calendars[0].ID != id {
		return nil, errors.New("EventKit returned an unexpected calendar scope")
	}
	configured := s.calendar(id)
	if configured == nil || out.Calendars[0].Name != configured.Name {
		return nil, errors.New("configured calendar name changed or is unavailable")
	}
	items := out.Calendars[0].Events
	if len(items) > 10000 {
		return nil, errors.New("calendar snapshot exceeds 10000 events")
	}
	seen := make(map[string]bool)
	for _, item := range items {
		layout := time.RFC3339
		if item.AllDay {
			layout = "2006-01-02"
		}
		a, aerr := time.Parse(layout, item.Start)
		b, berr := time.Parse(layout, item.End)
		if item.UID == "" || seen[item.UID] || aerr != nil || berr != nil || !b.After(a) {
			return nil, errors.New("EventKit returned an invalid or duplicate event")
		}
		seen[item.UID] = true
	}
	if items == nil {
		items = []model.Event{}
	}
	return items, nil
}

func (s *Store) calendar(id string) *config.List {
	for index := range s.calendars {
		if s.calendars[index].ID == id {
			return &s.calendars[index]
		}
	}
	return nil
}
