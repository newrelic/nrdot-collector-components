// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelicsqlserverreceiver

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

// Configuration constants - default values and validation ranges
const (
	// Connection defaults
	defaultHostname = "127.0.0.1"
	defaultPort     = "1433"

	// Concurrency and timeout defaults
	defaultMaxConcurrentWorkers = 10
	defaultTimeout              = 30 * time.Second
	defaultCollectionInterval   = 15 * time.Second

	// Validation ranges
	minTimeout              = 5 * time.Second
	maxTimeout              = 300 * time.Second
	minMaxConcurrentWorkers = 1
	maxMaxConcurrentWorkers = 100
	minCollectionInterval   = 1 * time.Second
	maxCollectionInterval   = 3600 * time.Second

	// Query monitoring defaults
	defaultEnableQueryMonitoring                = true
	defaultQueryMonitoringResponseTimeThreshold = 500 // 500ms = 0.5 seconds (capture queries >= 500ms)
	defaultQueryMonitoringCountThreshold        = 30  // Top 30 slow queries
	defaultQueryMonitoringFetchInterval         = 15
	minQueryMonitoringResponseTimeThreshold     = 0   // 0 = capture all queries (no minimum)
	minQueryMonitoringCountThreshold            = 20
	maxQueryMonitoringCountThreshold            = 50

	// Active queries defaults
	defaultActiveRunningQueriesElapsedTimeThreshold = 0  // 0ms = capture all
	defaultActiveRunningQueriesCountThreshold       = 40 // Top 40 active queries
	minActiveRunningQueriesCountThreshold           = 20
	maxActiveRunningQueriesCountThreshold           = 100

	// Slow query smoothing defaults (EWMA-based)
	defaultEnableSlowQuerySmoothing         = false // Disabled by default
	defaultSlowQuerySmoothingFactor         = 0.3   // 30% new, 70% historical
	defaultSlowQuerySmoothingDecayThreshold = 3     // Remove after 3 misses
	defaultSlowQuerySmoothingMaxAgeMinutes  = 5     // 5 minute max age

	// Interval-based averaging defaults
	defaultEnableIntervalBasedAveraging      = true // Enabled by default
	defaultIntervalCalculatorCacheTTLMinutes = 10   // 10 minute cache TTL

	// Wait resource enrichment defaults
	defaultEnableWaitResourceEnrichment       = true // Enabled by default
	defaultWaitResourceMetadataRefreshMinutes = 5    // 5 minute refresh

	// Database buffer metrics defaults
	defaultEnableDatabaseBufferMetrics = true // Enabled by default

	// Metric category defaults - all enabled by default for backward compatibility
	defaultEnableInstanceMetrics               = true
	defaultEnableDatabaseMetrics               = true
	defaultEnableUserConnectionMetrics         = true
	defaultEnableWaitTimeMetrics               = true
	defaultEnableFailoverClusterMetrics        = true
	defaultEnableDatabasePrincipalsMetrics     = true
	defaultEnableDatabaseRoleMembershipMetrics = true
	defaultEnableSecurityMetrics               = true
	defaultEnableLockMetrics                   = true
	defaultEnableThreadPoolMetrics             = true
	defaultEnableTempDBMetrics                 = true
)

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`

	// Connection configuration
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Hostname string `mapstructure:"hostname"`
	Port     string `mapstructure:"port"`
	Instance string `mapstructure:"instance"`

	// Azure AD authentication
	ClientID     string `mapstructure:"client_id"`
	TenantID     string `mapstructure:"tenant_id"`
	ClientSecret string `mapstructure:"client_secret"`

	// SSL configuration
	EnableSSL              bool   `mapstructure:"enable_ssl"`
	TrustServerCertificate bool   `mapstructure:"trust_server_certificate"`
	CertificateLocation    string `mapstructure:"certificate_location"`

	// Concurrency and timeouts
	MaxConcurrentWorkers int           `mapstructure:"max_concurrent_workers"`
	Timeout              time.Duration `mapstructure:"timeout"`

	// Custom queries
	CustomMetricsQuery  string `mapstructure:"custom_metrics_query"`
	CustomMetricsConfig string `mapstructure:"custom_metrics_config"`

	// Additional connection parameters
	ExtraConnectionURLArgs string `mapstructure:"extra_connection_url_args"`

	// Query monitoring configuration
	EnableQueryMonitoring                bool `mapstructure:"enable_query_monitoring"`
	QueryMonitoringResponseTimeThreshold int  `mapstructure:"query_monitoring_response_time_threshold"` // Minimum elapsed time in milliseconds (default: 500ms, min: 0 = capture all)
	QueryMonitoringCountThreshold        int  `mapstructure:"query_monitoring_count_threshold"`        // Maximum number of slow queries to emit (default: 30, range: 20-50)
	QueryMonitoringFetchInterval         int  `mapstructure:"query_monitoring_fetch_interval"`         // Scrape interval in seconds

	// Active running queries configuration
	ActiveRunningQueriesElapsedTimeThreshold int `mapstructure:"active_running_queries_elapsed_time_threshold"` // Minimum elapsed time in milliseconds (default: 0 = capture all)
	ActiveRunningQueriesCountThreshold       int `mapstructure:"active_running_queries_count_threshold"`        // Maximum number of active queries to fetch (default: 40, range: 20-100)

	// Slow query smoothing configuration (EWMA-based smoothing)
	EnableSlowQuerySmoothing         bool    `mapstructure:"enable_slow_query_smoothing"`          // Enable/disable EWMA smoothing algorithm
	SlowQuerySmoothingFactor         float64 `mapstructure:"slow_query_smoothing_factor"`          // Weight for new data (0.0-1.0, default: 0.3)
	SlowQuerySmoothingDecayThreshold int     `mapstructure:"slow_query_smoothing_decay_threshold"` // Consecutive misses before removal (default: 3)
	SlowQuerySmoothingMaxAgeMinutes  int     `mapstructure:"slow_query_smoothing_max_age_minutes"` // Maximum age in minutes (default: 5)

	// Interval-based averaging configuration (Simplified delta-based interval calculations)
	// This addresses the problem of cumulative averages masking recent query optimizations
	EnableIntervalBasedAveraging      bool `mapstructure:"enable_interval_based_averaging"`       // Enable/disable interval-based averaging
	IntervalCalculatorCacheTTLMinutes int  `mapstructure:"interval_calculator_cache_ttl_minutes"` // State cache TTL in minutes (default: 10)

	// Wait resource enrichment configuration
	// Enriches wait_resource with human-readable names (database names, object names, file names, etc.)
	EnableWaitResourceEnrichment       bool `mapstructure:"enable_wait_resource_enrichment"`        // Enable/disable wait_resource name enrichment
	WaitResourceMetadataRefreshMinutes int  `mapstructure:"wait_resource_metadata_refresh_minutes"` // Metadata cache refresh interval in minutes (default: 5)

	// KEY lock cross-database resolution configuration
	// Specify which databases to resolve KEY lock index names from
	MonitoredDatabases []string `mapstructure:"monitored_databases"` // List of database names for KEY lock resolution (empty = all databases)

	// Database buffer metrics configuration
	EnableDatabaseBufferMetrics bool `mapstructure:"enable_database_buffer_metrics"` // Enable/disable database buffer pool metrics collection

	// Metric Category Toggles - Enable/disable entire categories of metrics
	EnableInstanceMetrics               bool `mapstructure:"enable_instance_metrics"`                 // Enable/disable instance-level metrics (memory, processes, buffer pool, disk)
	EnableDatabaseMetrics               bool `mapstructure:"enable_database_metrics"`                 // Enable/disable database-level metrics (size, IO, transaction logs, page files)
	EnableUserConnectionMetrics         bool `mapstructure:"enable_user_connection_metrics"`          // Enable/disable user connection and authentication metrics
	EnableWaitTimeMetrics               bool `mapstructure:"enable_wait_time_metrics"`                // Enable/disable wait statistics metrics
	EnableFailoverClusterMetrics        bool `mapstructure:"enable_failover_cluster_metrics"`         // Enable/disable Always On Availability Group metrics
	EnableDatabasePrincipalsMetrics     bool `mapstructure:"enable_database_principals_metrics"`      // Enable/disable database security and principal metrics
	EnableDatabaseRoleMembershipMetrics bool `mapstructure:"enable_database_role_membership_metrics"` // Enable/disable database role and membership metrics
	EnableSecurityMetrics               bool `mapstructure:"enable_security_metrics"`                 // Enable/disable server-level security metrics
	EnableLockMetrics                   bool `mapstructure:"enable_lock_metrics"`                     // Enable/disable lock analysis metrics
	EnableThreadPoolMetrics             bool `mapstructure:"enable_thread_pool_metrics"`              // Enable/disable thread pool health monitoring metrics
	EnableTempDBMetrics                 bool `mapstructure:"enable_tempdb_metrics"`                   // Enable/disable TempDB contention monitoring metrics
}

// DefaultConfig returns a Config struct with default values
func DefaultConfig() component.Config {
	cfg := &Config{
		ControllerConfig: scraperhelper.NewDefaultControllerConfig(),

		// Connection settings
		Hostname: defaultHostname,
		Port:     defaultPort,

		// Concurrency and timeout
		MaxConcurrentWorkers: defaultMaxConcurrentWorkers,
		Timeout:              defaultTimeout,

		// SSL settings
		EnableSSL:              false,
		TrustServerCertificate: false,

		// Query monitoring settings
		EnableQueryMonitoring:                defaultEnableQueryMonitoring,
		QueryMonitoringResponseTimeThreshold: defaultQueryMonitoringResponseTimeThreshold,
		QueryMonitoringCountThreshold:        defaultQueryMonitoringCountThreshold,
		QueryMonitoringFetchInterval:         defaultQueryMonitoringFetchInterval,

		// Active running queries settings
		ActiveRunningQueriesElapsedTimeThreshold: defaultActiveRunningQueriesElapsedTimeThreshold,
		ActiveRunningQueriesCountThreshold:       defaultActiveRunningQueriesCountThreshold,

		// Slow query smoothing settings (EWMA-based)
		EnableSlowQuerySmoothing:         defaultEnableSlowQuerySmoothing,
		SlowQuerySmoothingFactor:         defaultSlowQuerySmoothingFactor,
		SlowQuerySmoothingDecayThreshold: defaultSlowQuerySmoothingDecayThreshold,
		SlowQuerySmoothingMaxAgeMinutes:  defaultSlowQuerySmoothingMaxAgeMinutes,

		// Interval-based averaging settings
		EnableIntervalBasedAveraging:      defaultEnableIntervalBasedAveraging,
		IntervalCalculatorCacheTTLMinutes: defaultIntervalCalculatorCacheTTLMinutes,

		// Wait resource enrichment settings
		EnableWaitResourceEnrichment:       defaultEnableWaitResourceEnrichment,
		WaitResourceMetadataRefreshMinutes: defaultWaitResourceMetadataRefreshMinutes,

		// Database buffer metrics settings
		EnableDatabaseBufferMetrics: defaultEnableDatabaseBufferMetrics,

		// Metric category toggle settings
		EnableInstanceMetrics:               defaultEnableInstanceMetrics,
		EnableDatabaseMetrics:               defaultEnableDatabaseMetrics,
		EnableUserConnectionMetrics:         defaultEnableUserConnectionMetrics,
		EnableWaitTimeMetrics:               defaultEnableWaitTimeMetrics,
		EnableFailoverClusterMetrics:        defaultEnableFailoverClusterMetrics,
		EnableDatabasePrincipalsMetrics:     defaultEnableDatabasePrincipalsMetrics,
		EnableDatabaseRoleMembershipMetrics: defaultEnableDatabaseRoleMembershipMetrics,
		EnableSecurityMetrics:               defaultEnableSecurityMetrics,
		EnableLockMetrics:                   defaultEnableLockMetrics,
		EnableThreadPoolMetrics:             defaultEnableThreadPoolMetrics,
		EnableTempDBMetrics:                 defaultEnableTempDBMetrics,
	}

	// Set default collection interval
	cfg.ControllerConfig.CollectionInterval = defaultCollectionInterval

	return cfg
}

// SetDefaults sets default values for configuration fields that are zero-valued
// This ensures consistent defaults even if fields are not explicitly set
func (cfg *Config) SetDefaults() {
	// Connection defaults
	if cfg.Hostname == "" {
		cfg.Hostname = defaultHostname
	}
	if cfg.Port == "" && cfg.Instance == "" {
		cfg.Port = defaultPort
	}

	// Concurrency and timeout defaults
	if cfg.MaxConcurrentWorkers == 0 {
		cfg.MaxConcurrentWorkers = defaultMaxConcurrentWorkers
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.ControllerConfig.CollectionInterval == 0 {
		cfg.ControllerConfig.CollectionInterval = defaultCollectionInterval
	}

	// Query monitoring defaults
	if cfg.QueryMonitoringCountThreshold == 0 {
		cfg.QueryMonitoringCountThreshold = defaultQueryMonitoringCountThreshold
	}
	if cfg.QueryMonitoringFetchInterval == 0 {
		cfg.QueryMonitoringFetchInterval = defaultQueryMonitoringFetchInterval
	}

	// Active running queries defaults
	if cfg.ActiveRunningQueriesCountThreshold == 0 {
		cfg.ActiveRunningQueriesCountThreshold = defaultActiveRunningQueriesCountThreshold
	}

	// Slow query smoothing defaults
	if cfg.SlowQuerySmoothingFactor == 0 {
		cfg.SlowQuerySmoothingFactor = defaultSlowQuerySmoothingFactor
	}
	if cfg.SlowQuerySmoothingDecayThreshold == 0 {
		cfg.SlowQuerySmoothingDecayThreshold = defaultSlowQuerySmoothingDecayThreshold
	}
	if cfg.SlowQuerySmoothingMaxAgeMinutes == 0 {
		cfg.SlowQuerySmoothingMaxAgeMinutes = defaultSlowQuerySmoothingMaxAgeMinutes
	}

	// Interval calculator defaults
	if cfg.IntervalCalculatorCacheTTLMinutes == 0 {
		cfg.IntervalCalculatorCacheTTLMinutes = defaultIntervalCalculatorCacheTTLMinutes
	}

	// Wait resource enrichment defaults
	if cfg.WaitResourceMetadataRefreshMinutes == 0 {
		cfg.WaitResourceMetadataRefreshMinutes = defaultWaitResourceMetadataRefreshMinutes
	}
}

// Validate validates the configuration and sets defaults where needed
func (cfg *Config) Validate() error {
	// Apply defaults first
	cfg.SetDefaults()

	// Hostname validation
	if cfg.Hostname == "" {
		return errors.New("hostname cannot be empty")
	}

	// Port/Instance validation
	if cfg.Port != "" && cfg.Instance != "" {
		return errors.New("specify either port or instance but not both")
	} else if cfg.Port == "" && cfg.Instance == "" {
		// Default to port if neither is specified
		cfg.Port = defaultPort
	}

	// Timeout validation
	if cfg.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if cfg.Timeout < minTimeout {
		return fmt.Errorf("timeout must be at least %v", minTimeout)
	}
	if cfg.Timeout > maxTimeout {
		return fmt.Errorf("timeout must be at most %v", maxTimeout)
	}

	// Concurrency validation
	if cfg.MaxConcurrentWorkers <= 0 {
		return errors.New("max_concurrent_workers must be positive")
	}
	if cfg.MaxConcurrentWorkers < minMaxConcurrentWorkers {
		return fmt.Errorf("max_concurrent_workers must be at least %d", minMaxConcurrentWorkers)
	}
	if cfg.MaxConcurrentWorkers > maxMaxConcurrentWorkers {
		return fmt.Errorf("max_concurrent_workers must be at most %d", maxMaxConcurrentWorkers)
	}

	// Collection interval validation
	if cfg.ControllerConfig.CollectionInterval > 0 {
		if cfg.ControllerConfig.CollectionInterval < minCollectionInterval {
			return fmt.Errorf("collection_interval must be at least %v", minCollectionInterval)
		}
		if cfg.ControllerConfig.CollectionInterval > maxCollectionInterval {
			return fmt.Errorf("collection_interval must be at most %v", maxCollectionInterval)
		}
	}

	// Query monitoring validation
	if cfg.EnableQueryMonitoring {
		if cfg.QueryMonitoringResponseTimeThreshold < minQueryMonitoringResponseTimeThreshold {
			return fmt.Errorf("query_monitoring_response_time_threshold must be >= %d (0 = no threshold)",
				minQueryMonitoringResponseTimeThreshold)
		}
		if cfg.QueryMonitoringCountThreshold < minQueryMonitoringCountThreshold {
			return fmt.Errorf("query_monitoring_count_threshold must be >= %d", minQueryMonitoringCountThreshold)
		}
		if cfg.QueryMonitoringCountThreshold > maxQueryMonitoringCountThreshold {
			return fmt.Errorf("query_monitoring_count_threshold must be <= %d", maxQueryMonitoringCountThreshold)
		}
	}

	// Active running queries validation
	if cfg.ActiveRunningQueriesElapsedTimeThreshold < 0 {
		return errors.New("active_running_queries_elapsed_time_threshold must be >= 0 (0 = no threshold)")
	}
	if cfg.ActiveRunningQueriesCountThreshold < minActiveRunningQueriesCountThreshold {
		return fmt.Errorf("active_running_queries_count_threshold must be >= %d", minActiveRunningQueriesCountThreshold)
	}
	if cfg.ActiveRunningQueriesCountThreshold > maxActiveRunningQueriesCountThreshold {
		return fmt.Errorf("active_running_queries_count_threshold must be <= %d", maxActiveRunningQueriesCountThreshold)
	}

	// SSL validation
	if cfg.EnableSSL && (!cfg.TrustServerCertificate && cfg.CertificateLocation == "") {
		return errors.New("must specify a certificate file when using SSL and not trusting server certificate")
	}

	// Custom metrics validation
	if len(cfg.CustomMetricsConfig) > 0 {
		if len(cfg.CustomMetricsQuery) > 0 {
			return errors.New("cannot specify both custom_metrics_query and custom_metrics_config")
		}
		if _, err := os.Stat(cfg.CustomMetricsConfig); err != nil {
			return fmt.Errorf("custom_metrics_config file error: %w", err)
		}
	}

	return nil
}

// Unmarshal implements the confmap.Unmarshaler interface
func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}
	return cfg.Validate()
}

// GetMaxConcurrentWorkers returns the configured max concurrent workers with fallback
func (cfg *Config) GetMaxConcurrentWorkers() int {
	if cfg.MaxConcurrentWorkers <= 0 {
		return 10 // Default max concurrent workers
	}
	return cfg.MaxConcurrentWorkers
}

// IsAzureADAuth checks if Azure AD Service Principal authentication is configured
func (cfg *Config) IsAzureADAuth() bool {
	return cfg.ClientID != "" && cfg.TenantID != "" && cfg.ClientSecret != ""
}

// CreateConnectionURL creates a connection string for SQL Server authentication
func (cfg *Config) CreateConnectionURL(dbName string) string {
	// Use ADO.NET connection string format instead of URL format
	// This avoids URL encoding issues with special characters in passwords
	connStr := fmt.Sprintf("server=%s", cfg.Hostname)

	// Add port or instance
	if cfg.Port != "" {
		connStr += fmt.Sprintf(";port=%s", cfg.Port)
	} else if cfg.Instance != "" {
		connStr += fmt.Sprintf("\\%s", cfg.Instance)
	}

	// Add authentication
	if cfg.Username != "" && cfg.Password != "" {
		connStr += fmt.Sprintf(";user id=%s;password=%s", cfg.Username, cfg.Password)
	}

	// Add database
	if dbName != "" {
		connStr += fmt.Sprintf(";database=%s", dbName)
	} else {
		connStr += ";database=master"
	}

	// Add timeouts
	connStr += fmt.Sprintf(";dial timeout=%.0f;connection timeout=%.0f",
		cfg.Timeout.Seconds(), cfg.Timeout.Seconds())

	// Add SSL settings
	if cfg.EnableSSL {
		connStr += ";encrypt=true"
		if cfg.TrustServerCertificate {
			connStr += ";TrustServerCertificate=true"
		}
		if !cfg.TrustServerCertificate && cfg.CertificateLocation != "" {
			connStr += fmt.Sprintf(";certificate=%s", cfg.CertificateLocation)
		}
	} else {
		// Explicitly disable encryption when SSL is not enabled
		connStr += ";encrypt=disable"
	}

	// Add extra connection args
	if cfg.ExtraConnectionURLArgs != "" {
		extraArgsMap, err := url.ParseQuery(cfg.ExtraConnectionURLArgs)
		if err == nil {
			for k, v := range extraArgsMap {
				connStr += fmt.Sprintf(";%s=%s", k, v[0])
			}
		}
	}

	return connStr
}

// CreateAzureADConnectionURL creates a connection string for Azure AD authentication
func (cfg *Config) CreateAzureADConnectionURL(dbName string) string {
	connectionString := fmt.Sprintf(
		"server=%s;port=%s;fedauth=ActiveDirectoryServicePrincipal;applicationclientid=%s;clientsecret=%s;database=%s",
		cfg.Hostname,
		cfg.Port,
		cfg.ClientID,     // Client ID
		cfg.ClientSecret, // Client Secret
		dbName,           // Database
	)

	if cfg.ExtraConnectionURLArgs != "" {
		extraArgsMap, err := url.ParseQuery(cfg.ExtraConnectionURLArgs)
		if err == nil {
			for k, v := range extraArgsMap {
				connectionString += fmt.Sprintf(";%s=%s", k, v[0])
			}
		}
	}

	if cfg.EnableSSL {
		connectionString += ";encrypt=true"
		if cfg.TrustServerCertificate {
			connectionString += ";TrustServerCertificate=true"
		} else {
			connectionString += ";TrustServerCertificate=false"
			if cfg.CertificateLocation != "" {
				connectionString += fmt.Sprintf(";certificate=%s", cfg.CertificateLocation)
			}
		}
	}

	return connectionString
}

// GetEnableInstanceMetrics returns whether instance metrics are enabled
func (cfg *Config) GetEnableInstanceMetrics() bool {
	return cfg.EnableInstanceMetrics
}

// GetEnableWaitTimeMetrics returns whether wait time metrics are enabled
func (cfg *Config) GetEnableWaitTimeMetrics() bool {
	return cfg.EnableWaitTimeMetrics
}

// GetEnableDatabaseMetrics returns whether database metrics are enabled
func (cfg *Config) GetEnableDatabaseMetrics() bool {
	return cfg.EnableDatabaseMetrics
}
