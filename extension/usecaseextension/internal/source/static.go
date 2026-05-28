// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package source // import "github.com/newrelic/nrdot-collector-components/extension/usecaseextension/internal/source"

import "context"

var _ Source = (*StaticSource)(nil)

type StaticSource struct {
	Value string
}

func (ss *StaticSource) Get(_ context.Context) (string, error) {
	return ss.Value, nil
}
