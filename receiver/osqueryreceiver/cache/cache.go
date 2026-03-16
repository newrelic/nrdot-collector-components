package cache

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

type CacheManager struct {
	cache      map[string]CachedResult
	cacheMutex sync.RWMutex
	logger     *zap.Logger
}

type CachedResult struct {
	Data     any
	CachedAt time.Time
	TTL      time.Duration
	IsValid  bool
}

func NewCacheManager(logger *zap.Logger) CacheManager {
	return CacheManager{
		cache:  make(map[string]CachedResult),
		logger: logger,
	}
}

// UpdateCache stores collection results in cache
func (m *CacheManager) UpdateCache(collectionName string, data any) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	m.cache[collectionName] = CachedResult{
		Data:     data,
		CachedAt: time.Now(),
		TTL:      5 * time.Minute, // Configurable TTL
		IsValid:  true,
	}

	m.logger.Debug("Updated cache", zap.String("collection", collectionName))
}

// GetCachedResult retrieves cached collection result if valid
func (m *CacheManager) GetCachedResult(collectionName string) (any, bool) {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	cached, exists := m.cache[collectionName]
	if !exists || !cached.IsValid {
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cached.CachedAt) > cached.TTL {
		return nil, false
	}

	return cached.Data, true
}

func (m *CacheManager) InvalidateCache(collectionName string) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	if cached, exists := m.cache[collectionName]; exists {
		cached.IsValid = false
		m.cache[collectionName] = cached
		m.logger.Debug("Invalidated cache", zap.String("collection", collectionName))
	}
}

func (m *CacheManager) GetCacheSize() int {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()
	return len(m.cache)
}
