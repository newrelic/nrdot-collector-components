// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension"

import (
	"errors"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/newrelic/nrdot-collector-components/extension/usecaseextension/internal/source"
)

var _ extension.Extension = (*useCaseSetterExtension)(nil)

type useCaseSetterExtension struct {
	component.StartFunc
	component.ShutdownFunc

	source source.Source
}

func newUseCaseSetterExtension(cfg *Config) (*useCaseSetterExtension, error) {
	if cfg == nil {
		return nil, errors.New("extension configuration is not provided")
	}
	if cfg.UseCaseConfig == nil {
		return nil, errMissingUseCaseConfig
	}

	return &useCaseSetterExtension{
		source: &source.StaticSource{
			Value: *cfg.UseCaseConfig.Value,
		},
	}, nil
}

func (e *useCaseSetterExtension) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return &useCaseRoundTripper{
		base:   base,
		source: e.source,
	}, nil
}
