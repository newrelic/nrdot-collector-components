// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package osqueryreceiver

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

const (
	defaultSocket   = "/var/osquery/osquery.em"
	defaultInterval = "60s"
)

// validCollections defines the allowed collection names
// we don't need the value, so we use an empty struct to save memory
var validCollections = map[string]struct{}{
	"system_info":     {},
	"package_info":    {},
	"os_info":         {},
	"secureboot_info": {},
	"users_info":      {},
}

type Config struct {
	TmpDir             string   `mapstructure:"tmp_dir"`
	CollectionInterval string   `mapstructure:"interval"`
	ExtensionsSocket   string   `mapstructure:"extensions_socket"`
	Collections        []string `mapstructure:"collections"`
	CustomQueries      []string `mapstructure:"custom_queries"`
}

func createDefaultConfig() component.Config {
	return &Config{
		TmpDir:             "/tmp/osqueryreceiver/tmp_data/",
		CollectionInterval: defaultInterval,
		ExtensionsSocket:   defaultSocket,
		Collections:        []string{},
		CustomQueries:      []string{},
	}
}

func (c *Config) Validate() error {
	if len(c.CustomQueries) == 0 && len(c.Collections) == 0 {
		return fmt.Errorf("either custom_queries or collections must be specified")
	}

	for _, collection := range c.Collections {
		if _, valid := validCollections[collection]; !valid {
			return fmt.Errorf("invalid collection %q: must be one of %s", collection, validCollections)
		}
	}

	return nil
}
