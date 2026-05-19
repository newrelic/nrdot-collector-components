// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package source // import "github.com/newrelic/nrdot-collector-components/extension/usecasesetterextension/internal/source"

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/collector/client"
)

var _ Source = (*AttributeSource)(nil)

type AttributeSource struct {
	Key          string
	DefaultValue string
}

func (ts *AttributeSource) Get(ctx context.Context) (string, error) {
	cl := client.FromContext(ctx)
	attr := cl.Auth.GetAttribute(ts.Key)

	switch a := attr.(type) {
	case string:
		return a, nil
	case nil:
		return ts.DefaultValue, nil
	default:
		b, err := json.Marshal(attr)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
