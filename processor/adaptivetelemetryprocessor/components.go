package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"go.opentelemetry.io/collector/processor"
)

// Components returns the processor factory for registration with the collector
func Components() []processor.Factory {
	return []processor.Factory{
		NewFactory(),
	}
}
