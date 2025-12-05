// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nopexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewNopFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &struct{}{}, cfg)

	traces, err := factory.CreateTraces(t.Context(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NoError(t, traces.Start(t.Context(), componenttest.NewNopHost()))
	assert.NoError(t, traces.ConsumeTraces(t.Context(), ptrace.NewTraces()))
	assert.NoError(t, traces.Shutdown(t.Context()))

	metrics, err := factory.CreateMetrics(t.Context(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NoError(t, metrics.Start(t.Context(), componenttest.NewNopHost()))
	assert.NoError(t, metrics.ConsumeMetrics(t.Context(), pmetric.NewMetrics()))
	assert.NoError(t, metrics.Shutdown(t.Context()))

	logs, err := factory.CreateLogs(t.Context(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NoError(t, logs.Start(t.Context(), componenttest.NewNopHost()))
	assert.NoError(t, logs.ConsumeLogs(t.Context(), plog.NewLogs()))
	assert.NoError(t, logs.Shutdown(t.Context()))
}
