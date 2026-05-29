// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/obfuscate.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package oracledbreceiver // import "github.com/newrelic/nrdot-collector-components/receiver/oracledbreceiver"

import (
	"github.com/DataDog/datadog-agent/pkg/obfuscate"
)

type obfuscator obfuscate.Obfuscator

func newObfuscator() *obfuscator {
	return (*obfuscator)(obfuscate.NewObfuscator(obfuscate.Config{
		SQL: obfuscate.SQLConfig{
			DBMS: "oracle",
		},
	}))
}

func (o *obfuscator) obfuscateSQLString(sql string) (string, error) {
	obfuscatedQuery, err := (*obfuscate.Obfuscator)(o).ObfuscateSQLString(sql)
	if err != nil {
		return "", err
	}
	return obfuscatedQuery.Query, nil
}
