package model

import "time"

const (
	ProtocolVersion        = 1
	CapabilityReminders    = "reminders"
	CapabilityCalendars    = "calendars"
	CapabilityCommandQueue = "command_queue"
)

type Event struct {
	UID         string `json:"uid"`
	Summary     string `json:"summary"`
	Start       string `json:"start"`
	End         string `json:"end"`
	AllDay      bool   `json:"all_day"`
	TimeZone    string `json:"time_zone"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
}

type Calendar struct {
	WindowStart string  `json:"window_start,omitempty"`
	WindowEnd   string  `json:"window_end,omitempty"`
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Source      string  `json:"source,omitempty"`
	Events      []Event `json:"events"`
}

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
	Calendars         []Calendar `json:"calendars,omitempty"`
	Capabilities      []string   `json:"capabilities,omitempty"`
	Version           int        `json:"version"`
	BridgeID          string     `json:"bridge_id"`
	SentAt            time.Time  `json:"sent_at"`
	Lists             []List     `json:"lists"`
	AppliedCommandIDs []string   `json:"applied_command_ids,omitempty"`
}

type Command struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	ListID string `json:"list_id"`
	Item   Item   `json:"item"`
}

type SyncResponse struct {
	Version      int       `json:"version"`
	Capabilities []string  `json:"capabilities,omitempty"`
	Commands     []Command `json:"commands"`
}
