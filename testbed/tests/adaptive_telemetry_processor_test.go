// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/newrelic/nrdot-collector-components/internal/common/testutil"
)

// atpProcessorConfig returns a standard ATP YAML config block.
// Storage is disabled because the testbed environment does not have /var/lib/nrdot-collector/.
func atpProcessorConfig(cpuThreshold float64) string {
	return fmt.Sprintf(`
  adaptivetelemetry:
    metric_thresholds:
      process.cpu.utilization: %.1f
    retention_minutes: 1
    enable_storage: false
`, cpuThreshold)
}

// processEntry describes one process to include in a makeProcessMetrics batch.
// execPath is optional; it sets process.executable.path on the resource, which is
// required for include-list matching (isProcessInIncludeList only accepts full paths).
type processEntry struct {
	pid      int64
	name     string
	cpuUtil  float64
	execPath string // optional absolute path; set when testing include-list bypass
}

// makeProcessMetrics builds a pmetric.Metrics with one ResourceMetrics per process entry.
// Each resource carries the minimal attributes ATP needs to identify a process:
//   - process.pid              (triggers buildProcessIdentity → isResourceTargeted)
//   - process.executable.name  (used for logging/debug)
//   - host.name                (part of the identity key)
//   - process.executable.path  (optional; required for include-list matching because
//                               isProcessInIncludeList only matches full-path entries)
func makeProcessMetrics(processes []processEntry) pmetric.Metrics {
	md := pmetric.NewMetrics()
	for _, p := range processes {
		rm := md.ResourceMetrics().AppendEmpty()
		attrs := rm.Resource().Attributes()
		attrs.PutInt("process.pid", p.pid)
		attrs.PutStr("process.executable.name", p.name)
		attrs.PutStr("host.name", "test-host")
		if p.execPath != "" {
			attrs.PutStr("process.executable.path", p.execPath)
		}

		sm := rm.ScopeMetrics().AppendEmpty()
		m := sm.Metrics().AppendEmpty()
		m.SetName("process.cpu.utilization")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(p.cpuUtil)
	}
	return md
}

// makeHostMetrics returns a single non-process resource metric (system.memory.usage).
// ATP must always forward these (stageDefaultInclusion) because they have no process.pid.
func makeHostMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("host.name", "test-host")

	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("system.memory.usage")
	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetIntValue(1024 * 1024 * 512) // 512 MB
	return md
}

// setupATPTestCase wires a full testbed pipeline with the ATP processor and returns the
// TestCase, a MetricDataSender ready to call ConsumeMetrics, and a cleanup func.
func setupATPTestCase(t *testing.T, processorBody string) (
	tc *testbed.TestCase,
	metricSender testbed.MetricDataSender,
	cleanup func(),
) {
	t.Helper()

	resultDir, err := filepath.Abs(filepath.Join("results", t.Name()))
	require.NoError(t, err)

	otlpSender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))
	receiver := testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t))

	agentProc := testbed.NewChildProcessCollector(testbed.WithEnvVar("GOMAXPROCS", "2"))
	processors := []ProcessorNameAndConfigBody{
		{Name: "adaptivetelemetry", Body: processorBody},
	}

	configStr := createConfigYaml(t, otlpSender, receiver, resultDir, processors, nil)
	configCleanup, err := agentProc.PrepareConfig(t, configStr)
	require.NoError(t, err)

	options := testbed.LoadOptions{DataItemsPerSecond: 100, ItemsPerBatch: 10}
	dataProvider := testbed.NewPerfTestDataProvider(options)

	tc = testbed.NewTestCase(
		t,
		dataProvider,
		otlpSender,
		receiver,
		agentProc,
		&testbed.PerfTestValidator{},
		performanceResultsSummary,
	)

	tc.StartBackend()
	tc.StartAgent()
	tc.EnableRecording()
	require.NoError(t, otlpSender.Start())

	providerSender, ok := tc.LoadGenerator.(*testbed.ProviderSender)
	require.True(t, ok, "load generator must be a ProviderSender")
	metricSender, ok = providerSender.Sender.(testbed.MetricDataSender)
	require.True(t, ok, "sender must implement MetricDataSender")

	cleanup = func() {
		configCleanup()
		tc.Stop()
	}
	return tc, metricSender, cleanup
}

// containsPID returns true if any of the received metrics batches contains
// a ResourceMetrics whose process.pid attribute equals the given pid.
func containsPID(received []pmetric.Metrics, pid int64) bool {
	for _, md := range received {
		for i := 0; i < md.ResourceMetrics().Len(); i++ {
			pidVal, ok := md.ResourceMetrics().At(i).Resource().Attributes().Get("process.pid")
			if ok && pidVal.Int() == pid {
				return true
			}
		}
	}
	return false
}

// hasATPMarkerForPID returns whether a resource for pid has an ATP marker and a
// diagnostic summary of what ATP-related attributes were seen.
// Compatibility note: some binaries set only process.atp.enabled, while others
// also populate process.atp JSON details.
func hasATPMarkerForPID(received []pmetric.Metrics, pid int64) (bool, string) {
	seen := []string{}
	matchedPID := false

	for _, md := range received {
		for i := 0; i < md.ResourceMetrics().Len(); i++ {
			rm := md.ResourceMetrics().At(i)
			pidVal, hasPID := rm.Resource().Attributes().Get("process.pid")
			if !hasPID || pidVal.Int() != pid {
				continue
			}
			matchedPID = true
			attrs := rm.Resource().Attributes()

			if atpVal, hasATP := attrs.Get("process.atp"); hasATP {
				if atpVal.AsString() != "" {
					seen = append(seen, "process.atp(non-empty)")
					return true, fmt.Sprintf("%v", seen)
				}
				seen = append(seen, "process.atp(empty)")
			}

			if enabledVal, hasEnabled := attrs.Get("process.atp.enabled"); hasEnabled {
				if enabledVal.AsString() == "true" {
					seen = append(seen, "process.atp.enabled=true")
					return true, fmt.Sprintf("%v", seen)
				}
				seen = append(seen, "process.atp.enabled="+enabledVal.AsString())
			}
		}
	}

	if !matchedPID {
		return false, "pid not found"
	}
	if len(seen) == 0 {
		return false, "no ATP marker attributes present"
	}
	return false, fmt.Sprintf("%v", seen)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 1: Process metrics ABOVE threshold are forwarded.
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_HighCPUProcessIsForwarded(t *testing.T) {
	tc, sender, cleanup := setupATPTestCase(t, atpProcessorConfig(5.0))
	defer cleanup()

	md := makeProcessMetrics([]processEntry{
		{pid: 1001, name: "high-cpu-proc", cpuUtil: 80.0},
	})

	before := tc.MockBackend.DataItemsReceived()
	require.NoError(t, sender.ConsumeMetrics(context.Background(), md))
	tc.LoadGenerator.IncDataItemsSent()
	tc.WaitFor(func() bool {
		return tc.MockBackend.DataItemsReceived() > before
	}, "high-CPU metric must arrive at backend")

	assert.True(t,
		containsPID(tc.MockBackend.ReceivedMetrics, 1001),
		"high-CPU process (pid=1001) must be forwarded through ATP",
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 2: Process metrics BELOW threshold are filtered out.
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_LowCPUProcessIsFiltered(t *testing.T) {
	tc, sender, cleanup := setupATPTestCase(t, atpProcessorConfig(5.0))
	defer cleanup()

	// pid=9999 is an above-threshold anchor that guarantees the filtered result set is
	// non-empty. This prevents ATP's zero-output safety guard (present in the current
	// binary) from re-emitting the original batch when all resources would otherwise be
	// filtered. The anchor arrival also gives us a deterministic "batch fully processed"
	// signal, replacing the fragile fixed tc.Sleep approach.
	md := makeProcessMetrics([]processEntry{
		{pid: 9999, name: "anchor-proc", cpuUtil: 99.0}, // above threshold (99% > 5%) → forwarded
		{pid: 2001, name: "idle-proc", cpuUtil: 0.1},    // below threshold (0.1% < 5%) → must be filtered
	})

	tc.MockBackend.ClearReceivedItems()
	require.NoError(t, sender.ConsumeMetrics(context.Background(), md))
	tc.LoadGenerator.IncDataItemsSent()

	// Wait until the anchor arrives — this proves ATP has fully processed the batch.
	// Only then check that the idle process was not forwarded.
	tc.WaitFor(func() bool {
		return containsPID(tc.MockBackend.ReceivedMetrics, 9999)
	}, "anchor (pid=9999) must arrive at backend; proves the batch was processed by ATP")

	assert.False(t,
		containsPID(tc.MockBackend.ReceivedMetrics, 2001),
		"idle process (pid=2001) should have been filtered by ATP",
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 3: Mixed batch — only the high-CPU process survives.
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_MixedBatch_OnlyHighCPUForwarded(t *testing.T) {
	tc, sender, cleanup := setupATPTestCase(t, atpProcessorConfig(5.0))
	defer cleanup()

	md := makeProcessMetrics([]processEntry{
		{pid: 3001, name: "heavy-proc", cpuUtil: 50.0},
		{pid: 3002, name: "light-proc", cpuUtil: 1.0},
	})

	before := tc.MockBackend.DataItemsReceived()
	require.NoError(t, sender.ConsumeMetrics(context.Background(), md))
	tc.LoadGenerator.IncDataItemsSent()
	tc.WaitFor(func() bool {
		return tc.MockBackend.DataItemsReceived() > before
	}, "at least the heavy process must arrive")

	assert.True(t,
		containsPID(tc.MockBackend.ReceivedMetrics, 3001),
		"heavy-proc (pid=3001) must be forwarded through ATP",
	)
	assert.False(t,
		containsPID(tc.MockBackend.ReceivedMetrics, 3002),
		"light-proc (pid=3002) should have been filtered by ATP",
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 4: Non-process (host) metrics always pass through (stageDefaultInclusion).
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_HostMetricsPassThrough(t *testing.T) {
	tc, sender, cleanup := setupATPTestCase(t, atpProcessorConfig(5.0))
	defer cleanup()

	md := makeHostMetrics()

	before := tc.MockBackend.DataItemsReceived()
	require.NoError(t, sender.ConsumeMetrics(context.Background(), md))
	tc.LoadGenerator.IncDataItemsSent()
	tc.WaitFor(func() bool {
		return tc.MockBackend.DataItemsReceived() > before
	}, "host metric must arrive at backend")

	// Confirm a resource with host.name=test-host and NO process.pid was received.
	found := false
	for _, received := range tc.MockBackend.ReceivedMetrics {
		for i := 0; i < received.ResourceMetrics().Len(); i++ {
			rm := received.ResourceMetrics().At(i)
			attrs := rm.Resource().Attributes()
			_, hasPID := attrs.Get("process.pid")
			hostVal, hasHost := attrs.Get("host.name")
			if !hasPID && hasHost && hostVal.AsString() == "test-host" {
				found = true
			}
		}
	}
	assert.True(t, found, "host memory metric must pass through ATP unchanged (stageDefaultInclusion)")
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 5: Process in the include list bypasses the CPU threshold.
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_IncludeListBypassesThreshold(t *testing.T) {
	// IMPORTANT: isProcessInIncludeList only matches entries that contain a path separator
	// (/ or \) — bare process names are rejected as a security measure against spoofing.
	// The test resource must also carry process.executable.path so the matcher has a path
	// to compare against.
	processorBody := `
  adaptivetelemetry:
    metric_thresholds:
      process.cpu.utilization: 5.0
    include_process_list:
      - "/test/critical-agent"
    retention_minutes: 1
    enable_storage: false
`
	tc, sender, cleanup := setupATPTestCase(t, processorBody)
	defer cleanup()

	md := makeProcessMetrics([]processEntry{
		{pid: 4001, name: "critical-agent", cpuUtil: 0.0, execPath: "/test/critical-agent"}, // in include list, zero CPU → must pass
		{pid: 4002, name: "normal-proc", cpuUtil: 0.1},                                      // not in list, below threshold → filtered
	})

	before := tc.MockBackend.DataItemsReceived()
	require.NoError(t, sender.ConsumeMetrics(context.Background(), md))
	tc.LoadGenerator.IncDataItemsSent()
	tc.WaitFor(func() bool {
		return tc.MockBackend.DataItemsReceived() > before
	}, "critical-agent must arrive at backend via include list")

	assert.True(t,
		containsPID(tc.MockBackend.ReceivedMetrics, 4001),
		"critical-agent (pid=4001) must bypass threshold via include list",
	)
	assert.False(t,
		containsPID(tc.MockBackend.ReceivedMetrics, 4002),
		"normal-proc (pid=4002) should have been filtered by ATP",
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 6: ATP attaches marker attributes to targeted, forwarded resources.
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_ProcessATPAttributePresent(t *testing.T) {
	tc, sender, cleanup := setupATPTestCase(t, atpProcessorConfig(5.0))
	defer cleanup()

	md := makeProcessMetrics([]processEntry{
		{pid: 5001, name: "monitored-proc", cpuUtil: 90.0},
	})

	before := tc.MockBackend.DataItemsReceived()
	require.NoError(t, sender.ConsumeMetrics(context.Background(), md))
	tc.LoadGenerator.IncDataItemsSent()
	tc.WaitFor(func() bool {
		return tc.MockBackend.DataItemsReceived() > before
	}, "monitored-proc metric must arrive at backend")

	markerFound, markerDetails := hasATPMarkerForPID(tc.MockBackend.ReceivedMetrics, 5001)
	assert.True(t, markerFound,
		"an ATP marker attribute must be present on targeted process pid=5001 (seen=%s)", markerDetails)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 7: Performance — ATP with 10 k DPS stays within CPU/RAM limits.
// ─────────────────────────────────────────────────────────────────────────────

func TestATP_10kDPS_Performance(t *testing.T) {
	sender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))
	receiver := testbed.NewOTLPDataReceiver(testutil.GetAvailablePort(t))

	processors := []ProcessorNameAndConfigBody{
		{
			Name: "adaptivetelemetry",
			Body: `
  adaptivetelemetry:
    metric_thresholds:
      process.cpu.utilization: 5.0
    retention_minutes: 1
    enable_storage: false
`,
		},
	}

	Scenario10kItemsPerSecond(
		t,
		sender,
		receiver,
		testbed.ResourceSpec{
			ExpectedMaxCPU: 60,  // 60 % CPU at most
			ExpectedMaxRAM: 150, // 150 MB RAM at most
		},
		performanceResultsSummary,
		processors,
		nil,
		nil,
	)
}
