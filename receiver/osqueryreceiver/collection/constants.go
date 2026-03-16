// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

const (
	// system_info
	SystemInfoCollectionName  = "system_info"
	SystemInfoCollectionQuery = `SELECT hostname, uuid, cpu_type, cpu_subtype, cpu_brand, cpu_physical_cores, cpu_logical_cores, physical_memory, hardware_vendor, hardware_model 
FROM system_info;`

	// package_info
	PackageInfoCollectionName          = "package_info"
	PackageInfoCollectionQueryHomebrew = `SELECT * from homebrew_packages;`
	PackageInfoCollectionQueryDebian   = `SELECT * from deb_packages;`
	PackageInfoCollectionQueryRPM      = `SELECT * from rpm_packages;`

	// os_info
	OSInfoCollectionName  = "os_info"
	OSInfoCollectionQuery = `SELECT * FROM os_version;`

	// secureboot
	SecureBootCollectionName  = "secureboot_info"
	SecureBootCollectionQuery = `SELECT * FROM secureboot;`

	// users
	UserCollectionName       = "users_info"
	UserCollectionQueryLinux = `SELECT u.username, GROUP_CONCAT(g.groupname, ', ') AS groups
FROM users u
JOIN user_groups ug ON u.uid = ug.uid
JOIN groups g ON ug.gid = g.gid
WHERE
    u.uid >= 1000
    AND u.uid != 65534 -- Exclude 'nobody'
    AND u.shell NOT IN ('/usr/sbin/nologin', '/bin/false')
GROUP BY u.username;`
	UserCollectionQueryDarwin = `SELECT u.username, GROUP_CONCAT(g.groupname, ', ') AS groups
FROM users u
JOIN user_groups ug ON u.uid = ug.uid
JOIN groups g ON ug.gid = g.gid
WHERE
    u.uid >= 500
    AND u.uid != 65534 -- Exclude 'nobody'
    AND u.shell NOT IN ('/usr/sbin/nologin', '/bin/false')
GROUP BY u.username;`
	UserCollectionQueryWindows = `SELECT u.username, GROUP_CONCAT(g.groupname, ', ') AS groups
FROM users u
JOIN user_groups ug ON u.uid = ug.uid
JOIN groups g ON ug.gid = g.gid
WHERE
    u.directory LIKE 'C:\\Users\\%'
    AND
    u.username NOT IN ('Administrator', 'Guest', 'SYSTEM', 'LOCAL SERVICE', 'NETWORK SERVICE')
GROUP BY u.username;`
)
