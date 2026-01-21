// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package nopexporter // import "github.com/newrelic/nrdot-collector-components/exporter/nopexporter"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

var nopInstance = &nop{
	Consumer: consumertest.NewNop(),
}

type nop struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}
