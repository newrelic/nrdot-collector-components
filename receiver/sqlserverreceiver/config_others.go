// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/config_others.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package sqlserverreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/sqlserverreceiver"

func (*Config) validateInstanceAndComputerName() error {
	return nil
}
