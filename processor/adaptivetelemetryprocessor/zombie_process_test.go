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

func TestZombieProcessAlwaysIncluded(t *testing.T) {
	// Create processor with high thresholds
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0, // High threshold
		},
		EnableStorage: ptrBool(false),
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

	// Create metrics for a zombie process with low CPU usage (below threshold)
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()

	// Set resource attributes with process.state = "Z"
	attrs := rm.Resource().Attributes()
	attrs.PutStr("process.executable.name", "defunct-app")
	attrs.PutInt("process.pid", 1234)
	attrs.PutStr("process.state", "Z")
	attrs.PutStr("host.name", "testhost")

	// Add scope metrics - low value
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("process.cpu.utilization")
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetDoubleValue(5.0) // Well below threshold

	// Process metrics
	result, err := proc.processMetrics(t.Context(), md)
	require.NoError(t, err)

	// Verify zombie process is included despite being below threshold
	assert.Equal(t, 1, countNonSummaryResources(result), "Zombie process should be included")

	// Check filter stage attribute
	found := false
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
		// Skip summary metrics
		if val, ok := rm.Resource().Attributes().Get("process.atp.metric_type"); ok && val.Str() == "filter_summary" {
			continue
		}

		found = true
		stageVal, ok := rm.Resource().Attributes().Get("process.atp.filter.stage")
		assert.True(t, ok)
		assert.Equal(t, stageZombieProcess, stageVal.Str())
	}
	assert.True(t, found, "Should have found the process resource")
}

func TestNormalProcessFiltered(t *testing.T) {
	// Create processor with high thresholds
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0,
		},
		EnableStorage: ptrBool(false),
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

	// Create metrics for a normal process (Running) with low CPU usage
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()

	// Set resource attributes with process.state = "R" (Running)
	attrs := rm.Resource().Attributes()
	attrs.PutStr("process.executable.name", "normal-app")
	attrs.PutInt("process.pid", 5678)
	attrs.PutStr("process.state", "R")
	attrs.PutStr("host.name", "testhost")

	// Add scope metrics - low value
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("process.cpu.utilization")
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetDoubleValue(5.0) // Below threshold

	// Process metrics
	result, err := proc.processMetrics(t.Context(), md)
	require.NoError(t, err)

	// Verify normal process is filtered out
	assert.Equal(t, 0, countNonSummaryResources(result), "Normal process below threshold should be filtered")
}
