package state

import (
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
