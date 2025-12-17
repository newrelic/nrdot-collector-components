// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "adaptivetelemetry", factory.Type().String())
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg)

	// Cast to *Config to call Validate
	pCfg, ok := cfg.(*Config)
	require.True(t, ok)
	assert.NoError(t, pCfg.Validate())
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	set := processortest.NewNopSettings(component.MustNewType("adaptivetelemetry"))
	processor, err := factory.CreateMetrics(t.Context(), set, cfg, consumertest.NewNop())

	require.NoError(t, err)
	assert.NotNil(t, processor)
}
