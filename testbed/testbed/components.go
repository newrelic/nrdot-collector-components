// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testbed // import "github.com/newrelic/nrdot-plus-collector-components/testbed/testbed"

import (
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/nrdotplustcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
	"go.uber.org/multierr"

	"github.com/newrelic/nrdot-plus-collector-components/connector/routingconnector"
	"github.com/newrelic/nrdot-plus-collector-components/connector/spanmetricsconnector"
	"github.com/newrelic/nrdot-plus-collector-components/exporter/stefexporter"
	"github.com/newrelic/nrdot-plus-collector-components/exporter/syslogexporter"
	"github.com/newrelic/nrdot-plus-collector-components/exporter/zipkinexporter"
	"github.com/newrelic/nrdot-plus-collector-components/receiver/jaegerreceiver"
	"github.com/newrelic/nrdot-plus-collector-components/receiver/otelarrowreceiver"
	"github.com/newrelic/nrdot-plus-collector-components/receiver/stefreceiver"
	"github.com/newrelic/nrdot-plus-collector-components/receiver/syslogreceiver"
	"github.com/newrelic/nrdot-plus-collector-components/receiver/zipkinreceiver"
)

// Components returns the set of components for tests
func Components() (
	nrdotplustcol.Factories,
	error,
) {
	var errs error

	extensions, err := nrdotplustcol.MakeFactoryMap[extension.Factory](
		zpagesextension.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	receivers, err := nrdotplustcol.MakeFactoryMap[receiver.Factory](
		jaegerreceiver.NewFactory(),
		otlpreceiver.NewFactory(),
		otelarrowreceiver.NewFactory(),
		stefreceiver.NewFactory(),
		syslogreceiver.NewFactory(),
		zipkinreceiver.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	exporters, err := nrdotplustcol.MakeFactoryMap[exporter.Factory](
		debugexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
		stefexporter.NewFactory(),
		syslogexporter.NewFactory(),
		zipkinexporter.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	processors, err := nrdotplustcol.MakeFactoryMap[processor.Factory](
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	connectors, err := nrdotplustcol.MakeFactoryMap[connector.Factory](
		spanmetricsconnector.NewFactory(),
		routingconnector.NewFactory(),
	)
	errs = multierr.Append(errs, err)

	factories := nrdotplustcol.Factories{
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
		Connectors: connectors,
		Telemetry:  otelconftelemetry.NewFactory(),
	}

	return factories, errs
}
