// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package newrelicsqlserverreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper/scrapererror"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/helpers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/models"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/queries"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/newrelicsqlserverreceiver/scrapers"
)

// sqlServerScraper handles SQL Server metrics collection
type sqlServerScraper struct {
	connection              *SQLConnection
	config                  *Config
	logger                  *zap.Logger
	startTime               pcommon.Timestamp
	settings                receiver.Settings
	mb                      *metadata.MetricsBuilder // Shared MetricsBuilder for all scrapers (Oracle pattern)
	instanceScraper         *scrapers.InstanceScraper
	queryPerformanceScraper *scrapers.QueryPerformanceScraper
	// slowQueryScraper  *scrapers.SlowQueryScraper
	databaseScraper               *scrapers.DatabaseScraper
	userConnectionScraper         *scrapers.UserConnectionScraper
	failoverClusterScraper        *scrapers.FailoverClusterScraper
	databasePrincipalsScraper     *scrapers.DatabasePrincipalsScraper
	databaseRoleMembershipScraper *scrapers.DatabaseRoleMembershipScraper
	waitTimeScraper               *scrapers.WaitTimeScraper         // Add this line
	securityScraper               *scrapers.SecurityScraper         // Security metrics scraper
	lockScraper                   *scrapers.LockScraper             // Lock analysis metrics scraper
	threadPoolHealthScraper       *scrapers.ThreadPoolHealthScraper // Thread pool health monitoring
	tempdbContentionScraper       *scrapers.TempDBContentionScraper // TempDB contention monitoring
	metadataCache                 *helpers.MetadataCache            // Metadata cache for wait resource enrichment
	engineEdition                 int                               // SQL Server engine edition (0=Unknown, 5=Azure DB, 8=Azure MI)
}

// newSqlServerScraper creates a new SQL Server scraper with structured approach
func newSqlServerScraper(settings receiver.Settings, cfg *Config) *sqlServerScraper {
	return &sqlServerScraper{
		config:   cfg,
		logger:   settings.Logger,
		settings: settings,
	}
}

// Start initializes the scraper and establishes database connection
func (s *sqlServerScraper) Start(ctx context.Context, _ component.Host) error {
	s.logger.Info("Starting SQL Server receiver")

	connection, err := NewSQLConnection(ctx, s.config, s.logger)
	if err != nil {
		s.logger.Error("Failed to connect to SQL Server", zap.Error(err))
		return err
	}
	s.connection = connection
	s.startTime = pcommon.NewTimestampFromTime(time.Now())

	if err := s.connection.Ping(ctx); err != nil {
		s.logger.Error("Failed to ping SQL Server", zap.Error(err))
		return err
	}

	// Get EngineEdition
	s.engineEdition = 0 // Default to 0 (Unknown)
	s.engineEdition, err = s.detectEngineEdition(ctx)
	if err != nil {
		s.logger.Debug("Failed to get engine edition, using default", zap.Error(err))
		s.engineEdition = queries.StandardSQLServerEngineEdition
	} else {
		s.logger.Info("Detected SQL Server engine edition",
			zap.Int("engine_edition", s.engineEdition),
			zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))
	}

	// Create ONE MetricsBuilder that will be shared across all scrapers (Oracle pattern)
	s.mb = metadata.NewMetricsBuilder(metadata.DefaultMetricsBuilderConfig(), s.settings)

	// Initialize metadata cache for wait resource enrichment if enabled
	if s.config.EnableWaitResourceEnrichment {
		refreshInterval := time.Duration(s.config.WaitResourceMetadataRefreshMinutes) * time.Minute
		s.metadataCache = helpers.NewMetadataCache(s.connection.Connection.DB, refreshInterval, s.config.MonitoredDatabases)

		// Perform initial cache refresh
		s.logger.Info("Initializing metadata cache for wait resource enrichment",
			zap.Int("refresh_interval_minutes", s.config.WaitResourceMetadataRefreshMinutes),
			zap.Strings("monitored_databases", s.config.MonitoredDatabases))

		if err := s.metadataCache.Refresh(ctx); err != nil {
			s.logger.Warn("Failed to perform initial metadata cache refresh",
				zap.Error(err))
			// Continue - cache will retry on next scrape
		} else {
			stats := s.metadataCache.GetCacheStats()
			s.logger.Info("Metadata cache initialized successfully",
				zap.Int("databases", stats["databases"]),
				zap.Int("objects", stats["objects"]),
				zap.Int("hobts", stats["hobts"]),
				zap.Int("partitions", stats["partitions"]))
		}
	} else {
		s.logger.Info("Wait resource enrichment disabled, skipping metadata cache initialization")
	}

	// Initialize instance scraper with engine edition for engine-specific queries
	// Create instance scraper for instance-level metrics
	s.instanceScraper = scrapers.NewInstanceScraper(s.connection, s.logger, s.mb, s.engineEdition, s.config)

	// Create database scraper for database-level metrics
	s.databaseScraper = scrapers.NewDatabaseScraper(s.connection, s.logger, s.mb, s.engineEdition, s.config)

	// Create failover cluster scraper for Always On Availability Group metrics
	s.failoverClusterScraper = scrapers.NewFailoverClusterScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Create database principals scraper for database security metrics
	s.databasePrincipalsScraper = scrapers.NewDatabasePrincipalsScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Create database role membership scraper for database role and membership metrics
	s.databaseRoleMembershipScraper = scrapers.NewDatabaseRoleMembershipScraper(s.logger, s.connection, s.mb, s.engineEdition)

	// Initialize query performance scraper for blocking sessions and performance monitoring
	// Pass smoothing, interval calculator, and execution plan cache configuration parameters from config
	s.queryPerformanceScraper = scrapers.NewQueryPerformanceScraper(
		s.connection,
		s.logger,
		s.mb,
		s.engineEdition,
		s.config.EnableSlowQuerySmoothing,
		s.config.SlowQuerySmoothingFactor,
		s.config.SlowQuerySmoothingDecayThreshold,
		s.config.SlowQuerySmoothingMaxAgeMinutes,
		s.config.EnableIntervalBasedAveraging,
		s.config.IntervalCalculatorCacheTTLMinutes,
		s.metadataCache,
	)
	// s.slowQueryScraper = scrapers.NewSlowQueryScraper(s.logger, s.connection)

	// Initialize user connection scraper for user connection and authentication metrics
	s.userConnectionScraper = scrapers.NewUserConnectionScraper(s.connection, s.logger, s.engineEdition, s.mb)

	// Initialize wait time scraper for wait time metrics
	s.waitTimeScraper = scrapers.NewWaitTimeScraper(s.connection, s.logger, s.engineEdition, s.mb, s.config)

	// Initialize security scraper for server-level security metrics
	s.securityScraper = scrapers.NewSecurityScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Initialize lock scraper for lock analysis metrics
	s.lockScraper = scrapers.NewLockScraper(s.connection, s.logger, s.mb, s.engineEdition)

	// Initialize thread pool health scraper for thread pool monitoring
	s.threadPoolHealthScraper = scrapers.NewThreadPoolHealthScraper(s.connection, s.logger, s.mb)

	// Initialize TempDB contention scraper for TempDB monitoring
	s.tempdbContentionScraper = scrapers.NewTempDBContentionScraper(s.connection, s.logger, s.mb)

	s.logger.Info("Successfully connected to SQL Server",
		zap.String("hostname", s.config.Hostname),
		zap.String("port", s.config.Port),
		zap.Int("engine_edition", s.engineEdition),
		zap.String("engine_type", queries.GetEngineTypeName(s.engineEdition)))

	return nil
}

// Shutdown closes the database connection
func (s *sqlServerScraper) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down SQL Server receiver")
	if s.connection != nil {
		s.connection.Close()
	}
	return nil
}

// detectEngineEdition detects the SQL Server engine edition following nri-mssql pattern
// detectEngineEdition detects the SQL Server engine edition
func (s *sqlServerScraper) detectEngineEdition(ctx context.Context) (int, error) {
	queryFunc := func(query string) (int, error) {
		var results []struct {
			EngineEdition int `db:"EngineEdition"`
		}

		err := s.connection.Query(ctx, &results, query)
		if err != nil {
			return 0, err
		}

		if len(results) == 0 {
			s.logger.Debug("EngineEdition query returned empty output.")
			return 0, nil
		}

		s.logger.Debug("Detected EngineEdition", zap.Int("engine_edition", results[0].EngineEdition))
		return results[0].EngineEdition, nil
	}

	return queries.DetectEngineEdition(queryFunc)
}

// scrape collects SQL Server instance metrics using structured approach
func (s *sqlServerScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	s.logger.Debug("Starting SQL Server metrics collection",
		zap.String("hostname", s.config.Hostname),
		zap.String("port", s.config.Port))

	// Track scraping errors but continue with partial results
	var scrapeErrors []error

	// Check connection health and refresh metadata cache
	if err := s.healthCheck(ctx); err != nil {
		scrapeErrors = collectErrors(scrapeErrors, fmt.Errorf("connection health check failed: %w", err))
	}
	s.refreshMetadataCache(ctx)

	// === Database Metrics Category ===
<<<<<<< HEAD
	if s.config.EnableDatabaseMetrics {
		// Scrape database-level buffer pool metrics (bufferpool.sizePerDatabaseInBytes)
		if s.config.EnableDatabaseBufferMetrics {
			scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
			defer cancel()

			if err := s.databaseScraper.ScrapeDatabaseBufferMetrics(scrapeCtx); err != nil {
				s.logger.Error("Failed to scrape database buffer metrics",
					zap.Error(err),
					zap.Duration("timeout", s.config.Timeout))
				scrapeErrors = append(scrapeErrors, err)
				// Don't return here - continue with other metrics
			} else {
				s.logger.Debug("Successfully scraped database buffer metrics")
			}
		} else {
			s.logger.Debug("Database buffer metrics scraping SKIPPED - EnableDatabaseBufferMetrics is false")
		}

		// Scrape database-level IO metrics (io.stallInMilliseconds)
		s.logger.Debug("Starting database IO metrics scraping")
		scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabaseIOMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database IO metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database IO metrics")
		}

		// Scrape database-level log growth metrics (log.transactionGrowth)
		s.logger.Debug("Starting database log growth metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabaseLogGrowthMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database log growth metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database log growth metrics")
		}

		// Scrape database-level page file metrics (pageFileAvailable)
		s.logger.Debug("Starting database page file metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabasePageFileMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database page file metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database page file metrics")
		}

		// Scrape database-level page file total metrics (pageFileTotal)
		s.logger.Debug("Starting database page file total metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabasePageFileTotalMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database page file total metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database page file total metrics")
		}

		// Scrape instance-level memory metrics (memoryTotal, memoryAvailable, memoryUtilization)
		s.logger.Debug("Starting database memory metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabaseMemoryMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database memory metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database memory metrics")
		}

		// Scrape database size metrics (total size and data size in MB)
		s.logger.Debug("Starting database size metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabaseSizeMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database size metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database size metrics")
		}

		// Scrape database disk metrics (max disk size for Azure SQL Database)
		s.logger.Debug("Starting database disk metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabaseDiskMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database disk metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database disk metrics")
		}

		// Scrape database transaction log metrics (flushes, bytes flushed, flush waits, active transactions)
		s.logger.Debug("Starting database transaction log metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		if err := s.databaseScraper.ScrapeDatabaseTransactionLogMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape database transaction log metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped database transaction log metrics")
		}

		// Scrape database log space usage metrics (used log space in MB)
		s.logger.Debug("Starting database log space usage metrics scraping")
		scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

	if err := s.databaseScraper.ScrapeDatabaseLogSpaceUsageMetrics(scrapeCtx); err != nil {
		s.logger.Error("Failed to scrape database log space usage metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		scrapeErrors = append(scrapeErrors, err)
		// Don't return here - continue with other metrics
	} else {
		s.logger.Debug("Successfully scraped database log space usage metrics")
=======
	// ALWAYS scrape database metrics - mandatory metrics will always be emitted

	// Scrape database metrics concurrently (independent metrics)
	databaseScrapers := map[string]scrapeFunc{
		"database IO metrics":              s.databaseScraper.ScrapeDatabaseIOMetrics,
		"database log growth metrics":      s.databaseScraper.ScrapeDatabaseLogGrowthMetrics,
		"database page file metrics":       s.databaseScraper.ScrapeDatabasePageFileMetrics,
		"database page file total metrics": s.databaseScraper.ScrapeDatabasePageFileTotalMetrics,
		"database memory metrics":          s.databaseScraper.ScrapeDatabaseMemoryMetrics,
		"database size metrics":            s.databaseScraper.ScrapeDatabaseSizeMetrics,
		"database disk metrics":            s.databaseScraper.ScrapeDatabaseDiskMetrics,
		"database transaction log metrics": s.databaseScraper.ScrapeDatabaseTransactionLogMetrics,
		"database log space usage metrics": s.databaseScraper.ScrapeDatabaseLogSpaceUsageMetrics,
	}

	// Add conditional buffer metrics if enabled
	if s.config.EnableDatabaseBufferMetrics {
		databaseScrapers["database buffer metrics"] = s.databaseScraper.ScrapeDatabaseBufferMetrics
	}

	// Execute all database scrapers concurrently
	dbErrors := s.concurrentScrape(ctx, databaseScrapers)
	for _, err := range dbErrors {
		scrapeErrors = collectErrors(scrapeErrors, err)
>>>>>>> 3e1480d5f8 (feat: add active query top N config and implement cache-first APM metadata extraction)
	}

	// // Scrape blocking session metrics if query monitoring is enabled
	// if s.config.EnableQueryMonitoring {
	// 	scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	// 	defer cancel()

	// 	// Use config values for blocking session parameters
	// 	limit := s.config.QueryMonitoringCountThreshold
	// 	textTruncateLimit := s.config.QueryMonitoringTextTruncateLimit // Use config value

	// 	if err := s.queryPerformanceScraper.ScrapeBlockingSessionMetrics(scrapeCtx, scopeMetrics, limit, textTruncateLimit); err != nil {
	// 		s.logger.Warn("Failed to scrape blocking session metrics - continuing with other metrics",
	// 			zap.Error(err),
	// 			zap.Duration("timeout", s.config.Timeout),
	// 			zap.Int("limit", limit),
	// 			zap.Int("text_truncate_limit", textTruncateLimit))
	// 		// Don't add to scrapeErrors - just warn and continue
	// 	} else {
	// 		s.logger.Debug("Successfully scraped blocking session metrics",
	// 			zap.Int("limit", limit),
	// 			zap.Int("text_truncate_limit", textTruncateLimit))
	// 	}
	// }

	// Scrape slow query metrics if query monitoring is enabled
	// Store query IDs and lightweight plan data (5 fields only) for correlation with active queries
	// Create a fresh APM metadata cache for this scrape cycle
	// This cache will be shared between active and slow query scrapers and discarded at scrape end
	apmMetadataCache := helpers.NewAPMMetadataCache(s.logger)
	s.logger.Debug("Created fresh APM metadata cache for current scrape cycle")

	var slowQueryIDs []string
	var slowQueryPlanDataMap map[string]models.SlowQueryPlanData
	if s.config.EnableQueryMonitoring {
		scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()

		// Use config value for slow query interval only (topN and elapsedTimeThreshold removed)
		intervalSeconds := s.config.QueryMonitoringFetchInterval

		s.logger.Info("Attempting to scrape slow query metrics (NO filters - all queries)",
			zap.Int("interval_seconds", intervalSeconds))

		slowQueries, err := s.queryPerformanceScraper.ScrapeSlowQueryMetrics(scrapeCtx, intervalSeconds, true, apmMetadataCache)
		if err != nil {
			s.logger.Warn("Failed to scrape slow query metrics - continuing with other metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout),
				zap.Int("interval_seconds", intervalSeconds))
			// Don't add to scrapeErrors - just warn and continue
		} else {
			s.logger.Info("Successfully scraped slow query metrics (ALL queries - no filters)",
				zap.Int("interval_seconds", intervalSeconds),
				zap.Int("slow_query_count", len(slowQueries)))

			// Extract query IDs and lightweight plan data (5 fields only) for active query correlation
			slowQueryIDs, slowQueryPlanDataMap = s.queryPerformanceScraper.ExtractQueryDataFromSlowQueries(slowQueries)
			s.logger.Info("Extracted query IDs and lightweight plan data (5 fields only, in-memory)",
				zap.Int("unique_query_id_count", len(slowQueryIDs)),
				zap.Int("plan_data_map_size", len(slowQueryPlanDataMap)))

		}
	} else {
		s.logger.Info("Slow query scraping SKIPPED - EnableQueryMonitoring is false")
	}

	// Scrape active running queries metrics
	scrapeCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Use config values for active running queries parameters
	limit := s.config.QueryMonitoringCountThreshold                           // Reuse count threshold for active queries limit

	s.logger.Info("Attempting to scrape active running queries metrics",
		zap.Int("limit", limit))

	// Step 1: Fetch active queries from database (NO filters: no limit, no threshold, no slow query filter)
	activeQueries, err := s.queryPerformanceScraper.ScrapeActiveRunningQueriesMetrics(scrapeCtx)
	if err != nil {
		s.logger.Warn("Failed to fetch active running queries - continuing with other metrics",
			zap.Error(err),
			zap.Duration("timeout", s.config.Timeout))
		// Don't add to scrapeErrors - just warn and continue
	} else if len(activeQueries) == 0 {
		s.logger.Info("No active queries found (all queries are fetched without filters)")
	} else {
		// Log correlation statistics
		matchedCount := 0
		for _, activeQuery := range activeQueries {
			if activeQuery.QueryID != nil && !activeQuery.QueryID.IsEmpty() {
				queryIDStr := activeQuery.QueryID.String()
				if _, found := slowQueryPlanDataMap[queryIDStr]; found {
					matchedCount++
				}
			}
		}

		s.logger.Info("Active queries fetched with correlation statistics",
			zap.Int("total_active_queries", len(activeQueries)),
			zap.Int("matched_with_slow_queries", matchedCount),
			zap.Int("unmatched_active_queries", len(activeQueries)-matchedCount))

		// Phase 1: Identify active queries missing from slow query map (need backfill)
		// Collect unique query_hashes that don't have plan data yet
		missingQueryHashes := make(map[string]bool) // Use map for automatic deduplication
		for _, activeQuery := range activeQueries {
			if activeQuery.QueryID != nil && !activeQuery.QueryID.IsEmpty() {
				queryIDStr := activeQuery.QueryID.String()
				if _, found := slowQueryPlanDataMap[queryIDStr]; !found {
					// Not found in slow query map - mark for backfill
					missingQueryHashes[queryIDStr] = true
				}
			}
		}

		if len(missingQueryHashes) > 0 {
			s.logger.Info("Identified active queries missing plan data - will attempt backfill",
				zap.Int("missing_count", len(missingQueryHashes)))

			// Phase 2: Backfill missing plan handles from dm_exec_requests / dm_exec_query_stats
			// Convert map keys to slice for backfill function
			missingHashList := make([]string, 0, len(missingQueryHashes))
			for queryHash := range missingQueryHashes {
				missingHashList = append(missingHashList, queryHash)
			}

			// Create a new context for backfill (separate timeout from active query fetch)
			backfillCtx, backfillCancel := context.WithTimeout(ctx, s.config.Timeout)
			defer backfillCancel()

			// Call backfill function to fetch plan_handles for missing queries
			backfilledPlanData, err := s.queryPerformanceScraper.BackfillPlanHandlesForActiveQueries(
				backfillCtx, missingHashList)

			if err != nil {
				s.logger.Warn("Failed to backfill plan handles - continuing without backfill",
					zap.Error(err),
					zap.Int("missing_count", len(missingQueryHashes)))
			} else if len(backfilledPlanData) > 0 {
				// Phase 3: Merge backfilled data into slowQueryPlanDataMap
				originalSize := len(slowQueryPlanDataMap)
				for queryHash, planData := range backfilledPlanData {
					slowQueryPlanDataMap[queryHash] = planData
				}

				s.logger.Info("Merged backfilled plan data into slow query map",
					zap.Int("original_size", originalSize),
					zap.Int("backfilled_count", len(backfilledPlanData)),
					zap.Int("new_size", len(slowQueryPlanDataMap)),
					zap.Int("coverage_percent", (len(slowQueryPlanDataMap)*100)/len(activeQueries)))
			} else {
				s.logger.Info("Backfill completed but no plan handles found (queries not in dm_exec_requests or dm_exec_query_stats)",
					zap.Int("missing_count", len(missingQueryHashes)))
			}
		} else {
			s.logger.Info("All active queries matched with slow queries - no backfill needed")
		}

		// Step 2: Emit metrics for active queries (using lightweight plan data from memory and APM metadata cache)
		if err := s.queryPerformanceScraper.EmitActiveRunningQueriesMetrics(scrapeCtx, activeQueries, slowQueryPlanDataMap, apmMetadataCache); err != nil {
			s.logger.Warn("Failed to emit active running queries metrics",
				zap.Error(err))
		} else {
			s.logger.Info("Successfully emitted active running queries metrics",
				zap.Int("active_query_count", len(activeQueries)))
		}

		// Step 2.5: Emit blocking queries as custom events (metrics → logs via metricsaslogs connector)
		if err := s.queryPerformanceScraper.EmitBlockingQueriesAsCustomEvents(activeQueries); err != nil {
			s.logger.Warn("Failed to emit blocking query events",
				zap.Error(err))
		} else {
			s.logger.Info("Successfully emitted blocking query events")
		}

		// Step 3: Emit execution plan statistics using lightweight plan data from memory (5 fields only, NO database query)
		if err := s.queryPerformanceScraper.ScrapeActiveQueryPlanStatistics(scrapeCtx, activeQueries, slowQueryPlanDataMap); err != nil {
			s.logger.Warn("Failed to scrape active query execution plan statistics - continuing with other metrics",
				zap.Error(err))
			// Don't fail the entire scrape, just log the warning
		} else {
			s.logger.Info("Successfully emitted execution plan statistics as metrics",
				zap.Int("active_query_count", len(activeQueries)))
		}
	}
	} // end EnableDatabaseMetrics

	// === Instance Metrics Category ===
<<<<<<< HEAD
	if s.config.EnableInstanceMetrics {
		s.logger.Debug("Starting instance buffer pool hit percent metrics scraping")
		scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
		if err := s.instanceScraper.ScrapeInstanceComprehensiveStats(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape instance comprehensive statistics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
		s.logger.Debug("Instance comprehensive statistics collection is disabled")
		}
=======
	// ALWAYS scrape instance metrics - mandatory metrics will always be emitted

	// Scrape all instance metrics concurrently (all independent)
	instanceScrapers := map[string]scrapeFunc{
		"instance comprehensive stats":     s.instanceScraper.ScrapeInstanceComprehensiveStats,
		"instance memory metrics":          s.instanceScraper.ScrapeInstanceMemoryMetrics,
		"instance process counts":          s.instanceScraper.ScrapeInstanceProcessCounts,
		"instance runnable tasks":          s.instanceScraper.ScrapeInstanceRunnableTasks,
		"instance active connections":      s.instanceScraper.ScrapeInstanceActiveConnections,
		"instance buffer pool hit percent": s.instanceScraper.ScrapeInstanceBufferPoolHitPercent,
		"instance disk metrics":            s.instanceScraper.ScrapeInstanceDiskMetrics,
		"instance buffer pool size":        s.instanceScraper.ScrapeInstanceBufferPoolSize,
		"instance target memory":           s.instanceScraper.ScrapeInstanceTargetMemoryMetrics,
		"instance performance ratios":      s.instanceScraper.ScrapeInstancePerformanceRatiosMetrics,
		"instance index metrics":           s.instanceScraper.ScrapeInstanceIndexMetrics,
		"instance lock metrics":            s.instanceScraper.ScrapeInstanceLockMetrics,
	}
>>>>>>> 3e1480d5f8 (feat: add active query top N config and implement cache-first APM metadata extraction)

	// Execute all instance scrapers concurrently
	instanceErrors := s.concurrentScrape(ctx, instanceScrapers)
	for _, err := range instanceErrors {
		scrapeErrors = collectErrors(scrapeErrors, err)
	}

	// === User Connection Metrics Category ===
	if s.config.EnableUserConnectionMetrics {
		userConnectionScrapers := map[string]scrapeFunc{
			"user connection summary":        s.userConnectionScraper.ScrapeUserConnectionSummaryMetrics,
			"user connection utilization":    s.userConnectionScraper.ScrapeUserConnectionUtilizationMetrics,
			"user connection by client":      s.userConnectionScraper.ScrapeUserConnectionByClientMetrics,
			"user connection client summary": s.userConnectionScraper.ScrapeUserConnectionClientSummaryMetrics,
			"user connection stats":          s.userConnectionScraper.ScrapeUserConnectionStatsMetrics,
			"login logout summary":           s.userConnectionScraper.ScrapeLoginLogoutSummaryMetrics,
			"failed login summary":           s.userConnectionScraper.ScrapeFailedLoginSummaryMetrics,
		}

		userConnErrors := s.concurrentScrape(ctx, userConnectionScrapers)
		for _, err := range userConnErrors {
			scrapeErrors = collectErrors(scrapeErrors, err)
		}
	} else {
		s.logger.Info("User connection metrics scraping SKIPPED - EnableUserConnectionMetrics is false")
	}

	// === Failover Cluster Metrics Category ===
	if s.config.EnableFailoverClusterMetrics {
		failoverScrapers := map[string]scrapeFunc{
			"failover cluster replica":                  s.failoverClusterScraper.ScrapeFailoverClusterMetrics,
			"failover availability group health":        s.failoverClusterScraper.ScrapeFailoverClusterAvailabilityGroupHealthMetrics,
			"failover availability group configuration": s.failoverClusterScraper.ScrapeFailoverClusterAvailabilityGroupMetrics,
			"failover cluster redo queue":               s.failoverClusterScraper.ScrapeFailoverClusterRedoQueueMetrics,
		}

		failoverErrors := s.concurrentScrape(ctx, failoverScrapers)
		for _, err := range failoverErrors {
			scrapeErrors = collectErrors(scrapeErrors, err)
		}
	} else {
		s.logger.Info("Failover cluster metrics scraping SKIPPED - EnableFailoverClusterMetrics is false")
	}

	// === Database Principals Metrics Category ===
	if s.config.EnableDatabasePrincipalsMetrics {
		principalsScrapers := map[string]scrapeFunc{
			"database principals summary":  s.databasePrincipalsScraper.ScrapeDatabasePrincipalsSummaryMetrics,
			"database principals activity": s.databasePrincipalsScraper.ScrapeDatabasePrincipalActivityMetrics,
		}

		principalsErrors := s.concurrentScrape(ctx, principalsScrapers)
		for _, err := range principalsErrors {
			scrapeErrors = collectErrors(scrapeErrors, err)
		}
	} else {
		s.logger.Info("Database principals metrics scraping SKIPPED - EnableDatabasePrincipalsMetrics is false")
	}

	// === Database Role Membership Metrics Category ===
	if s.config.EnableDatabaseRoleMembershipMetrics {
		roleMembershipScrapers := map[string]scrapeFunc{
			"database role membership summary": s.databaseRoleMembershipScraper.ScrapeDatabaseRoleMembershipSummaryMetrics,
			"database role activity":           s.databaseRoleMembershipScraper.ScrapeDatabaseRoleActivityMetrics,
			"database role permission matrix":  s.databaseRoleMembershipScraper.ScrapeDatabaseRolePermissionMatrixMetrics,
		}

		roleMembershipErrors := s.concurrentScrape(ctx, roleMembershipScrapers)
		for _, err := range roleMembershipErrors {
			scrapeErrors = collectErrors(scrapeErrors, err)
		}
	} else {
		s.logger.Info("Database role membership metrics scraping SKIPPED - EnableDatabaseRoleMembershipMetrics is false")
	}
	} // end EnableInstanceMetrics

	// === Wait Time Metrics Category ===
<<<<<<< HEAD
	if s.config.EnableWaitTimeMetrics {
		// Scrape wait time metrics
		s.logger.Debug("Starting wait time metrics scraping")
		scrapeCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
		if err := s.waitTimeScraper.ScrapeWaitTimeMetrics(scrapeCtx); err != nil {
			s.logger.Error("Failed to scrape wait time metrics",
				zap.Error(err),
				zap.Duration("timeout", s.config.Timeout))
			scrapeErrors = append(scrapeErrors, err)
			// Don't return here - continue with other metrics
		} else {
			s.logger.Debug("Successfully scraped wait time metrics")
		}
=======
	// ALWAYS scrape wait time metrics - mandatory metrics will always be emitted
	waitTimeScrapers := map[string]scrapeFunc{
		"wait time metrics":       s.waitTimeScraper.ScrapeWaitTimeMetrics,
		"latch wait time metrics": s.waitTimeScraper.ScrapeLatchWaitTimeMetrics,
	}
>>>>>>> 3e1480d5f8 (feat: add active query top N config and implement cache-first APM metadata extraction)

	waitTimeErrors := s.concurrentScrape(ctx, waitTimeScrapers)
	for _, err := range waitTimeErrors {
		scrapeErrors = collectErrors(scrapeErrors, err)
	}
	} // end EnableWaitTimeMetrics

	// === Security Metrics Category ===
	if s.config.EnableSecurityMetrics {
		securityScrapers := map[string]scrapeFunc{
			"security principals":   s.securityScraper.ScrapeSecurityPrincipalsMetrics,
			"security role members": s.securityScraper.ScrapeSecurityRoleMembersMetrics,
		}

		securityErrors := s.concurrentScrape(ctx, securityScrapers)
		for _, err := range securityErrors {
			scrapeErrors = collectErrors(scrapeErrors, err)
		}
	} else {
		s.logger.Info("Security metrics scraping SKIPPED - EnableSecurityMetrics is false")
	}

	// === Lock Metrics Category ===
	if s.config.EnableLockMetrics {
		lockScrapers := map[string]scrapeFunc{
			"lock resource metrics": s.lockScraper.ScrapeLockResourceMetrics,
			"lock mode metrics":     s.lockScraper.ScrapeLockModeMetrics,
		}

		lockErrors := s.concurrentScrape(ctx, lockScrapers)
		for _, err := range lockErrors {
			scrapeErrors = collectErrors(scrapeErrors, err)
		}
	} else {
		s.logger.Info("Lock metrics scraping SKIPPED - EnableLockMetrics is false")
	}

	// === Thread Pool Metrics Category ===
	err := s.executeConditionalScrape(ctx, s.config.EnableThreadPoolMetrics,
		"thread pool health metrics", s.threadPoolHealthScraper.ScrapeThreadPoolHealthMetrics)
	scrapeErrors = collectErrors(scrapeErrors, err)

	// === TempDB Metrics Category ===
	err = s.executeConditionalScrape(ctx, s.config.EnableTempDBMetrics,
		"TempDB contention metrics", s.tempdbContentionScraper.ScrapeTempDBContentionMetrics)
	scrapeErrors = collectErrors(scrapeErrors, err)

	// Build final metrics using MetricsBuilder
	metrics := s.buildMetrics(ctx)

	// Log summary of scraping results
	if len(scrapeErrors) > 0 {
		s.logger.Warn("Completed scraping with errors",
			zap.Int("error_count", len(scrapeErrors)),
			zap.Int("metrics_collected", metrics.MetricCount()))

		// Return all errors combined as a PartialScrapeError with partial metrics
		return metrics, scrapererror.NewPartialScrapeError(multierr.Combine(scrapeErrors...), len(scrapeErrors))
	}

	s.logger.Debug("Successfully completed SQL Server metrics collection",
		zap.Int("metrics_collected", metrics.MetricCount()))

	return metrics, nil
}

// buildMetrics constructs the final metrics output with resource attributes
func (s *sqlServerScraper) buildMetrics(ctx context.Context) pmetric.Metrics {
	rb := s.mb.NewResourceBuilder()
	rb.SetServerAddress(fmt.Sprintf("%s:%s", s.config.Hostname, s.config.Port))
	return s.mb.Emit(metadata.WithResource(rb.Emit()))
}

// Helper functions to safely extract values from pointers for logging
func getStringValueFromMap(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func getIntValueFromMap(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getInt64ValueFromMap(ptr *int64) int64 {
	if ptr != nil {
		return *ptr
	}
	return 0
}

func getBoolValueFromMap(ptr *bool) bool {
	if ptr != nil {
		return *ptr
	}
	return false
}

// CollectSystemInformation retrieves comprehensive system and host information
// This information should be included as resource attributes with all metrics
func (s *sqlServerScraper) CollectSystemInformation(ctx context.Context) (*models.SystemInformation, error) {
	s.logger.Debug("Collecting SQL Server system and host information")

	var results []models.SystemInformation
	if err := s.connection.Query(ctx, &results, queries.SystemInformationQuery); err != nil {
		s.logger.Error("Failed to execute system information query",
			zap.Error(err),
			zap.String("query", queries.TruncateQuery(queries.SystemInformationQuery, 100)),
			zap.Int("engine_edition", s.engineEdition))
		return nil, fmt.Errorf("failed to execute system information query: %w", err)
	}

	if len(results) == 0 {
		s.logger.Warn("No results returned from system information query - SQL Server may not be ready")
		return nil, fmt.Errorf("no results returned from system information query")
	}

	if len(results) > 1 {
		s.logger.Warn("Multiple results returned from system information query",
			zap.Int("result_count", len(results)))
	}

	result := results[0]

	// Log collected system information for debugging
	s.logger.Info("Successfully collected system information",
		zap.String("server_name", getStringValueFromMap(result.ServerName)),
		zap.String("computer_name", getStringValueFromMap(result.ComputerName)),
		zap.String("edition", getStringValueFromMap(result.Edition)),
		zap.Int("engine_edition", getIntValueFromMap(result.EngineEdition)),
		zap.String("product_version", getStringValueFromMap(result.ProductVersion)),
		zap.Int("cpu_count", getIntValueFromMap(result.CPUCount)),
		zap.Int64("server_memory_kb", getInt64ValueFromMap(result.ServerMemoryKB)),
		zap.Bool("is_clustered", getBoolValueFromMap(result.IsClustered)),
		zap.Bool("is_hadr_enabled", getBoolValueFromMap(result.IsHadrEnabled)))

	return &result, nil
}
