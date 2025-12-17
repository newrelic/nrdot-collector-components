// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

const (
	typeStr = "adaptivetelemetry"
)

// NewFactory creates the processor.Factory used by the Collector to construct this processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType(typeStr),
		createDefaultConfig,
		processor.WithMetrics(createMetricsProcessor, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig returns the default configuration for this processor.
func createDefaultConfig() component.Config {
	return &Config{
		// TODO: Add default configuration values as fields are added
	}
}

// createMetricsProcessor constructs the processor for metrics pipelines.
func createMetricsProcessor(
	_ context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	pCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: expected *Config, got %T", cfg)
	}

	return newProcessor(pCfg, set.Logger, nextConsumer), nil
}
