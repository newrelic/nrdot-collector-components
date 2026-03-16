package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockOSQueryExecutor struct{}

func (m *mockOSQueryExecutor) Run(query string) ([]map[string]any, error) {
	return []map[string]any{
		{
			"name":    "Ubuntu",
			"version": "20.04",
		},
	}, nil
}

func TestRunOSQuery(t *testing.T) {
	tests := map[string]struct {
		query    string
		wantErr  bool
		expected []map[string]any
	}{
		"simple test": {
			query:   "SELECT * FROM os_version;",
			wantErr: false,
			expected: []map[string]any{
				{
					"name":    "Ubuntu",
					"version": "20.04",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			executor := &mockOSQueryExecutor{}
			result, err := executor.Run(tc.query)
			assert.Equal(tt, tc.wantErr, err != nil)
			assert.Equal(tt, tc.expected, result)
		})
	}
}
