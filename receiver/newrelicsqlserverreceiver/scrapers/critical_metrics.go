// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package scrapers

// This file documents critical metrics that must ALWAYS be emitted regardless of toggle settings.
// These metrics power the New Relic UI (golden metrics and dashboard metrics).
//
// Critical Metrics (37 total):
//   - 6 Golden instance metrics (main entity health indicators)
//   - 21 Dashboard instance metrics (out-of-box dashboard requirements)
//   - 1 Golden wait time metric
//   - 1 Golden database metric
//   - 8 Dashboard database metrics
//
// These metrics are emitted even when feature toggles are disabled to ensure
// New Relic UI functionality remains intact.

// Metric name constants for instance metrics
const (
	// Golden instance metrics
	MetricStatsConnections              = "sqlserver.stats.connections"
	MetricInstanceBlockedProcessesCount = "sqlserver.instance.blocked_processes_count"
	MetricInstanceConnectionsActive     = "sqlserver.instance.connections_active"
	MetricInstanceTransactionsPerSec    = "sqlserver.instance.transactions_per_sec"
	MetricStatsSQLCompilationsPerSec    = "sqlserver.stats.sql_compilations_per_sec"
	MetricInstanceBufferPoolHitPercent  = "sqlserver.instance.buffer_pool_hit_percent"

	// Dashboard instance metrics
	MetricBufferpoolPageLifeExpectancyMs          = "sqlserver.bufferpool.page_life_expectancy_ms"
	MetricInstanceMemoryUtilizationPercent        = "sqlserver.instance.memory_utilization_percent"
	MetricInstanceMemoryAvailable                 = "sqlserver.instance.memory_available"
	MetricInstanceBufferPoolSize                  = "sqlserver.instance.buffer_pool_size"
	MetricInstanceTargetMemoryKb                  = "sqlserver.instance.target_memory_kb"
	MetricStatsSQLRecompilationsPerSec            = "sqlserver.stats.sql_recompilations_per_sec"
	MetricInstanceCompilationsPerBatch            = "sqlserver.instance.compilations_per_batch"
	MetricInstanceForcedParameterizationsPerSec   = "sqlserver.instance.forced_parameterizations_per_sec"
	MetricInstancePageSplitsPerBatch              = "sqlserver.instance.page_splits_per_batch"
	MetricAccessPageSplitsPerSec                  = "sqlserver.access.page_splits_per_sec"
	MetricInstanceFullScansRate                   = "sqlserver.instance.full_scans_rate"
	MetricBufferCheckpointPagesPerSec             = "sqlserver.buffer.checkpoint_pages_per_sec"
	MetricStatsLockWaitsPerSec                    = "sqlserver.stats.lock_waits_per_sec"
	MetricInstanceLockTimeoutsRate                = "sqlserver.instance.lock_timeouts_rate"
	MetricStatsDeadlocksPerSec                    = "sqlserver.stats.deadlocks_per_sec"
	MetricStatsUserErrorsPerSec                   = "sqlserver.stats.user_errors_per_sec"
	MetricStatsKillConnectionErrorsPerSec         = "sqlserver.stats.kill_connection_errors_per_sec"

	// Wait time metrics
	MetricWaitStatsWaitTimeMs = "sqlserver.wait_stats.wait_time_ms"

	// Database metrics
	MetricDatabaseLogFlushesPerSec                 = "sqlserver.database.log.flushes_per_sec"
	MetricDatabaseSizeTotalMb                      = "sqlserver.database.size.total_mb"
	MetricDatabaseSizeDataMb                       = "sqlserver.database.size.data_mb"
	MetricDatabasePageFileAvailableBytes           = "sqlserver.database.page_file_available_bytes"
	MetricDatabasePageFileTotalBytes               = "sqlserver.database.page_file_total_bytes"
	MetricDatabaseIoStallMs                        = "sqlserver.database.io.stall_ms"
	MetricDatabaseBufferpoolSizePerDatabaseBytes   = "sqlserver.database.bufferpool.size_per_database_bytes"
	MetricDatabaseMaxDiskSizeBytes                 = "sqlserver.database.max_disk_size_bytes"
	MetricDatabaseLogTransactionGrowth             = "sqlserver.database.log.transaction_growth"
)

// CriticalInstanceMetrics lists instance metrics that must always be emitted
// These are required for golden metrics and dashboard functionality
// NOTE: This map is for documentation and validation purposes only.
// The actual emission logic uses grouped blocks for performance (no runtime map lookups).
var CriticalInstanceMetrics = map[string]bool{
	// Golden Metrics (6 metrics from instance category)
	MetricStatsConnections:              true, // Golden: Current user connections
	MetricInstanceBlockedProcessesCount: true, // Golden: Number of blocked processes
	MetricInstanceConnectionsActive:     true, // Golden: Number of active connections
	MetricInstanceTransactionsPerSec:    true, // Golden: Transactions per second
	MetricStatsSQLCompilationsPerSec:    true, // Golden: SQL compilations per second
	MetricInstanceBufferPoolHitPercent:  true, // Golden: Buffer pool hit percentage

	// Dashboard Metrics (21 metrics from instance category)
	MetricBufferpoolPageLifeExpectancyMs:        true, // Dashboard: Page life expectancy
	MetricInstanceMemoryUtilizationPercent:      true, // Dashboard: Memory utilization
	MetricInstanceMemoryAvailable:               true, // Dashboard: Available memory
	MetricInstanceBufferPoolSize:                true, // Dashboard: Buffer pool size
	MetricInstanceTargetMemoryKb:                true, // Dashboard: Target memory
	MetricStatsSQLRecompilationsPerSec:          true, // Dashboard: SQL recompilations
	MetricInstanceCompilationsPerBatch:          true, // Dashboard: Compilations per batch
	MetricInstanceForcedParameterizationsPerSec: true, // Dashboard: Forced parameterizations
	MetricInstancePageSplitsPerBatch:            true, // Dashboard: Page splits per batch
	MetricAccessPageSplitsPerSec:                true, // Dashboard: Page splits per second
	MetricInstanceFullScansRate:                 true, // Dashboard: Full scans rate
	MetricBufferCheckpointPagesPerSec:           true, // Dashboard: Checkpoint pages
	MetricStatsLockWaitsPerSec:                  true, // Dashboard: Lock waits
	MetricInstanceLockTimeoutsRate:              true, // Dashboard: Lock timeouts
	MetricStatsDeadlocksPerSec:                  true, // Dashboard: Deadlocks
	MetricStatsUserErrorsPerSec:                 true, // Dashboard: User errors
	MetricStatsKillConnectionErrorsPerSec:       true, // Dashboard: Kill connection errors
}

// CriticalWaitTimeMetrics lists wait time metrics that must always be emitted
// NOTE: This map is for documentation and validation purposes only.
var CriticalWaitTimeMetrics = map[string]bool{
	// Golden Metrics (1 metric)
	MetricWaitStatsWaitTimeMs: true, // Golden: Total wait time in milliseconds
}

// CriticalDatabaseMetrics lists database metrics that must always be emitted
// NOTE: This map is for documentation and validation purposes only.
var CriticalDatabaseMetrics = map[string]bool{
	// Golden Metrics (1 metric)
	MetricDatabaseLogFlushesPerSec: true, // Golden: Log flush operations per second

	// Dashboard Metrics (8 metrics)
	MetricDatabaseSizeTotalMb:                    true, // Dashboard: Total database size
	MetricDatabaseSizeDataMb:                     true, // Dashboard: Data file size
	MetricDatabasePageFileAvailableBytes:         true, // Dashboard: Page file available
	MetricDatabasePageFileTotalBytes:             true, // Dashboard: Page file total
	MetricDatabaseIoStallMs:                      true, // Dashboard: IO stall time
	MetricDatabaseBufferpoolSizePerDatabaseBytes: true, // Dashboard: Buffer pool size per database
	MetricDatabaseMaxDiskSizeBytes:               true, // Dashboard: Max disk size
	MetricDatabaseLogTransactionGrowth:           true, // Dashboard: Transaction log growth
}



// IsCriticalMetric checks if a metric name is critical (must always be emitted)
// NOTE: This function is for testing and validation purposes only.
// Production code uses grouped blocks instead of runtime map lookups for performance.
func IsCriticalMetric(metricName string) bool {
	return CriticalInstanceMetrics[metricName] ||
		CriticalWaitTimeMetrics[metricName] ||
		CriticalDatabaseMetrics[metricName]
}
