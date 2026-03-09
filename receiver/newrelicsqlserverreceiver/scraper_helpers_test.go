// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newrelicsqlserverreceiver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExecuteScrape(t *testing.T) {
	tests := []struct {
		name         string
		metricName   string
		scrapeFunc   scrapeFunc
		expectError  bool
		expectLog    bool
	}{
		{
			name:       "Successful scrape",
			metricName: "test metrics",
			scrapeFunc: func(ctx context.Context) error {
				return nil
			},
			expectError: false,
			expectLog:   true,
		},
		{
			name:       "Failed scrape",
			metricName: "test metrics",
			scrapeFunc: func(ctx context.Context) error {
				return errors.New("scrape failed")
			},
			expectError: true,
			expectLog:   true,
		},
		{
			name:       "Scrape with context timeout",
			metricName: "slow metrics",
			scrapeFunc: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Second):
					return nil
				}
			},
			expectError: true, // Should timeout
			expectLog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := &sqlServerScraper{
				logger: zap.NewNop(),
				config: &Config{
					Timeout: 100 * time.Millisecond,
				},
			}

			ctx := context.Background()
			err := scraper.executeScrape(ctx, tt.metricName, tt.scrapeFunc)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteConditionalScrape(t *testing.T) {
	tests := []struct {
		name          string
		condition     bool
		metricName    string
		scrapeFunc    scrapeFunc
		expectError   bool
		expectSkipped bool
	}{
		{
			name:       "Condition true - successful scrape",
			condition:  true,
			metricName: "test metrics",
			scrapeFunc: func(ctx context.Context) error {
				return nil
			},
			expectError:   false,
			expectSkipped: false,
		},
		{
			name:       "Condition false - skipped",
			condition:  false,
			metricName: "test metrics",
			scrapeFunc: func(ctx context.Context) error {
				t.Fatal("Should not be called")
				return nil
			},
			expectError:   false,
			expectSkipped: true,
		},
		{
			name:       "Condition true - failed scrape",
			condition:  true,
			metricName: "test metrics",
			scrapeFunc: func(ctx context.Context) error {
				return errors.New("scrape failed")
			},
			expectError:   true,
			expectSkipped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := &sqlServerScraper{
				logger: zap.NewNop(),
				config: &Config{
					Timeout: 1 * time.Second,
				},
			}

			ctx := context.Background()
			err := scraper.executeConditionalScrape(ctx, tt.condition, tt.metricName, tt.scrapeFunc)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectErrors(t *testing.T) {
	tests := []struct {
		name          string
		initialErrors []error
		newError      error
		expectedCount int
	}{
		{
			name:          "Add error to empty slice",
			initialErrors: []error{},
			newError:      errors.New("new error"),
			expectedCount: 1,
		},
		{
			name:          "Add error to existing errors",
			initialErrors: []error{errors.New("error 1"), errors.New("error 2")},
			newError:      errors.New("error 3"),
			expectedCount: 3,
		},
		{
			name:          "Add nil error - should not append",
			initialErrors: []error{errors.New("error 1")},
			newError:      nil,
			expectedCount: 1,
		},
		{
			name:          "Add nil to empty slice",
			initialErrors: []error{},
			newError:      nil,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectErrors(tt.initialErrors, tt.newError)
			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

func TestScrapeWithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		scrapeFunc  scrapeFunc
		expectError bool
	}{
		{
			name:    "Successful scrape within timeout",
			timeout: 1 * time.Second,
			scrapeFunc: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			expectError: false,
		},
		{
			name:    "Scrape exceeds timeout",
			timeout: 50 * time.Millisecond,
			scrapeFunc: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(1 * time.Second):
					return nil
				}
			},
			expectError: true,
		},
		{
			name:    "Scrape returns error",
			timeout: 1 * time.Second,
			scrapeFunc: func(ctx context.Context) error {
				return errors.New("scrape error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := scrapeWithTimeout(ctx, tt.timeout, tt.scrapeFunc)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConcurrentScrape(t *testing.T) {
	tests := []struct {
		name          string
		scrapers      map[string]scrapeFunc
		expectedCount int // Expected number of errors
	}{
		{
			name: "All scrapers succeed",
			scrapers: map[string]scrapeFunc{
				"metric1": func(ctx context.Context) error { return nil },
				"metric2": func(ctx context.Context) error { return nil },
				"metric3": func(ctx context.Context) error { return nil },
			},
			expectedCount: 0,
		},
		{
			name: "Some scrapers fail",
			scrapers: map[string]scrapeFunc{
				"metric1": func(ctx context.Context) error { return nil },
				"metric2": func(ctx context.Context) error { return errors.New("error 2") },
				"metric3": func(ctx context.Context) error { return errors.New("error 3") },
			},
			expectedCount: 2,
		},
		{
			name: "All scrapers fail",
			scrapers: map[string]scrapeFunc{
				"metric1": func(ctx context.Context) error { return errors.New("error 1") },
				"metric2": func(ctx context.Context) error { return errors.New("error 2") },
			},
			expectedCount: 2,
		},
		{
			name:          "Empty scrapers map",
			scrapers:      map[string]scrapeFunc{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := &sqlServerScraper{
				logger: zap.NewNop(),
				config: &Config{
					Timeout: 1 * time.Second,
				},
			}

			ctx := context.Background()
			errors := scraper.concurrentScrape(ctx, tt.scrapers)

			assert.Equal(t, tt.expectedCount, len(errors))
		})
	}
}

func TestConcurrentScrapePerformance(t *testing.T) {
	// Test that concurrent scraping is actually faster than sequential
	scraper := &sqlServerScraper{
		logger: zap.NewNop(),
		config: &Config{
			Timeout: 5 * time.Second,
		},
	}

	// Each scraper takes 100ms
	scrapers := map[string]scrapeFunc{
		"metric1": func(ctx context.Context) error { time.Sleep(100 * time.Millisecond); return nil },
		"metric2": func(ctx context.Context) error { time.Sleep(100 * time.Millisecond); return nil },
		"metric3": func(ctx context.Context) error { time.Sleep(100 * time.Millisecond); return nil },
		"metric4": func(ctx context.Context) error { time.Sleep(100 * time.Millisecond); return nil },
		"metric5": func(ctx context.Context) error { time.Sleep(100 * time.Millisecond); return nil },
	}

	ctx := context.Background()
	start := time.Now()
	errors := scraper.concurrentScrape(ctx, scrapers)
	elapsed := time.Since(start)

	// Sequential would take 500ms, concurrent should take ~100ms
	// Allow some overhead, but should be much faster than 500ms
	assert.Less(t, elapsed, 300*time.Millisecond, "Concurrent scraping should be significantly faster than sequential")
	assert.Equal(t, 0, len(errors), "All scrapers should succeed")
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		connection     *SQLConnection
		setupMock      func() *SQLConnection
		expectError    bool
		expectNilError bool // For nil connection case
	}{
		{
			name:           "Nil connection",
			connection:     nil,
			expectError:    false,
			expectNilError: true,
		},
		// Note: Testing actual SQLConnection would require sqlmock
		// which is better suited for integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := &sqlServerScraper{
				logger:     zap.NewNop(),
				connection: tt.connection,
			}

			ctx := context.Background()
			err := scraper.healthCheck(ctx)

			if tt.expectNilError {
				assert.NoError(t, err)
			} else if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRefreshMetadataCache(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		cache       interface{} // Using interface{} since we can't easily mock MetadataCache
		shouldSkip  bool
	}{
		{
			name: "Enrichment disabled",
			config: &Config{
				EnableWaitResourceEnrichment: false,
			},
			cache:      nil,
			shouldSkip: true,
		},
		{
			name: "Cache is nil",
			config: &Config{
				EnableWaitResourceEnrichment: true,
			},
			cache:      nil,
			shouldSkip: true,
		},
		// Note: Testing actual cache refresh would require a mock cache
		// which is better suited for integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := &sqlServerScraper{
				logger:        zap.NewNop(),
				config:        tt.config,
				metadataCache: nil, // Would be *helpers.MetadataCache in real usage
			}

			ctx := context.Background()
			// Should not panic
			scraper.refreshMetadataCache(ctx)
		})
	}
}

func TestScrapeResultChannel(t *testing.T) {
	// Test that scrapeResult can be sent through channels
	results := make(chan scrapeResult, 1)

	result := scrapeResult{
		MetricName: "test_metric",
		Error:      errors.New("test error"),
	}

	results <- result
	received := <-results

	assert.Equal(t, "test_metric", received.MetricName)
	assert.Error(t, received.Error)
	assert.Equal(t, "test error", received.Error.Error())
}

func TestConcurrentScrapeOrdering(t *testing.T) {
	// Test that concurrent scraping handles scrapers completing in different orders
	scraper := &sqlServerScraper{
		logger: zap.NewNop(),
		config: &Config{
			Timeout: 5 * time.Second,
		},
	}

	scrapers := map[string]scrapeFunc{
		"fast": func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
		"medium": func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
		"slow": func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	ctx := context.Background()
	errors := scraper.concurrentScrape(ctx, scrapers)

	// All should complete successfully regardless of order
	assert.Equal(t, 0, len(errors))
}

func BenchmarkExecuteScrape(b *testing.B) {
	scraper := &sqlServerScraper{
		logger: zap.NewNop(),
		config: &Config{
			Timeout: 1 * time.Second,
		},
	}

	fn := func(ctx context.Context) error {
		return nil
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = scraper.executeScrape(ctx, "benchmark", fn)
	}
}

func BenchmarkConcurrentScrape(b *testing.B) {
	scraper := &sqlServerScraper{
		logger: zap.NewNop(),
		config: &Config{
			Timeout: 1 * time.Second,
		},
	}

	scrapers := map[string]scrapeFunc{
		"metric1": func(ctx context.Context) error { return nil },
		"metric2": func(ctx context.Context) error { return nil },
		"metric3": func(ctx context.Context) error { return nil },
		"metric4": func(ctx context.Context) error { return nil },
		"metric5": func(ctx context.Context) error { return nil },
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = scraper.concurrentScrape(ctx, scrapers)
	}
}

func TestExecuteScrapeContextCancellation(t *testing.T) {
	scraper := &sqlServerScraper{
		logger: zap.NewNop(),
		config: &Config{
			Timeout: 5 * time.Second,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fn := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	err := scraper.executeScrape(ctx, "test", fn)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
