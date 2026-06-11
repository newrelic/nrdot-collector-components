// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/tests/log_test.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

// Package tests contains test cases. To run the tests go to tests directory and run:
// RUN_TESTBED=1 go test -v

package tests

import (
	"testing"
	"time"

	"github.com/newrelic/nrdot-collector-components/internal/common/testutil"
	"github.com/newrelic/nrdot-collector-components/testbed/datasenders"
	"github.com/newrelic/nrdot-collector-components/testbed/testbed"
)

func TestLog10kDPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
		extensions   map[string]string
	}{
		{
			name:     "OTLP",
			sender:   testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 30,
				ExpectedMaxRAM: 120,
			},
		},
		{
			name:     "OTLP-HTTP",
			sender:   testbed.NewOTLPHTTPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
			receiver: testbed.NewOTLPHTTPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 30,
				ExpectedMaxRAM: 120,
			},
		},
		{
			name:     "kubernetes containers",
			sender:   datasenders.NewKubernetesContainerWriter(),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 110,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "kubernetes containers parser",
			sender:   datasenders.NewKubernetesContainerParserWriter(),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 110,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "k8s CRI-Containerd",
			sender:   datasenders.NewKubernetesCRIContainerdWriter(),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "k8s CRI-Containerd no attr ops",
			sender:   datasenders.NewKubernetesCRIContainerdNoAttributesOpsWriter(),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "CRI-Containerd",
			sender:   datasenders.NewCRIContainerdWriter(),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 100,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "syslog-tcp-batch-1",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testutil.GetAvailablePort(t), 1),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "syslog-tcp-batch-100",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testutil.GetAvailablePort(t), 100),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "tcp-batch-1",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testutil.GetAvailablePort(t), 1),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
		{
			name:     "tcp-batch-100",
			sender:   datasenders.NewTCPUDPWriter("tcp", testbed.DefaultHost, testutil.GetAvailablePort(t), 100),
			receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
			resourceSpec: testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 150,
			},
		},
	}

	processors := []ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				processors,
				test.extensions,
				nil,
			)
		})
	}
}

func TestLogOtlpSendingQueue(t *testing.T) {
	otlpreceiver10 := testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t))
	otlpreceiver10.WithRetry(`
    retry_on_failure:
      enabled: true
`)
	otlpreceiver10.WithQueue(`
    sending_queue:
      enabled: true
      queue_size: 10
`)
	t.Run("OTLP-sending-queue-full", func(t *testing.T) {
		ScenarioSendingQueuesFull(
			t,
			testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
			otlpreceiver10,
			testbed.LoadOptions{
				DataItemsPerSecond: 100,
				ItemsPerBatch:      10,
				Parallel:           1,
			},
			testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 120,
			}, 10,
			performanceResultsSummary,
			nil,
			nil,
		)
	})

	otlpreceiver100 := testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t))
	otlpreceiver100.WithRetry(`
    retry_on_failure:
      enabled: true
`)
	otlpreceiver100.WithQueue(`
    sending_queue:
      enabled: true
      queue_size: 100
`)
	t.Run("OTLP-sending-queue-not-full", func(t *testing.T) {
		ScenarioSendingQueuesNotFull(
			t,
			testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
			otlpreceiver100,
			testbed.LoadOptions{
				DataItemsPerSecond: 100,
				ItemsPerBatch:      10,
				Parallel:           1,
			},
			testbed.ResourceSpec{
				ExpectedMaxCPU: 80,
				ExpectedMaxRAM: 120,
			}, 10,
			performanceResultsSummary,
			nil,
			nil,
		)
	})
}

func TestMemoryLimiterHit(t *testing.T) {
	tests := []struct {
		name   string
		sender testbed.DataSender
	}{
		{
			name:   "otlp",
			sender: testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			otlpreceiver := testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t))
			otlpreceiver.WithRetry(`
    retry_on_failure:
      enabled: true
      max_interval: 5s
`)
			otlpreceiver.WithQueue(`
    sending_queue:
       enabled: true
       queue_size: 100000
       num_consumers: 20
`)
			otlpreceiver.WithTimeout(`
    timeout: 0s
`)
			processors := []ProcessorNameAndConfigBody{
				{
					Name: "memory_limiter",
					Body: `
  memory_limiter:
    check_interval: 1s
    limit_mib: 300
    spike_limit_mib: 150
`,
				},
			}
			ScenarioMemoryLimiterHit(
				t,
				test.sender,
				otlpreceiver,
				testbed.LoadOptions{
					DataItemsPerSecond: 100000,
					ItemsPerBatch:      1000,
					Parallel:           1,
					MaxDelay:           20 * time.Second,
				},
				performanceResultsSummary, 100, processors,
			)
		})
	}
}
