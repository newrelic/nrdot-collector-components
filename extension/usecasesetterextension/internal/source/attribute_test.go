// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/client"
)

type authData struct {
	attrs map[string]any
}

func (a authData) GetAttribute(s string) any {
	return a.attrs[s]
}

func (a authData) GetAttributeNames() []string {
	keys := make([]string, 0, len(a.attrs))
	for key := range a.attrs {
		keys = append(keys, key)
	}
	return keys
}

func TestAttributeSourceSuccessString(t *testing.T) {
	ts := &AttributeSource{Key: "X-Scope-OrgID"}
	cl := client.FromContext(t.Context())
	cl.Auth = authData{attrs: map[string]any{"X-Scope-OrgID": "acme"}}
	ctx := client.NewContext(t.Context(), cl)

	val, err := ts.Get(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "acme", val)
}

func TestAttributeSourceSuccessStruct(t *testing.T) {
	ts := &AttributeSource{Key: "X-Scope-OrgID"}
	cl := client.FromContext(t.Context())
	cl.Auth = authData{attrs: map[string]any{"X-Scope-OrgID": struct {
		Foo string
	}{
		Foo: "bar",
	}}}
	ctx := client.NewContext(t.Context(), cl)

	val, err := ts.Get(ctx)

	assert.NoError(t, err)
	assert.JSONEq(t, "{\"Foo\":\"bar\"}", val)
}

func TestAttributeSourceMarshalError(t *testing.T) {
	ts := &AttributeSource{Key: "X-Scope-OrgID"}
	cl := client.FromContext(t.Context())
	// channels cannot be marshaled to JSON, triggering the error path
	cl.Auth = authData{attrs: map[string]any{"X-Scope-OrgID": make(chan int)}}
	ctx := client.NewContext(t.Context(), cl)

	val, err := ts.Get(ctx)

	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestAttributeSourceNotFound(t *testing.T) {
	ts := &AttributeSource{Key: "X-Scope-OrgID"}
	cl := client.FromContext(t.Context())
	cl.Auth = authData{attrs: map[string]any{"Not-Scope-OrgID": "acme"}}
	ctx := client.NewContext(t.Context(), cl)

	val, err := ts.Get(ctx)

	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestAttributeSourceNotFoundWithDefault(t *testing.T) {
	ts := &AttributeSource{Key: "X-Scope-OrgID", DefaultValue: "fallback"}
	cl := client.FromContext(t.Context())
	cl.Auth = authData{attrs: map[string]any{"Not-Scope-OrgID": "acme"}}
	ctx := client.NewContext(t.Context(), cl)

	val, err := ts.Get(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "fallback", val)
}
