// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/fake_db_client.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package oracledbreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/oracledbreceiver"

import (
	"context"
)

type fakeDbClient struct {
	Err            error
	Responses      [][]metricRow
	RequestCounter int
}

func (c *fakeDbClient) metricRows(context.Context, ...any) ([]metricRow, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	idx := c.RequestCounter
	c.RequestCounter++
	return c.Responses[idx], nil
}
