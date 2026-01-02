package adaptivetelemetryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestIncludeListBypassesAllFilters(t *testing.T) {
	// Create processor with include list
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0, // High threshold
		},
		IncludeProcessList: []string{"nginx", "postgres"},
		EnableStorage:      ptrBool(false),
	}
	cfg.Normalize()

	proc := &processorImp{
		logger:              zap.NewNop(),
		config:              cfg,
		trackedEntities:     make(map[string]*TrackedEntity),
		nextConsumer:        &mockMetricsConsumer{},
		persistenceEnabled:  false,
		dynamicThresholdsEnabled: false,
		multiMetricEnabled:  false,
		dynamicCustomThresholds: make(map[string]float64),
	}

	// Create metrics for nginx (in include list) with low CPU usage
	md := createTestProcessMetrics("nginx", 1234, 5.0) // Well below threshold

	// Process metrics
	ctx := context.Background()
	result, err := proc.processMetrics(ctx, md)
	require.NoError(t, err)

	// Verify nginx is included despite being below threshold
	assert.Equal(t, 1, result.ResourceMetrics().Len())
	
	// Check filter stage attribute
	rm := result.ResourceMetrics().At(0)
	stageVal, ok := rm.Resource().Attributes().Get("process.atp.filter.stage")
	assert.True(t, ok)
	assert.Equal(t, "include_list", stageVal.Str())

	// Verify entity is tracked
	proc.mu.Lock()
	defer proc.mu.Unlock()
	assert.Equal(t, 1, len(proc.trackedEntities))
}

func TestIncludeListWithMultipleProcesses(t *testing.T) {
	// Create processor with include list
	cfg := &Config{
		MetricThresholds: map[string]float64{
			"process.cpu.utilization": 50.0,
		},
		IncludeProcessList: []string{"nginx", "postgres"},
		EnableStorage:      ptrBool(false),
	}
	cfg.Normalize()

	proc := &processorImp{
		logger:              zap.NewNop(),
		config:              cfg,
		trackedEntities:     make(map[string]*TrackedEntity),
		nextConsumer:        &mockMetricsConsumer{},
		persistenceEnabled:  false,
		dynamicThresholdsEnabled: false,
		multiMetricEnabled:  false,
		dynamicCustomThresholds: make(map[string]float64),
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
	ctx := context.Background()
	result, err := proc.processMetrics(ctx, md)
	require.NoError(t, err)

	// Only nginx and postgres should be included (apache filtered out)
	assert.Equal(t, 2, result.ResourceMetrics().Len())

	// Verify both included processes have correct filter stage
	includedProcesses := make(map[string]bool)
	for i := 0; i < result.ResourceMetrics().Len(); i++ {
		rm := result.ResourceMetrics().At(i)
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
		IncludeProcessList: []string{"nginx"},
		EnableStorage:      ptrBool(false),
	}
	cfg.Normalize()

	proc := &processorImp{
		logger:              zap.NewNop(),
		config:              cfg,
		trackedEntities:     make(map[string]*TrackedEntity),
		nextConsumer:        &mockMetricsConsumer{},
		persistenceEnabled:  false,
		dynamicThresholdsEnabled: false,
		multiMetricEnabled:  false,
		dynamicCustomThresholds: make(map[string]float64),
	}

	// Create metrics for apache (not in include list) with high CPU
	md := createTestProcessMetrics("apache2", 9999, 80.0) // Above threshold

	// Process metrics
	ctx := context.Background()
	result, err := proc.processMetrics(ctx, md)
	require.NoError(t, err)

	// Apache should be included because it exceeds threshold
	assert.Equal(t, 1, result.ResourceMetrics().Len())
	
	// Check filter stage - should be static_threshold, not include_list
	rm := result.ResourceMetrics().At(0)
	stageVal, ok := rm.Resource().Attributes().Get("process.atp.filter.stage")
	assert.True(t, ok)
	assert.Equal(t, "static_threshold", stageVal.Str())
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
	
	// Add scope metrics
	sm := rm.ScopeMetrics().AppendEmpty()
	
	// Add CPU utilization metric
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("process.cpu.utilization")
	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetDoubleValue(cpuUtilization)
}

// Mock consumer for testing
type mockMetricsConsumer struct{}

func (m *mockMetricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (m *mockMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return nil
}

// Helper to create bool pointer
func ptrBool(b bool) *bool {
	return &b
}

