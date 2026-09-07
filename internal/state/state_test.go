package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRoundTripAndDeduplication(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	state := &State{}
	state.MarkApplied("one")
	state.MarkApplied("one")
	if err := state.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AppliedCommandIDs) != 1 || !loaded.Has("one") {
		t.Fatalf("unexpected state: %#v", loaded)
	}
}

func TestStateBoundsAppliedLedger(t *testing.T) {
	t.Parallel()
	state := &State{}
	for i := 0; i < maxAppliedCommands+10; i++ {
		state.MarkApplied(string(rune(i)))
	}
	if len(state.AppliedCommandIDs) != maxAppliedCommands {
		t.Fatalf("got %d ledger entries", len(state.AppliedCommandIDs))
	}
}

func TestStateRejectsUntrustedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := (&State{AppliedCommandIDs: []string{"one"}}).Save(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o620); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("group-writable state accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlinked state accepted")
	}
}
