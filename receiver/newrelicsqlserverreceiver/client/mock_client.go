// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// MockClient implements SQLServerClient interface for testing
// It allows tests to verify database operations without requiring an actual SQL Server instance
type MockClient struct {
	// QueryFunc allows tests to customize Query behavior
	QueryFunc func(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// QueryRowFunc allows tests to customize QueryRow behavior
	QueryRowFunc func(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// ExecFunc allows tests to customize Exec behavior
	ExecFunc func(ctx context.Context, query string, args ...interface{}) error

	// PingFunc allows tests to customize Ping behavior
	PingFunc func(ctx context.Context) error

	// CloseFunc allows tests to customize Close behavior
	CloseFunc func() error

	// GetDBFunc allows tests to customize GetDB behavior
	GetDBFunc func() *sql.DB

	// Call tracking
	mu         sync.Mutex
	QueryCalls []MockCall
	ExecCalls  []MockCall
	PingCalls  int
	CloseCalls int
	GetDBCalls int
	IsClosed   bool
}

// MockCall represents a recorded call to Query, QueryRow, or Exec
type MockCall struct {
	Query string
	Args  []interface{}
	Error error
}

// NewMockClient creates a new mock client with default no-op implementations
func NewMockClient() *MockClient {
	return &MockClient{
		QueryCalls: make([]MockCall, 0),
		ExecCalls:  make([]MockCall, 0),
	}
}

// Query records the call and invokes QueryFunc if set
func (m *MockClient) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsClosed {
		return fmt.Errorf("client is closed")
	}

	var err error
	if m.QueryFunc != nil {
		err = m.QueryFunc(ctx, dest, query, args...)
	}

	m.QueryCalls = append(m.QueryCalls, MockCall{
		Query: query,
		Args:  args,
		Error: err,
	})

	return err
}

// QueryRow records the call and invokes QueryRowFunc if set
func (m *MockClient) QueryRow(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsClosed {
		return fmt.Errorf("client is closed")
	}

	var err error
	if m.QueryRowFunc != nil {
		err = m.QueryRowFunc(ctx, dest, query, args...)
	}

	m.QueryCalls = append(m.QueryCalls, MockCall{
		Query: query,
		Args:  args,
		Error: err,
	})

	return err
}

// Exec records the call and invokes ExecFunc if set
func (m *MockClient) Exec(ctx context.Context, query string, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsClosed {
		return fmt.Errorf("client is closed")
	}

	var err error
	if m.ExecFunc != nil {
		err = m.ExecFunc(ctx, query, args...)
	}

	m.ExecCalls = append(m.ExecCalls, MockCall{
		Query: query,
		Args:  args,
		Error: err,
	})

	return err
}

// Ping records the call and invokes PingFunc if set
func (m *MockClient) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsClosed {
		return fmt.Errorf("client is closed")
	}

	m.PingCalls++

	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

// Close records the call and invokes CloseFunc if set
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CloseCalls++
	m.IsClosed = true

	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// GetDB records the call and invokes GetDBFunc if set
func (m *MockClient) GetDB() *sql.DB {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetDBCalls++

	if m.GetDBFunc != nil {
		return m.GetDBFunc()
	}
	return nil
}

// Reset clears all recorded calls
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueryCalls = make([]MockCall, 0)
	m.ExecCalls = make([]MockCall, 0)
	m.PingCalls = 0
	m.CloseCalls = 0
	m.GetDBCalls = 0
	m.IsClosed = false
}

// GetQueryCallCount returns the number of times Query or QueryRow was called
func (m *MockClient) GetQueryCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.QueryCalls)
}

// GetExecCallCount returns the number of times Exec was called
func (m *MockClient) GetExecCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.ExecCalls)
}

// GetLastQuery returns the last query that was executed, or empty string if none
func (m *MockClient) GetLastQuery() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.QueryCalls) == 0 {
		return ""
	}
	return m.QueryCalls[len(m.QueryCalls)-1].Query
}

// GetLastExec returns the last exec query that was executed, or empty string if none
func (m *MockClient) GetLastExec() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.ExecCalls) == 0 {
		return ""
	}
	return m.ExecCalls[len(m.ExecCalls)-1].Query
}

// SetQueryError configures the mock to return an error for Query calls
func (m *MockClient) SetQueryError(err error) {
	m.QueryFunc = func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
		return err
	}
}

// SetExecError configures the mock to return an error for Exec calls
func (m *MockClient) SetExecError(err error) {
	m.ExecFunc = func(ctx context.Context, query string, args ...interface{}) error {
		return err
	}
}

// SetPingError configures the mock to return an error for Ping calls
func (m *MockClient) SetPingError(err error) {
	m.PingFunc = func(ctx context.Context) error {
		return err
	}
}
