// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/recorders_test.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package sqlserverreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/sqlserverreceiver"

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPerfCounterRecorders requires each perf counter record created to exist in the available counters.
func TestPerfCounterRecorders(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		expectedCounters := getAvailableCounters(t)

		for _, counter := range perfCounterRecorders {
			for counterPath := range counter.recorders {
				counterFullName := fmt.Sprintf("%s:%s %s", defaultObjectName, counter.object, counterPath)
				_, ok := expectedCounters[counterFullName]
				require.Truef(t, ok, "counter %s not found", counterFullName)
			}
		}
	})

	t.Run("named", func(t *testing.T) {
		expectedCounters := getAvailableNamedInstanceCounters(t)

		for _, counter := range perfCounterRecorders {
			for counterPath := range counter.recorders {
				counterFullName := fmt.Sprintf("MSSQL$TEST_NAME:%s %s", counter.object, counterPath)
				_, ok := expectedCounters[counterFullName]
				require.Truef(t, ok, "counter %s not found", counterFullName)
			}
		}
	})
}

// getAvailableCounters populates a map containing all available counters.
func getAvailableCounters(t *testing.T) map[string]bool {
	file, err := os.Open(filepath.Join("testdata", "counters.txt"))
	require.NoError(t, err)

	defer file.Close()

	scanner := bufio.NewScanner(file)
	counterNames := map[string]bool{}
	for scanner.Scan() {
		if scanner.Text() != "" {
			nameNoBackslash := strings.ReplaceAll(scanner.Text(), "\\", " ")
			nameNoInstance := strings.TrimSpace(strings.ReplaceAll(nameNoBackslash, "(*)", ""))
			counterNames[nameNoInstance] = true
		}
	}

	return counterNames
}

// getAvailableCounters populates a map containing all available counters.
func getAvailableNamedInstanceCounters(t *testing.T) map[string]bool {
	file, err := os.Open(filepath.Join("testdata", "named-instance-counters.txt"))
	require.NoError(t, err)

	defer file.Close()

	scanner := bufio.NewScanner(file)
	counterNames := map[string]bool{}
	for scanner.Scan() {
		if scanner.Text() != "" {
			nameNoBackslash := strings.ReplaceAll(scanner.Text(), "\\", " ")
			nameNoInstance := strings.TrimSpace(strings.ReplaceAll(nameNoBackslash, "(*)", ""))
			counterNames[nameNoInstance] = true
		}
	}

	return counterNames
}
