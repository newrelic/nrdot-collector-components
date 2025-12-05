// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package nopreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/nopreceiver"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewNopFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &struct{}{}, cfg)

	traces, err := factory.CreateTraces(t.Context(), receivertest.NewNopSettings(receivertest.NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, traces.Start(t.Context(), componenttest.NewNopHost()))
	assert.NoError(t, traces.Shutdown(t.Context()))

	metrics, err := factory.CreateMetrics(t.Context(), receivertest.NewNopSettings(receivertest.NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metrics.Start(t.Context(), componenttest.NewNopHost()))
	assert.NoError(t, metrics.Shutdown(t.Context()))

	logs, err := factory.CreateLogs(t.Context(), receivertest.NewNopSettings(receivertest.NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logs.Start(t.Context(), componenttest.NewNopHost()))
	assert.NoError(t, logs.Shutdown(t.Context()))
}
