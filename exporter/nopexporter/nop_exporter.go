// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
