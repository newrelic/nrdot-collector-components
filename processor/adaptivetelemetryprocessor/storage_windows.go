// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"os"
	"syscall"
	"unsafe"
)

// isWindowsReparsePoint checks if a path is a Windows reparse point (junction or mount point).
// These are similar to symlinks but are not detected by the os.ModeSymlink flag.
// This is a Windows-specific security check to prevent redirection attacks.
//
// On Windows:
// - Symbolic links are detected by os.ModeSymlink
// - Directory junctions are NOT detected by os.ModeSymlink (they use reparse points)
// - Mount points are NOT detected by os.ModeSymlink (they also use reparse points)
//
// We need to check the FILE_ATTRIBUTE_REPARSE_POINT flag to detect these.
func isWindowsReparsePoint(path string, info os.FileInfo) bool {
	// Note: We don't check info.IsDir() first because junctions may not appear as
	// regular directories via os.Lstat(). Instead, we always check the reparse point
	// attribute directly from Windows syscalls for any path that exists.

	// Get the raw syscall.Win32FileAttributeData to check the attributes
	// We need to use syscall package to access Windows-specific file attributes
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		// If we can't convert the path, we can't check - be safe and assume it's suspicious
		return true
	}

	var data syscall.Win32FileAttributeData
	err = syscall.GetFileAttributesEx(pathPtr, syscall.GetFileExInfoStandard, (*byte)(unsafe.Pointer(&data)))
	if err != nil {
		// If we can't get attributes, we can't verify safety - be safe and assume it's suspicious
		return true
	}

	// fileAttributeReparsePoint = 0x400
	// This flag is set for junctions, mount points, and symbolic links
	const fileAttributeReparsePoint = 0x400
	return (data.FileAttributes & fileAttributeReparsePoint) != 0
}
