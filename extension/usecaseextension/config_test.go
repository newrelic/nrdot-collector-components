// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id            component.ID
		expected      component.Config
		expectedError error
	}{
		{
			id:            component.NewIDWithName(component.MustNewType("usecase"), ""),
			expectedError: errMissingUseCaseConfig,
		},
		{
			id: component.NewIDWithName(component.MustNewType("usecase"), "1"),
			expected: &Config{
				UseCaseConfig: &UseCaseConfig{
					Value: stringp("static_value"),
				},
			},
		},
		{
			id:            component.NewIDWithName(component.MustNewType("usecase"), "2"),
			expectedError: errMissingSource,
		},
	}
	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
			require.NoError(t, err)
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			sub, err := cm.Sub(tt.id.String())

			require.NoError(t, err)
			require.NoError(t, sub.Unmarshal(cfg))

			if tt.expectedError != nil {
				assert.ErrorIs(t, xconfmap.Validate(cfg), tt.expectedError)
				return
			}
			assert.NoError(t, xconfmap.Validate(cfg))
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		usecase     *UseCaseConfig
		expectedErr error
	}{
		{
			name:        "use case value from config property",
			usecase:     &UseCaseConfig{Value: stringp("from config")},
			expectedErr: nil,
		},
		{
			name:        "use case source is missing",
			usecase:     &UseCaseConfig{},
			expectedErr: errMissingSource,
		},
		{
			name:        "use case configuration is missing",
			usecase:     nil,
			expectedErr: errMissingUseCaseConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{UseCaseConfig: tt.usecase}
			require.ErrorIs(t, cfg.Validate(), tt.expectedErr)
		})
	}
}
