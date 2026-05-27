// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension"

import "errors"

var (
	errMissingUseCaseConfig = errors.New("missing use case configuration")
	errMissingSource        = errors.New("missing use case source, must set 'value'")
)

type Config struct {
	UseCaseConfig *UseCaseConfig `mapstructure:"usecase"`

	// prevent unkeyed literal initialization
	_ struct{}
}

type UseCaseConfig struct {
	Value *string `mapstructure:"value"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.UseCaseConfig == nil {
		return errMissingUseCaseConfig
	}
	if cfg.UseCaseConfig.Value == nil {
		return errMissingSource
	}
	return nil
}
