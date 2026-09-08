package reminderstore

import (
	"context"
	"testing"

	"github.com/pdfowler/fruit-forwarder/internal/config"
	"github.com/pdfowler/fruit-forwarder/internal/model"
)

func TestUpdatePatchPreservesOmittedFieldsAndClearsExplicitValues(t *testing.T) {
	store := NewWithRunner(&config.Config{Lists: []config.List{{ID: "allowed", Name: "Tasks"}}}, &patchRunner{})
	item, err := store.UpdatePatch(context.Background(), "allowed", model.ItemPatch{
		UID: "item", Summary: "Renamed", Status: "needs_action",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Description != "preserved" || item.Due != "2026-01-01" {
		t.Fatalf("omitted fields were not preserved: %#v", item)
	}
	empty := ""
	item, err = store.UpdatePatch(context.Background(), "allowed", model.ItemPatch{
		UID: "item", Summary: "Renamed", Status: "needs_action", Description: &empty, Due: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Description != "" || item.Due != "" {
		t.Fatalf("explicit clears were not preserved: %#v", item)
	}
}

type patchRunner struct{}

func (*patchRunner) Run(_ context.Context, input request) (response, error) {
	switch input.Action {
	case "snapshot":
		return response{Lists: []model.List{{ID: "allowed", Name: "Tasks", Items: []model.Item{{
			UID: "item", Summary: "Original", Status: "needs_action", Description: "preserved", Due: "2026-01-01",
		}}}}}, nil
	case "update":
		return response{Item: &input.Item}, nil
	default:
		return response{}, nil
	}
}
