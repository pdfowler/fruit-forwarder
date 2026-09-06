package config

import "testing"

func TestSyncPreflight(t *testing.T) {
	for _, tc := range []struct {
		name  string
		url   string
		lists []List
		valid bool
	}{
		{"configured", "https://ha.example.com", []List{{ID: "one", Name: "Tasks"}}, true},
		{"missing URL", "", []List{{ID: "one", Name: "Tasks"}}, false},
		{"example URL", "https://home-assistant.example.invalid", []List{{ID: "one", Name: "Tasks"}}, false},
		{"empty allowlist", "https://ha.example.com", nil, false},
		{"embedded credentials", "https://user:password@ha.example.com", nil, false},
		{"missing host", "https://", nil, false},
		{"query", "https://ha.example.com?token=value", nil, false},
		{"fragment", "https://ha.example.com#fragment", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{BridgeID: "mac", HomeAssistantURL: tc.url, Lists: tc.lists}
			if err := cfg.ValidateSync(); (err == nil) != tc.valid {
				t.Fatalf("ValidateSync() = %v; valid = %v", err, tc.valid)
			}
		})
	}
	local := Config{BridgeID: "mac"}
	if err := local.Validate(); err != nil {
		t.Fatalf("local MCP config must not require HA: %v", err)
	}
}

func TestValidateSecurityAndAllowlist(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid focused config",
			cfg:  Config{BridgeID: "mac", HomeAssistantURL: "https://home.example", PollInterval: "30s", Lists: []List{{ID: "one", Name: "Example List"}}},
		},
		{
			name:    "reject plaintext HTTP",
			cfg:     Config{BridgeID: "mac", HomeAssistantURL: "http://home.example", PollInterval: "30s"},
			wantErr: true,
		},
		{
			name:    "reject ambiguous list names",
			cfg:     Config{BridgeID: "mac", HomeAssistantURL: "https://home.example", PollInterval: "30s", Lists: []List{{ID: "one", Name: "Example List"}, {ID: "two", Name: "example list"}}},
			wantErr: true,
		},
		{
			name:    "reject aggressive polling",
			cfg:     Config{BridgeID: "mac", HomeAssistantURL: "https://home.example", PollInterval: "2s"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
