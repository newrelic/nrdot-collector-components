// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap/zaptest"
)

func TestProcessorLifecycle(t *testing.T) {
	// Create processor
	cfg := &Config{}
	logger := zaptest.NewLogger(t)
	nextConsumer := &consumertest.MetricsSink{}

	proc := newProcessor(cfg, logger, nextConsumer)
	require.NotNil(t, proc)

	// Test Start
	ctx := t.Context()
	err := proc.Start(ctx, nil)
	assert.NoError(t, err)

	// Test ConsumeMetrics (pass-through)
	metrics := pmetric.NewMetrics()
	err = proc.ConsumeMetrics(ctx, metrics)
	assert.NoError(t, err)

	// Verify metrics were passed through
	allMetrics := nextConsumer.AllMetrics()
	assert.Len(t, allMetrics, 1)

	// Test Shutdown
	err = proc.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProcessorCapabilities(t *testing.T) {
	cfg := &Config{}
	logger := zaptest.NewLogger(t)
	nextConsumer := &consumertest.MetricsSink{}

	proc := newProcessor(cfg, logger, nextConsumer)
	require.NotNil(t, proc)

	capabilities := proc.Capabilities()
	assert.False(t, capabilities.MutatesData, "processor should not mutate data yet")
}
