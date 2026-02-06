// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package adaptivetelemetryprocessor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowsReparsePoint_Junction tests detection of Windows directory junctions
func TestWindowsReparsePoint_Junction(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a target directory
	targetDir := filepath.Join(tmpDir, "target")
	err = os.MkdirAll(targetDir, 0o755)
	require.NoError(t, err)

	// Create a junction pointing to the target directory
	// Windows junctions are created using mklink /J
	junctionDir := filepath.Join(baseDir, "junction")
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionDir, targetDir)
	err = cmd.Run()
	if err != nil {
		// If we can't create junctions (permission issue), skip the test
		t.Skipf("Cannot create directory junction (admin rights may be required): %v", err)
	}

	// Ensure cleanup
	defer func() {
		// Remove junction using rmdir (not del, as del would delete target contents)
		_ = exec.Command("cmd", "/c", "rmdir", junctionDir).Run()
	}()

	// Verify the junction was created
	info, err := os.Lstat(junctionDir)
	require.NoError(t, err)
	// Note: On Windows, junctions may not always appear as regular directories via os.Lstat()
	// The important thing is that they exist and can be detected as reparse points

	// Test that isWindowsReparsePoint detects the junction
	isReparse := isWindowsReparsePoint(junctionDir, info)
	assert.True(t, isReparse, "Junction should be detected as a reparse point")

	// Test that checkPathForSymlinks catches it
	testFile := filepath.Join(junctionDir, "test.db")
	err = checkPathForSymlinks(testFile, baseDir)
	assert.Error(t, err, "checkPathForSymlinks should detect junction")
	assert.Contains(t, err.Error(), "reparse point", "Error should mention reparse point")
}

// TestWindowsReparsePoint_SymbolicLink tests detection of Windows symbolic links
func TestWindowsReparsePoint_SymbolicLink(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a target directory
	targetDir := filepath.Join(tmpDir, "target")
	err = os.MkdirAll(targetDir, 0o755)
	require.NoError(t, err)

	// Create a symbolic link (requires admin rights on Windows < 10, or Developer Mode on Windows 10+)
	symlinkDir := filepath.Join(baseDir, "symlink")
	err = os.Symlink(targetDir, symlinkDir)
	if err != nil {
		// If we can't create symlinks (permission issue), skip the test
		t.Skipf("Cannot create symbolic link (admin rights or Developer Mode may be required): %v", err)
	}

	// Verify the symlink was created
	_, err = os.Lstat(symlinkDir)
	require.NoError(t, err)

	// On Windows, symbolic links can be detected by both os.ModeSymlink and FILE_ATTRIBUTE_REPARSE_POINT
	// Our code checks both, so either method should work

	// Test that checkPathForSymlinks catches it
	testFile := filepath.Join(symlinkDir, "test.db")
	err = checkPathForSymlinks(testFile, baseDir)
	assert.Error(t, err, "checkPathForSymlinks should detect symbolic link")
	// Error message might be "symlink" or "reparse point" depending on which check triggers first
	assert.NotEmpty(t, err.Error(), "Error message should not be empty")
}

// TestWindowsReparsePoint_NormalDirectory tests that normal directories are not flagged
func TestWindowsReparsePoint_NormalDirectory(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	normalDir := filepath.Join(baseDir, "normal", "subdir")
	err := os.MkdirAll(normalDir, 0o755)
	require.NoError(t, err)

	// Get directory info
	info, err := os.Lstat(normalDir)
	require.NoError(t, err)
	require.True(t, info.IsDir(), "Should be a directory")

	// Test that isWindowsReparsePoint does NOT flag normal directories
	isReparse := isWindowsReparsePoint(normalDir, info)
	assert.False(t, isReparse, "Normal directory should NOT be detected as a reparse point")

	// Test that checkPathForSymlinks allows normal directories
	testFile := filepath.Join(normalDir, "test.db")
	err = checkPathForSymlinks(testFile, baseDir)
	assert.NoError(t, err, "checkPathForSymlinks should allow normal directories")
}

// TestWindowsReparsePoint_NormalFile tests that normal files are not flagged
func TestWindowsReparsePoint_NormalFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a normal file
	testFile := filepath.Join(baseDir, "test.db")
	err = os.WriteFile(testFile, []byte("test data"), 0o600)
	require.NoError(t, err)

	// Get file info
	info, err := os.Lstat(testFile)
	require.NoError(t, err)
	require.False(t, info.IsDir(), "Should be a file, not a directory")

	// Test that isWindowsReparsePoint does NOT flag normal files
	// (normal files don't have the FILE_ATTRIBUTE_REPARSE_POINT attribute)
	isReparse := isWindowsReparsePoint(testFile, info)
	assert.False(t, isReparse, "Normal file should NOT be detected as a reparse point")

	// Test that checkPathForSymlinks allows normal files
	err = checkPathForSymlinks(testFile, baseDir)
	assert.NoError(t, err, "checkPathForSymlinks should allow normal files")
}

// TestWindowsReparsePoint_SaveWithJunctionProtection tests that file storage
// prevents writing through junctions
func TestWindowsReparsePoint_SaveWithJunctionProtection(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a target directory (simulating a sensitive location like C:\Windows\System32)
	targetDir := filepath.Join(tmpDir, "sensitive_target")
	err = os.MkdirAll(targetDir, 0o755)
	require.NoError(t, err)

	// Create a junction in our allowed directory pointing to the sensitive location
	junctionDir := filepath.Join(baseDir, "data")
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionDir, targetDir)
	err = cmd.Run()
	if err != nil {
		t.Skipf("Cannot create directory junction (admin rights may be required): %v", err)
	}

	// Ensure cleanup
	defer func() {
		_ = exec.Command("cmd", "/c", "rmdir", junctionDir).Run()
	}()

	// Try to create storage with a path through the junction
	storagePath := filepath.Join(junctionDir, "evil.db")
	storage := &fileStorage{
		filePath:       storagePath,
		allowedBaseDir: baseDir,
		skipValidation: false,
	}

	// Try to save - this should fail due to junction detection
	testData := map[string]*trackedEntity{
		"test": {
			Identity: "test-entity",
		},
	}

	err = storage.Save(testData)
	require.Error(t, err, "Save should fail when path goes through a junction")
	assert.Contains(t, err.Error(), "reparse point", "Error should mention reparse point detection")

	// Verify that no file was written to the target directory
	targetFilePath := filepath.Join(targetDir, "evil.db")
	_, err = os.Stat(targetFilePath)
	assert.True(t, os.IsNotExist(err), "File should not be written through junction")
}

// TestWindowsReparsePoint_MountPoint tests detection of Windows mount points
// Note: Creating mount points requires admin privileges, so this test may be skipped
func TestWindowsReparsePoint_MountPoint(t *testing.T) {
	t.Skip("Mount point creation requires admin privileges - manual testing recommended")

	// This test documents what should be tested manually with admin privileges:
	//
	// 1. Create a mount point using mountvol or diskpart
	// 2. Verify that isWindowsReparsePoint detects it
	// 3. Verify that checkPathForSymlinks rejects paths through it
	//
	// Example setup (requires admin cmd):
	//   mkdir C:\TestMount
	//   mountvol C:\TestMount \\?\Volume{GUID}\
	//
	// Then test that our code detects it as a reparse point
}

// TestWindowsReparsePoint_ValidationIntegration tests the full validation flow
func TestWindowsReparsePoint_ValidationIntegration(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Test 1: Normal path should validate successfully
	normalPath := filepath.Join(baseDir, "data", "state.db")
	err = validateStoragePath(normalPath, nil)
	assert.NoError(t, err, "Normal path should validate successfully")

	// Test 2: Path with junction should fail validation
	targetDir := filepath.Join(tmpDir, "target")
	err = os.MkdirAll(targetDir, 0o755)
	require.NoError(t, err)

	junctionDir := filepath.Join(baseDir, "junction_data")
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionDir, targetDir)
	err = cmd.Run()
	if err != nil {
		t.Skipf("Cannot create directory junction (admin rights may be required): %v", err)
	}
	defer func() {
		_ = exec.Command("cmd", "/c", "rmdir", junctionDir).Run()
	}()

	junctionPath := filepath.Join(junctionDir, "state.db")
	err = validateStoragePath(junctionPath, nil)
	assert.Error(t, err, "Path with junction should fail validation")
	assert.Contains(t, err.Error(), "reparse point", "Error should mention reparse point")
}

// TestWindowsReparsePoint_NestedJunctions tests detection of junctions in nested paths
func TestWindowsReparsePoint_NestedJunctions(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "nrdot-collector")
	err := os.MkdirAll(baseDir, 0o755)
	require.NoError(t, err)

	// Create a normal subdirectory
	normalDir := filepath.Join(baseDir, "data")
	err = os.MkdirAll(normalDir, 0o755)
	require.NoError(t, err)

	// Create a target directory
	targetDir := filepath.Join(tmpDir, "target")
	err = os.MkdirAll(targetDir, 0o755)
	require.NoError(t, err)

	// Create a junction deeper in the path
	junctionDir := filepath.Join(normalDir, "junction_subdir")
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionDir, targetDir)
	err = cmd.Run()
	if err != nil {
		t.Skipf("Cannot create directory junction (admin rights may be required): %v", err)
	}
	defer func() {
		_ = exec.Command("cmd", "/c", "rmdir", junctionDir).Run()
	}()

	// Test that a path through the nested junction is detected
	nestedPath := filepath.Join(junctionDir, "deeper", "state.db")
	err = checkPathForSymlinks(nestedPath, baseDir)
	assert.Error(t, err, "Nested junction should be detected")
	assert.Contains(t, err.Error(), "reparse point", "Error should mention reparse point")
}
