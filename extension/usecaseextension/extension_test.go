// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:      "nil use case config returns error",
			cfg:       &Config{},
			expectErr: true,
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