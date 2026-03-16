// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package osqueryreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("osqueryreceiver")
)

func createLogsReceiver(_ context.Context, params receiver.Settings, baseCfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	logger := params.Logger
	logRcvrConfig := baseCfg.(*Config)

	logRcvr := &osqueryReceiver{
		logger:       logger,
		nextConsumer: consumer,
		config:       logRcvrConfig,
	}

	return logRcvr, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelAlpha),
	)
}
