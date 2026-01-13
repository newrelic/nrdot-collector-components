// Copyright 2023 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

// detectAnomaly is a method on the processorImp struct that delegates to the utility function.
// This method is used by the process_metrics.go code.
func (p *processorImp) detectAnomaly(entity *trackedEntity, values map[string]float64) (bool, string) {
	return detectAnomalyUtil(p, entity, values)
}
