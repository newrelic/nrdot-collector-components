// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/correctnesstests/metrics/metric_index.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package metrics // import "github.com/newrelic/nrdot-collector-components/testbed/correctnesstests/metrics"

import (
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type metricReceived struct {
	pdm      pmetric.Metrics
	received bool
}

type metricsReceivedIndex struct {
	m map[string]*metricReceived
}

func newMetricsReceivedIndex(pdms []pmetric.Metrics) *metricsReceivedIndex {
	mi := &metricsReceivedIndex{m: map[string]*metricReceived{}}
	for _, pdm := range pdms {
		metrics := pdm.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
		name := metrics.At(0).Name()
		mi.m[name] = &metricReceived{pdm: pdm}
	}
	return mi
}

func (mi *metricsReceivedIndex) lookup(name string) (*metricReceived, bool) {
	mr, ok := mi.m[name]
	return mr, ok
}

func (mi *metricsReceivedIndex) allReceived() bool {
	for _, m := range mi.m {
		if !m.received {
			return false
		}
	}
	return true
}
