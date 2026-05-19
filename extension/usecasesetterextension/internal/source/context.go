// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package source // import "github.com/newrelic/nrdot-collector-components/extension/usecasesetterextension/internal/source"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/client"
)

var _ Source = (*ContextSource)(nil)

type ContextSource struct {
	Key          string
	DefaultValue string
}

func (ts *ContextSource) Get(ctx context.Context) (string, error) {
	cl := client.FromContext(ctx)
	ss := cl.Metadata.Get(ts.Key)

	if len(ss) == 0 {
		return ts.DefaultValue, nil
	}

	if len(ss) > 1 {
		return "", fmt.Errorf("%d source keys found in the context, can't determine which one to use", len(ss))
	}

	return ss[0], nil
}
