// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func FuzzUpdateProcessATPAttribute(f *testing.F) {
	f.Fuzz(func(_ *testing.T, existing string) {
		resource := pcommon.NewResource()
		resource.Attributes().PutStr("process.atp", existing)
		updateProcessATPAttribute(resource, "testkey", "testvalue", nil)
	})
}
