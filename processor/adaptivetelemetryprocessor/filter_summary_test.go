package adaptivetelemetryprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// TestGenerateFilteringSummaryMetrics tests the generation of filter summary metrics
func TestGenerateFilteringSummaryMetrics(t *testing.T) {
	logger := zap.NewNop()
	processor := &processorImp{
		logger: logger,
		config: &Config{},
	}

	tests := []struct {
		name                   string
		inputResourceCount     int
		outputResourceCount    int
		inputMetricCount       int
		outputMetricCount      int
		stageHits              map[string]int
		expectedMetricsCount   int
		expectedEfficiency     float64
		validateSummaryMetrics func(t *testing.T, filteredMetrics pmetric.Metrics)
	}{
		{
			name:                 "Empty batch - no summary generated",
			inputResourceCount:   0,
			outputResourceCount:  0,
			inputMetricCount:     0,
			outputMetricCount:    0,
			stageHits:            map[string]int{},
			expectedMetricsCount: 0,
			validateSummaryMetrics: func(t *testing.T, filteredMetrics pmetric.Metrics) {
				// No summary metrics should be generated for empty batch
				assert.Equal(t, 0, filteredMetrics.ResourceMetrics().Len())
			},
		},
		{
			name:                 "All resources filtered out",
			inputResourceCount:   10,
			outputResourceCount:  0,
			inputMetricCount:     100,
			outputMetricCount:    0,
			stageHits:            map[string]int{},
			expectedMetricsCount: 2,   // efficiency_ratio + resource_count
			expectedEfficiency:   1.0, // 100% filtered
			validateSummaryMetrics: func(t *testing.T, filteredMetrics pmetric.Metrics) {
				assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())

				summaryRM := filteredMetrics.ResourceMetrics().At(0)
				validateSummaryResource(t, summaryRM)

				// Should have 2 metrics: efficiency_ratio + resource_count
				assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
				sm := summaryRM.ScopeMetrics().At(0)
				assert.Equal(t, 2, sm.Metrics().Len())

				// Validate efficiency metric
				efficiencyMetric := findMetricByName(sm, filteringEfficiencyRatioMetric)
				assert.NotNil(t, efficiencyMetric)
				assert.Equal(t, 1.0, efficiencyMetric.Gauge().DataPoints().At(0).DoubleValue())

				// Validate resource count metric
				resourceCountMetric := findMetricByName(sm, filteringResourceCountMetric)
				assert.NotNil(t, resourceCountMetric)
				validateResourceCountMetric(t, resourceCountMetric, 0, 10)
			},
		},
		{
			name:                 "No resources filtered",
			inputResourceCount:   5,
			outputResourceCount:  5,
			inputMetricCount:     50,
			outputMetricCount:    50,
			stageHits:            map[string]int{"static_threshold": 3, "retention": 2},
			expectedMetricsCount: 3,   // efficiency_ratio + resource_count + threshold_triggers
			expectedEfficiency:   0.0, // 0% filtered
			validateSummaryMetrics: func(t *testing.T, filteredMetrics pmetric.Metrics) {
				assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())

				summaryRM := filteredMetrics.ResourceMetrics().At(0)
				validateSummaryResource(t, summaryRM)

				// Should have 3 metrics: efficiency_ratio + resource_count + threshold_triggers
				assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
				sm := summaryRM.ScopeMetrics().At(0)
				assert.Equal(t, 3, sm.Metrics().Len())

				// Validate efficiency metric
				efficiencyMetric := findMetricByName(sm, filteringEfficiencyRatioMetric)
				assert.NotNil(t, efficiencyMetric)
				assert.Equal(t, 0.0, efficiencyMetric.Gauge().DataPoints().At(0).DoubleValue())

				// Validate resource count metric
				resourceCountMetric := findMetricByName(sm, filteringResourceCountMetric)
				assert.NotNil(t, resourceCountMetric)
				validateResourceCountMetric(t, resourceCountMetric, 5, 0)

				// Validate threshold triggers metric
				triggersMetric := findMetricByName(sm, filteringThresholdTriggersMetric)
				assert.NotNil(t, triggersMetric)
				validateThresholdTriggersMetric(t, triggersMetric, map[string]int{"static_threshold": 3, "retention": 2})
			},
		},
		{
			name:                 "Partial filtering",
			inputResourceCount:   20,
			outputResourceCount:  8,
			inputMetricCount:     200,
			outputMetricCount:    80,
			stageHits:            map[string]int{"dynamic_threshold": 5, "multi_metric": 2, "anomaly_detection": 1},
			expectedMetricsCount: 3,   // efficiency_ratio + resource_count + threshold_triggers
			expectedEfficiency:   0.6, // 60% filtered (12 out of 20)
			validateSummaryMetrics: func(t *testing.T, filteredMetrics pmetric.Metrics) {
				assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())

				summaryRM := filteredMetrics.ResourceMetrics().At(0)
				validateSummaryResource(t, summaryRM)

				// Should have 3 metrics
				assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
				sm := summaryRM.ScopeMetrics().At(0)
				assert.Equal(t, 3, sm.Metrics().Len())

				// Validate efficiency metric
				efficiencyMetric := findMetricByName(sm, filteringEfficiencyRatioMetric)
				assert.NotNil(t, efficiencyMetric)
				assert.InDelta(t, 0.6, efficiencyMetric.Gauge().DataPoints().At(0).DoubleValue(), 0.01)

				// Validate resource count metric
				resourceCountMetric := findMetricByName(sm, filteringResourceCountMetric)
				assert.NotNil(t, resourceCountMetric)
				validateResourceCountMetric(t, resourceCountMetric, 8, 12)

				// Validate threshold triggers metric
				triggersMetric := findMetricByName(sm, filteringThresholdTriggersMetric)
				assert.NotNil(t, triggersMetric)
				validateThresholdTriggersMetric(t, triggersMetric, map[string]int{"dynamic_threshold": 5, "multi_metric": 2, "anomaly_detection": 1})
			},
		},
		{
			name:                 "Only stages with zero hits - no triggers metric",
			inputResourceCount:   3,
			outputResourceCount:  1,
			inputMetricCount:     30,
			outputMetricCount:    10,
			stageHits:            map[string]int{}, // No stage hits
			expectedMetricsCount: 2,                // Only efficiency_ratio + resource_count (no threshold_triggers)
			expectedEfficiency:   0.6667,           // 66.67% filtered (2 out of 3)
			validateSummaryMetrics: func(t *testing.T, filteredMetrics pmetric.Metrics) {
				assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())

				summaryRM := filteredMetrics.ResourceMetrics().At(0)
				validateSummaryResource(t, summaryRM)

				// Should have only 2 metrics (no threshold_triggers)
				assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
				sm := summaryRM.ScopeMetrics().At(0)
				assert.Equal(t, 2, sm.Metrics().Len())

				// Validate efficiency metric
				efficiencyMetric := findMetricByName(sm, filteringEfficiencyRatioMetric)
				assert.NotNil(t, efficiencyMetric)
				assert.InDelta(t, 0.6667, efficiencyMetric.Gauge().DataPoints().At(0).DoubleValue(), 0.01)

				// Validate resource count metric
				resourceCountMetric := findMetricByName(sm, filteringResourceCountMetric)
				assert.NotNil(t, resourceCountMetric)
				validateResourceCountMetric(t, resourceCountMetric, 1, 2)

				// Should NOT have threshold triggers metric
				triggersMetric := findMetricByName(sm, filteringThresholdTriggersMetric)
				assert.Nil(t, triggersMetric)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create empty filtered metrics
			filteredMetrics := pmetric.NewMetrics()

			// Call the function under test
			processor.generateFilteringSummaryMetrics(&filteredMetrics,
				tt.inputResourceCount, tt.outputResourceCount,
				tt.inputMetricCount, tt.outputMetricCount, tt.stageHits)

			// Run test-specific validations
			tt.validateSummaryMetrics(t, filteredMetrics)
		})
	}
}

// TestFilterSummaryResourceAttributes tests that filter summary resources have correct attributes
func TestFilterSummaryResourceAttributes(t *testing.T) {
	logger := zap.NewNop()
	processor := &processorImp{
		logger: logger,
		config: &Config{},
	}

	filteredMetrics := pmetric.NewMetrics()

	processor.generateFilteringSummaryMetrics(&filteredMetrics, 5, 3, 50, 30,
		map[string]int{"static_threshold": 2, "retention": 1})

	assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())

	summaryRM := filteredMetrics.ResourceMetrics().At(0)
	resource := summaryRM.Resource()

	// Validate resource attributes
	sourceAttr, exists := resource.Attributes().Get(atpSourceAttribute)
	assert.True(t, exists, "Should have source attribute")
	assert.Equal(t, "adaptive_telemetry_processor", sourceAttr.AsString())

	metricTypeAttr, exists := resource.Attributes().Get(atpMetricTypeAttribute)
	assert.True(t, exists, "Should have metric_type attribute")
	assert.Equal(t, "filter_summary", metricTypeAttr.AsString())
}

// TestFilterSummaryMetricProperties tests detailed properties of each summary metric
func TestFilterSummaryMetricProperties(t *testing.T) {
	logger := zap.NewNop()
	processor := &processorImp{
		logger: logger,
		config: &Config{},
	}

	filteredMetrics := pmetric.NewMetrics()

	processor.generateFilteringSummaryMetrics(&filteredMetrics, 10, 4, 100, 40,
		map[string]int{"static_threshold": 2, "dynamic_threshold": 1, "retention": 1})

	assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())

	summaryRM := filteredMetrics.ResourceMetrics().At(0)
	assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())

	sm := summaryRM.ScopeMetrics().At(0)

	// Validate scope properties
	assert.Equal(t, atpScopeName, sm.Scope().Name())
	assert.Equal(t, atpScopeVersion, sm.Scope().Version())

	// Should have 3 metrics
	assert.Equal(t, 3, sm.Metrics().Len())

	t.Run("Efficiency ratio metric properties", func(t *testing.T) {
		efficiencyMetric := findMetricByName(sm, filteringEfficiencyRatioMetric)
		assert.NotNil(t, efficiencyMetric)
		assert.Equal(t, "Percentage of resources filtered out by adaptive telemetry processor", efficiencyMetric.Description())
		assert.Equal(t, "1", efficiencyMetric.Unit())
		assert.Equal(t, pmetric.MetricTypeGauge, efficiencyMetric.Type())

		// Validate data point
		gauge := efficiencyMetric.Gauge()
		assert.Equal(t, 1, gauge.DataPoints().Len())
		dp := gauge.DataPoints().At(0)
		assert.InDelta(t, 0.6, dp.DoubleValue(), 0.01) // 60% filtered (6 out of 10)
		assert.True(t, dp.Timestamp() > 0)             // Should have timestamp
	})

	t.Run("Resource count metric properties", func(t *testing.T) {
		resourceCountMetric := findMetricByName(sm, filteringResourceCountMetric)
		assert.NotNil(t, resourceCountMetric)
		assert.Equal(t, "Count of resources by filter status", resourceCountMetric.Description())
		assert.Equal(t, "1", resourceCountMetric.Unit())
		assert.Equal(t, pmetric.MetricTypeGauge, resourceCountMetric.Type())

		// Validate data points
		gauge := resourceCountMetric.Gauge()
		assert.Equal(t, 2, gauge.DataPoints().Len()) // included + filtered

		// Find included and filtered data points
		var includedDP, filteredDP pmetric.NumberDataPoint
		for i := 0; i < gauge.DataPoints().Len(); i++ {
			dp := gauge.DataPoints().At(i)
			statusAttr, exists := dp.Attributes().Get(atpStatusAttribute)
			assert.True(t, exists)

			if statusAttr.AsString() == statusIncluded {
				includedDP = dp
			} else if statusAttr.AsString() == statusFiltered {
				filteredDP = dp
			}
		}

		assert.Equal(t, int64(4), includedDP.IntValue()) // 4 included
		assert.Equal(t, int64(6), filteredDP.IntValue()) // 6 filtered
		assert.True(t, includedDP.Timestamp() > 0)
		assert.True(t, filteredDP.Timestamp() > 0)
	})

	t.Run("Threshold triggers metric properties", func(t *testing.T) {
		triggersMetric := findMetricByName(sm, filteringThresholdTriggersMetric)
		assert.NotNil(t, triggersMetric)
		assert.Equal(t, "Count of resources included by each filter stage", triggersMetric.Description())
		assert.Equal(t, "1", triggersMetric.Unit())
		assert.Equal(t, pmetric.MetricTypeGauge, triggersMetric.Type())

		// Validate data points
		gauge := triggersMetric.Gauge()
		assert.Equal(t, 3, gauge.DataPoints().Len()) // 3 stages

		stageValues := make(map[string]int64)
		for i := 0; i < gauge.DataPoints().Len(); i++ {
			dp := gauge.DataPoints().At(i)
			stageAttr, exists := dp.Attributes().Get(atpStageAttribute)
			assert.True(t, exists)
			stageValues[stageAttr.AsString()] = dp.IntValue()
			assert.True(t, dp.Timestamp() > 0)
		}

		assert.Equal(t, int64(2), stageValues["static_threshold"])
		assert.Equal(t, int64(1), stageValues["dynamic_threshold"])
		assert.Equal(t, int64(1), stageValues["retention"])
	})
}

// TestFilterSummarySkipsEmptyStageHits tests that empty stage hits don't create threshold_triggers metric
func TestFilterSummarySkipsEmptyStageHits(t *testing.T) {
	logger := zap.NewNop()
	processor := &processorImp{
		logger: logger,
		config: &Config{},
	}

	tests := []struct {
		name                     string
		stageHits                map[string]int
		shouldHaveTriggersMetric bool
	}{
		{
			name:                     "Empty stage hits map",
			stageHits:                map[string]int{},
			shouldHaveTriggersMetric: false,
		},
		{
			name:                     "Nil stage hits map",
			stageHits:                nil,
			shouldHaveTriggersMetric: false,
		},
		{
			name:                     "Only zero value stage hits",
			stageHits:                map[string]int{"static_threshold": 0, "retention": 0},
			shouldHaveTriggersMetric: true, // Metric is created but will have 0 data points
		},
		{
			name:                     "Mixed zero and non-zero stage hits",
			stageHits:                map[string]int{"static_threshold": 0, "retention": 1},
			shouldHaveTriggersMetric: true,
		},
		{
			name:                     "All non-zero stage hits",
			stageHits:                map[string]int{"static_threshold": 2, "retention": 3},
			shouldHaveTriggersMetric: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredMetrics := pmetric.NewMetrics()

			processor.generateFilteringSummaryMetrics(&filteredMetrics, 5, 2, 50, 20, tt.stageHits)

			assert.Equal(t, 1, filteredMetrics.ResourceMetrics().Len())
			summaryRM := filteredMetrics.ResourceMetrics().At(0)
			assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
			sm := summaryRM.ScopeMetrics().At(0)

			triggersMetric := findMetricByName(sm, filteringThresholdTriggersMetric)
			if tt.shouldHaveTriggersMetric {
				assert.NotNil(t, triggersMetric, "Should have threshold triggers metric")

				// Count non-zero expected stage hits
				expectedNonZeroStages := 0
				for _, count := range tt.stageHits {
					if count > 0 {
						expectedNonZeroStages++
					}
				}

				// Validate that only non-zero stages are included as data points
				gauge := triggersMetric.Gauge()
				assert.Equal(t, expectedNonZeroStages, gauge.DataPoints().Len(), "Should only have data points for non-zero stages")

				for i := 0; i < gauge.DataPoints().Len(); i++ {
					dp := gauge.DataPoints().At(i)
					assert.True(t, dp.IntValue() > 0, "All data points should have positive values")
				}
			} else {
				assert.Nil(t, triggersMetric, "Should not have threshold triggers metric")
			}
		})
	}
}

// Helper functions for test validation

// validateSummaryResource validates the summary resource attributes and scope
func validateSummaryResource(t *testing.T, summaryRM pmetric.ResourceMetrics) {
	// Validate resource attributes
	resource := summaryRM.Resource()

	sourceAttr, exists := resource.Attributes().Get(atpSourceAttribute)
	assert.True(t, exists, "Should have source attribute")
	assert.Equal(t, "adaptive_telemetry_processor", sourceAttr.AsString())

	metricTypeAttr, exists := resource.Attributes().Get(atpMetricTypeAttribute)
	assert.True(t, exists, "Should have metric_type attribute")
	assert.Equal(t, "filter_summary", metricTypeAttr.AsString())

	// Validate scope
	assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
	sm := summaryRM.ScopeMetrics().At(0)
	assert.Equal(t, atpScopeName, sm.Scope().Name())
	assert.Equal(t, atpScopeVersion, sm.Scope().Version())
}

// findMetricByName finds a metric by its name in a scope metrics
func findMetricByName(sm pmetric.ScopeMetrics, name string) *pmetric.Metric {
	for i := 0; i < sm.Metrics().Len(); i++ {
		metric := sm.Metrics().At(i)
		if metric.Name() == name {
			return &metric
		}
	}
	return nil
}

// validateResourceCountMetric validates the resource count metric data points
func validateResourceCountMetric(t *testing.T, metric *pmetric.Metric, expectedIncluded, expectedFiltered int) {
	assert.Equal(t, pmetric.MetricTypeGauge, metric.Type())
	gauge := metric.Gauge()
	assert.Equal(t, 2, gauge.DataPoints().Len())

	includedFound := false
	filteredFound := false

	for i := 0; i < gauge.DataPoints().Len(); i++ {
		dp := gauge.DataPoints().At(i)
		statusAttr, exists := dp.Attributes().Get(atpStatusAttribute)
		assert.True(t, exists)

		switch statusAttr.AsString() {
		case statusIncluded:
			assert.Equal(t, int64(expectedIncluded), dp.IntValue())
			includedFound = true
		case statusFiltered:
			assert.Equal(t, int64(expectedFiltered), dp.IntValue())
			filteredFound = true
		}

		assert.True(t, dp.Timestamp() > 0, "Should have timestamp")
	}

	assert.True(t, includedFound, "Should have included status data point")
	assert.True(t, filteredFound, "Should have filtered status data point")
}

// validateThresholdTriggersMetric validates the threshold triggers metric data points
func validateThresholdTriggersMetric(t *testing.T, metric *pmetric.Metric, expectedStageHits map[string]int) {
	assert.Equal(t, pmetric.MetricTypeGauge, metric.Type())
	gauge := metric.Gauge()

	// Count non-zero expected stage hits
	expectedNonZeroStages := 0
	for _, count := range expectedStageHits {
		if count > 0 {
			expectedNonZeroStages++
		}
	}

	assert.Equal(t, expectedNonZeroStages, gauge.DataPoints().Len())

	foundStages := make(map[string]int64)
	for i := 0; i < gauge.DataPoints().Len(); i++ {
		dp := gauge.DataPoints().At(i)
		stageAttr, exists := dp.Attributes().Get(atpStageAttribute)
		assert.True(t, exists)

		stage := stageAttr.AsString()
		foundStages[stage] = dp.IntValue()
		assert.True(t, dp.Timestamp() > 0, "Should have timestamp")
		assert.True(t, dp.IntValue() > 0, "Should only include stages with positive counts")
	}

	// Validate all expected non-zero stages are present with correct values
	for stage, expectedCount := range expectedStageHits {
		if expectedCount > 0 {
			actualCount, found := foundStages[stage]
			assert.True(t, found, "Stage %s should be present", stage)
			assert.Equal(t, int64(expectedCount), actualCount, "Stage %s should have correct count", stage)
		}
	}
}

// TestFilterSummaryIntegrationWithProcessor tests integration with process metrics
func TestFilterSummaryIntegrationWithProcessor(t *testing.T) {
	logger := zap.NewNop()
	processor := &processorImp{
		logger: logger,
		config: &Config{
			MetricThresholds: map[string]float64{
				"system.cpu.utilization": 70.0,
			},
			EnableDynamicThresholds: false,
			EnableMultiMetric:       false,
		},
		trackedEntities: make(map[string]*TrackedEntity),
	}

	// Create test metrics - mix of above and below threshold
	inputMetrics := pmetric.NewMetrics()

	// Resource 1: Above threshold (should be included)
	rm1 := inputMetrics.ResourceMetrics().AppendEmpty()
	rm1.Resource().Attributes().PutStr("service.name", "high-cpu-service")
	sm1 := rm1.ScopeMetrics().AppendEmpty()
	metric1 := sm1.Metrics().AppendEmpty()
	metric1.SetName("system.cpu.utilization")
	metric1.SetEmptyGauge()
	dp1 := metric1.Gauge().DataPoints().AppendEmpty()
	dp1.SetDoubleValue(85.0) // Above threshold
	dp1.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Resource 2: Below threshold (should be filtered)
	rm2 := inputMetrics.ResourceMetrics().AppendEmpty()
	rm2.Resource().Attributes().PutStr("service.name", "low-cpu-service")
	sm2 := rm2.ScopeMetrics().AppendEmpty()
	metric2 := sm2.Metrics().AppendEmpty()
	metric2.SetName("system.cpu.utilization")
	metric2.SetEmptyGauge()
	dp2 := metric2.Gauge().DataPoints().AppendEmpty()
	dp2.SetDoubleValue(30.0) // Below threshold
	dp2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Process metrics
	ctx := context.Background()
	filteredMetrics, err := processor.processMetrics(ctx, inputMetrics)
	assert.NoError(t, err)

	// Should have 2 resources: 1 filtered + 1 summary
	assert.Equal(t, 2, filteredMetrics.ResourceMetrics().Len())

	// Find summary resource
	var summaryRM pmetric.ResourceMetrics
	var businessRM pmetric.ResourceMetrics
	found := false

	for i := 0; i < filteredMetrics.ResourceMetrics().Len(); i++ {
		rm := filteredMetrics.ResourceMetrics().At(i)
		if metricType, exists := rm.Resource().Attributes().Get(atpMetricTypeAttribute); exists {
			if metricType.AsString() == "filter_summary" {
				summaryRM = rm
				found = true
			}
		} else {
			businessRM = rm
		}
	}

	assert.True(t, found, "Should have summary resource")

	// Validate summary metrics exist and have correct values
	validateSummaryResource(t, summaryRM)
	assert.Equal(t, 1, summaryRM.ScopeMetrics().Len())
	sm := summaryRM.ScopeMetrics().At(0)

	// Should have efficiency_ratio, resource_count, and threshold_triggers
	assert.Equal(t, 3, sm.Metrics().Len())

	// Efficiency should be 0.5 (1 out of 2 resources filtered)
	efficiencyMetric := findMetricByName(sm, filteringEfficiencyRatioMetric)
	assert.NotNil(t, efficiencyMetric)
	assert.InDelta(t, 0.5, efficiencyMetric.Gauge().DataPoints().At(0).DoubleValue(), 0.01)

	// Resource count should show 1 included, 1 filtered
	resourceCountMetric := findMetricByName(sm, filteringResourceCountMetric)
	assert.NotNil(t, resourceCountMetric)
	validateResourceCountMetric(t, resourceCountMetric, 1, 1)

	// Threshold triggers should show 1 static_threshold hit
	triggersMetric := findMetricByName(sm, filteringThresholdTriggersMetric)
	assert.NotNil(t, triggersMetric)
	validateThresholdTriggersMetric(t, triggersMetric, map[string]int{"static_threshold": 1})

	// Validate business resource has filter stage attribute
	assert.Equal(t, 1, businessRM.ScopeMetrics().Len())
	stageAttr, exists := businessRM.Resource().Attributes().Get(adaptiveFilterStageAttributeKey)
	assert.True(t, exists, "Business resource should have filter stage attribute")
	assert.Equal(t, stageStaticThreshold, stageAttr.AsString())
}

// TestFilterSummaryHelperFunctions tests the helper functions used by the main tests
func TestFilterSummaryHelperFunctions(t *testing.T) {
	t.Run("findMetricByName", func(t *testing.T) {
		// Create test scope metrics
		sm := pmetric.NewScopeMetrics()

		// Add a metric
		metric1 := sm.Metrics().AppendEmpty()
		metric1.SetName("test.metric.1")

		metric2 := sm.Metrics().AppendEmpty()
		metric2.SetName("test.metric.2")

		// Test finding existing metric
		found := findMetricByName(sm, "test.metric.1")
		assert.NotNil(t, found)
		assert.Equal(t, "test.metric.1", found.Name())

		// Test finding non-existent metric
		notFound := findMetricByName(sm, "non.existent.metric")
		assert.Nil(t, notFound)
	})

	t.Run("validateSummaryResource", func(t *testing.T) {
		// Create test resource metrics
		rm := pmetric.NewResourceMetrics()
		rm.Resource().Attributes().PutStr(atpSourceAttribute, "adaptive_telemetry_processor")
		rm.Resource().Attributes().PutStr(atpMetricTypeAttribute, "filter_summary")

		sm := rm.ScopeMetrics().AppendEmpty()
		sm.Scope().SetName(atpScopeName)
		sm.Scope().SetVersion(atpScopeVersion)

		// Should not panic and should validate correctly
		assert.NotPanics(t, func() {
			validateSummaryResource(t, rm)
		})
	})
}
