// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type EntityStateStorage interface {
	Load() (map[string]*trackedEntity, error)

	Save(map[string]*trackedEntity) error

	Close() error
}

type fileStorage struct {
	filePath string
	mu       sync.Mutex
}

func newFileStorage(filePath string) *fileStorage {
	return &fileStorage{
		filePath: filePath,
	}
}

func (s *fileStorage) Load() (map[string]*trackedEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// Return empty map if file doesn't exist yet
		return make(map[string]*trackedEntity), nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var entities map[string]*trackedEntity
	if err := json.Unmarshal(data, &entities); err != nil {
		return nil, err
	}

	return entities, nil
}

func (s *fileStorage) Save(entities map[string]*trackedEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entities, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0o600)
}

func (*fileStorage) Close() error {
	// No cleanup needed for file storage
	return nil
}

// createDirectoryIfNotExists creates a directory if it doesn't exist
func createDirectoryIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0o700)
	}
	return nil
}

// getAllowedStorageDirectory returns the only allowed directory for storage paths.
// Following Linux Filesystem Hierarchy Standard (FHS), application state data
// should be stored under /var/lib/<appname>/.
// This prevents:
// - Writing to world-writable directories (like /tmp)
// - Path traversal attacks
// - Symlink redirection to sensitive locations
const allowedStorageDirectory = "/var/lib/nrdot-collector/"

func getAllowedStorageDirectory() string {
	return allowedStorageDirectory
}

// validateStoragePath validates that the storage path is secure and within /var/lib/nrdot-collector/.
// Security checks performed:
// 1. Path must be under /var/lib/nrdot-collector/ (no exceptions)
// 2. No component in the path can be a symlink (prevents redirection attacks)
// 3. Path traversal is prevented (no .. escapes)
//
// Parameters:
// - storagePath: The configured storage path to validate
// - additionalAllowedDirs: Ignored - only /var/lib/nrdot-collector/ is allowed
//
// Returns an error if validation fails, nil otherwise.
func validateStoragePath(storagePath string, additionalAllowedDirs []string) error {
	if storagePath == "" {
		return errors.New("storage_path cannot be empty")
	}

	allowedDir := getAllowedStorageDirectory()

	// Clean the path to resolve . and ..
	cleanPath := filepath.Clean(storagePath)

	// Path must be absolute
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("storage_path must be an absolute path under %s, got relative path: %q", allowedDir, storagePath)
	}

	// Check if path is under the allowed directory
	if !strings.HasPrefix(cleanPath+string(filepath.Separator), allowedDir) && cleanPath != strings.TrimSuffix(allowedDir, string(filepath.Separator)) {
		return fmt.Errorf("storage_path must be under %s, got: %q", allowedDir, cleanPath)
	}

	// Check for symlinks in the entire path
	// We need to check each component from /var/lib/nrdot-collector/ onwards
	if err := checkPathForSymlinks(cleanPath, allowedDir); err != nil {
		return fmt.Errorf("symlink detected in storage path: %w", err)
	}

	return nil
}

// checkPathForSymlinks checks if any component in the path starting from the base directory is a symlink.
// This prevents symlink-based redirection attacks.
//
// For example, if path is /var/lib/nrdot-collector/data/test.db and base is /var/lib/nrdot-collector/:
// - We skip checking /var, /var/lib (system directories)
// - We check /var/lib/nrdot-collector/data
// - We check /var/lib/nrdot-collector/data/test.db
//
// If any of these is a symlink, the function returns an error.
func checkPathForSymlinks(path, baseDir string) error {
	// Ensure both paths are clean
	path = filepath.Clean(path)
	baseDir = filepath.Clean(strings.TrimSuffix(baseDir, string(filepath.Separator)))

	// If path equals base, no additional components to check
	if path == baseDir {
		return nil
	}

	// Path must be under baseDir
	if !strings.HasPrefix(path, baseDir+string(filepath.Separator)) {
		return fmt.Errorf("path %q is not under base directory %q", path, baseDir)
	}

	// Get the relative path from base to target
	relPath := strings.TrimPrefix(path, baseDir+string(filepath.Separator))

	// Split the relative path into components
	parts := strings.Split(relPath, string(filepath.Separator))

	// Check each component under the base path
	currentPath := baseDir
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		// Check for path traversal attempts
		if part == ".." {
			return fmt.Errorf("path traversal detected: .. in path")
		}

		currentPath = filepath.Join(currentPath, part)

		// Check if this component is a symlink using Lstat (doesn't follow symlinks)
		info, err := os.Lstat(currentPath)
		if err != nil {
			// If the path doesn't exist yet, that's okay - we'll create it later
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to stat %q: %w", currentPath, err)
		}

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component %q is a symlink", currentPath)
		}
	}

	return nil
}
