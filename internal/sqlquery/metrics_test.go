// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/metrics_test.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package sqlquery // import "github.com/newrelic/nrdot-collector-components/internal/sqlquery"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestSetDataPointValue(t *testing.T) {
	err := setDataPointValue(&MetricCfg{
		ValueType:   MetricValueTypeInt,
		ValueColumn: "some-col",
	}, "", pmetric.NewNumberDataPoint())
	assert.EqualError(t, err, `setDataPointValue: col "some-col": error converting to integer: strconv.Atoi: parsing "": invalid syntax`)
}
