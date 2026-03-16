// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

import "runtime"

// table_name("homebrew_packages")
// description("The installed homebrew package database.")
// schema([
//     Column("name", TEXT, "Package name"),
//     Column("path", TEXT, "Package install path"),
//     Column("version", TEXT, "Current 'linked' version", collate="version"),
//     Column("type", TEXT, "Package type ('formula' or 'cask')"),
//     Column("auto_updates", INTEGER, "1 if the cask auto-updates otherwise 0"),
//     Column("app_name", TEXT, "Name of the installed App (for Casks)"),
//     Column("prefix", TEXT, "Homebrew install prefix", hidden=True, additional=True, optimized=True),
// ])
// attributes(cacheable=True)
// implementation("system/homebrew_packages@genHomebrewPackages")

// table_name("deb_packages")
// description("The installed DEB package database.")
// schema([
//     Column("name", TEXT, "Package name"),
//     Column("version", TEXT, "Package version", collate="version_dpkg"),
//     Column("source", TEXT, "Package source"),
//     Column("size", BIGINT, "Package size in bytes"),
//     Column("arch", TEXT, "Package architecture"),
//     Column("revision", TEXT, "Package revision"),
//     Column("status", TEXT, "Package status"),
//     Column("maintainer", TEXT, "Package maintainer"),
//     Column("section", TEXT, "Package section"),
//     Column("priority", TEXT, "Package priority"),
//     Column("admindir", TEXT, "libdpkg admindir. Defaults to /var/lib/dpkg", additional=True, optimized=True),
// ])
// extended_schema(LINUX, [
//     Column("pid_with_namespace", INTEGER, "Pids that contain a namespace", additional=True, hidden=True),
//     Column("mount_namespace_id", TEXT, "Mount namespace id", hidden=True),
// ])
// attributes(cacheable=True)
// implementation("system/deb_packages@genDebPackages")
// fuzz_paths([
//     "/var/lib/dpkg",
// ])

// table_name("rpm_packages")
// description("RPM packages that are currently installed on the host system.")
// schema([
//     Column("name", TEXT, "RPM package name", index=True, optimized=True),
//     Column("version", TEXT, "Package version" ,index=True, collate="version_rhel"),
//     Column("release", TEXT, "Package release", index=True),
//     Column("source", TEXT, "Source RPM package name (optional)"),
//     Column("size", BIGINT, "Package size in bytes"),
//     Column("sha1", TEXT, "SHA1 hash of the package contents"),
//     Column("arch", TEXT, "Architecture(s) supported", index=True),
//     Column("epoch", INTEGER, "Package epoch value", index=True),
//     Column("install_time", INTEGER, "When the package was installed"),
//     Column("vendor", TEXT, "Package vendor"),
//     Column("package_group", TEXT, "Package group")
// ])
// extended_schema(LINUX, [
//     Column("pid_with_namespace", INTEGER, "Pids that contain a namespace", additional=True, hidden=True),
//     Column("mount_namespace_id", TEXT, "Mount namespace id", hidden=True),
// ])
// attributes(cacheable=True)
// implementation("@genRpmPackages")

// PackageInfoCollection represents OS package information
// Supports multiple package managers across different operating systems:
// - Darwin: homebrew_packages, packages
// - Linux (Debian): deb_packages
// - Linux (RPM): rpm_packages
type PackageInfoCollection struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Type        string `json:"type,omitempty"`     // homebrew package type: formula or cask
	Path        string `json:"path,omitempty"`     // homebrew package install path
	AppName     string `json:"app_name,omitempty"` // homebrew app name for casks
	Source      string `json:"source,omitempty"`
	Vendor      string `json:"vendor,omitempty"`
	Arch        string `json:"arch,omitempty"`
	Size        int64  `json:"size,omitempty"` // in bytes for deb and rpm
	InstallTime string `json:"install_time,omitempty"`
}

var (
	packageInfoStringFields = []string{
		"name",
		"version",
		"type",
		"path",
		"app_name",
		"source",
		"vendor",
		"arch",
		"install_time",
	}
	packageInfoInt64Fields = []string{
		"size",
	}
)

func (p PackageInfoCollection) GetName() string {
	return PackageInfoCollectionName
}

func (p PackageInfoCollection) GetQuery() string {
	// Otherwise, auto-detect based on OS
	return p.queryForOS(runtime.GOOS)
}

// TODO: User util function for this pattern
func (p PackageInfoCollection) queryForOS(os string) string {
	switch os {
	case "darwin":
		// Default to homebrew on macOS as it's most common
		return PackageInfoCollectionQueryHomebrew
	case "linux":
		// TODO: Improve detection between deb and rpm based systems
		// For now, default to debian-based systems
		return PackageInfoCollectionQueryDebian
	default:
		// Fallback to homebrew query
		return PackageInfoCollectionQueryHomebrew
	}
}

func (p PackageInfoCollection) Unmarshal(result any) any {
	resultSlice, ok := result.([]map[string]any)
	if !ok {
		return nil
	}

	packages := make([]map[string]any, 0, len(resultSlice))
	for _, resultMap := range resultSlice {
		sanitized := sanitizeRow(resultMap, packageInfoStringFields, nil, packageInfoInt64Fields)
		if len(sanitized) == 0 {
			continue
		}
		packages = append(packages, sanitized)
	}

	if len(packages) == 0 {
		return nil
	}

	return packages
}

// NewPackageInfoCollection creates a new package info collection with OS detection
func NewPackageInfoCollection() ICollection {
	return PackageInfoCollection{}
}
