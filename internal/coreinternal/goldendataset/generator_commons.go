// Copyright The OpenTelemetry Authors
// Modifications copyright New Relic, Inc.
//
// Modifications can be found at the following URL:
// https://github.com/newrelic/nrdot-collector-components/commits/main/internal/coreinternal/goldendataset/generator_commons.go?since=2025-11-26
//
// SPDX-License-Identifier: Apache-2.0

package goldendataset // import "github.com/newrelic/nrdot-collector-components/internal/coreinternal/goldendataset"

import (
	"encoding/csv"
	"os"
	"path/filepath"
)

func loadPictOutputFile(fileName string) ([][]string, error) {
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	reader := csv.NewReader(file)
	reader.Comma = '\t'

	return reader.ReadAll()
}
