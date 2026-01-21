// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

// This file contains all constant definitions for the adaptivetelemetryprocessor.

const (
	defaultStoragePath        = "/var/lib/nrdot-collector/adaptivetelemetry.db"
	dynamicSmoothingFactor    = 0.2
	dynamicUpdateIntervalSecs = 60
	genericScalingFactor      = 0.2

	// Attribute key denoting which filtering stage allowed the resource through
	// Allowed values: static_threshold | dynamic_threshold | multi_metric | anomaly_detection | anomaly_retention | standard_retention
	adaptiveFilterStageAttributeKey = "process.atp.filter.stage"

	// Multi-metric composite score attributes
	multiMetricCompositeScoreKey = "multi_metric.composite_score"
	multiMetricThresholdKey      = "multi_metric.threshold"

	// Filtering stage values
	stageIncludeList               = "include_list"   // Explicitly included process
	stageZombieProcess             = "zombie_process" // Zombie/Defunct process (always included)
	stageStaticThreshold           = "static_threshold"
	stageDynamicThreshold          = "dynamic_threshold"
	stageMultiMetric               = "multi_metric"
	stageAnomalyDetection          = "anomaly_detection"
	stageAnomalyRetention          = "anomaly_retention"           // Retention after anomaly was detected
	stageStandardRetention         = "standard_retention"          // Retention after threshold exceeded
	stageRetention                 = "retention"                   // Legacy/fallback retention stage
	stageResourceProcessingTimeout = "resource_processing_timeout" // Used for all resource types during timeout

	// Summary metric names with process. prefix to match HOST entity synthesis rules
	filteringEfficiencyRatioMetric   = "process.atp.filter.efficiency_ratio"
	filteringResourceCountMetric     = "process.atp.filter.resource_count"
	filteringThresholdTriggersMetric = "process.atp.filter.threshold_triggers"

	// Summary resource attributes following nr.atp pattern
	atpSourceAttribute     = "process.atp.source"
	atpMetricTypeAttribute = "process.atp.metric_type"
	atpScopeName           = "process.atp.processor"
	atpScopeVersion        = "1.0.0"

	// Attribute values for summary metrics
	atpStatusAttribute = "process.atp.status"
	// for storing just the count of each stage
	atpStageAttribute = "process.atp.stage"
	statusIncluded    = "included"
	statusFiltered    = "filtered"

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
