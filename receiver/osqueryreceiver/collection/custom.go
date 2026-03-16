// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

type CustomCollection struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

func (c CustomCollection) GetName() string {
	return c.Name
}

func (c CustomCollection) GetQuery() string {
	return c.Query
}

func (c CustomCollection) Unmarshal(results any) any {
	return results
}

func NewCustomCollection(name, query string) ICollection {
	return CustomCollection{
		Name:  name,
		Query: query,
	}
}
