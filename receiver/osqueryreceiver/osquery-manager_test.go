package osqueryreceiver

import (
	"testing"

	"go.uber.org/zap"
)

func TestOSQueryManagerInitialization(t *testing.T) {

	tests := map[string]struct {
		config                 *Config
		expectedNumCollections int
	}{
		"Single Collection": {
			config: &Config{
				ExtensionsSocket:   "/var/osquery/osquery.em",
				CollectionInterval: "60s",
				Collections:        []string{"system_info"},
			},
			expectedNumCollections: 1,
		},
		// "Multiple Collections": {
		// 	config: &Config{
		// 		ExtensionsSocket:   "/var/osquery/osquery.em",
		// 		CollectionInterval: "60s",
		// 		Collections: []string{"system_info", "processes", "users"},
		// 	},
		// 	expectedNumTasks: 3,
		// 	expectedNumCollections: 3,
		// },
		"Custom Queries Only": {
			config: &Config{
				ExtensionsSocket:   "/var/osquery/osquery.em",
				CollectionInterval: "60s",
				CustomQueries: []string{
					"select * from processes limit 5;",
					"select * from os_version;",
				},
			},
			expectedNumCollections: 2,
		},
		"Collections and Custom Queries": {
			config: &Config{
				ExtensionsSocket:   "/var/osquery/osquery.em",
				CollectionInterval: "60s",
				Collections:        []string{"system_info"},
				CustomQueries: []string{
					"select * from processes limit 5;",
				},
			},
			expectedNumCollections: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			manager, err := NewOSQueryManager(tc.config, zap.NewNop())
			if err != nil {
				tt.Fatalf("Failed to initialize OSQueryManager: %v", err)
			}

			if manager.extensionsSocket != tc.config.ExtensionsSocket {
				tt.Errorf("Expected extensions socket %s, got %s", tc.config.ExtensionsSocket, manager.extensionsSocket)
			}

			if len(manager.executor.Collections) != tc.expectedNumCollections {
				tt.Errorf("Expected %d collections, got %d", tc.expectedNumCollections, len(manager.executor.Collections))
			}
		})
	}
}

// func TestCollect(t *testing.T) {
// 	config := &Config{
// 		ExtensionsSocket:   "/var/osquery/osquery.em",
// 		CollectionInterval: "60s",
// 		Collections: []string{"system_info"},
// 		CustomQueries: []string{
// 			"select * from processes limit 5;",
// 		},
// 	}

// 	manager, err := NewOSQueryManager(config, zap.NewNop())
// 	if err != nil {
// 		t.Fatalf("Failed to initialize OSQueryManager: %v", err)
// 	}

// 	// Since we cannot run actual osquery commands in tests, we will just ensure that
// 	// the collect method runs without errors and processes all tasks.
// 	err = manager.collect(zap.NewNop(), config, nil)
// 	if err != nil {
// 		t.Errorf("Collect method returned an error: %v", err)
// 	}
// }
