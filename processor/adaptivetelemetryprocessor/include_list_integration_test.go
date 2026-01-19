// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestIncludeListBypassesAllFilters(t *testing.T) {
	// Create processor with include list
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0, // High threshold
		},
		IncludeProcessList: []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
		EnableStorage:      ptrBool(false),
	}
	cfg.Normalize()

	proc := &processorImp{
		logger:                   zap.NewNop(),
		config:                   cfg,
		trackedEntities:          make(map[string]*trackedEntity),
		nextConsumer:             &mockMetricsConsumer{},
		persistenceEnabled:       false,
		dynamicThresholdsEnabled: false,
		multiMetricEnabled:       false,
		dynamicCustomThresholds:  make(map[string]float64),
	}

	// Create metrics for nginx (in include list) with low CPU usage
	md := createTestProcessMetrics("nginx", 1234, 5.0) // Well below threshold

	// Process metrics
	result, err := proc.processMetrics(t.Context(), md)
	require.NoError(t, err)

	// Verify nginx is included despite being below threshold
	assert.Equal(t, 1, countNonSummaryResources(result), "Should have 1 resource (excluding summary metrics)")

	// Check filter stage attribute
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		// Skip summary metrics
		if val, ok := rm.Resource().Attributes().Get("process.atp.metric_type"); ok && val.Str() == "filter_summary" {
			continue
		}

		stageVal, ok := rm.Resource().Attributes().Get("process.atp.filter.stage")
		assert.True(t, ok)
		assert.Equal(t, "include_list", stageVal.Str())
	}

	// Verify entity is tracked
	proc.mu.Lock()
	defer proc.mu.Unlock()
	assert.Len(t, proc.trackedEntities, 1)
}

func TestIncludeListWithMultipleProcesses(t *testing.T) {
	// Create processor with include list
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0,
		},
		IncludeProcessList: []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
		EnableStorage:      ptrBool(false),
	}
	cfg.Normalize()

	proc := &processorImp{
		logger:                   zap.NewNop(),
		config:                   cfg,
		trackedEntities:          make(map[string]*trackedEntity),
		nextConsumer:             &mockMetricsConsumer{},
		persistenceEnabled:       false,
		dynamicThresholdsEnabled: false,
		multiMetricEnabled:       false,
		dynamicCustomThresholds:  make(map[string]float64),
	}

	// Create batch with multiple processes
	md := pmetric.NewMetrics()

	// Add nginx (in include list, low CPU)
	addProcessToMetrics(md, "nginx", 1234, 5.0)

	// Add postgres (in include list, low CPU)
	addProcessToMetrics(md, "postgres", 5678, 10.0)

	// Add apache (not in include list, low CPU)
	addProcessToMetrics(md, "apache2", 9999, 8.0)

	// Process metrics
	result, err := proc.processMetrics(t.Context(), md)
	require.NoError(t, err)

	// Only nginx and postgres should be included (apache filtered out)
	assert.Equal(t, 2, countNonSummaryResources(result))

	// Verify both included processes have correct filter stage
	includedProcesses := make(map[string]bool)
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		// Skip summary metrics
		if val, ok := rm.Resource().Attributes().Get("process.atp.metric_type"); ok && val.Str() == "filter_summary" {
			continue
		}

		stageVal, ok := rm.Resource().Attributes().Get("process.atp.filter.stage")
		assert.True(t, ok)
		assert.Equal(t, "include_list", stageVal.Str())

		// Track which process was included
		if execName, ok := rm.Resource().Attributes().Get("process.executable.name"); ok {
			includedProcesses[execName.Str()] = true
		}
	}

	assert.True(t, includedProcesses["nginx"])
	assert.True(t, includedProcesses["postgres"])
	assert.False(t, includedProcesses["apache2"])
}

func TestProcessExceedsThresholdButNotInIncludeList(t *testing.T) {
	// Create processor with include list
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0,
		},
		IncludeProcessList: []string{"/usr/sbin/nginx"},
		EnableStorage:      ptrBool(false),
	}
	cfg.Normalize()

	proc := &processorImp{
		logger:                   zap.NewNop(),
		config:                   cfg,
		trackedEntities:          make(map[string]*trackedEntity),
		nextConsumer:             &mockMetricsConsumer{},
		persistenceEnabled:       false,
		dynamicThresholdsEnabled: false,
		multiMetricEnabled:       false,
		dynamicCustomThresholds:  make(map[string]float64),
	}

	// Create metrics for apache (not in include list) with high CPU
	md := createTestProcessMetrics("apache2", 9999, 80.0) // Above threshold

	// Process metrics
	result, err := proc.processMetrics(t.Context(), md)
	require.NoError(t, err)

	// Apache should be included because it exceeds threshold
	assert.Equal(t, 1, countNonSummaryResources(result))

	// Check filter stage - should be static_threshold, not include_list
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		// Skip summary metrics
		if val, ok := rm.Resource().Attributes().Get("process.atp.metric_type"); ok && val.Str() == "filter_summary" {
			continue
		}

		stageVal, ok := rm.Resource().Attributes().Get("process.atp.filter.stage")
		assert.True(t, ok)
		assert.Equal(t, "static_threshold", stageVal.Str())
	}
}

// Helper function to create test process metrics
func createTestProcessMetrics(processName string, pid int, cpuUtilization float64) pmetric.Metrics {
	md := pmetric.NewMetrics()
	addProcessToMetrics(md, processName, pid, cpuUtilization)
	return md
}

// Helper function to add a process to metrics
func addProcessToMetrics(md pmetric.Metrics, processName string, pid int, cpuUtilization float64) {
	rm := md.ResourceMetrics().AppendEmpty()

	// Set resource attributes
	attrs := rm.Resource().Attributes()
	attrs.PutStr("process.executable.name", processName)
	attrs.PutInt("process.pid", int64(pid))
	attrs.PutStr("host.name", "testhost")

	// Set full executable path based on process name
	// This matches the include list paths used in tests
	var execPath string
	switch processName {
	case "nginx":
		execPath = "/usr/sbin/nginx"
	case "postgres":
		execPath = "/usr/bin/postgres"
	case "apache2":
		execPath = "/usr/sbin/apache2"
	default:
		execPath = "/usr/bin/" + processName
	}
	attrs.PutStr("process.executable.path", execPath)

	// Add scope metrics
	sm := rm.ScopeMetrics().AppendEmpty()

	// Add CPU utilization metric
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("process.cpu.utilization")
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetDoubleValue(cpuUtilization)
}

// Helper to create bool pointer
func ptrBool(b bool) *bool {
	return &b
}
