// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package usecasesetterextension

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

func TestRoundTripperAppendsUseCaseToExistingUserAgent(t *testing.T) {
	ext, err := newUseCaseSetterExtension(&Config{
		UseCaseConfig: &UseCaseConfig{Value: stringp("my-use-case")},
	})
	require.NoError(t, err)

	mock := &mockRoundTripper{}
	rt, err := ext.RoundTripper(mock)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "existing-agent")

	_, err = rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "existing-agent my-use-case", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSetsUseCaseWhenNoUserAgent(t *testing.T) {
	ext, err := newUseCaseSetterExtension(&Config{
		UseCaseConfig: &UseCaseConfig{Value: stringp("my-use-case")},
	})
	require.NoError(t, err)

	mock := &mockRoundTripper{}
	rt, err := ext.RoundTripper(mock)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	_, err = rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "my-use-case", mock.req.Header.Get("User-Agent"))
}

func TestRoundTripperSkipsEmptyUseCase(t *testing.T) {
	ext, err := newUseCaseSetterExtension(&Config{
		UseCaseConfig: &UseCaseConfig{Value: stringp("")},
	})
	require.NoError(t, err)

	mock := &mockRoundTripper{}
	rt, err := ext.RoundTripper(mock)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "existing-agent")

	_, err = rt.RoundTrip(req)
	require.NoError(t, err)

	assert.Equal(t, "existing-agent", mock.req.Header.Get("User-Agent"))
}
