// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

import (
	"strings"

	"github.com/newrelic/nrdot-collector-components/receiver/osqueryreceiver/util"
)

// sanitizeRow trims string values and removes empty/zero values for the provided keys.
// It ensures that only meaningful data is kept so state comparisons ignore empty fields.
// It is not scalable, and needs to be changed, but leaving it for now as it is POC 😎
func sanitizeRow(resultMap map[string]any, stringKeys, intKeys, int64Keys []string) map[string]any {
	sanitized := make(map[string]any)

	for _, key := range stringKeys {
		value := strings.TrimSpace(util.GetString(resultMap, key))
		if value != "" {
			sanitized[key] = value
		}
	}

	for _, key := range intKeys {
		value := util.GetInt(resultMap, key)
		if value > 0 {
			sanitized[key] = float64(value)
		}
	}

	for _, key := range int64Keys {
		value := util.GetInt64(resultMap, key)
		if value > 0 {
			sanitized[key] = float64(value)
		}
	}

	return sanitized
}
