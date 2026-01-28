// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
		IncludeProcessList:       []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
		EnableStorage:            ptrBool(false),
		DebugShowAllFilterStages: true,
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
	// Using full path to match strict security check
	md := createTestProcessMetrics("/usr/sbin/nginx", 1234, 5.0) // Well below threshold

	// Verify attributes are set correctly
	rm := md.ResourceMetrics().At(0)
	val, ok := rm.Resource().Attributes().Get("process.executable.name")
	require.True(t, ok)
	require.Equal(t, "/usr/sbin/nginx", val.Str())

	// Verify config
	require.Equal(t, []string{"/usr/sbin/nginx", "/usr/bin/postgres"}, proc.config.IncludeProcessList)

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

		stageVal, ok := rm.Resource().Attributes().Get(internalFilterStageAttributeKey)
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
			"process.cpu.utilization": 50.0, // High threshold
		},
		IncludeProcessList:       []string{"/usr/sbin/nginx", "/usr/bin/postgres"},
		EnableStorage:            ptrBool(false),
		DebugShowAllFilterStages: true,
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
	addProcessToMetrics(md, "/usr/sbin/nginx", 1234, 5.0)

	// Add postgres (in include list, low CPU)
	addProcessToMetrics(md, "/usr/bin/postgres", 5678, 10.0)

	// Add apache (not in include list, low CPU)
	addProcessToMetrics(md, "/usr/sbin/apache2", 9999, 8.0)

	// Process metrics
	result, err := proc.processMetrics(t.Context(), md)
	require.NoError(t, err)

	// Verify both included processes have correct filter stage
	// With DebugShowAllFilterStages=true, apache2 might be returned with a debug stage
	includedCount := 0
	includedProcesses := make(map[string]bool)
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		// Skip summary metrics
		if val, ok := rm.Resource().Attributes().Get("process.atp.metric_type"); ok && val.Str() == "filter_summary" {
			continue
		}

		stageVal, ok := rm.Resource().Attributes().Get(internalFilterStageAttributeKey)
		assert.True(t, ok)

		// Only count resources that were included by the include list logic
		if stageVal.Str() == "include_list" {
			includedCount++
			// Track which process was included
			if execName, ok := rm.Resource().Attributes().Get("process.executable.name"); ok {
				includedProcesses[execName.Str()] = true
			}
		}
	}

	// Only nginx and postgres should be included (apache filtered out or debug-included)
	assert.Equal(t, 2, includedCount)

	assert.True(t, includedProcesses["/usr/sbin/nginx"])
	assert.True(t, includedProcesses["/usr/bin/postgres"])
	assert.False(t, includedProcesses["/usr/sbin/apache2"])
}

func TestProcessExceedsThresholdButNotInIncludeList(t *testing.T) {
	// Create processor with include list
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0,
		},
		IncludeProcessList:       []string{"/usr/sbin/nginx"},
		EnableStorage:            ptrBool(false),
		DebugShowAllFilterStages: true,
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
	md := createTestProcessMetrics("/usr/sbin/apache2", 9999, 80.0) // Above threshold

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

		stageVal, ok := rm.Resource().Attributes().Get(internalFilterStageAttributeKey)
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
	case "/usr/sbin/nginx":
		execPath = "/usr/sbin/nginx"
	case "/usr/bin/postgres":
		execPath = "/usr/bin/postgres"
	case "/usr/sbin/apache2":
		execPath = "/usr/sbin/apache2"
	default:
		// If it looks like a path, use it directly
		if len(processName) > 0 && (processName[0] == '/' || (len(processName) > 1 && processName[1] == ':')) {
			execPath = processName
		} else {
			execPath = "/usr/bin/" + processName
		}
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
