// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension"

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	errMissingSource         = errors.New("missing use case source, must set 'id'")
	errEmptyUseCaseID        = errors.New("use case id cannot be empty")
	errInvalidUseCaseIDChars = errors.New("use case id contains invalid characters, only alphanumeric, forward slash, underscore, hyphen, and period are allowed")

	// useCaseIDPattern defines allowed characters for use case ID
	// Only alphanumeric, forward slash, underscore, hyphen, and period are allowed
	useCaseIDPattern = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)
)

type Config struct {
	Id *string `mapstructure:"id"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Id == nil {
		return errMissingSource
	}

	id := *cfg.Id
	if id == "" {
		return errEmptyUseCaseID
	}

	if !useCaseIDPattern.MatchString(id) {
		return fmt.Errorf("%w: %q", errInvalidUseCaseIDChars, id)
	}

	return nil
}
