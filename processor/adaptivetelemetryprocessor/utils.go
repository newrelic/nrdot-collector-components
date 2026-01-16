// Copyright New Relic, Inc. All rights reserved.
// New Relic Software License

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// buildResourceIdentity returns a stable, human-readable identity string for a resource.
// Priority order:
// - service or entity identifiers
// - hostmetrics specific identifiers (CPU, disk, filesystem, etc)
// - service.instance.id or service.name(+service.namespace)
// - fallback: sorted concatenation of all resource attributes
func buildResourceIdentity(res pcommon.Resource) string {
	attrs := res.Attributes()

	// Get host information if available for any resource
	host := getHostName(attrs)

	// Try host metrics identification first
	if identity := buildHostMetricIdentity(attrs, host); identity != "" {
		return identity
	}

	// Try service identification
	if identity := buildServiceIdentity(attrs); identity != "" {
		return identity
	}

	// Fallback to attribute concatenation
	return buildFallbackIdentity(attrs)
}

// getHostName extracts the host name from resource attributes
func getHostName(attrs pcommon.Map) string {
	if hv, ok := attrs.Get("host.name"); ok {
		return hv.AsString()
	}
	return ""
}

// buildHostMetricIdentity builds identity for host metrics with specific formatting
func buildHostMetricIdentity(attrs pcommon.Map, host string) string {
	hostMetricType, isHostMetric := identifyHostMetricType(attrs)
	if !isHostMetric {
		return ""
	}

	switch hostMetricType {
	case resourceTypeCPU:
		return buildCPUIdentity(attrs, host)
	case resourceTypeProcess:
		return buildProcessIdentity(attrs, host)
	case resourceTypeDisk, resourceTypeNetwork:
		return buildDeviceBasedIdentity(attrs, host, hostMetricType)
	case resourceTypeFilesystem:
		return buildFilesystemIdentity(attrs, host)
	case resourceTypePaging:
		return buildPagingIdentity(attrs, host)
	case resourceTypeLoad, resourceTypeMemory, resourceTypeProcesses, resourceTypeSystem:
		return buildSimpleHostIdentity(host, hostMetricType)
	default:
		return ""
	}
}

// buildCPUIdentity builds identity for CPU metrics
func buildCPUIdentity(attrs pcommon.Map, host string) string {
	cpuNum := ""
	if cpu, ok := attrs.Get("cpu"); ok {
		cpuNum = "." + cpu.AsString()
	}

	if host != "" {
		return fmt.Sprintf("cpu%s@%s", cpuNum, host)
	}
	return fmt.Sprintf("cpu%s", cpuNum)
}

// buildProcessIdentity builds identity for individual process metrics
// Uses process.pid as the primary identifier (unique per process per host)
func buildProcessIdentity(attrs pcommon.Map, host string) string {
	pid := ""
	if pidVal, ok := attrs.Get("process.pid"); ok {
		pid = fmt.Sprintf("%v", pidVal.AsRaw())
	}

	// Fallback: if no PID, try process.command (shouldn't happen in practice)
	if pid == "" {
		if cmd, ok := attrs.Get("process.command"); ok {
			pid = cmd.AsString()
		}
	}

	// Still no identifier? Use "unknown"
	if pid == "" {
		pid = "unknown"
	}

	if host != "" {
		return fmt.Sprintf("process.%s@%s", pid, host)
	}
	return fmt.Sprintf("process.%s", pid)
}

// buildPagingIdentity builds identity for paging metrics
// Paging metrics can have different attribute combinations:
// - system.paging.faults: type (major/minor) - system-wide
// - system.paging.operations: direction (page_in/page_out), type - system-wide
// - system.paging.usage/utilization: device, state (cached/free/used) - per device
func buildPagingIdentity(attrs pcommon.Map, host string) string {
	// Check for device attribute (page file/swap device specific)
	if dev, hasDevice := attrs.Get("device"); hasDevice {
		device := dev.AsString()
		if host != "" {
			return fmt.Sprintf("paging.%s@%s", device, host)
		}
		return fmt.Sprintf("paging.%s", device)
	}

	// No device - system-wide paging metric (faults, operations)
	if host != "" {
		return fmt.Sprintf("paging@%s", host)
	}
	return "paging"
}

// buildFilesystemIdentity builds identity for filesystem metrics
// Uses mountpoint as the primary identifier (most meaningful for filesystem tracking)
// Falls back to device if no mountpoint available
func buildFilesystemIdentity(attrs pcommon.Map, host string) string {
	identifier := ""

	// Primary: Use mountpoint (e.g., "/", "/home", "/mnt/data")
	if mountpoint, ok := attrs.Get("mountpoint"); ok {
		identifier = mountpoint.AsString()
	}

	// Fallback: Use device if no mountpoint
	if identifier == "" {
		if device, ok := attrs.Get("device"); ok {
			identifier = device.AsString()
		}
	}

	// Still no identifier? Use "unknown"
	if identifier == "" {
		identifier = "unknown"
	}

	if host != "" {
		return fmt.Sprintf("filesystem.%s@%s", identifier, host)
	}
	return fmt.Sprintf("filesystem.%s", identifier)
}

// buildDeviceBasedIdentity builds identity for device-based metrics (disk, network)
func buildDeviceBasedIdentity(attrs pcommon.Map, host, hostMetricType string) string {
	device := ""
	if dev, ok := attrs.Get("device"); ok {
		device = "." + dev.AsString()
	}

	if host != "" {
		return fmt.Sprintf("%s%s@%s", hostMetricType, device, host)
	}
	return fmt.Sprintf("%s%s", hostMetricType, device)
}

// buildSimpleHostIdentity builds identity for simple host metrics (load, memory, processes, system)
func buildSimpleHostIdentity(host, hostMetricType string) string {
	if host != "" {
		return fmt.Sprintf("%s@%s", hostMetricType, host)
	}
	return hostMetricType
}

// buildServiceIdentity builds identity for service-based resources
func buildServiceIdentity(attrs pcommon.Map) string {
	// Check for service.instance.id first
	if v, ok := attrs.Get("service.instance.id"); ok {
		return "service.instance.id:" + v.AsString()
	}

	// Check for service.name with optional namespace
	if v, ok := attrs.Get("service.name"); ok {
		serviceName := v.AsString()
		namespace := ""
		if nv, ok := attrs.Get("service.namespace"); ok {
			namespace = nv.AsString()
		}

		if namespace != "" {
			return "service:" + namespace + "/" + serviceName
		}
		return "service:" + serviceName
	}

	return ""
}

// buildFallbackIdentity builds identity from all resource attributes as fallback
func buildFallbackIdentity(attrs pcommon.Map) string {
	// Collect all attribute keys
	keys := make([]string, 0, attrs.Len())
	attrs.Range(func(k string, _ pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})

	if len(keys) == 0 {
		return "resource:empty"
	}

	// Sort keys for deterministic output
	sort.Strings(keys)

	// Build key=value pairs
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		if v, ok := attrs.Get(k); ok {
			parts = append(parts, k+"="+v.AsString())
		}
	}

	// Join and truncate if necessary
	id := strings.Join(parts, ",")
	if len(id) > 512 {
		id = id[:512]
	}
	return id
}

// snapshotResourceAttributes copies resource attributes into a plain map for logging/debugging
func snapshotResourceAttributes(res pcommon.Resource) map[string]string {
	out := map[string]string{}
	res.Attributes().Range(func(k string, v pcommon.Value) bool {
		out[k] = v.AsString()
		return true
	})
	return out
}

// extractProcessName extracts the process name from resource attributes.
// It tries the following in order:
// 1. process.executable.name - the process binary name (e.g., "nginx", "postgres")
// 2. process.command - the full command (may include path, we extract basename)
// 3. Empty string if neither is available
func extractProcessName(attrs pcommon.Map) string {
	// Try process.executable.name first (most reliable)
	if execName, ok := attrs.Get("process.executable.name"); ok && execName.Str() != "" {
		return execName.Str()
	}

	// Fallback to process.command (extract basename if path is present)
	if cmd, ok := attrs.Get("process.command"); ok && cmd.Str() != "" {
		cmdStr := cmd.Str()
		// Extract basename from path (handle both Unix and Windows paths)
		if idx := strings.LastIndexAny(cmdStr, "/\\"); idx >= 0 {
			return cmdStr[idx+1:]
		}
		return cmdStr
	}

	return ""
}

// isProcessInIncludeList checks if a process should be included based on the include list.
// It matches the process name against entries in the includeList.
// Returns true if the process is in the include list, false otherwise.
func isProcessInIncludeList(attrs pcommon.Map, includeList []string) bool {
	if len(includeList) == 0 {
		return false
	}

	// Extract process name
	processName := extractProcessName(attrs)
	if processName == "" {
		return false
	}

	// Check if process name matches any entry in the include list
	for _, includedProcess := range includeList {
		if includedProcess == processName {
			return true
		}
	}

	return false
}
