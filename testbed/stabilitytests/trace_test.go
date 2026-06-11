// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/stabilitytests/trace_test.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

// Package stabilitytests contains long-running test cases verifying that otel-collector can run
// sustainably for long time, 1 hour by default.
// Tests supposed to be run on CircleCI, each tests must be allocated to exactly one runner
// to make sure that the whole test suit will not take longer than one hour.
// Because of that, every time overall number of stability tests changed,
// make sure to update CircleCI parameter: run-stability-tests.runners-number

package tests

import (
	"testing"
	"time"

	"github.com/newrelic/nrdot-collector-components/testbed/testbed"

	"github.com/newrelic/nrdot-collector-components/internal/common/testutil"
	scenarios "github.com/newrelic/nrdot-collector-components/testbed/tests"
)

var (
	contribPerfResultsSummary = &testbed.PerformanceResults{}
	resourceCheckPeriod, _    = time.ParseDuration("1m")
	processorsConfig          = []scenarios.ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
`,
		},
	}
)

// TestMain is used to initiate setup, execution and tear down of testbed.
func TestMain(m *testing.M) {
	testbed.DoTestMain(m, contribPerfResultsSummary)
}

func TestStabilityTracesOTLP(t *testing.T) {
	scenarios.Scenario10kItemsPerSecond(
		t,
		testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
		testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
		testbed.ResourceSpec{
			ExpectedMaxCPU:      20,
			ExpectedMaxRAM:      80,
			ResourceCheckPeriod: resourceCheckPeriod,
		},
		contribPerfResultsSummary,
		processorsConfig,
		nil,
		nil,
	)
}
