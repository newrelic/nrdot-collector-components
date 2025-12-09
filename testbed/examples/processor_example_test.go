// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package examples provides example tests to help contributors get started with testbed.
//
// This file demonstrates how to write testbed tests for a new processor component.
// For most contributors adding a new processor, this is the pattern you'll follow.
package examples // import "github.com/newrelic/nrdot-collector-components/testbed/examples"

import (
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/datareceivers"
	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/datasenders"
	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
	"github.com/stretchr/testify/require"

	"github.com/newrelic/nrdot-collector-components/internal/common/testutil"
	"github.com/newrelic/nrdot-collector-components/testbed/correctnesstests"
)

// TestProcessorExample demonstrates the minimal setup needed to test a processor
// in the testbed framework.
//
// This example shows:
//  1. Setting up OTLP sender/receiver (simplest protocol)
//  2. Configuring your custom processor
//  3. Creating and running a correctness test
//  4. Validating data through the pipeline
//
// For processor testing, you typically want to verify that:
//  - Data flows through your processor without errors
//  - Your processor transforms data as expected
//  - All data sent is received (no data loss)
func TestProcessorExample(t *testing.T) {
	// STEP 1: Set up the data sender
	// OTLP is the simplest protocol and supports all signal types (traces, metrics, logs)
	// Use testutil.GetAvailablePort() to avoid port conflicts
	sender := datasenders.NewOTLPMetricDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))

	// STEP 2: Set up the data receiver
	// The receiver should match the sender protocol
	// This receives data from the collector after processing
	receiver := datareceivers.NewOTLPDataReceiver(testutil.GetAvailablePort(t))

	// STEP 3: Define processor configuration
	// This is where you configure YOUR processor that you're testing
	// Replace "batch" with your processor name and configuration
	processors := []correctnesstests.ProcessorNameAndConfigBody{
		{
			Name: "batch", // Change to your processor name (e.g., "myprocessor")
			Body: `
  batch:
    # Your processor configuration goes here
    # For example:
    # send_batch_size: 1024
    # timeout: 1s
`,
		},
		// You can add multiple processors to test processor chains:
		// {
		//     Name: "attributes",
		//     Body: `
		//   attributes:
		//     actions:
		//       - key: test.attribute
		//         action: insert
		//         value: test_value
		// `,
		// },
	}

	// STEP 4: Create a data provider
	// PerfTestDataProvider generates synthetic test data
	// For correctness tests with real data, use GoldenDataProvider instead
	dataProvider := testbed.NewPerfTestDataProvider()

	// STEP 5: Get collector component factories
	// This loads all available components (receivers, processors, exporters)
	factories, err := testbed.Components()
	require.NoError(t, err, "default components resulted in: %v", err)

	// STEP 6: Create in-process collector
	// This runs the collector as a goroutine in the same process as the test
	// For more realistic testing, use ChildProcess instead
	runner := testbed.NewInProcessCollector(factories)

	// STEP 7: Generate collector configuration YAML
	// This creates a complete collector config with your processors
	config := correctnesstests.CreateConfigYaml(t, sender, receiver, nil, processors)

	// Optional: Print the config for debugging
	// t.Logf("Collector config:\n%s", config)

	// STEP 8: Prepare the collector with the configuration
	configCleanup, err := runner.PrepareConfig(t, config)
	require.NoError(t, err, "collector configuration resulted in: %v", err)
	defer configCleanup()

	// STEP 9: Create a validator
	// CorrectTestValidator ensures data correctness through the pipeline
	validator := testbed.NewCorrectTestValidator(
		sender.ProtocolName(),
		receiver.ProtocolName(),
		dataProvider,
	)

	// STEP 10: Create the test case
	// TestCase orchestrates the entire test lifecycle
	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		runner,
		validator,
		&testbed.CorrectnessResults{},
	)
	defer tc.Stop()

	// STEP 11: Run the test
	// Start the backend (mock receiver)
	tc.StartBackend()

	// Start the collector agent
	tc.StartAgent()

	// Start generating and sending load
	tc.StartLoad(testbed.LoadOptions{
		DataItemsPerSecond: 100, // Rate of data generation
		ItemsPerBatch:      10,  // Number of items per batch
	})

	// Let the test run for a bit
	tc.Sleep(2 * time.Second)

	// Stop generating load
	tc.StopLoad()

	// STEP 12: Wait for all data to be received and validate
	// This ensures no data was lost during processing
	tc.WaitForN(func() bool {
		return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived()
	}, 10*time.Second, "all data items received")

	// Validate the data correctness
	// This checks that data was correctly transformed by your processor
	tc.ValidateData()
}

// TestProcessorWithCustomValidation shows how to add custom validation logic
// beyond the standard testbed validators.
//
// Use this pattern when you need to verify specific transformations
// that your processor performs on the data.
func TestProcessorWithCustomValidation(t *testing.T) {
	t.Skip("This is an example test - remove this line to enable")

	sender := datasenders.NewOTLPMetricDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))
	receiver := datareceivers.NewOTLPDataReceiver(testutil.GetAvailablePort(t))

	processors := []correctnesstests.ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
    send_batch_size: 100
`,
		},
	}

	dataProvider := testbed.NewPerfTestDataProvider()
	factories, err := testbed.Components()
	require.NoError(t, err)
	runner := testbed.NewInProcessCollector(factories)
	config := correctnesstests.CreateConfigYaml(t, sender, receiver, nil, processors)
	configCleanup, err := runner.PrepareConfig(t, config)
	require.NoError(t, err)
	defer configCleanup()

	validator := testbed.NewCorrectTestValidator(
		sender.ProtocolName(),
		receiver.ProtocolName(),
		dataProvider,
	)

	tc := testbed.NewTestCase(
		t,
		dataProvider,
		sender,
		receiver,
		runner,
		validator,
		&testbed.CorrectnessResults{},
	)
	defer tc.Stop()

	tc.StartBackend()
	tc.StartAgent()
	tc.StartLoad(testbed.LoadOptions{
		DataItemsPerSecond: 100,
		ItemsPerBatch:      10,
	})

	tc.Sleep(2 * time.Second)
	tc.StopLoad()
	tc.WaitForN(func() bool {
		return tc.LoadGenerator.DataItemsSent() == tc.MockBackend.DataItemsReceived()
	}, 10*time.Second, "all data items received")

	// Standard validation
	tc.ValidateData()

	// CUSTOM VALIDATION: Add your own checks here
	// For example, verify specific attributes or transformations:
	receivedMetrics := tc.MockBackend.ReceivedMetrics
	require.NotEmpty(t, receivedMetrics, "should have received metrics")

	// Add your custom assertions here
	// Example:
	// for _, metric := range receivedMetrics {
	//     // Verify your processor added/modified expected attributes
	//     require.Contains(t, metric.Attributes, "your_attribute")
	// }
}

// Additional Tips for Processor Testing:
//
// 1. CHOOSING PROTOCOLS:
//    - OTLP: Use for most tests (supports all signal types)
//    - Jaeger/Zipkin: Use if testing trace-specific interoperability
//    - Prometheus: Use if testing metrics-specific interoperability
//
// 2. DATA PROVIDERS:
//    - PerfTestDataProvider: Good for basic functional tests
//    - GoldenDataProvider: Use for comprehensive correctness tests with known data
//
// 3. RESOURCE SPECS:
//    Add resource specifications to test performance characteristics:
//    tc := testbed.NewTestCase(...,
//        testbed.WithResourceLimits(testbed.ResourceSpec{
//            ExpectedMaxCPU: 20,
//            ExpectedMaxRAM: 100,
//        }),
//    )
//
// 4. PERFORMANCE TESTING:
//    For performance tests, see testbed/tests/trace_test.go and metric_test.go
//    for examples using PerformanceResults instead of CorrectnessResults.
//
// 5. DEBUGGING:
//    - Use t.Logf() to print the generated config
//    - Enable verbose logging with: tc.EnableRecording()
//    - Check collector logs if tests fail
//
// 6. RUNNING TESTS:
//    From repo root:
//    - All testbed tests: make e2e-test
//    - This example: go test ./testbed/examples/...
