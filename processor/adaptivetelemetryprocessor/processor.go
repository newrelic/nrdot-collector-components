// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// adaptiveProcessor is the main processor implementation.
type adaptiveProcessor struct {
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Metrics
}

// newProcessor creates a new adaptive telemetry processor.
func newProcessor(cfg *Config, logger *zap.Logger, nextConsumer consumer.Metrics) *adaptiveProcessor {
	return &adaptiveProcessor{
		logger:       logger,
		config:       cfg,
		nextConsumer: nextConsumer,
	}
}

// Start is invoked during service startup.
func (p *adaptiveProcessor) Start(_ context.Context, _ component.Host) error {
	p.logger.Info("Starting adaptive telemetry processor")
	return nil
}

// Shutdown is invoked during service shutdown.
func (p *adaptiveProcessor) Shutdown(_ context.Context) error {
	p.logger.Info("Shutting down adaptive telemetry processor")
	return nil
}

// Capabilities returns the consumer capabilities of this processor.
func (*adaptiveProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics processes the incoming metrics.
// Currently, this is a pass-through implementation.
// TODO: Implement adaptive telemetry filtering logic.
func (p *adaptiveProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// TODO: Add adaptive telemetry processing logic here
	// For now, just pass metrics through unchanged
	return p.nextConsumer.ConsumeMetrics(ctx, md)
}
