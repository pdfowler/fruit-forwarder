package model

import "time"

const ProtocolVersion = 1

type Item struct {
	UID         string `json:"uid"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Due         string `json:"due,omitempty"`
	Completed   string `json:"completed,omitempty"`
	Modified    string `json:"modified,omitempty"`
}

type List struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Source   string `json:"source,omitempty"`
	ReadOnly bool   `json:"read_only"`
	Items    []Item `json:"items"`
}

type Snapshot struct {
	Version           int       `json:"version"`
	BridgeID          string    `json:"bridge_id"`
	SentAt            time.Time `json:"sent_at"`
	Lists             []List    `json:"lists"`
	AppliedCommandIDs []string  `json:"applied_command_ids,omitempty"`
}

type Command struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	ListID string `json:"list_id"`
	Item   Item   `json:"item"`
}

type SyncResponse struct {
	Version  int       `json:"version"`
	Commands []Command `json:"commands"`
}
