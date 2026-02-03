package tests

import (
	"testing"

	"github.com/newrelic/nrdot-collector-components/internal/common/testutil"
	"github.com/newrelic/nrdot-collector-components/testbed/testbed"
)

/*
Sandbox: We could dynamically fetch processors instead.
*/
var processors = []string{
	"processor/adaptivetelemetryprocessor",
}

func TestNRDOTProcessors10kDPS(t *testing.T) {
	test := struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
		skipMessage  string
	}{
		name:     "ATP",
		sender:   testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
		receiver: testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t)),
		resourceSpec: testbed.ResourceSpec{
			ExpectedMaxCPU: 60,
			ExpectedMaxRAM: 105,
		},
	}

	/*
		Sandbox: We might want to consider per-use-case load tests instead of (or alongside) per-component load tests.
		For example, passing non-host metrics into a host processor might tell us if something's gone catastrophically wrong,
		but there might be more value in passing common host metrics. There may also be value in load-testing multiple
		"host use case" processors together, when it comes to it.
	*/
	for _, processor := range processors {
		metadata := testbed.NewComponentMetadata(processor)
		configBody, err := metadata.GetTestConfigBody()
		if err != nil {
			t.Errorf("failed to get test config for %s: %v", processor, err)
			continue
		}
		processorConfigs := []ProcessorNameAndConfigBody{
			{
				Name: metadata.GetFullComponentName(),
				Body: configBody,
			},
		}
		t.Run(test.name, func(t *testing.T) {
			if test.skipMessage != "" {
				t.Skip(test.skipMessage)
			}
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				processorConfigs,
				nil,
				nil,
			)
		})
	}

}
