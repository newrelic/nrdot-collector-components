// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

// Config defines configuration for the adaptive telemetry processor.
//
// Example configuration:
//
//	processors:
//	  adaptivetelemetry:
//	    # Add configuration options here as the processor is developed
type Config struct {
	// TODO: Add configuration fields as features are implemented
	// Example:
	// MetricThresholds map[string]float64 `mapstructure:"metric_thresholds"`
}

// Validate checks the configuration for errors.
func (*Config) Validate() error {
	// TODO: Add validation logic as configuration fields are added
	return nil
}
