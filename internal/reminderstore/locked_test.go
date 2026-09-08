package reminderstore

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfowler/fruit-forwarder/internal/config"
	"github.com/pdfowler/fruit-forwarder/internal/state"
)

func TestLockedStoreRespectsBridgeStateLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "state.lock")
	store := NewLocked(NewWithRunner(&config.Config{}, &fakeRunner{}), lockPath)
	release, err := state.Acquire(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := store.Snapshot(context.Background()); err == nil || !strings.Contains(err.Error(), "state lock") {
		t.Fatalf("locked store ignored the bridge lock: %v", err)
	}
}
