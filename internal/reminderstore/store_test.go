package reminderstore

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pdfowler/fruit-forwarder/internal/config"
	"github.com/pdfowler/fruit-forwarder/internal/model"
)

func TestValidateHelperPathChecksExecutableSafety(t *testing.T) {
	path := t.TempDir() + "/helper"
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHelperPath(path); err != nil {
		t.Fatalf("valid helper rejected: %v", err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHelperPath(path); err != nil {
		t.Fatalf("owner-executable helper rejected: %v", err)
	}
	if err := ValidateHelperPath(path + ".missing"); err == nil {
		t.Fatal("missing helper accepted")
	}
}

func TestDiscoverRejectsExcessiveHelperOutput(t *testing.T) {
	path := t.TempDir() + "/helper"
	script := "#!/bin/sh\nhead -c 8388609 /dev/zero\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := Discover(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "output exceeds") {
		t.Fatalf("excessive helper output was accepted: %v", err)
	}
}

func TestAuthorizationIsNonPromptingAndValidatesStatuses(t *testing.T) {
	path := t.TempDir() + "/helper"
	script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '{\"authorization\":{\"reminders\":\"full_access\",\"calendars\":\"not_determined\"}}'\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	status, err := Authorization(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if status.Reminders != "full_access" || status.Calendars != "not_determined" {
		t.Fatalf("unexpected authorization status: %#v", status)
	}

	badPath := t.TempDir() + "/helper"
	badScript := "#!/bin/sh\nprintf '%s\\n' '{\"authorization\":{\"reminders\":\"unexpected\",\"calendars\":\"full_access\"}}'\n"
	if err := os.WriteFile(badPath, []byte(badScript), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Authorization(context.Background(), badPath); err == nil || !strings.Contains(err.Error(), "invalid authorization") {
		t.Fatalf("invalid authorization status accepted: %v", err)
	}
}

type fakeRunner struct {
	lists    []model.List
	snapshot []model.List
	requests []request
	item     *model.Item
}

func (f *fakeRunner) Run(_ context.Context, input request) (response, error) {
	f.requests = append(f.requests, input)
	switch input.Action {
	case "lists":
		return response{Lists: f.lists}, nil
	case "snapshot":
		return response{Lists: f.snapshot}, nil
	default:
		if f.item != nil {
			copy := *f.item
			return response{Item: &copy}, nil
		}
		return response{Item: &model.Item{UID: "result", Summary: input.Item.Summary}}, nil
	}
}

func testConfig() *config.Config {
	return &config.Config{CompletedRetention: "720h", Lists: []config.List{{ID: "allowed", Name: "Example List"}}}
}

func TestEmptyAllowlistDoesNotInvokeEventKit(t *testing.T) {
	runner := &fakeRunner{}
	lists, err := NewWithRunner(&config.Config{}, runner).Snapshot(context.Background())
	if err != nil || len(lists) != 0 || len(runner.requests) != 0 {
		t.Fatalf("empty allowlist touched EventKit: lists=%v err=%v calls=%d", lists, err, len(runner.requests))
	}
}

func TestValidateListsRejectsChangedName(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{lists: []model.List{{ID: "allowed", Name: "Renamed"}}}
	err := NewWithRunner(testConfig(), runner).ValidateLists(context.Background())
	if err == nil || !strings.Contains(err.Error(), "changed name") {
		t.Fatalf("expected renamed-list error, got %v", err)
	}
}

func TestSnapshotRejectsCrossListLeak(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{snapshot: []model.List{{ID: "not-allowed", Name: "Private"}}}
	_, err := NewWithRunner(testConfig(), runner).Snapshot(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unexpected list") {
		t.Fatalf("expected list-scope error, got %v", err)
	}
}

func TestSnapshotOmitsBlankTitleItems(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{snapshot: []model.List{{
		ID:   "allowed",
		Name: "Example List",
		Items: []model.Item{
			{UID: "blank", Summary: "  ", Status: "completed"},
			{UID: "milk", Summary: "Milk", Status: "needs_action"},
		},
	}}}
	lists, err := NewWithRunner(testConfig(), runner).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || len(lists[0].Items) != 1 || lists[0].Items[0].UID != "milk" {
		t.Fatalf("unexpected filtered snapshot: %#v", lists)
	}
}

func TestSnapshotOmitsOldCompletedItems(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{snapshot: []model.List{{
		ID:   "allowed",
		Name: "Example List",
		Items: []model.Item{
			{UID: "old", Summary: "Old", Status: "completed", Completed: "2020-01-01T00:00:00Z"},
			{UID: "recent", Summary: "Recent", Status: "completed", Completed: time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)},
			{UID: "open", Summary: "Open", Status: "needs_action"},
		},
	}}}
	lists, err := NewWithRunner(testConfig(), runner).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(lists[0].Items) != 2 {
		t.Fatalf("unexpected retained snapshot: %#v", lists[0].Items)
	}
	for _, item := range lists[0].Items {
		if item.UID == "old" {
			t.Fatalf("old completed item was retained: %#v", lists[0].Items)
		}
	}
}

func TestCreateUsesExactAllowlistedID(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{}
	store := NewWithRunner(testConfig(), runner)
	if _, err := store.Create(context.Background(), "not-allowed", model.Item{Summary: "No"}); err == nil {
		t.Fatal("create outside allowlist succeeded")
	}
	if _, err := store.Create(context.Background(), "allowed", model.Item{Summary: "Milk"}); err != nil {
		t.Fatal(err)
	}
	last := runner.requests[len(runner.requests)-1]
	if last.Action != "create" || last.ListID != "allowed" || last.Item.Summary != "Milk" {
		t.Fatalf("unexpected request: %#v", last)
	}
}

func TestUpdateRequiresUID(t *testing.T) {
	t.Parallel()
	store := NewWithRunner(testConfig(), &fakeRunner{})
	_, err := store.Update(context.Background(), "allowed", model.Item{Summary: "Changed"})
	if err == nil || !strings.Contains(err.Error(), "uid") {
		t.Fatalf("expected uid error, got %v", err)
	}
}

func TestWriteRejectsOversizedFields(t *testing.T) {
	store := NewWithRunner(testConfig(), &fakeRunner{})
	for _, item := range []model.Item{
		{Summary: strings.Repeat("x", 257)},
		{Summary: "Task", Description: strings.Repeat("x", 4097)},
	} {
		if _, err := store.Create(context.Background(), "allowed", item); err == nil {
			t.Fatal("oversized reminder field was accepted")
		}
	}
}

func TestSnapshotRejectsOversizedHelperFields(t *testing.T) {
	runner := &fakeRunner{snapshot: []model.List{{
		ID: "allowed", Name: "Example List",
		Items: []model.Item{{UID: "item", Summary: strings.Repeat("x", 257), Status: "needs_action"}},
	}}}
	if _, err := NewWithRunner(testConfig(), runner).Snapshot(context.Background()); err == nil {
		t.Fatal("oversized helper reminder field was accepted")
	}
}

func TestSetCompletedUsesScopedAction(t *testing.T) {
	t.Parallel()
	runner := &fakeRunner{item: &model.Item{UID: "item", Summary: "Task", Status: "completed"}}
	store := NewWithRunner(testConfig(), runner)
	if _, err := store.SetCompleted(context.Background(), "allowed", "item", true); err != nil {
		t.Fatal(err)
	}
	last := runner.requests[len(runner.requests)-1]
	if last.Action != "complete" || last.ListID != "allowed" || last.Item.UID != "item" {
		t.Fatalf("unexpected request: %#v", last)
	}
}
