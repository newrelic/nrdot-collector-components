// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package usecaseextension

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct {
	req *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.req = req
	return &http.Response{StatusCode: http.StatusOK}, nil
}

// newTestRT creates a round tripper from the given ID and returns
// it alongside the mock transport so callers can inspect the forwarded request.
func newTestRT(t *testing.T, id *string) (http.RoundTripper, *mockRoundTripper) {
	t.Helper()
	ext, err := newUseCaseSetterExtension(&Config{ID: id})
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
	rt, mock := newTestRT(t, stringp("my-use-case"))
	req := newTestRequest(t)
	req.Header.Set("User-Agent", "existing-agent")

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "existing-agent my-use-case", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSetsUseCaseWhenNoUserAgent(t *testing.T) {
	rt, mock := newTestRT(t, stringp("my-use-case"))
	req := newTestRequest(t)

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "my-use-case", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSkipsEmptyUseCase(t *testing.T) {
	rt, mock := newTestRT(t, stringp(""))
	req := newTestRequest(t)
	req.Header.Set("User-Agent", "existing-agent")

	_, err := rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "existing-agent", mock.req.Header.Get("User-Agent"))
}
