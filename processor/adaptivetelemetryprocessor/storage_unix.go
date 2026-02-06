// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"os"
)

// isWindowsReparsePoint is a no-op on non-Windows platforms.
// It always returns false since reparse points (junctions, mount points) are Windows-specific.
func isWindowsReparsePoint(_ string, _ os.FileInfo) bool {
	return false
}
