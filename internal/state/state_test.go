package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
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
		state.MarkApplied(fmt.Sprintf("command-%d", i))
	}
	if len(state.AppliedCommandIDs) != maxAppliedCommands {
		t.Fatalf("got %d ledger entries", len(state.AppliedCommandIDs))
	}
}

func TestStateRejectsMalformedLedger(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(`{"applied_command_ids":["ok","ok"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("duplicate applied command identifiers accepted")
	}
	if err := os.WriteFile(path, []byte(`{"applied_command_ids":["   "]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("blank applied command identifier accepted")
	}
	if err := os.WriteFile(path, []byte(`{"applied_command_ids":["`+strings.Repeat("x", maxCommandIDLength+1)+`"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("oversized applied command identifier accepted")
	}
}

func TestStateSaveRejectsOversizedLedger(t *testing.T) {
	t.Parallel()
	state := &State{AppliedCommandIDs: make([]string, maxAppliedCommands+1)}
	if err := state.Save(filepath.Join(t.TempDir(), "state.json")); err == nil {
		t.Fatal("oversized ledger saved")
	}
}

func TestStateRoundTripsInFlightCommand(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "state.json")
	state := &State{InFlight: &model.Command{
		ID: "command-1", Action: "create", ListID: "list-1",
		Item: model.Item{Summary: "Task"},
	}}
	if err := state.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.InFlight == nil || loaded.InFlight.ID != "command-1" {
		t.Fatalf("in-flight command lost: %#v", loaded.InFlight)
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
