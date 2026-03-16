package osqueryreceiver

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		expectErr bool
	}{
		{
			name: "valid",
			cfg: Config{
				CollectionInterval: "30s",
				ExtensionsSocket:   "/var/osquery/osquery.em",
				CustomQueries:      []string{"SELECT * FROM processes;"},
			},
			expectErr: false,
		},
		{
			name: "invalid - empty queries and collections",
			cfg: Config{
				CollectionInterval: "30s",
				ExtensionsSocket:   "/var/osquery/osquery.em",
				CustomQueries:      []string{},
			},
			expectErr: true,
		},
		{
			name: "invalid - invalid collection",
			cfg: Config{
				CollectionInterval: "30s",
				ExtensionsSocket:   "/var/osquery/osquery.em",
				Collections:        []string{"invalid_collection"},
			},
			expectErr: true,
		},
		{
			name: "valid - with collections",
			cfg: Config{
				CollectionInterval: "30s",
				ExtensionsSocket:   "/var/osquery/osquery.em",
				Collections:        []string{"system_info", "package_info"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
