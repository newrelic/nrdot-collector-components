// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package mockdatadogagentexporter // import "github.com/newrelic/nrdot-collector-components/testbed/mockdatasenders/mockdatadogagentexporter"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

// Config defines configuration for datadog receiver.
type Config struct {
	component.Config
	// client to send to the agent
	confighttp.ClientConfig `mapstructure:",squash"`
}
