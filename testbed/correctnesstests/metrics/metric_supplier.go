// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/correctnesstests/metrics/metric_supplier.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package metrics // import "github.com/newrelic/nrdot-collector-components/testbed/correctnesstests/metrics"

import (
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type metricSupplier struct {
	pdms    []pmetric.Metrics
	currIdx int
}

func newMetricSupplier(pdms []pmetric.Metrics) *metricSupplier {
	return &metricSupplier{pdms: pdms}
}

func (p *metricSupplier) nextMetrics() (pdm pmetric.Metrics, done bool) {
	if p.currIdx == len(p.pdms) {
		return pmetric.Metrics{}, true
	}
	pdm = p.pdms[p.currIdx]
	p.currIdx++
	return pdm, false
}
