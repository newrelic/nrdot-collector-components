// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"context"
	"path/filepath"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

// newProcessor constructs the processor with configured features and storage.
func newProcessor(logger *zap.Logger, config *Config, nextConsumer consumer.Metrics) (*processorImp, error) {
	// Normalize & validate config first
	config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Check if storage is enabled (defaults to true if not specified)
	storageEnabled := true
	if config.EnableStorage != nil {
		logger.Info("DEBUG: EnableStorage config field is set", zap.Bool("value", *config.EnableStorage))
		storageEnabled = *config.EnableStorage
	} else {
		logger.Info("DEBUG: EnableStorage config field is nil, using default", zap.Bool("default", true))
	}

	// Use default storage path based on platform
	storagePath := getDefaultStoragePath()

	logger.Info("Initializing adaptivetelemetryprocessor",
		zap.Int("metric_thresholds_count", len(config.MetricThresholds)),
		zap.Bool("dynamic_thresholds_enabled", config.EnableDynamicThresholds),
		zap.Bool("multi_metric_enabled", config.EnableMultiMetric),
		zap.Bool("anomaly_detection_enabled", config.EnableAnomalyDetection),
		zap.Bool("storage_enabled", storageEnabled),
		zap.Int64("retention_minutes", config.RetentionMinutes),
		zap.Float64("composite_threshold", config.CompositeThreshold),
		zap.Float64("anomaly_change_threshold", config.AnomalyChangeThreshold),
		zap.String("storage_path", storagePath))

	logger.Info("Resource agnostic processing enabled",
		zap.String("supported_resources", "cpu, disk, filesystem, load, memory, network, process, processes, paging"))

	p := &processorImp{
		logger:                   logger,
		config:                   config,
		nextConsumer:             nextConsumer,
		trackedEntities:          make(map[string]*trackedEntity),
		persistenceEnabled:       storageEnabled,
		dynamicThresholdsEnabled: config.EnableDynamicThresholds,
		multiMetricEnabled:       config.EnableMultiMetric,
		lastThresholdUpdate:      time.Now(),
		dynamicCustomThresholds:  make(map[string]float64),
	}

	// Seed dynamic thresholds with configured static thresholds
	for m, base := range config.MetricThresholds {
		if base > 0 {
			p.dynamicCustomThresholds[m] = base
			logger.Debug("Seeded dynamic threshold", zap.String("metric", m), zap.Float64("initial_threshold", base))
		}
	}

	if p.dynamicThresholdsEnabled {
		logger.Info("Dynamic thresholds enabled", zap.Float64("smoothing_factor", config.DynamicSmoothingFactor), zap.Int("metrics_tracked", len(p.dynamicCustomThresholds)))
	}
	if p.multiMetricEnabled {
		logger.Info("Multi-metric evaluation enabled", zap.Float64("composite_threshold", config.CompositeThreshold), zap.Int("weights_count", len(config.Weights)))
	}
	if config.EnableAnomalyDetection {
		logger.Info("Anomaly detection enabled", zap.Int("history_size", config.AnomalyHistorySize), zap.Float64("change_threshold", config.AnomalyChangeThreshold))
	}

	if p.persistenceEnabled {
		logger.Info("Setting up persistent storage", zap.String("path", storagePath))
		storageDir := filepath.Dir(storagePath)
		if err := createDirectoryIfNotExists(storageDir); err != nil {
			logger.Warn("Failed to create storage directory, continuing without persistence",
				zap.String("path", storageDir),
				zap.Error(err))
			p.persistenceEnabled = false
		} else {
			p.storage = newFileStorage(storagePath)
			if err := p.loadTrackedEntities(); err != nil {
				logger.Warn("Failed to load tracked entities, starting with empty state",
					zap.Error(err))
				// Continue without loaded state - don't disable persistence for future saves
			}
		}
	}

	// Log processor configuration
	configSummary := map[string]any{
		"dynamic_thresholds_enabled": p.dynamicThresholdsEnabled,
		"multi_metric_enabled":       p.multiMetricEnabled,
		"anomaly_detection_enabled":  config.EnableAnomalyDetection,
		"persistence_enabled":        p.persistenceEnabled,
		"retention_minutes":          float64(config.RetentionMinutes), // Convert int64 to float64 for consistent display
		"metric_thresholds_count":    len(config.MetricThresholds),
		"min_thresholds_count":       len(config.MinThresholds),
		"max_thresholds_count":       len(config.MaxThresholds),
		"weights_count":              len(config.Weights),
		"dynamic_smoothing_factor":   config.DynamicSmoothingFactor,
		"composite_threshold":        config.CompositeThreshold,
		"anomaly_history_size":       config.AnomalyHistorySize,
		"anomaly_change_threshold":   config.AnomalyChangeThreshold,
	}

	logger.Info("adaptivetelemetryprocessor initialized successfully with this configuration", zap.Any("config", configSummary))

	return p, nil
}

// Shutdown cleans up processor resources
func (p *processorImp) Shutdown(_ context.Context) error {
	if p.persistenceEnabled && p.storage != nil {
		if err := p.persistTrackedEntities(); err != nil {
			p.logger.Warn("Failed to persist tracked entities during shutdown", zap.Error(err))
		}
		if err := p.storage.Close(); err != nil {
			p.logger.Warn("Failed to close storage during shutdown", zap.Error(err))
		}
	}
	return nil
}

// Start is a no-op for this processor
func (*processorImp) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Capabilities indicates that this processor mutates data
func (*processorImp) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
