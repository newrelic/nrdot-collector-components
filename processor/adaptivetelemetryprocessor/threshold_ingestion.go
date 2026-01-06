// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"encoding/json"
	"math"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

// isValidMetricValue checks if a metric value is valid (not NaN or Inf)
func isValidMetricValue(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// isValidThreshold checks if a threshold value is valid and positive
func isValidThreshold(threshold float64) bool {
	return isValidMetricValue(threshold) && threshold > 0
}

// determineEffectiveThreshold selects the appropriate threshold (dynamic or static)
func (p *processorImp) determineEffectiveThreshold(metricName string, staticThreshold float64) (float64, string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Try dynamic threshold first if enabled
	if p.dynamicThresholdsEnabled && p.dynamicCustomThresholds != nil {
		if dt, exists := p.dynamicCustomThresholds[metricName]; exists && isValidThreshold(dt) {
			return dt, "dynamic", true
		}
	}

	// Fallback to static threshold
	if isValidThreshold(staticThreshold) {
		return staticThreshold, "static", true
	}

	return 0, "", false
}

// addThresholdAttributes adds threshold-related attributes to the resource
func addThresholdAttributes(attrs pcommon.Map, metricName string, threshold, value float64, thresholdType string) {
	thresholdData := map[string]interface{}{
		"threshold":            threshold,
		"observed_value":       value,
		"threshold_type":       thresholdType,
		"evaluation_timestamp": time.Now().Unix(),
	}

	if jsonData, err := json.Marshal(thresholdData); err == nil {
		attrs.PutStr(atpThresholdPrefix+metricName, string(jsonData))
	}
}

// captureUsedMetricThresholds captures only metric thresholds that are actually evaluated
func (p *processorImp) captureUsedMetricThresholds(resource pcommon.Resource, values map[string]float64) {
	defer func() {
		if r := recover(); r != nil && p.logger != nil {
			p.logger.Error("panic in captureUsedMetricThresholds", zap.Any("error", r))
		}
	}()

	// Early returns for invalid states
	if p == nil || p.config == nil || p.config.MetricThresholds == nil || len(values) == 0 {
		return
	}

	attrs := resource.Attributes()
	capturedCount := 0

	for metricName, metricValue := range values {
		if metricName == "" {
			continue
		}

		staticThreshold, hasStatic := p.config.MetricThresholds[metricName]
		if !hasStatic || !isValidMetricValue(metricValue) {
			continue
		}

		effectiveThreshold, thresholdType, isValid := p.determineEffectiveThreshold(metricName, staticThreshold)
		if !isValid {
			continue
		}

		addThresholdAttributes(attrs, metricName, effectiveThreshold, metricValue, thresholdType)
		capturedCount++
	}

	if p.logger != nil && capturedCount > 0 {
		p.logger.Debug("Metric thresholds captured", zap.Int("metrics_count", capturedCount))
	}
}
