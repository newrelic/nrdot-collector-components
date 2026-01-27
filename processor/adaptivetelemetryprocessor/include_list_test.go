// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
			name: "process in include list - full path match",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
				"process.executable.name": "nginx",
				"process.pid":             "1234",
			},
			includeList: []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
			expected:    true,
		},
		{
			name: "process in include list - command with full path",
			attrs: map[string]string{
				"process.command": "/usr/bin/postgres -D /data",
				"process.pid":     "5678",
			},
			includeList: []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
			expected:    true,
		},
		{
			name: "process not in include list - different path",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/apache2",
				"process.executable.name": "apache2",
				"process.pid":             "9999",
			},
			includeList: []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
			expected:    false,
		},
		{
			name: "empty include list",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
			},
			includeList: []string{},
			expected:    false,
		},
		{
			name: "no process path attributes",
			attrs: map[string]string{
				"host.name": "server1",
			},
			includeList: []string{"/usr/sbin/nginx"},
			expected:    false,
		},
		{
			name: "basename only in include list - no match",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
				"process.executable.name": "nginx",
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
		IncludeProcessList: []string{"/usr/sbin/nginx", "/usr/bin/postgres", "/usr/bin/redis-server"},
	}

	cfg.Normalize()
	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Len(t, cfg.IncludeProcessList, 3)
	assert.Contains(t, cfg.IncludeProcessList, "/usr/sbin/nginx")
	assert.Contains(t, cfg.IncludeProcessList, "/usr/bin/postgres")
	assert.Contains(t, cfg.IncludeProcessList, "/usr/bin/redis-server")
}

// TestExtractProcessExecutablePath tests the extraction of full executable paths
func TestExtractProcessExecutablePath(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]string
		expected string
	}{
		{
			name: "process.executable.path present",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
				"process.command":         "/usr/sbin/nginx -c /etc/nginx.conf",
			},
			expected: "/usr/sbin/nginx",
		},
		{
			name: "only command with full path and arguments",
			attrs: map[string]string{
				"process.command": "/usr/bin/postgres -D /var/lib/postgresql/data",
			},
			expected: "/usr/bin/postgres",
		},
		{
			name: "only command with full path no arguments",
			attrs: map[string]string{
				"process.command": "/usr/bin/redis-server",
			},
			expected: "/usr/bin/redis-server",
		},
		{
			name: "windows path with arguments",
			attrs: map[string]string{
				"process.command": "C:\\Program Files\\App\\app.exe --config file.conf",
			},
			expected: "C:\\Program Files\\App\\app.exe",
		},
		{
			name: "relative path - not returned",
			attrs: map[string]string{
				"process.command": "bin/app",
			},
			expected: "",
		},
		{
			name: "basename only - not returned",
			attrs: map[string]string{
				"process.command": "nginx",
			},
			expected: "",
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
			result := extractProcessExecutablePath(attrs)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIsProcessInIncludeList_SecurityScenarios tests security-focused scenarios
// to ensure the enhanced matching logic prevents process name spoofing
func TestIsProcessInIncludeList_SecurityScenarios(t *testing.T) {
	tests := []struct {
		name        string
		attrs       map[string]string
		includeList []string
		expected    bool
		description string
	}{
		{
			name: "SECURITY: Full path match - legitimate nginx",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
				"process.executable.name": "nginx",
			},
			includeList: []string{"/usr/sbin/nginx"},
			expected:    true,
			description: "Legitimate nginx at correct path should match",
		},
		{
			name: "SECURITY: Full path mismatch - spoofed nginx",
			attrs: map[string]string{
				"process.executable.path": "/tmp/nginx",
				"process.executable.name": "nginx",
			},
			includeList: []string{"/usr/sbin/nginx"},
			expected:    false,
			description: "Malicious /tmp/nginx should NOT match when full path is specified",
		},
		{
			name: "SECURITY: Basename-only entry does not match",
			attrs: map[string]string{
				"process.executable.path": "/tmp/nginx",
				"process.executable.name": "nginx",
			},
			includeList: []string{"nginx"},
			expected:    false,
			description: "Basename-only entries (without path separator) will not match any process",
		},
		{
			name: "SECURITY: Full path from command attribute",
			attrs: map[string]string{
				"process.command": "/usr/bin/postgres -D /data",
			},
			includeList: []string{"/usr/bin/postgres"},
			expected:    true,
			description: "Full path extracted from command should match",
		},
		{
			name: "SECURITY: Spoofed postgres with different path",
			attrs: map[string]string{
				"process.command": "/home/attacker/postgres -D /data",
			},
			includeList: []string{"/usr/bin/postgres"},
			expected:    false,
			description: "Postgres in unusual location should NOT match full path",
		},
		{
			name: "SECURITY: Mixed include list - full and basename",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
				"process.executable.name": "nginx",
			},
			includeList: []string{"/usr/sbin/nginx", "postgres"},
			expected:    true,
			description: "Should match against full path entry",
		},
		{
			name: "SECURITY: Windows path spoofing prevented",
			attrs: map[string]string{
				"process.command": "C:\\Users\\Public\\nginx.exe",
			},
			includeList: []string{"C:\\Program Files\\nginx\\nginx.exe"},
			expected:    false,
			description: "Windows path spoofing should be prevented with full paths",
		},
		{
			name: "SECURITY: Empty string and basename entries ignored",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/nginx",
				"process.executable.name": "nginx",
			},
			includeList: []string{"", "nginx", ""},
			expected:    false,
			description: "Empty entries and basename-only entries (without path separator) are ignored",
		},
		{
			name: "SECURITY: Case sensitive matching",
			attrs: map[string]string{
				"process.executable.path": "/usr/sbin/NGINX",
			},
			includeList: []string{"/usr/sbin/nginx"},
			expected:    false,
			description: "Path matching should be case sensitive",
		},
		{
			name: "SECURITY: Relative path in include list",
			attrs: map[string]string{
				"process.command": "./nginx",
			},
			includeList: []string{"./nginx"},
			expected:    false,
			description: "Relative paths should not match (no absolute path extracted)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs := pcommon.NewMap()
			for k, v := range tc.attrs {
				attrs.PutStr(k, v)
			}
			result := isProcessInIncludeList(attrs, tc.includeList)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}
