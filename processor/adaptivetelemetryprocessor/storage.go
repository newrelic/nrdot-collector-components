// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"encoding/json"
	"os"
	"path/filepath"
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
