// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecasesetterextension // import "github.com/newrelic/nrdot-collector-components/extension/usecasesetterextension"

import (
	"errors"

	"go.opentelemetry.io/collector/config/configopaque"
)

var (
	errMissingUseCaseConfig = errors.New("missing use case configuration")
	errMissingSource        = errors.New("missing use case source, must be 'from_context', 'from_attribute' or 'value'")
	errConflictingSources   = errors.New("invalid use case source, must either 'from_context', 'from_attribute' or 'value'")
)

type Config struct {
	UseCaseConfig *UseCaseConfig `mapstructure:"usecase"`

	// prevent unkeyed literal initialization
	_ struct{}
}

type UseCaseConfig struct {
	Value         *string              `mapstructure:"value"`
	FromContext   *string              `mapstructure:"from_context"`
	FromAttribute *string              `mapstructure:"from_attribute"`
	DefaultValue  *configopaque.String `mapstructure:"default_value"`
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.UseCaseConfig == nil {
		return errMissingUseCaseConfig
	}
	u := cfg.UseCaseConfig
	if u.FromContext == nil && u.FromAttribute == nil && u.Value == nil {
		return errMissingSource
	}
	if (u.FromContext != nil && u.FromAttribute != nil) ||
		(u.FromContext != nil && u.Value != nil) ||
		(u.Value != nil && u.FromAttribute != nil) {
		return errConflictingSources
	}
	return nil
}
