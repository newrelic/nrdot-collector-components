package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"go.opentelemetry.io/collector/processor"
)

// components returns the processor factory for registration with the collector
func components() []processor.Factory {
	return []processor.Factory{
		NewFactory(),
	}
}
