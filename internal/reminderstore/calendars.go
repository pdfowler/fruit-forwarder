package reminderstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pdfowler/fruit-forwarder/internal/config"
	"github.com/pdfowler/fruit-forwarder/internal/model"
)

func DiscoverCalendars(ctx context.Context, helperPath string) ([]model.Calendar, error) {
	if err := validateExecutable(helperPath); err != nil {
		return nil, err
	}
	out, err := (&commandRunner{helperPath: helperPath}).Run(ctx, request{Action: "calendars"})
	if err != nil {
		return nil, err
	}
	for _, calendar := range out.Calendars {
		if err := validateCalendar(calendar); err != nil {
			return nil, fmt.Errorf("EventKit returned an invalid calendar: %w", err)
		}
	}
	return out.Calendars, nil
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
	if err := validateCalendar(out.Calendars[0]); err != nil {
		return nil, fmt.Errorf("EventKit returned an invalid calendar: %w", err)
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
		if item.UID == "" || seen[item.UID] || aerr != nil || berr != nil || !b.After(a) ||
			len(item.UID) > maxFieldLength || len(item.Summary) > maxFieldLength ||
			len(item.Start) > maxFieldLength || len(item.End) > maxFieldLength ||
			len(item.TimeZone) > maxFieldLength || len(item.Description) > maxFieldLength ||
			len(item.Location) > maxFieldLength {
			return nil, errors.New("EventKit returned an invalid or duplicate event")
		}
		seen[item.UID] = true
	}
	if items == nil {
		items = []model.Event{}
	}
	return items, nil
}

func validateCalendar(calendar model.Calendar) error {
	if strings.TrimSpace(calendar.ID) == "" || len(calendar.ID) > maxFieldLength {
		return errors.New("calendar id is empty or too long")
	}
	if strings.TrimSpace(calendar.Name) == "" || len(calendar.Name) > maxSummaryLength {
		return errors.New("calendar name is empty or too long")
	}
	if len(calendar.Source) > maxFieldLength {
		return errors.New("calendar source is too long")
	}
	if len(calendar.Events) > 10000 {
		return errors.New("calendar contains too many events")
	}
	return nil
}

func (s *Store) calendar(id string) *config.List {
	for index := range s.calendars {
		if s.calendars[index].ID == id {
			return &s.calendars[index]
		}
	}
	return nil
}
