package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

// This file contains all constant definitions for the adaptivetelemetryprocessor.

const (
	defaultStoragePath        = "/var/lib/nrdot-collector/adaptivetelemetry.db"
	dynamicSmoothingFactor    = 0.2
	dynamicUpdateIntervalSecs = 60
	genericScalingFactor      = 0.2

	// Internal attribute key denoting which filtering stage allowed the resource through
	// Used only for internal tracking, removed before export
	internalFilterStageAttributeKey = "ProcessATPFilterStage"

	atpScopeName    = "process.atp.processor"
	atpScopeVersion = "1.0.0"

	// Filtering stage values
	stageIncludeList               = "include_list" // Explicitly included process
	stageStaticThreshold           = "static_threshold"
	stageDynamicThreshold          = "dynamic_threshold"
	stageMultiMetric               = "multi_metric"
	stageAnomalyDetection          = "anomaly_detection"
	stageAnomalyRetention          = "anomaly_retention"           // Retention after anomaly was detected
	stageStandardRetention         = "standard_retention"          // Retention after threshold exceeded
	stageRetention                 = "retention"                   // Legacy/fallback retention stage
	stageResourceProcessingTimeout = "resource_processing_timeout" // Used for all resource types during timeout

	// Hostmetrics resource types
	resourceTypeCPU        = "cpu"
	resourceTypeDisk       = "disk"
	resourceTypeFilesystem = "filesystem"
	resourceTypeLoad       = "load"
	resourceTypeMemory     = "memory"
	resourceTypeNetwork    = "network"
	resourceTypeProcess    = "process"
	resourceTypeProcesses  = "processes"
	resourceTypePaging     = "paging"
	resourceTypeSystem     = "system"

	// Common resource attribute keys
	attrHostID             = "host.id"
	attrHostName           = "host.name"
	attrMetricName         = "metricName"
	attrNewRelicSource     = "newrelic.source"
	attrContainerID        = "container.id"
	attrServiceName        = "service.name"
	attrOtelLibraryName    = "otel.library.name"
	attrOtelLibraryVersion = "otel.library.version"
)
