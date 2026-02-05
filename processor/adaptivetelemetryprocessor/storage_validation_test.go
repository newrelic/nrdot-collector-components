// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
		name          string
		storagePath   string
		description   string
		skipOnWindows bool
	}{
		{
			name:          "valid path directly under allowed directory",
			storagePath:   "/var/lib/nrdot-collector/state.db",
			description:   "Files directly under /var/lib/nrdot-collector/ should be allowed",
			skipOnWindows: true,
		},
		{
			name:          "valid nested path",
			storagePath:   "/var/lib/nrdot-collector/data/subdir/state.db",
			description:   "Nested paths under /var/lib/nrdot-collector/ should be allowed",
			skipOnWindows: true,
		},
		{
			name:          "valid path with multiple levels",
			storagePath:   "/var/lib/nrdot-collector/level1/level2/level3/state.db",
			description:   "Deep nesting should be allowed",
			skipOnWindows: true,
		},
	}

	// Add Windows-specific test cases
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		baseDir := filepath.Join(localAppData, "nrdot-collector")

		windowsTests := []struct {
			name        string
			storagePath string
			description string
		}{
			{
				name:        "valid Windows path directly under allowed directory",
				storagePath: filepath.Join(baseDir, "state.db"),
				description: "Files directly under %LOCALAPPDATA%\\nrdot-collector\\ should be allowed",
			},
			{
				name:        "valid Windows nested path",
				storagePath: filepath.Join(baseDir, "data", "subdir", "state.db"),
				description: "Nested paths under %LOCALAPPDATA%\\nrdot-collector\\ should be allowed",
			},
			{
				name:        "valid Windows path with multiple levels",
				storagePath: filepath.Join(baseDir, "level1", "level2", "level3", "state.db"),
				description: "Deep nesting should be allowed on Windows",
			},
		}

		for _, tc := range windowsTests {
			t.Run(tc.name, func(t *testing.T) {
				err := validateStoragePath(tc.storagePath, nil)
				assert.NoError(t, err, tc.description)
			})
		}
	} else {
		// Run Linux tests only on non-Windows
		for _, tc := range tests {
			if tc.skipOnWindows {
				t.Run(tc.name, func(t *testing.T) {
					err := validateStoragePath(tc.storagePath, nil)
					assert.NoError(t, err, tc.description)
				})
			}
		}
	}
}

// TestGetDefaultStoragePath_Windows tests Windows-specific default storage path generation
func TestGetDefaultStoragePath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	defaultPath := getDefaultStoragePath()

	// Verify it's an absolute path
	assert.True(t, filepath.IsAbs(defaultPath), "Default path should be absolute on Windows")

	// Verify it contains the expected directory
	assert.Contains(t, defaultPath, "nrdot-collector", "Default path should contain nrdot-collector directory")

	// Verify it contains the database filename
	assert.Contains(t, defaultPath, "adaptiveprocess.db", "Default path should contain the database filename")

	// Verify it's in the user's local app data
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		assert.Contains(t, defaultPath, localAppData, "Default path should be under LOCALAPPDATA")
	}
}

// TestCreateDirectoryIfNotExists_Windows tests directory creation on Windows
func TestCreateDirectoryIfNotExists_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	// Use a temporary directory for testing
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_dir", "nested", "deep")

	// Directory should not exist yet
	_, err := os.Stat(testDir)
	assert.True(t, os.IsNotExist(err), "Test directory should not exist initially")

	// Create the directory
	err = createDirectoryIfNotExists(testDir)
	assert.NoError(t, err, "createDirectoryIfNotExists should succeed")

	// Verify directory was created
	info, err := os.Stat(testDir)
	assert.NoError(t, err, "Directory should exist after creation")
	assert.True(t, info.IsDir(), "Created path should be a directory")

	// Calling again should be idempotent
	err = createDirectoryIfNotExists(testDir)
	assert.NoError(t, err, "createDirectoryIfNotExists should be idempotent")
}

func TestValidateStoragePath_InvalidPaths(t *testing.T) {
	tests := []struct {
		name          string
		storagePath   string
		errorContains string
		description   string
		skipOnWindows bool
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
			errorContains: "must be under",
			description:   "/tmp is not allowed",
			skipOnWindows: true,
		},
		{
			name:          "path outside allowed directory - /etc",
			storagePath:   "/etc/state.db",
			errorContains: "must be under",
			description:   "/etc is not allowed",
			skipOnWindows: true,
		},
		{
			name:          "path outside allowed directory - /var/lib/other",
			storagePath:   "/var/lib/other-app/state.db",
			errorContains: "must be under",
			description:   "Other directories under /var/lib/ are not allowed",
			skipOnWindows: true,
		},
		{
			name:          "path traversal attempt - parent directory",
			storagePath:   "/var/lib/nrdot-collector/../../../etc/passwd",
			errorContains: "must be under",
			description:   "Path traversal with .. should be rejected after cleaning",
			skipOnWindows: true,
		},
		{
			name:          "root directory",
			storagePath:   "/",
			errorContains: "must be under",
			description:   "Root directory should be rejected",
			skipOnWindows: true,
		},
		{
			name:          "home directory",
			storagePath:   "/home/user/state.db",
			errorContains: "must be under",
			description:   "Home directories should be rejected",
			skipOnWindows: true,
		},
	}

	// Add Windows-specific invalid paths
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		allowedDir := filepath.Join(localAppData, "nrdot-collector")

		windowsTests := []struct {
			name          string
			storagePath   string
			errorContains string
			description   string
		}{
			{
				name:          "Windows path outside allowed directory - C:\\Temp",
				storagePath:   "C:\\Temp\\state.db",
				errorContains: "must be under",
				description:   "C:\\Temp is not allowed on Windows",
			},
			{
				name:          "Windows path outside allowed directory - C:\\Windows",
				storagePath:   "C:\\Windows\\state.db",
				errorContains: "must be under",
				description:   "C:\\Windows is not allowed on Windows",
			},
			{
				name:          "Windows path traversal attempt",
				storagePath:   filepath.Join(allowedDir, "..", "..", "Windows", "System32", "config"),
				errorContains: "must be under",
				description:   "Path traversal should be rejected on Windows",
			},
			{
				name:          "Windows path in Program Files",
				storagePath:   "C:\\Program Files\\nrdot-collector\\state.db",
				errorContains: "must be under",
				description:   "Program Files is not allowed on Windows",
			},
			{
				name:          "Windows path in different user's AppData",
				storagePath:   "C:\\Users\\OtherUser\\AppData\\Local\\nrdot-collector\\state.db",
				errorContains: "must be under",
				description:   "Different user's AppData should be rejected on Windows",
			},
		}

		for _, tc := range windowsTests {
			t.Run(tc.name, func(t *testing.T) {
				err := validateStoragePath(tc.storagePath, nil)
				assert.Error(t, err, tc.description)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			})
		}
	}

	// Run platform-appropriate tests
	for _, tc := range tests {
		if runtime.GOOS == "windows" && tc.skipOnWindows {
			continue
		}
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

	if runtime.GOOS == "windows" {
		// On Windows, should return %LOCALAPPDATA%\nrdot-collector\
		assert.Contains(t, allowedDir, "nrdot-collector")
		assert.True(t, filepath.IsAbs(allowedDir), "Windows path should be absolute")
		assert.True(t, len(allowedDir) > 0, "Windows path should not be empty")
	} else {
		// On Linux/Unix, should return /var/lib/nrdot-collector/
		assert.Equal(t, "/var/lib/nrdot-collector/", allowedDir)
	}
}

func TestConfigValidation_StoragePath(t *testing.T) {
	// Get platform-specific paths
	var validPath1, validPath2, invalidPath1, invalidPath2 string

	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		baseDir := filepath.Join(localAppData, "nrdot-collector")
		validPath1 = filepath.Join(baseDir, "state.db")
		validPath2 = filepath.Join(baseDir, "data", "state.db")
		invalidPath1 = "C:\\Temp\\state.db"
		invalidPath2 = "C:\\Windows\\state.db"
	} else {
		validPath1 = "/var/lib/nrdot-collector/state.db"
		validPath2 = "/var/lib/nrdot-collector/data/state.db"
		invalidPath1 = "/tmp/state.db"
		invalidPath2 = "/etc/state.db"
	}

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		description string
	}{
		{
			name: "valid storage path",
			config: &Config{
				StoragePath:   validPath1,
				EnableStorage: ptrBool(true),
			},
			expectError: false,
			description: "Valid path under allowed directory should pass",
		},
		{
			name: "valid nested storage path",
			config: &Config{
				StoragePath:   validPath2,
				EnableStorage: ptrBool(true),
			},
			expectError: false,
			description: "Valid nested path should pass",
		},
		{
			name: "invalid storage path 1",
			config: &Config{
				StoragePath:   invalidPath1,
				EnableStorage: ptrBool(true),
			},
			expectError: true,
			description: "Path outside allowed directory should fail",
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
				StoragePath:   invalidPath2,
				EnableStorage: ptrBool(false),
			},
			expectError: false,
			description: "When storage is disabled, path validation should be skipped",
		},
		{
			name: "invalid storage path 2",
			config: &Config{
				StoragePath:   invalidPath2,
				EnableStorage: ptrBool(true),
			},
			expectError: true,
			description: "Path outside allowed directory should fail",
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
	var traversalPath string
	var expectedErrorSubstring string

	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		baseDir := filepath.Join(localAppData, "nrdot-collector")
		traversalPath = filepath.Join(baseDir, "..", "..", "Windows", "System32", "config")
		expectedErrorSubstring = "must be under"
	} else {
		traversalPath = "/var/lib/nrdot-collector/../../etc/passwd"
		expectedErrorSubstring = "must be under /var/lib/nrdot-collector/"
	}

	err := validateStoragePath(traversalPath, nil)
	assert.Error(t, err, "Path traversal should be rejected")
	assert.Contains(t, err.Error(), expectedErrorSubstring)
}

func TestPathValidation_EdgeCases(t *testing.T) {
	// Get platform-specific base path
	var basePath string
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		basePath = filepath.Join(localAppData, "nrdot-collector")
	} else {
		basePath = "/var/lib/nrdot-collector"
	}

	tests := []struct {
		name          string
		storagePath   string
		expectError   bool
		errorContains string
	}{
		{
			name:        "path with trailing slash",
			storagePath: filepath.Join(basePath, "state.db") + string(filepath.Separator),
			expectError: false, // filepath.Clean will remove trailing slash
		},
		{
			name:        "path with dot",
			storagePath: filepath.Join(basePath, ".", "state.db"),
			expectError: false, // filepath.Clean handles this
		},
	}

	// Add Linux-specific test for double slashes
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name          string
			storagePath   string
			expectError   bool
			errorContains string
		}{
			name:        "double slashes in path",
			storagePath: "/var/lib/nrdot-collector//data//state.db",
			expectError: false, // filepath.Clean handles this
		})
	}

	// Add Windows-specific edge cases
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			name          string
			storagePath   string
			expectError   bool
			errorContains string
		}{
			{
				name:        "Windows path with forward slashes",
				storagePath: basePath + "/data/state.db",
				expectError: false, // filepath.Clean handles mixed separators
			},
			{
				name:        "Windows path with mixed separators",
				storagePath: basePath + "\\data/subdir\\state.db",
				expectError: false, // filepath.Clean normalizes this
			},
		}
		tests = append(tests, windowsTests...)
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
