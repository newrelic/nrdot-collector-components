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
			expectedError: errMissingSource,
		},
		{
			id: component.NewIDWithName(component.MustNewType("usecase"), "1"),
			expected: &Config{
				Id: stringp("host-monitoring/1.15.1"),
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
		id          *string
		expectedErr error
	}{
		{
			name:        "use case id from config property",
			id:          stringp("from-config"),
			expectedErr: nil,
		},
		{
			name:        "use case id with valid special characters",
			id:          stringp("host-monitoring/1.15.1"),
			expectedErr: nil,
		},
		{
			name:        "use case id with underscores and hyphens",
			id:          stringp("my_use-case.v1"),
			expectedErr: nil,
		},
		{
			name:        "use case source is missing",
			id:          nil,
			expectedErr: errMissingSource,
		},
		{
			name:        "empty use case id",
			id:          stringp(""),
			expectedErr: errEmptyUseCaseID,
		},
		{
			name:        "use case id with newline",
			id:          stringp("test\n"),
			expectedErr: errInvalidUseCaseIDChars,
		},
		{
			name:        "use case id with parentheses",
			id:          stringp("test(prod)"),
			expectedErr: errInvalidUseCaseIDChars,
		},
		{
			name:        "use case id with spaces",
			id:          stringp("test case"),
			expectedErr: errInvalidUseCaseIDChars,
		},
		{
			name:        "use case id with special chars",
			id:          stringp("test@case"),
			expectedErr: errInvalidUseCaseIDChars,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Id: tt.id}
			require.ErrorIs(t, cfg.Validate(), tt.expectedErr)
		})
	}
}
