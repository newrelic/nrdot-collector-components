// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/testbed/in_process_collector_test.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package testbed

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/newrelic/nrdot-collector-components/internal/common/testutil"
)

func TestNewInProcessPipeline(t *testing.T) {
	factories, err := Components()
	assert.NoError(t, err)
	sender := NewOTLPTraceDataSender(DefaultHost, testutil.GetAvailablePort(t))
	receiver := NewOTLPDataReceiver(DefaultOTLPPort)
	runner, ok := NewInProcessCollector(factories).(*inProcessCollector)
	require.True(t, ok)

	format := `
receivers:%v
exporters:%v
processors:
  batch:

extensions:

service:
  extensions:
  pipelines:
    traces:
      receivers: [%v]
      processors: [batch]
      exporters: [%v]
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: '127.0.0.1'
                port: %d
`
	config := fmt.Sprintf(
		format,
		sender.GenConfigYAMLStr(),
		receiver.GenConfigYAMLStr(),
		sender.ProtocolName(),
		receiver.ProtocolName(),
		testutil.GetAvailablePort(t),
	)
	configCleanup, cfgErr := runner.PrepareConfig(t, config)
	defer configCleanup()
	assert.NoError(t, cfgErr)
	assert.NotNil(t, configCleanup)
	assert.NotEmpty(t, runner.configStr)
	args := StartParams{}
	defer func() {
		_, err := runner.Stop()
		require.NoError(t, err)
	}()
	assert.NoError(t, runner.Start(args))
	assert.NotNil(t, runner.svc)
}
