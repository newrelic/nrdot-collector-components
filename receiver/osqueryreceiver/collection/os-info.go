// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

// https://github.com/osquery/osquery/blob/master/specs/os_version.table
type OSInfoCollection struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Build        string `json:"build"`
	Platform     string `json:"platform"`
	PlatformLike string `json:"platform_like"`
	Codename     string `json:"codename,omitempty"`
	Arch         string `json:"arch,omitempty"`
}

func (o OSInfoCollection) GetName() string {
	return OSInfoCollectionName
}

func (o OSInfoCollection) GetQuery() string {
	return OSInfoCollectionQuery
}

func (o OSInfoCollection) Unmarshal(result any) any {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil
	}

	sanitized := sanitizeRow(
		resultMap,
		[]string{
			"name",
			"platform",
			"platform_like",
			"build",
			"version",
			"codename",
			"arch",
		},
		nil,
		nil,
	)

	if len(sanitized) == 0 {
		return nil
	}

	return sanitized
}

func NewOSInfoCollection() ICollection {
	return &OSInfoCollection{}
}
