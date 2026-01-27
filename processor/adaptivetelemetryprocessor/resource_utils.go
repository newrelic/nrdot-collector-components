// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adaptivetelemetryprocessor // import "github.com/newrelic/nrdot-collector-components/processor/adaptivetelemetryprocessor"

import (
	"encoding/json"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// identifyHostMetricType tries to identify the type of host metric from attributes
func identifyHostMetricType(attrs pcommon.Map) (string, bool) {
	// Check for specific resource attributes that identify different hostmetric types

	// CPU - look for cpu and state attributes
	if _, hasCPU := attrs.Get("cpu"); hasCPU {
		if stateVal, hasState := attrs.Get("state"); hasState && stateVal.Type() == pcommon.ValueTypeStr {
			state := stateVal.AsString()
			if state == "idle" || state == "interrupt" || state == "nice" ||
				state == "softirq" || state == "steal" || state == "system" ||
				state == "user" || state == "wait" {
				return resourceTypeCPU, true
			}
		}
		// Even without state, if cpu attribute exists, it's likely CPU metric
		return resourceTypeCPU, true
	}

	// Disk - look for device and direction attributes
	// Some disk metrics have both device + direction (system.disk.io, system.disk.operations)
	// with direction="read" or "write"
	// Others have only device (system.disk.io_time, system.disk.pending_operations, system.disk.weighted_io_time)
	if deviceVal, hasDevice := attrs.Get("device"); hasDevice && deviceVal.Type() == pcommon.ValueTypeStr {
		if dirVal, hasDir := attrs.Get("direction"); hasDir && dirVal.Type() == pcommon.ValueTypeStr {
			dir := dirVal.AsString()
			if dir == "read" || dir == "write" {
				return resourceTypeDisk, true
			}
		}
	}

	// Filesystem - look for mountpoint, device, type, state attributes
	if _, hasMountpoint := attrs.Get("mountpoint"); hasMountpoint {
		return resourceTypeFilesystem, true
	}
	if _, hasType := attrs.Get("type"); hasType {
		if stateVal, hasState := attrs.Get("state"); hasState && stateVal.Type() == pcommon.ValueTypeStr {
			state := stateVal.AsString()
			if state == "free" || state == "reserved" || state == "used" {
				return resourceTypeFilesystem, true
			}
		}
	}

	// Memory - look for state attribute with memory-specific values
	if stateVal, hasState := attrs.Get("state"); hasState && stateVal.Type() == pcommon.ValueTypeStr {
		state := stateVal.AsString()
		if state == "buffered" || state == "cached" || state == "inactive" ||
			state == "free" || state == "slab_reclaimable" || state == "slab_unreclaimable" ||
			state == "used" {
			return resourceTypeMemory, true
		}
	}

	// Network - look for device, direction, protocol attributes
	// Two types of network metrics:
	// 1. Per-device: system.network.io, system.network.dropped, etc. (has device + direction)
	// 2. System-wide: system.network.connections (protocol + state), system.network.conntrack.* (no attributes)
	if _, hasDevice := attrs.Get("device"); hasDevice {
		if dirVal, hasDir := attrs.Get("direction"); hasDir && dirVal.Type() == pcommon.ValueTypeStr {
			dir := dirVal.AsString()
			if dir == "receive" || dir == "transmit" {
				return resourceTypeNetwork, true
			}
		}
		if _, hasProtocol := attrs.Get("protocol"); hasProtocol {
			return resourceTypeNetwork, true
		}
		if stateVal, hasState := attrs.Get("state"); hasState && stateVal.Type() == pcommon.ValueTypeStr {
			return resourceTypeNetwork, true
		}
	}
	// System-wide network metrics without device attribute
	// Example: system.network.connections (protocol + state)
	if _, hasProtocol := attrs.Get("protocol"); hasProtocol {
		if stateVal, hasState := attrs.Get("state"); hasState && stateVal.Type() == pcommon.ValueTypeStr {
			return resourceTypeNetwork, true
		}
	}

	// Paging - look for direction, state, type attributes
	if dirVal, hasDir := attrs.Get("direction"); hasDir && dirVal.Type() == pcommon.ValueTypeStr {
		dir := dirVal.AsString()
		if dir == "page_in" || dir == "page_out" {
			return resourceTypePaging, true
		}
	}
	if stateVal, hasState := attrs.Get("state"); hasState && stateVal.Type() == pcommon.ValueTypeStr {
		state := stateVal.AsString()
		if state == "cached" || state == "free" || state == "used" {
			// Look for device to distinguish from memory
			if _, hasDevice := attrs.Get("device"); hasDevice {
				return resourceTypePaging, true
			}
		}
	}
	if typeVal, hasType := attrs.Get("type"); hasType && typeVal.Type() == pcommon.ValueTypeStr {
		typ := typeVal.AsString()
		if typ == "major" || typ == "minor" {
			return resourceTypePaging, true
		}
	}

	// Process - look for process.pid attribute (individual process metrics)
	// This must come BEFORE "Processes" check to distinguish between:
	// - Individual process metrics (process.cpu.utilization, process.memory.usage) - has process.pid
	// - System-wide process count metrics (system.processes.count) - has status attribute
	if _, hasPID := attrs.Get("process.pid"); hasPID {
		return resourceTypeProcess, true
	}

	// Processes - look for status attribute (system-wide process count metrics)
	if statusVal, hasStatus := attrs.Get("status"); hasStatus && statusVal.Type() == pcommon.ValueTypeStr {
		status := statusVal.AsString()
		if status == "blocked" || status == "daemon" || status == "detached" ||
			status == "idle" || status == "locked" || status == "orphan" ||
			status == "paging" || status == "running" || status == "sleeping" ||
			status == "stopped" || status == "system" || status == "unknown" ||
			status == "zombies" {
			return resourceTypeProcesses, true
		}
	}

	// Disk (device-only metrics) - catch metrics with only device attribute
	// This check must come AFTER network, paging, and filesystem checks to avoid false positives
	// Examples: system.disk.io_time, system.disk.pending_operations, system.disk.weighted_io_time
	// These have only [device] attribute, no direction/mountpoint/protocol/state
	if deviceVal, hasDevice := attrs.Get("device"); hasDevice && deviceVal.Type() == pcommon.ValueTypeStr {
		// Verify no other specific attributes that would indicate network/paging/filesystem
		_, hasDirection := attrs.Get("direction")
		_, hasMountpoint := attrs.Get("mountpoint")
		_, hasProtocol := attrs.Get("protocol")
		_, hasState := attrs.Get("state")
		_, hasType := attrs.Get("type")

		// If device exists alone without these other attributes, it's a disk metric
		if !hasDirection && !hasMountpoint && !hasProtocol && !hasState && !hasType {
			return resourceTypeDisk, true
		}
	}

	// System - system-wide metrics with NO resource-level attributes
	// Examples: system.uptime (time system has been running)
	// These metrics have only host-level attributes (host.name, host.id, cloud provider info)
	// and represent the entire system/host, not a specific component
	// If we have host.name but none of the above specific resource attributes, it's a system metric
	if _, hasHostName := attrs.Get("host.name"); hasHostName {
		// Make sure it's not one of the specific types above
		// (This is a catch-all for host-level metrics without distinguishing attributes)
		return resourceTypeSystem, true
	}

	return "", false
}

// getResourceType determines the type of resource based on its attributes
func getResourceType(attrs pcommon.Map) string {
	// Try to identify if it's a host metric resource
	if resourceType, isHostMetric := identifyHostMetricType(attrs); isHostMetric {
		return resourceType
	}

	// Look for service.name as a fallback
	if val, ok := attrs.Get("service.name"); ok {
		return "service:" + val.AsString()
	}

	// Generic fallback
	return "unknown"
}

// countMetricsInResource counts the total number of metrics in a resource
func countMetricsInResource(rm pmetric.ResourceMetrics) int {
	count := 0
	for i := 0; i < rm.ScopeMetrics().Len(); i++ {
		sm := rm.ScopeMetrics().At(i)
		count += sm.Metrics().Len()
	}
	return count
}

// updateProcessATPAttribute updates the process.atp JSON attribute with new data under the given key
func updateProcessATPAttribute(resource pcommon.Resource, key string, data interface{}) {
	attrs := resource.Attributes()
	var atpData map[string]interface{}

	// Check if attribute exists and parse it
	if val, ok := attrs.Get("process.atp"); ok {
		// Try to unmarshal existing data
		// If unmarshalling fails, we'll start with a clean map to avoid propagating corruption
		// but we lose existing data (which shouldn't happen if we control writes)
		_ = json.Unmarshal([]byte(val.AsString()), &atpData)
	}

	if atpData == nil {
		atpData = make(map[string]interface{})
	}

	atpData[key] = data

	if jsonData, err := json.Marshal(atpData); err == nil {
		attrs.PutStr("process.atp", string(jsonData))
	}
}
