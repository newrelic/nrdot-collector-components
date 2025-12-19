// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/testbed/correctnesstests/metrics/results_dir.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package metrics // import "github.com/newrelic/nrdot-collector-components/testbed/correctnesstests/metrics"

import (
	"os"
	"path"
	"path/filepath"
)

type resultsDir struct {
	dir string
}

func newResultsDir(dirName string) (*resultsDir, error) {
	dir, err := filepath.Abs(path.Join("results", dirName))
	if err != nil {
		return nil, err
	}
	return &resultsDir{dir: dir}, nil
}

func (d *resultsDir) mkDir() error {
	return os.MkdirAll(d.dir, os.ModePerm)
}

func (d *resultsDir) fullPath(name string) (string, error) {
	return filepath.Abs(path.Join(d.dir, name))
}
