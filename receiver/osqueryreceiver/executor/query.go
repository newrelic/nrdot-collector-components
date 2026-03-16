// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"time"
)

type QueryResult map[string]any

type QueryExecution struct {
	Query         string
	ExecutedAt    time.Time
	ResultCount   int
	TransformInto any
	State         any
	Error         error
}
