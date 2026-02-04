// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/correctnesstests/metrics/metrics_correctness_test.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/newrelic/nrdot-collector-components/testbed/correctnesstests"
	"github.com/newrelic/nrdot-collector-components/testbed/testbed"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/newrelic/nrdot-collector-components/internal/coreinternal/goldendataset"
	"github.com/newrelic/nrdot-collector-components/internal/coreinternal/metricstestutil"
)

// tests with the prefix "TestHarness_" get run in the "correctnesstests-metrics" ci job
func TestHarness_MetricsGoldenData(t *testing.T) {
	tests, err := correctnesstests.LoadPictOutputPipelineDefs(
		"testdata/generated_pict_pairs_metrics_pipeline.txt",
	)
	require.NoError(t, err)

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	res := results{}
	res.Init("results")
	for _, test := range tests {
		test.TestName = fmt.Sprintf("%s-%s", test.Receiver, test.Exporter)
		test.DataSender = correctnesstests.ConstructMetricsSender(t, test.Receiver)
		test.DataReceiver = correctnesstests.ConstructReceiver(t, test.Exporter)
		t.Run(test.TestName, func(t *testing.T) {
			r := testWithMetricsGoldenDataset(
				t,
				test.DataSender.(testbed.MetricDataSender),
				test.DataReceiver,
			)
			res.Add("", r)
		})
	}
	res.Save()
}

func testWithMetricsGoldenDataset(
	t *testing.T,
	sender testbed.MetricDataSender,
	receiver testbed.DataReceiver,
) result {
	mds := getTestMetrics(t)
	accumulator := newDiffAccumulator()
	h := newTestHarness(
		t,
		newMetricSupplier(mds),
		newMetricsReceivedIndex(mds),
		sender,
		accumulator,
	)

	/*
		Sandbox: Pull in custom component, dynamically extract config from test metadata, see traces output
	*/
	atpMetadata := testbed.NewComponentMetadata("processor/adaptivetelemetryprocessor")

	atpConfig, err := atpMetadata.GetTestConfigBody()
	if err != nil {
		return handleTestError(err, t)
	}

	fmt.Println(atpConfig)

	processors := []correctnesstests.ProcessorNameAndConfigBody{
		{
			Name: atpMetadata.GetFullComponentName(),
			Body: atpConfig,
		},
	}

	tc := newCorrectnessTestCase(t, sender, receiver, processors, h)

	tc.startTestbedReceiver()
	tc.startCollector()
	tc.startTestbedSender()

	tc.sendFirstMetric()
	tc.waitForAllMetrics()

	tc.stopTestbedReceiver()
	tc.stopCollector()

	r := result{
		testName:   t.Name(),
		testResult: "PASS",
		numDiffs:   accumulator.numDiffs,
	}
	if accumulator.numDiffs > 0 {
		r.testResult = "FAIL"
		t.Fail()
	}
	return r
}

func handleTestError(err error, t *testing.T) result {
	fmt.Fprint(os.Stderr, err)
	t.Fail()
	return result{
		testName:   t.Name(),
		testResult: "FAIL",
		numDiffs:   0,
	}
}

func getTestMetrics(t *testing.T) []pmetric.Metrics {
	const file = "../../../internal/coreinternal/goldendataset/testdata/generated_pict_pairs_metrics.txt"
	mds, err := goldendataset.GenerateMetrics(file)
	require.NoError(t, err)
	return mds
}

type diffAccumulator struct {
	numDiffs int
}

var _ diffConsumer = (*diffAccumulator)(nil)

func newDiffAccumulator() *diffAccumulator {
	return &diffAccumulator{}
}

func (d *diffAccumulator) accept(metricName string, diffs []*metricstestutil.MetricDiff) {
	if len(diffs) > 0 {
		d.numDiffs++
		log.Printf("Found diffs for [%v]\n%v", metricName, diffs)
	}
}
