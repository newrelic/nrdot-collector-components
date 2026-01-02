package adaptivetelemetryprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestExtractProcessName(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]string
		expected string
	}{
		{
			name: "executable.name present",
			attrs: map[string]string{
				"process.executable.name": "nginx",
				"process.command":         "/usr/sbin/nginx",
			},
			expected: "nginx",
		},
		{
			name: "only command with path",
			attrs: map[string]string{
				"process.command": "/usr/bin/postgres",
			},
			expected: "postgres",
		},
		{
			name: "only command without path",
			attrs: map[string]string{
				"process.command": "redis-server",
			},
			expected: "redis-server",
		},
		{
			name: "windows path",
			attrs: map[string]string{
				"process.command": "C:\\Program Files\\App\\app.exe",
			},
			expected: "app.exe",
		},
		{
			name:     "no process attributes",
			attrs:    map[string]string{},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			for k, v := range tc.attrs {
				attrs.PutStr(k, v)
			}
			result := extractProcessName(attrs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsProcessInIncludeList(t *testing.T) {
	tests := []struct {
		name        string
		attrs       map[string]string
		includeList []string
		expected    bool
	}{
		{
			name: "process in include list - executable.name match",
			attrs: map[string]string{
				"process.executable.name": "nginx",
				"process.pid":             "1234",
			},
			includeList: []string{"nginx", "postgres"},
			expected:    true,
		},
		{
			name: "process in include list - command match",
			attrs: map[string]string{
				"process.command": "/usr/bin/postgres",
				"process.pid":     "5678",
			},
			includeList: []string{"nginx", "postgres"},
			expected:    true,
		},
		{
			name: "process not in include list",
			attrs: map[string]string{
				"process.executable.name": "apache2",
				"process.pid":             "9999",
			},
			includeList: []string{"nginx", "postgres"},
			expected:    false,
		},
		{
			name: "empty include list",
			attrs: map[string]string{
				"process.executable.name": "nginx",
			},
			includeList: []string{},
			expected:    false,
		},
		{
			name: "no process attributes",
			attrs: map[string]string{
				"host.name": "server1",
			},
			includeList: []string{"nginx"},
			expected:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			for k, v := range tc.attrs {
				attrs.PutStr(k, v)
			}
			result := isProcessInIncludeList(attrs, tc.includeList)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIncludeListInProcessorConfig(t *testing.T) {
	// Test that config properly handles include list
	cfg := &Config{
		IncludeProcessList: []string{"nginx", "postgres", "redis-server"},
	}

	cfg.Normalize()
	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(cfg.IncludeProcessList))
	assert.Contains(t, cfg.IncludeProcessList, "nginx")
	assert.Contains(t, cfg.IncludeProcessList, "postgres")
	assert.Contains(t, cfg.IncludeProcessList, "redis-server")
}

