// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStoragePath_ValidPaths(t *testing.T) {
	tests := []struct {
		name        string
		storagePath string
		description string
	}{
		{
			name:        "valid path directly under allowed directory",
			storagePath: "/var/lib/nrdot-collector/state.db",
			description: "Files directly under /var/lib/nrdot-collector/ should be allowed",
		},
		{
			name:        "valid nested path",
			storagePath: "/var/lib/nrdot-collector/data/subdir/state.db",
			description: "Nested paths under /var/lib/nrdot-collector/ should be allowed",
		},
		{
			name:        "valid path with multiple levels",
			storagePath: "/var/lib/nrdot-collector/level1/level2/level3/state.db",
			description: "Deep nesting should be allowed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStoragePath(tc.storagePath, nil)
			assert.NoError(t, err, tc.description)
		})
	}
}

func TestValidateStoragePath_InvalidPaths(t *testing.T) {
	tests := []struct {
		name          string
		storagePath   string
		errorContains string
		description   string
	}{
		{
			name:          "empty storage path",
			storagePath:   "",
			errorContains: "cannot be empty",
			description:   "Empty paths should be rejected",
		},
		{
			name:          "relative path",
			storagePath:   "./state.db",
			errorContains: "must be an absolute path",
			description:   "Relative paths should be rejected",
		},
		{
			name:          "relative path with parent traversal",
			storagePath:   "../state.db",
			errorContains: "must be an absolute path",
			description:   "Relative paths with .. should be rejected",
		},
		{
			name:          "path outside allowed directory - /tmp",
			storagePath:   "/tmp/state.db",
			errorContains: "must be under /var/lib/nrdot-collector/",
			description:   "/tmp is not allowed",
		},
		{
			name:          "path outside allowed directory - /etc",
			storagePath:   "/etc/state.db",
			errorContains: "must be under /var/lib/nrdot-collector/",
			description:   "/etc is not allowed",
		},
		{
			name:          "path outside allowed directory - /var/lib/other",
			storagePath:   "/var/lib/other-app/state.db",
			errorContains: "must be under /var/lib/nrdot-collector/",
			description:   "Other directories under /var/lib/ are not allowed",
		},
		{
			name:          "path traversal attempt - parent directory",
			storagePath:   "/var/lib/nrdot-collector/../../../etc/passwd",
			errorContains: "must be under /var/lib/nrdot-collector/",
			description:   "Path traversal with .. should be rejected after cleaning",
		},
		{
			name:          "root directory",
			storagePath:   "/",
			errorContains: "must be under /var/lib/nrdot-collector/",
			description:   "Root directory should be rejected",
		},
		{
			name:          "home directory",
			storagePath:   "/home/user/state.db",
			errorContains: "must be under /var/lib/nrdot-collector/",
			description:   "Home directories should be rejected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStoragePath(tc.storagePath, nil)
			assert.Error(t, err, tc.description)
			if tc.errorContains != "" {
				assert.Contains(t, err.Error(), tc.errorContains)
			}
		})
	}
}

func TestCheckPathForSymlinks(t *testing.T) {
	// Skip on Windows as symlinks require admin privileges
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink tests on Windows")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a real directory
	realDir := filepath.Join(baseDir, "real_directory")
	err = os.MkdirAll(realDir, 0o755)
	require.NoError(t, err)

	// Create a symlink under baseDir
	symlinkPath := filepath.Join(baseDir, "symlink_dir")
	err = os.Symlink(realDir, symlinkPath)
	require.NoError(t, err)

	tests := []struct {
		name          string
		path          string
		baseDir       string
		expectError   bool
		errorContains string
	}{
		{
			name:        "normal path without symlinks",
			path:        filepath.Join(baseDir, "real_directory", "file.db"),
			baseDir:     baseDir,
			expectError: false,
		},
		{
			name:          "path with symlink component",
			path:          filepath.Join(baseDir, "symlink_dir", "file.db"),
			baseDir:       baseDir,
			expectError:   true,
			errorContains: "is a symlink",
		},
		{
			name:        "non-existent path (should not error)",
			path:        filepath.Join(baseDir, "nonexistent", "path", "file.db"),
			baseDir:     baseDir,
			expectError: false,
		},
		{
			name:        "path equals base",
			path:        baseDir,
			baseDir:     baseDir,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPathForSymlinks(tc.path, tc.baseDir)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAllowedStorageDirectory(t *testing.T) {
	allowedDir := getAllowedStorageDirectory()
	assert.Equal(t, "/var/lib/nrdot-collector/", allowedDir)
}

func TestConfigValidation_StoragePath(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		description string
	}{
		{
			name: "valid storage path",
			config: &Config{
				StoragePath:   "/var/lib/nrdot-collector/state.db",
				EnableStorage: ptrBool(true),
			},
			expectError: false,
			description: "Valid path under /var/lib/nrdot-collector/ should pass",
		},
		{
			name: "valid nested storage path",
			config: &Config{
				StoragePath:   "/var/lib/nrdot-collector/data/state.db",
				EnableStorage: ptrBool(true),
			},
			expectError: false,
			description: "Valid nested path should pass",
		},
		{
			name: "invalid storage path in /tmp",
			config: &Config{
				StoragePath:   "/tmp/state.db",
				EnableStorage: ptrBool(true),
			},
			expectError: true,
			description: "Path in /tmp should fail",
		},
		{
			name: "invalid relative path",
			config: &Config{
				StoragePath:   "./state.db",
				EnableStorage: ptrBool(true),
			},
			expectError: true,
			description: "Relative path should fail",
		},
		{
			name: "storage disabled - no validation",
			config: &Config{
				StoragePath:   "/etc/state.db",
				EnableStorage: ptrBool(false),
			},
			expectError: false,
			description: "When storage is disabled, path validation should be skipped",
		},
		{
			name: "invalid storage path in /etc",
			config: &Config{
				StoragePath:   "/etc/state.db",
				EnableStorage: ptrBool(true),
			},
			expectError: true,
			description: "Path in /etc should fail",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.Normalize()
			err := tc.config.Validate()

			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}
		})
	}
}

func TestSymlinkAttackPrevention(t *testing.T) {
	// Skip on Windows as symlinks require admin privileges
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink tests on Windows")
	}

	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a subdirectory
	dataDir := filepath.Join(baseDir, "data")
	err = os.MkdirAll(dataDir, 0o755)
	require.NoError(t, err)

	// Create malicious symlink pointing to /etc
	maliciousSymlink := filepath.Join(dataDir, "evil_link")
	err = os.Symlink("/etc", maliciousSymlink)
	require.NoError(t, err)

	// Try to use a path through the symlink
	attackPath := filepath.Join(maliciousSymlink, "passwd")

	err = checkPathForSymlinks(attackPath, baseDir)
	assert.Error(t, err, "Attack path through symlink should be rejected")
	assert.Contains(t, err.Error(), "is a symlink", "Error should mention symlink detection")
}

func TestPathTraversalPrevention(t *testing.T) {
	// Test that path traversal is prevented by filepath.Clean
	traversalPath := "/var/lib/nrdot-collector/../../etc/passwd"

	err := validateStoragePath(traversalPath, nil)
	assert.Error(t, err, "Path traversal should be rejected")
	assert.Contains(t, err.Error(), "must be under /var/lib/nrdot-collector/")
}

func TestPathValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		storagePath   string
		expectError   bool
		errorContains string
	}{
		{
			name:        "path with trailing slash",
			storagePath: "/var/lib/nrdot-collector/state.db/",
			expectError: false, // filepath.Clean will remove trailing slash
		},
		{
			name:        "path with dot",
			storagePath: "/var/lib/nrdot-collector/./state.db",
			expectError: false, // filepath.Clean handles this
		},
		{
			name:        "double slashes in path",
			storagePath: "/var/lib/nrdot-collector//data//state.db",
			expectError: false, // filepath.Clean handles this
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStoragePath(tc.storagePath, nil)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
