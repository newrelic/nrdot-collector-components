// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// processMetrics iterates resource metrics, applies threshold logic, and returns a filtered copy.
// Optimized for better performance with metrics batching and reduced memory allocations.
func (p *processorImp) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	start := time.Now()

	// Quick exit for empty metrics
	if md.ResourceMetrics().Len() == 0 {
		p.logger.Info("Received empty metrics batch, returning without processing")
		return md, nil
	}

	// Initialize processing context
	processCtx := p.initializeProcessingContext(ctx, md)

	// Check if context is already cancelled
	if processCtx.ctx.Err() != nil {
		return p.handleContextCancellation(md, processCtx)
	}

	// Update dynamic thresholds if needed
	p.updateDynamicThresholdsIfNeeded(md)

	// Process all resources
	filtered, _ := p.processAllResources(processCtx, md)

	// Perform post-processing tasks
	p.performPostProcessingTasks(processCtx, filtered, start)

	return filtered, nil
}

// processingContext holds context information for metrics processing
type processingContext struct {
	ctx              context.Context
	resourceCount    int
	totalMetricCount int
	metricTypeCount  map[string]int
	stageHits        map[string]int // Track which stages triggered inclusions
}

// initializeProcessingContext sets up the processing context and logs batch information
func (p *processorImp) initializeProcessingContext(ctx context.Context, md pmetric.Metrics) *processingContext {
	processCtx := &processingContext{
		ctx:             ctx,
		resourceCount:   md.ResourceMetrics().Len(),
		metricTypeCount: make(map[string]int),
		stageHits:       make(map[string]int),
	}

	// Count metrics by type for better visibility
	processCtx.totalMetricCount = countMetricsByType(md, processCtx.metricTypeCount)

	// Log batch information
	p.logger.Info("Processing metrics batch",
		zap.Int("resources", processCtx.resourceCount),
		zap.Int("metrics", processCtx.totalMetricCount),
		zap.Int("tracked_entities", len(p.trackedEntities)),
		zap.Bool("dynamic_thresholds", p.dynamicThresholdsEnabled),
		zap.Bool("multi_metric", p.multiMetricEnabled),
		zap.Bool("anomaly_detection", p.config.EnableAnomalyDetection))

	return processCtx
}

// countMetricsByType counts metrics by their OpenTelemetry type
func countMetricsByType(md pmetric.Metrics, metricTypeCount map[string]int) int {
	totalMetricCount := 0

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			totalMetricCount += sm.Metrics().Len()

			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					metricTypeCount["gauge"]++
				case pmetric.MetricTypeSum:
					metricTypeCount["sum"]++
				case pmetric.MetricTypeHistogram:
					metricTypeCount["histogram"]++
				case pmetric.MetricTypeSummary:
					metricTypeCount["summary"]++
				case pmetric.MetricTypeExponentialHistogram:
					metricTypeCount["exp_histogram"]++
				default:
					metricTypeCount["unknown"]++
				}
			}
		}
	}

	return totalMetricCount
}

// handleContextCancellation handles when context is cancelled before processing
func (p *processorImp) handleContextCancellation(md pmetric.Metrics, processCtx *processingContext) (pmetric.Metrics, error) {
	p.logger.Warn("Context cancelled before processing started",
		zap.Error(processCtx.ctx.Err()),
		zap.Int("resource_count", processCtx.resourceCount),
		zap.Int("metric_count", processCtx.totalMetricCount))
	return md, processCtx.ctx.Err()
}

// updateDynamicThresholdsIfNeeded updates dynamic thresholds if interval has passed
func (p *processorImp) updateDynamicThresholdsIfNeeded(md pmetric.Metrics) {
	if !p.dynamicThresholdsEnabled || time.Since(p.lastThresholdUpdate).Seconds() < dynamicUpdateIntervalSecs {
		return
	}

	// Update thresholds synchronously - simple and fast operation
	p.updateDynamicThresholds(md)
	p.lastThresholdUpdate = time.Now()
	p.logger.Debug("Dynamic thresholds updated")
}

// processAllResources processes all resource metrics and returns filtered results
func (p *processorImp) processAllResources(processCtx *processingContext, md pmetric.Metrics) (pmetric.Metrics, int) {
	filtered := pmetric.NewMetrics()
	rms := md.ResourceMetrics()
	includedCount := 0

	// Process all resources with a time limit per resource
	for i := 0; i < rms.Len(); i++ {
		// Check context occasionally to allow cancellation during long processing
		if i > 0 && i%25 == 0 {
			if processCtx.ctx.Err() != nil {
				p.logger.Warn("Context cancelled during resource processing", zap.Error(processCtx.ctx.Err()))
				return md, 0 // Return original metrics on timeout
			}
		}

		rm := rms.At(i)
		if p.processingSingleResource(rm, &filtered, processCtx.stageHits) {
			includedCount++
		}
	}

	p.logger.Debug("Resource filtering completed",
		zap.Int("included_count", includedCount),
		zap.Int("total_resources", processCtx.resourceCount))

	return filtered, includedCount
}

// processingSingleResource processes a single resource and returns whether it was included
func (p *processorImp) processingSingleResource(rm pmetric.ResourceMetrics, filtered *pmetric.Metrics, stageHits map[string]int) bool {
	resourceID := buildResourceIdentity(rm.Resource())

	// Evaluate resource through all filter stages - no artificial timeout
	includeResource := p.shouldIncludeResource(rm.Resource(), rm)

	// Get the filter stage from the resource attributes that was set by shouldIncludeResource
	includeReason := ""
	if stageAttr, hasStage := rm.Resource().Attributes().Get(adaptiveFilterStageAttributeKey); hasStage {
		includeReason = stageAttr.AsString()
	}

	if includeResource {
		p.handleIncludedResource(rm, resourceID, includeReason, filtered)
		// Track which stage allowed this resource through
		if stageHits != nil {
			stageHits[includeReason]++
		}
		return true
	}
	p.handleExcludedResource(rm, resourceID)
	return false
}

// handleIncludedResource processes a resource that should be included in output
func (p *processorImp) handleIncludedResource(rm pmetric.ResourceMetrics, resourceID, includeReason string, filtered *pmetric.Metrics) {
	resourceType := getResourceType(rm.Resource().Attributes())
	serviceName := "unknown"
	if val, ok := rm.Resource().Attributes().Get("service.name"); ok {
		serviceName = val.AsString()
	}

	p.logger.Info("Including resource in output",
		zap.String("resource_id", resourceID),
		zap.String("filter_stage", includeReason),
		zap.String("resource_type", resourceType),
		zap.String("service_name", serviceName),
		zap.Int("scope_count", rm.ScopeMetrics().Len()),
		zap.Int("metric_count_in_resource", countMetricsInResource(rm)))

	dest := filtered.ResourceMetrics().AppendEmpty()
	rm.CopyTo(dest)
	// Remove the internal filter stage attribute from the output
	dest.Resource().Attributes().Remove(internalFilterStageAttributeKey)
}

// handleExcludedResource processes a resource that should be excluded from output
func (p *processorImp) handleExcludedResource(rm pmetric.ResourceMetrics, resourceID string) {
	resourceType := getResourceType(rm.Resource().Attributes())
	p.logger.Info("Excluding resource from output",
		zap.String("resource_id", resourceID),
		zap.String("resource_type", resourceType),
		zap.Int("metric_count", countMetricsInResource(rm)))
}

// performPostProcessingTasks handles cleanup and final logging
func (p *processorImp) performPostProcessingTasks(processCtx *processingContext, filtered pmetric.Metrics, start time.Time) {
	// Perform cleanup of expired entities with controlled frequency
	if processCtx.resourceCount > 0 && p.config.RetentionMinutes > 0 && rand.Float64() < 0.01 {
		go p.cleanupExpiredEntities()
	}

	processingTime := time.Since(start)
	outputResourceCount := filtered.ResourceMetrics().Len()
	outputMetricCount := countOutputMetrics(filtered)

	// Generate summary metrics for customer visibility into filtering effectiveness
	p.generateFilteringSummaryMetrics(&filtered, processCtx.resourceCount, outputResourceCount,
		processCtx.totalMetricCount, outputMetricCount, processCtx.stageHits)

	p.logger.Info("Metrics processing completed",
		zap.Int("input_resources", processCtx.resourceCount),
		zap.Int("output_resources", outputResourceCount),
		zap.Int("input_metrics", processCtx.totalMetricCount),
		zap.Int("output_metrics", outputMetricCount),
		zap.Duration("processing_time", processingTime),
		zap.Int("tracked_entities", len(p.trackedEntities)))

	// Log warning if all resources were filtered out
	if processCtx.resourceCount > 0 && outputResourceCount == 0 {
		p.logger.Warn("All resources were filtered out - check configuration",
			zap.Int("input_resources", processCtx.resourceCount),
			zap.Int("input_metrics", processCtx.totalMetricCount))
	}
}

// countOutputMetrics counts metrics in the filtered output
func countOutputMetrics(filtered pmetric.Metrics) int {
	outputMetricCount := 0
	for i := 0; i < filtered.ResourceMetrics().Len(); i++ {
		rm := filtered.ResourceMetrics().At(i)
		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			outputMetricCount += rm.ScopeMetrics().At(j).Metrics().Len()
		}
	}
	return outputMetricCount
}

// shouldIncludeResource determines if a resource should be included in the filtered output
func (p *processorImp) shouldIncludeResource(resource pcommon.Resource, rm pmetric.ResourceMetrics) bool {
	// Get resource identity and basic info
	id := buildResourceIdentity(resource)
	resourceType := getResourceType(resource.Attributes())
	values := p.extractMetricValues(rm)

	// Log basic resource info
	p.logger.Debug("Evaluating resource",
		zap.String("resource_id", id),
		zap.String("resource_type", resourceType),
		zap.Int("metric_count", len(values)))

	// This tracks only the MetricThresholds values, not other config data
	p.captureUsedMetricThresholds(resource, values)

	// Bypass filtering if no metrics in the resource match configured thresholds
	// This ensures we default to INCLUSION for non-targeted resources (e.g., system metrics, unconfigured processes)
	if !p.isResourceTargeted(values) {
		p.logger.Debug("Resource included: no specified metrics found (default inclusion)", zap.String("resource_id", id))
		return true
	}

	// Check if this is a zombie process - always include if so
	if isZombieProcess(resource.Attributes()) {
		setResourceFilterStage(resource, stageZombieProcess)
		p.logger.Info("Resource included: zombie process",
			zap.String("resource_id", id))

		// Track the entity even if it's a zombie process for statistics
		p.mu.Lock()
		defer p.mu.Unlock()
		p.upsertTrackedEntityForIncludeList(id, values, resource)
		return true
	}

	// Check include list FIRST - bypass all filters if in include list
	if len(p.config.IncludeProcessList) > 0 && isProcessInIncludeList(resource.Attributes(), p.config.IncludeProcessList) {
		processName := extractProcessName(resource.Attributes())
		setResourceFilterStage(resource, stageIncludeList)
		p.logger.Info("Resource included: in include list (bypass filters)",
			zap.String("resource_id", id),
			zap.String("process_name", processName))

		// Track the entity even if it's in the include list for statistics
		p.mu.Lock()
		defer p.mu.Unlock()
		p.upsertTrackedEntityForIncludeList(id, values, resource)
		return true
	}

	// Take write lock for entity tracking
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if this is a known entity
	trackedEntity, exists := p.trackedEntities[id]

	if exists {
		return p.evaluateExistingEntity(resource, id, trackedEntity, values)
	}
	return p.evaluateNewEntity(resource, id, values)
}

// evaluateExistingEntity evaluates filter stages for an existing tracked entity
func (p *processorImp) evaluateExistingEntity(resource pcommon.Resource, id string, trackedEntity *trackedEntity, values map[string]float64) bool {
	// Update current and max values
	updateEntityValues(trackedEntity, values)

	// Check filter stages in order
	return p.checkAnomalyDetectionStage(resource, id, trackedEntity, values) ||
		p.checkThresholdStages(resource, id, trackedEntity, values) ||
		p.checkMultiMetricStage(resource, id, trackedEntity, values) ||
		p.checkRetentionStages(resource, id, trackedEntity)
}

// evaluateNewEntity evaluates filter stages for a new entity
func (p *processorImp) evaluateNewEntity(resource pcommon.Resource, id string, values map[string]float64) bool {
	// Create new tracked entity
	newEntity := p.createNewTrackedEntity(id, values, resource)

	// Check filter stages for new entity
	include, stage := p.checkNewEntityFilterStages(resource, id, newEntity, values)

	// Store entity if it should be included or if debug mode is enabled
	if include || p.config.DebugShowAllFilterStages {
		p.trackedEntities[id] = newEntity

		if include {
			setResourceFilterStage(resource, stage)
			p.logger.Info("Resource included: new resource",
				zap.String("resource_id", id),
				zap.String("filter_stage", stage))
			return true
		} else if p.config.DebugShowAllFilterStages {
			return p.handleDebugMode(resource, id, values)
		}
	}

	p.logger.Debug("Excluding new resource", zap.String("resource_id", id))
	return false
}

// checkNewEntityFilterStages checks all filter stages for a new entity
// Order matches requirement.md: Anomaly → Threshold → Multi-Metric
func (p *processorImp) checkNewEntityFilterStages(resource pcommon.Resource, id string, newEntity *trackedEntity, values map[string]float64) (bool, string) {
	// Stage 1: Check anomaly detection first (highest priority - detects sudden changes)
	if include, stage := p.checkNewEntityAnomaly(id, newEntity, values); include {
		return true, stage
	}

	// Stage 2: Check threshold stages (dynamic or static - absolute limits)
	if include, stage := p.checkNewEntityThresholds(id, values); include {
		newEntity.LastExceeded = time.Now() // Update timestamp for retention period tracking
		return true, stage
	}

	// Stage 3: Check multi-metric stage (composite scoring - combined stress)
	if include, stage := p.checkNewEntityMultiMetric(resource, id, values); include {
		newEntity.LastExceeded = time.Now() // Update timestamp for retention period tracking
		return true, stage
	}

	return false, ""
}

// updateEntityValues updates current and max values for a tracked entity
func updateEntityValues(trackedEntity *trackedEntity, values map[string]float64) {
	if trackedEntity.CurrentValues == nil {
		trackedEntity.CurrentValues = make(map[string]float64)
	}
	if trackedEntity.MaxValues == nil {
		trackedEntity.MaxValues = make(map[string]float64)
	}

	for m, v := range values {
		trackedEntity.CurrentValues[m] = v
		if v > trackedEntity.MaxValues[m] {
			trackedEntity.MaxValues[m] = v
		}
	}
}

// upsertTrackedEntityForIncludeList ensures a tracked entity exists or updates it for include-list resources.
// Sets LastExceeded so retention logic keeps the entity included.
func (p *processorImp) upsertTrackedEntityForIncludeList(id string, values map[string]float64, resource pcommon.Resource) {
	now := time.Now()
	if te, exists := p.trackedEntities[id]; !exists {
		p.trackedEntities[id] = &trackedEntity{
			Identity:      id,
			FirstSeen:     now,
			LastExceeded:  now,
			CurrentValues: values,
			MaxValues:     values,
			Attributes:    snapshotResourceAttributes(resource),
		}
	} else {
		updateEntityValues(te, values)
		te.LastExceeded = now
	}
}

// checkAnomalyDetectionStage checks for anomaly detection in existing entities
func (p *processorImp) checkAnomalyDetectionStage(resource pcommon.Resource, id string, trackedEntity *trackedEntity, values map[string]float64) bool {
	if !p.config.EnableAnomalyDetection {
		return false
	}

	if isAnomaly, anomalyReason := p.detectAnomaly(trackedEntity, values); isAnomaly {
		setResourceFilterStage(resource, stageAnomalyDetection)
		p.logger.Info("Resource included: anomaly detected",
			zap.String("resource_id", id),
			zap.String("details", anomalyReason))
		return true
	}
	return false
}

// checkThresholdStages checks dynamic and static threshold stages
func (p *processorImp) checkThresholdStages(resource pcommon.Resource, id string, trackedEntity *trackedEntity, values map[string]float64) bool {
	if p.dynamicThresholdsEnabled {
		return p.checkDynamicThresholds(resource, id, trackedEntity, values)
	}
	return p.checkStaticThresholds(resource, id, trackedEntity, values)
}

// checkDynamicThresholds checks dynamic threshold stage for existing entities
func (p *processorImp) checkDynamicThresholds(resource pcommon.Resource, id string, trackedEntity *trackedEntity, values map[string]float64) bool {
	for m, v := range values {
		if threshold, ok := p.dynamicCustomThresholds[m]; ok && v >= threshold {
			trackedEntity.LastExceeded = time.Now()
			setResourceFilterStage(resource, stageDynamicThreshold)
			p.logger.Info("Resource included: dynamic threshold",
				zap.String("resource_id", id),
				zap.String("metric", m),
				zap.Float64("value", v),
				zap.Float64("threshold", threshold))
			return true
		}
	}
	return false
}

// checkStaticThresholds checks static threshold stage for existing entities
func (p *processorImp) checkStaticThresholds(resource pcommon.Resource, id string, trackedEntity *trackedEntity, values map[string]float64) bool {
	for m, v := range values {
		threshold, ok := p.config.MetricThresholds[m]
		// Strict check: metric must exist in config (guaranteed by extractMetricValues logic, but explicit check ensures safety)
		if !ok {
			continue
		}

		if threshold == 0.0 || v >= threshold {
			trackedEntity.LastExceeded = time.Now()
			setResourceFilterStage(resource, stageStaticThreshold)
			p.logger.Debug("Resource included: static threshold",
				zap.String("resource_id", id),
				zap.String("metric", m),
				zap.Float64("value", v),
				zap.Float64("threshold", threshold))
			return true
		}
	}
	return false
}

// checkMultiMetricStage checks multi-metric stage for existing entities
func (p *processorImp) checkMultiMetricStage(resource pcommon.Resource, id string, trackedEntity *trackedEntity, values map[string]float64) bool {
	if !p.multiMetricEnabled {
		return false
	}

	compScore, reason := p.calculateCompositeGeneric(values)
	threshold := p.config.CompositeThreshold
	if threshold <= 0 {
		threshold = defaultCompositeThreshold
	}

	if compScore >= threshold {
		trackedEntity.LastExceeded = time.Now()
		setResourceFilterStage(resource, stageMultiMetric)

		// Add composite score and threshold to process.atp JSON
		multiMetricDetails := map[string]interface{}{
			"composite_score": compScore,
			"threshold":       threshold,
		}
		updateProcessATPAttribute(resource, "multi_metric", multiMetricDetails)

		p.logger.Info("Resource included: multi-metric",
			zap.String("resource_id", id),
			zap.Float64("score", compScore),
			zap.Float64("threshold", threshold),
			zap.String("calculation", reason))
		return true
	}
	return false
}

// checkRetentionStages checks anomaly and standard retention stages
func (p *processorImp) checkRetentionStages(resource pcommon.Resource, id string, trackedEntity *trackedEntity) bool {
	return p.checkAnomalyRetention(resource, id, trackedEntity) ||
		p.checkStandardRetention(resource, id, trackedEntity)
}

// checkAnomalyRetention checks anomaly retention stage
func (p *processorImp) checkAnomalyRetention(resource pcommon.Resource, id string, trackedEntity *trackedEntity) bool {
	if !p.config.EnableAnomalyDetection || trackedEntity.LastAnomalyDetected.IsZero() {
		return false
	}

	anomalyRetentionMins := 30 // Default
	if p.config.RetentionMinutes > 0 {
		anomalyRetentionMins = int(p.config.RetentionMinutes)
	}

	if time.Since(trackedEntity.LastAnomalyDetected).Minutes() < float64(anomalyRetentionMins) {
		setResourceFilterStage(resource, stageAnomalyRetention)
		p.logger.Info("Resource included: anomaly retention",
			zap.String("resource_id", id),
			zap.Float64("minutes_since_anomaly", time.Since(trackedEntity.LastAnomalyDetected).Minutes()),
			zap.Int("retention_minutes", anomalyRetentionMins))
		return true
	}
	return false
}

// checkStandardRetention checks standard retention stage
func (p *processorImp) checkStandardRetention(resource pcommon.Resource, id string, trackedEntity *trackedEntity) bool {
	if p.config.RetentionMinutes <= 0 || trackedEntity.LastExceeded.IsZero() {
		return false
	}

	retentionWindow := time.Duration(p.config.RetentionMinutes) * time.Minute
	if time.Since(trackedEntity.LastExceeded) < retentionWindow {
		setResourceFilterStage(resource, stageStandardRetention)
		p.logger.Info("Resource included: standard retention period",
			zap.String("resource_id", id),
			zap.Duration("time_since_exceeded", time.Since(trackedEntity.LastExceeded)),
			zap.Int64("retention_minutes", p.config.RetentionMinutes))
		return true
	}
	return false
}

// createNewTrackedEntity creates a new tracked entity
func (p *processorImp) createNewTrackedEntity(id string, values map[string]float64, resource pcommon.Resource) *trackedEntity {
	now := time.Now()
	newEntity := &trackedEntity{
		Identity:      id,
		FirstSeen:     now,
		LastExceeded:  time.Time{}, // Zero value - only set when threshold is actually exceeded
		CurrentValues: values,
		MaxValues:     values,
		Attributes:    snapshotResourceAttributes(resource),
	}

	// Initialize history if needed for anomaly detection
	if p.config.EnableAnomalyDetection {
		newEntity.MetricHistory = make(map[string][]float64)
		for m, v := range values {
			newEntity.MetricHistory[m] = []float64{v}
		}
	}

	return newEntity
}

// checkNewEntityThresholds checks threshold stages for new entities
func (p *processorImp) checkNewEntityThresholds(id string, values map[string]float64) (bool, string) {
	if p.dynamicThresholdsEnabled {
		for m, v := range values {
			if threshold, ok := p.dynamicCustomThresholds[m]; ok && v >= threshold {
				p.logger.Info("New resource exceeds dynamic threshold",
					zap.String("resource_id", id),
					zap.String("metric", m),
					zap.Float64("value", v),
					zap.Float64("threshold", threshold))
				return true, stageDynamicThreshold
			}
		}
	} else {
		for m, v := range values {
			threshold, ok := p.config.MetricThresholds[m]
			// Strict check: metric must exist in config (guaranteed by extractMetricValues logic, but explicit check hurts nothing)
			if !ok {
				continue
			}

			// If threshold is 0.0, it means "always include if metric present"
			if threshold == 0.0 || v >= threshold {
				p.logger.Info("New resource exceeds static threshold",
					zap.String("resource_id", id),
					zap.String("metric", m),
					zap.Float64("value", v),
					zap.Float64("threshold", threshold))
				return true, stageStaticThreshold
			}
		}
	}
	return false, ""
}

// checkNewEntityMultiMetric checks multi-metric stage for new entities
func (p *processorImp) checkNewEntityMultiMetric(resource pcommon.Resource, id string, values map[string]float64) (bool, string) {
	if !p.multiMetricEnabled {
		return false, ""
	}

	compScore, reason := p.calculateCompositeGeneric(values)
	threshold := p.config.CompositeThreshold
	if threshold <= 0 {
		threshold = defaultCompositeThreshold
	}

	if compScore >= threshold {
		// Add composite score and threshold to process.atp JSON
		multiMetricDetails := map[string]interface{}{
			"composite_score": compScore,
			"threshold":       threshold,
		}
		updateProcessATPAttribute(resource, "multi_metric", multiMetricDetails)

		p.logger.Info("New resource exceeds multi-metric threshold",
			zap.String("resource_id", id),
			zap.Float64("score", compScore),
			zap.Float64("threshold", threshold),
			zap.String("reason", reason))
		return true, stageMultiMetric
	}
	return false, ""
}

// checkNewEntityAnomaly checks anomaly detection stage for new entities
func (p *processorImp) checkNewEntityAnomaly(id string, newEntity *trackedEntity, values map[string]float64) (bool, string) {
	if !p.config.EnableAnomalyDetection {
		return false, ""
	}

	if isAnomaly, anomalyReason := p.detectAnomaly(newEntity, values); isAnomaly {
		p.logger.Info("New resource shows anomaly",
			zap.String("resource_id", id),
			zap.String("reason", anomalyReason))
		return true, stageAnomalyDetection
	}
	return false, ""
}

// handleDebugMode handles debug mode for resources that don't match any filter
func (p *processorImp) handleDebugMode(resource pcommon.Resource, id string, values map[string]float64) bool {
	// Build detailed debug reason showing why resource didn't match
	debugDetails := make([]string, 0, 4)

	// Check static/dynamic threshold stage
	if p.dynamicThresholdsEnabled {
		maxThresholdRatio := 0.0
		for m, v := range values {
			if threshold, ok := p.dynamicCustomThresholds[m]; ok && threshold > 0 {
				ratio := v / threshold
				if ratio > maxThresholdRatio {
					maxThresholdRatio = ratio
				}
			}
		}
		debugDetails = append(debugDetails, fmt.Sprintf("dynamic_max_ratio=%.2f", maxThresholdRatio))
	} else {
		maxThresholdRatio := 0.0
		for m, v := range values {
			if threshold := p.config.MetricThresholds[m]; threshold > 0 {
				ratio := v / threshold
				if ratio > maxThresholdRatio {
					maxThresholdRatio = ratio
				}
			}
		}
		debugDetails = append(debugDetails, fmt.Sprintf("static_max_ratio=%.2f", maxThresholdRatio))
	}

	// Check multi-metric stage
	if p.multiMetricEnabled {
		score, _ := p.calculateCompositeGeneric(values)
		threshold := p.config.CompositeThreshold
		if threshold <= 0 {
			threshold = defaultCompositeThreshold
		}
		debugDetails = append(debugDetails, fmt.Sprintf("multi_metric=%.2f/%.2f", score, threshold))
	}

	// Check anomaly detection
	if p.config.EnableAnomalyDetection {
		debugDetails = append(debugDetails, "anomaly=none")
	}

	debugReason := "debug_no_match:" + fmt.Sprintf("%v", debugDetails)

	setResourceFilterStage(resource, debugReason)
	p.logger.Debug("Including resource for debugging",
		zap.String("resource_id", id),
		zap.String("debug_reason", debugReason))
	return true
}

// setResourceFilterStage sets the filter stage attribute on a resource
func setResourceFilterStage(resource pcommon.Resource, stage string) {
	resource.Attributes().PutStr(internalFilterStageAttributeKey, stage)
}

// generateFilteringSummaryMetrics creates summary metrics about filtering performance
func (p *processorImp) generateFilteringSummaryMetrics(filtered *pmetric.Metrics, inputResourceCount, outputResourceCount, inputMetricCount, outputMetricCount int, stageHits map[string]int) {
	p.logger.Info("ATP Summary: generateFilteringSummaryMetrics called",
		zap.Int("input_resources", inputResourceCount),
		zap.Int("output_resources", outputResourceCount),
		zap.Int("input_metrics", inputMetricCount),
		zap.Int("output_metrics", outputMetricCount))

	if inputResourceCount == 0 {
		p.logger.Info("ATP Summary: Skipping summary generation - no input resources")
		return // No point generating summary for empty batch
	}

	// Create a new resource for summary metrics
	summaryRM := filtered.ResourceMetrics().AppendEmpty()
	summaryAttrs := summaryRM.Resource().Attributes()

	// Copy only the required attributes for HOST entity synthesis
	// Metric names starting with "process." will trigger HOST entity synthesis rules
	if filtered.ResourceMetrics().Len() > 1 {
		firstResource := filtered.ResourceMetrics().At(0)
		firstAttrs := firstResource.Resource().Attributes()

		// Copy host.id and host.name (required for HOST entity synthesis)
		if val, exists := firstAttrs.Get(attrHostID); exists {
			val.CopyTo(summaryAttrs.PutEmpty(attrHostID))
		}
		if val, exists := firstAttrs.Get(attrHostName); exists {
			val.CopyTo(summaryAttrs.PutEmpty(attrHostName))
		}

		// Copy newrelic.source (required condition for synthesis)
		if val, exists := firstAttrs.Get(attrNewRelicSource); exists {
			val.CopyTo(summaryAttrs.PutEmpty(attrNewRelicSource))
		}

		// Copy container.id if present (synthesis rule checks this is absent for hosts)
		if val, exists := firstAttrs.Get(attrContainerID); exists {
			val.CopyTo(summaryAttrs.PutEmpty(attrContainerID))
		}

		// Copy service.name if present (synthesis rule checks this is absent for hosts)
		if val, exists := firstAttrs.Get(attrServiceName); exists {
			val.CopyTo(summaryAttrs.PutEmpty(attrServiceName))
		}
	}

	// Add ATP-specific attributes (these will become entity tags)
	// These are now included in the JSON payload of the process.atp metric

	p.logger.Info("ATP Summary: Created summary resource",
		zap.String("atp_source", "adaptive_telemetry_processor"),
		zap.String("atp_metric_type", "filter_summary"))

	summaryScope := summaryRM.ScopeMetrics().AppendEmpty()
	summaryScope.Scope().SetName(atpScopeName)
	summaryScope.Scope().SetVersion(atpScopeVersion)

	// Add otel library attributes to scope
	summaryScope.Scope().Attributes().PutStr(attrOtelLibraryName, atpScopeName)
	summaryScope.Scope().Attributes().PutStr(attrOtelLibraryVersion, atpScopeVersion)

	// Create a single process.atp metric
	atpMetric := summaryScope.Metrics().AppendEmpty()
	atpMetric.SetName("process.atp")
	atpMetric.SetDescription("Adaptive Telemetry Processor summary metrics")
	atpMetric.SetUnit("1")
	atpGauge := atpMetric.SetEmptyGauge()
	atpDP := atpGauge.DataPoints().AppendEmpty()
	atpDP.SetDoubleValue(1.0)
	atpDP.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Metric 1: Efficiency ratio (percentage of resources filtered)
	filteredResourceCount := inputResourceCount - outputResourceCount
	efficiencyRatio := float64(filteredResourceCount) / float64(inputResourceCount)

	// Consolidate summary stats into a JSON object
	summaryDetails := map[string]interface{}{
		"source":                   "adaptive_telemetry_processor",
		"metric_type":              "filter_summary",
		"efficiency_ratio":         efficiencyRatio,
		"total_resource_count":     inputResourceCount,
		"resources_filtered_count": filteredResourceCount,
		"resources_included_count": outputResourceCount,
		"stage_hits":               stageHits,
		"evaluation_timestamp":     time.Now().Unix(),
	}

	atpData := map[string]interface{}{
		"filtering_summary": summaryDetails,
	}

	if jsonData, err := json.Marshal(atpData); err == nil {
		atpDP.Attributes().PutStr("process.atp", string(jsonData))
	}

	p.logger.Info("Generated filtering summary metrics",
		zap.Float64("efficiency_ratio", efficiencyRatio),
		zap.Int("filtered_resources", filteredResourceCount),
		zap.Int("included_resources", outputResourceCount),
		zap.Any("stage_hits", stageHits))
}

// isResourceTargeted checks if any metric in the values map is present in the configuration
func (p *processorImp) isResourceTargeted(values map[string]float64) bool {
	// Check static thresholds
	if p.config.MetricThresholds != nil {
		for m := range values {
			if _, ok := p.config.MetricThresholds[m]; ok {
				return true
			}
		}
	}

	// Check dynamic thresholds
	if p.dynamicThresholdsEnabled && p.dynamicCustomThresholds != nil {
		for m := range values {
			if _, ok := p.dynamicCustomThresholds[m]; ok {
				return true
			}
		}
	}

	return false
}
