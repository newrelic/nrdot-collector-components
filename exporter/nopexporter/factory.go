// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package nopexporter // import "github.com/newrelic/nrdot-collector-components/exporter/nopexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"

	"github.com/newrelic/nrdot-collector-components/exporter/nopexporter/internal/metadata"
)

// NewFactory returns an exporter.Factory that constructs nop exporters.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		func() component.Config { return &Config{} },
		exporter.WithTraces(createTraces, metadata.TracesStability),
		exporter.WithMetrics(createMetrics, metadata.MetricsStability),
		exporter.WithLogs(createLogs, metadata.LogsStability),
	)
}

func createTraces(context.Context, exporter.Settings, component.Config) (exporter.Traces, error) {
	return nopInstance, nil
}

func createMetrics(context.Context, exporter.Settings, component.Config) (exporter.Metrics, error) {
	return nopInstance, nil
}

func createLogs(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
	return nopInstance, nil
}
