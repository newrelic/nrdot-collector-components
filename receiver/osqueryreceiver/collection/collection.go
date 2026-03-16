// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package collection

import (
	"fmt"
)

type ICollection interface {
	GetName() string
	GetQuery() string
	Unmarshal(any) any
}

type Collection struct{}

func GetCollection(name string) (ICollection, error) {
	switch name {
	case "system_info":
		return NewSystemInfoCollection(), nil
	case "package_info":
		return NewPackageInfoCollection(), nil
	case "os_info":
		return NewOSInfoCollection(), nil
	case "secureboot_info":
		return NewSecureBootCollection(), nil
	case "users_info":
		return NewUserCollection(), nil
	default:
		return nil, fmt.Errorf("wrong collection name passed")
	}
}

func GetCustomCollection(name, query string) ICollection {
	return NewCustomCollection(name, query)
}
