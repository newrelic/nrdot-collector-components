// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/config_windows.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package sqlserverreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/sqlserverreceiver"

import "errors"

func (cfg *Config) validateInstanceAndComputerName() error {
	if cfg.InstanceName != "" && cfg.ComputerName == "" {
		return errors.New("'instance_name' may not be specified without 'computer_name'")
	}
	if cfg.InstanceName == "" && cfg.ComputerName != "" {
		return errors.New("'computer_name' may not be specified without 'instance_name'")
	}

	return nil
}
