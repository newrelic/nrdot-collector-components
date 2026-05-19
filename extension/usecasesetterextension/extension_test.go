// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package usecasesetterextension

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configopaque"
)

func TestNewUseCaseSetterExtension(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		expectErr bool
	}{
		{
			name:      "nil config returns error",
			cfg:       nil,
			expectErr: true,
		},
		{
			name: "static value source",
			cfg: &Config{
				UseCaseConfig: &UseCaseConfig{
					Value: stringp("some-value"),
				},
			},
		},
		{
			name: "from context source",
			cfg: &Config{
				UseCaseConfig: &UseCaseConfig{
					FromContext: stringp("tenant_id"),
				},
			},
		},
		{
			name: "from attribute source",
			cfg: &Config{
				UseCaseConfig: &UseCaseConfig{
					FromAttribute: stringp("attr_key"),
				},
			},
		},
		{
			name: "from context source with default value",
			cfg: &Config{
				UseCaseConfig: &UseCaseConfig{
					FromContext:  stringp("tenant_id"),
					DefaultValue: opaquep("default_tenant"),
				},
			},
		},
		{
			name: "from attribute source with default value",
			cfg: &Config{
				UseCaseConfig: &UseCaseConfig{
					FromAttribute: stringp("attr_key"),
					DefaultValue:  opaquep("default_value"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, err := newUseCaseSetterExtension(tt.cfg)
			if tt.expectErr {
				require.Error(t, err)
				assert.Nil(t, ext)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, ext)
		})
	}
}

func stringp(str string) *string {
	return &str
}

func opaquep(stro configopaque.String) *configopaque.String {
	return &stro
}
