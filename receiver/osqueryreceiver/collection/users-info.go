// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

import (
	"runtime"
)

var (
	UserCollectionQuery = map[string]string{
		"linux":   UserCollectionQueryLinux,
		"darwin":  UserCollectionQueryDarwin,
		"windows": UserCollectionQueryWindows,
	}
)

type UserCollection struct {
	Username string `json:"username"`
	Groups   string `json:"groups,omitempty"`
}

func (u UserCollection) GetName() string {
	return UserCollectionName
}

func (u UserCollection) GetQuery() string {
	return UserCollectionQuery[runtime.GOOS]
}

func (u UserCollection) Unmarshal(result any) any {
	resultSlice, ok := result.([]map[string]any)
	if !ok {
		return nil
	}

	usersList := make([]map[string]any, 0, len(resultSlice))
	for _, resultMap := range resultSlice {
		sanitized := sanitizeRow(
			resultMap,
			[]string{
				"username",
				"groups",
			},
			nil,
			nil,
		)
		if len(sanitized) == 0 {
			continue
		}
		usersList = append(usersList, sanitized)
	}

	if len(usersList) == 0 {
		return nil
	}

	return usersList
}

func NewUserCollection() ICollection {
	return &UserCollection{}
}
