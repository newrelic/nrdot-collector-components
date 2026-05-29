// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// NewFactory creates a factory for the use case setter extension.
func NewFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType("usecase"),
		createDefaultConfig,
		createExtension,
		component.StabilityLevelAlpha,
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createExtension(
	_ context.Context,
	_ extension.Settings,
	cfg component.Config,
) (extension.Extension, error) {
	return newUseCaseSetterExtension(cfg.(*Config))
}
