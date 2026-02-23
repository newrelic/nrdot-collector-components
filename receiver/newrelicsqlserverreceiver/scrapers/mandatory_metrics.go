// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package scrapers

// This file documents mandatory metrics that must ALWAYS be emitted regardless of toggle settings.
// These metrics power the New Relic UI (golden metrics and dashboard metrics).

// MandatoryInstanceMetrics lists instance metrics that must always be emitted
// These are required for golden metrics and dashboard functionality
var MandatoryInstanceMetrics = map[string]bool{
	// Golden Metrics (6 metrics from instance category)
	"sqlserver.stats.connections":                  true, // Golden: Current user connections
	"sqlserver.instance.blocked_processes_count":   true, // Golden: Number of blocked processes
	"sqlserver.instance.connections_active":        true, // Golden: Number of active connections
	"sqlserver.instance.transactions_per_sec":      true, // Golden: Transactions per second
	"sqlserver.stats.sql_compilations_per_sec":     true, // Golden: SQL compilations per second
	"sqlserver.instance.buffer_pool_hit_percent":   true, // Golden: Buffer pool hit percentage

	// Dashboard Metrics (21 metrics from instance category)
	"sqlserver.bufferpool.page_life_expectancy_ms": true, // Dashboard: Page life expectancy
	"sqlserver.instance.memory_utilization_percent": true, // Dashboard: Memory utilization
	"sqlserver.instance.memory_available":          true, // Dashboard: Available memory
	"sqlserver.instance.buffer_pool_size":          true, // Dashboard: Buffer pool size
	"sqlserver.instance.target_memory_kb":          true, // Dashboard: Target memory
	"sqlserver.stats.sql_recompilations_per_sec":   true, // Dashboard: SQL recompilations
	"sqlserver.instance.compilations_per_batch":    true, // Dashboard: Compilations per batch
	"sqlserver.instance.forced_parameterizations_per_sec": true, // Dashboard: Forced parameterizations
	"sqlserver.instance.page_splits_per_batch":     true, // Dashboard: Page splits per batch
	"sqlserver.access.page_splits_per_sec":         true, // Dashboard: Page splits per second
	"sqlserver.instance.full_scans_rate":           true, // Dashboard: Full scans rate
	"sqlserver.buffer.checkpoint_pages_per_sec":    true, // Dashboard: Checkpoint pages
	"sqlserver.stats.lock_waits_per_sec":           true, // Dashboard: Lock waits
	"sqlserver.instance.lock_timeouts_rate":        true, // Dashboard: Lock timeouts
	"sqlserver.stats.deadlocks_per_sec":            true, // Dashboard: Deadlocks
	"sqlserver.stats.user_errors_per_sec":          true, // Dashboard: User errors
	"sqlserver.stats.kill_connection_errors_per_sec": true, // Dashboard: Kill connection errors
}

// MandatoryWaitTimeMetrics lists wait time metrics that must always be emitted
var MandatoryWaitTimeMetrics = map[string]bool{
	// Golden Metrics (1 metric)
	"sqlserver.wait_stats.wait_time_ms": true, // Golden: Total wait time in milliseconds
}

// MandatoryDatabaseMetrics lists database metrics that must always be emitted
var MandatoryDatabaseMetrics = map[string]bool{
	// Golden Metrics (1 metric)
	"sqlserver.database.log.flushes_per_sec": true, // Golden: Log flush operations per second

	// Dashboard Metrics (8 metrics)
	"sqlserver.database.size.total_mb":                   true, // Dashboard: Total database size
	"sqlserver.database.size.data_mb":                    true, // Dashboard: Data file size
	"sqlserver.database.page_file_available_bytes":       true, // Dashboard: Page file available
	"sqlserver.database.page_file_total_bytes":           true, // Dashboard: Page file total
	"sqlserver.database.io.stall_ms":                     true, // Dashboard: IO stall time
	"sqlserver.database.bufferpool.size_per_database_bytes": true, // Dashboard: Buffer pool size per database
	"sqlserver.database.max_disk_size_bytes":             true, // Dashboard: Max disk size
	"sqlserver.database.log.transaction_growth":          true, // Dashboard: Transaction log growth
}



// IsMandatoryMetric checks if a metric name is mandatory (must always be emitted)
func IsMandatoryMetric(metricName string) bool {
	return MandatoryInstanceMetrics[metricName] ||
		MandatoryWaitTimeMetrics[metricName] ||
		MandatoryDatabaseMetrics[metricName] 
}
