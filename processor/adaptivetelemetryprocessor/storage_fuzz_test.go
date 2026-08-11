// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzFileStorageLoad(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		tmpFile := filepath.Join(t.TempDir(), "storage.json")
		if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
			return
		}
		s := newFileStorageForTesting(tmpFile, "")
		_, _ = s.Load()
	})
}
