// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor

import (
	"encoding/json"
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func FuzzUpdateProcessATPAttribute(f *testing.F) {
	f.Add(`{"a":1}`)
	f.Add(``)
	f.Add(`not json`)
	f.Add(`{"nested":{"x":[1,2,3]}}`)

	f.Fuzz(func(t *testing.T, existing string) {
		resource := pcommon.NewResource()
		resource.Attributes().PutStr("process.atp", existing)

		updateProcessATPAttribute(resource, "testkey", "testvalue", nil)

		val, ok := resource.Attributes().Get("process.atp")
		if !ok {
			t.Fatal("process.atp attribute missing after update")
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(val.AsString()), &out); err != nil {
			t.Fatalf("resulting process.atp is not valid JSON: %v", err)
		}
	})
}
