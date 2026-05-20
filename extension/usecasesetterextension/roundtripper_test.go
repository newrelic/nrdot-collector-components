// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecasesetterextension

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
)

type mockRoundTripper struct {
	req *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.req = req
	return &http.Response{StatusCode: http.StatusOK}, nil
}

type testAuth struct {
	attrs map[string]any
}

func (a testAuth) GetAttribute(s string) any   { return a.attrs[s] }
func (_ testAuth) GetAttributeNames() []string { return nil }

// newTestRT creates a round tripper from the given source config and returns
// it alongside the mock transport so callers can inspect the forwarded request.
func newTestRT(t *testing.T, cfg *UseCaseConfig) (http.RoundTripper, *mockRoundTripper) {
	t.Helper()
	ext, err := newUseCaseSetterExtension(&Config{UseCaseConfig: cfg})
	require.NoError(t, err)
	mock := &mockRoundTripper{}
	rt, err := ext.RoundTripper(mock)
	require.NoError(t, err)
	return rt, mock
}

func newTestRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://example.com", http.NoBody)
	require.NoError(t, err)
	return req
}

func TestRoundTripperAppendsUseCaseToExistingUserAgent(t *testing.T) {
	rt, mock := newTestRT(t, &UseCaseConfig{Value: stringp("my-use-case")})
	req := newTestRequest(t)
	req.Header.Set("User-Agent", "existing-agent")

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "existing-agent my-use-case", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSetsUseCaseWhenNoUserAgent(t *testing.T) {
	rt, mock := newTestRT(t, &UseCaseConfig{Value: stringp("my-use-case")})
	req := newTestRequest(t)

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "my-use-case", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSkipsEmptyUseCase(t *testing.T) {
	rt, mock := newTestRT(t, &UseCaseConfig{Value: stringp("")})
	req := newTestRequest(t)
	req.Header.Set("User-Agent", "existing-agent")

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "existing-agent", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperWithContextSource(t *testing.T) {
	rt, mock := newTestRT(t, &UseCaseConfig{FromContext: stringp("X-Scope-OrgID")})
	req := newTestRequest(t)
	cl := client.FromContext(req.Context())
	cl.Metadata = client.NewMetadata(map[string][]string{"X-Scope-OrgID": {"acme"}})
	req = req.WithContext(client.NewContext(req.Context(), cl))

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "acme", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperWithAttributeSource(t *testing.T) {
	rt, mock := newTestRT(t, &UseCaseConfig{FromAttribute: stringp("X-Scope-OrgID")})
	req := newTestRequest(t)
	cl := client.FromContext(req.Context())
	cl.Auth = testAuth{attrs: map[string]any{"X-Scope-OrgID": "acme"}}
	req = req.WithContext(client.NewContext(req.Context(), cl))

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "acme", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSourceError(t *testing.T) {
	// ContextSource returns an error when the key has multiple values.
	rt, _ := newTestRT(t, &UseCaseConfig{FromContext: stringp("X-Scope-OrgID")})
	req := newTestRequest(t)
	cl := client.FromContext(req.Context())
	cl.Metadata = client.NewMetadata(map[string][]string{"X-Scope-OrgID": {"acme", "globex"}})
	req = req.WithContext(client.NewContext(req.Context(), cl))

	_, err := rt.RoundTrip(req)
	assert.Error(t, err)
}
