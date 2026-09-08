package reminderstore

import (
	"context"

	"github.com/pdfowler/fruit-forwarder/internal/model"
	"github.com/pdfowler/fruit-forwarder/internal/state"
)

// LockedStore serializes EventKit calls made by MCP with the bridge's HA
// synchronizer and recovery commands. The lock is acquired per operation so a
// long-lived MCP session does not starve background polling.
type LockedStore struct {
	*Store
	lockPath string
}

func NewLocked(store *Store, lockPath string) *LockedStore {
	return &LockedStore{Store: store, lockPath: lockPath}
}

func (s *LockedStore) Lists(ctx context.Context) ([]model.List, error) {
	var result []model.List
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.Lists(ctx)
		return err
	})
	return result, err
}

func (s *LockedStore) Snapshot(ctx context.Context) ([]model.List, error) {
	var result []model.List
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.Snapshot(ctx)
		return err
	})
	return result, err
}

func (s *LockedStore) Create(ctx context.Context, listID string, item model.Item) (*model.Item, error) {
	var result *model.Item
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.Create(ctx, listID, item)
		return err
	})
	return result, err
}

func (s *LockedStore) Update(ctx context.Context, listID string, item model.Item) (*model.Item, error) {
	var result *model.Item
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.Update(ctx, listID, item)
		return err
	})
	return result, err
}

func (s *LockedStore) UpdatePatch(ctx context.Context, listID string, patch model.ItemPatch) (*model.Item, error) {
	var result *model.Item
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.UpdatePatch(ctx, listID, patch)
		return err
	})
	return result, err
}

func (s *LockedStore) SetCompleted(ctx context.Context, listID, uid string, completed bool) (*model.Item, error) {
	var result *model.Item
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.SetCompleted(ctx, listID, uid, completed)
		return err
	})
	return result, err
}

func (s *LockedStore) Calendars(ctx context.Context) ([]model.Calendar, error) {
	var result []model.Calendar
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.Calendars(ctx)
		return err
	})
	return result, err
}

func (s *LockedStore) CalendarEvents(ctx context.Context, id, start, end string) ([]model.Event, error) {
	var result []model.Event
	err := s.withLock(func() error {
		var err error
		result, err = s.Store.CalendarEvents(ctx, id, start, end)
		return err
	})
	return result, err
}

func (s *LockedStore) withLock(operation func() error) error {
	release, err := state.Acquire(s.lockPath)
	if err != nil {
		return err
	}
	defer release()
	return operation()
}
