// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package usecasesetterextension // import "github.com/newrelic/nrdot-collector-components/extension/usecasesetterextension"

import (
	"errors"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/newrelic/nrdot-collector-components/extension/usecasesetterextension/internal/source"
)

var (
	_ extension.Extension = (*useCaseSetterExtension)(nil)
)

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
		source: determineSource(cfg.UseCaseConfig),
	}, nil
}

func (e *useCaseSetterExtension) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return &useCaseRoundTripper{
		base:   base,
		source: e.source,
	}, nil
}

func determineSource(u *UseCaseConfig) source.Source {
	var s source.Source

	switch {
	case u.Value != nil:
		s = &source.StaticSource{
			Value: *u.Value,
		}
	case u.FromAttribute != nil:
		defaultValue := ""
		if u.DefaultValue != nil {
			defaultValue = string(*u.DefaultValue)
		}
		s = &source.AttributeSource{
			Key:          *u.FromAttribute,
			DefaultValue: defaultValue,
		}
	case u.FromContext != nil:
		defaultValue := ""
		if u.DefaultValue != nil {
			defaultValue = string(*u.DefaultValue)
		}
		s = &source.ContextSource{
			Key:          *u.FromContext,
			DefaultValue: defaultValue,
		}
	}

	return s
}
