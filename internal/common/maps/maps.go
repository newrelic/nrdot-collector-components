// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/internal/common/maps/maps.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package maps // import "github.com/newrelic/nrdot-collector-components/internal/common/maps"

import (
	extmaps "maps"
)

// MergeRawMaps merges n maps with a later map's keys overriding earlier maps.
func MergeRawMaps(maps ...map[string]any) map[string]any {
	ret := map[string]any{}

	for _, m := range maps {
		extmaps.Copy(ret, m)
	}

	return ret
}

// MergeStringMaps merges n maps with a later map's keys overriding earlier maps.
func MergeStringMaps(maps ...map[string]string) map[string]string {
	ret := map[string]string{}

	for _, m := range maps {
		extmaps.Copy(ret, m)
	}

	return ret
}

// CloneStringMap makes a shallow copy of a map[string]string.
func CloneStringMap(m map[string]string) map[string]string {
	m2 := make(map[string]string, len(m))
	extmaps.Copy(m2, m)
	return m2
}
