package config

import "testing"

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
