// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

// SystemInfoCollection represents the system_info collection
// https://github.com/osquery/osquery/blob/master/specs/system_info.table
type SystemInfoCollection struct {
	Hostname         string `json:"hostname"`
	UUID             string `json:"uuid"`
	CPUType          string `json:"cpu_type"`
	CPUSubtype       string `json:"cpu_subtype"`
	CPUBrand         string `json:"cpu_brand"`
	CPUPhysicalCores int    `json:"cpu_physical_cores"`
	CPULogicalCores  int    `json:"cpu_logical_cores"`
	PhysicalMemory   string `json:"physical_memory"`
	HardwareVendor   string `json:"hardware_vendor"`
	HardwareModel    string `json:"hardware_model"`
	ComputerName     string `json:"computer_name,omitempty"`
	EmulatedCPUType  string `json:"emulated_cpu_type,omitempty"`
}

func (s SystemInfoCollection) GetName() string {
	return SystemInfoCollectionName
}

func (s SystemInfoCollection) GetQuery() string {
	return SystemInfoCollectionQuery
}

func (s SystemInfoCollection) Unmarshal(result any) any {

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil
	}

	sanitized := sanitizeRow(
		resultMap,
		[]string{
			"hostname",
			"uuid",
			"cpu_type",
			"cpu_subtype",
			"cpu_brand",
			"physical_memory",
			"hardware_vendor",
			"hardware_model",
			"computer_name",
			"emulated_cpu_type",
		},
		[]string{
			"cpu_physical_cores",
			"cpu_logical_cores",
		},
		nil,
	)

	if len(sanitized) == 0 {
		return nil
	}

	return sanitized
}

func NewSystemInfoCollection() ICollection {
	return SystemInfoCollection{}
}
