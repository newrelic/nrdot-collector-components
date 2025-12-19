// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/internal/coreinternal/clientutil/client.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package clientutil // import "github.com/newrelic/nrdot-collector-components/internal/coreinternal/clientutil"

import (
	"net"
	"strings"

	"go.opentelemetry.io/collector/client"
)

// Address returns the address of the client connecting to the collector.
func Address(client client.Info) string {
	if client.Addr == nil {
		return ""
	}
	switch addr := client.Addr.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	case *net.IPAddr:
		return addr.IP.String()
	}

	// If this is not a known address type, check for known "untyped" formats.
	// 1.1.1.1:<port>

	lastColonIndex := strings.LastIndex(client.Addr.String(), ":")
	if lastColonIndex != -1 {
		ipString := client.Addr.String()[:lastColonIndex]
		ip := net.ParseIP(ipString)
		if ip != nil {
			return ip.String()
		}
	}

	return client.Addr.String()
}
