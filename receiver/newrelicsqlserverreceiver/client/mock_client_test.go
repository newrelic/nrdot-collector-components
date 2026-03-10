// Copyright New Relic, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMockClient(t *testing.T) {
	client := NewMockClient()

	assert.NotNil(t, client)
	assert.NotNil(t, client.QueryCalls)
	assert.NotNil(t, client.ExecCalls)
	assert.Equal(t, 0, client.PingCalls)
	assert.Equal(t, 0, client.CloseCalls)
	assert.Equal(t, 0, client.GetDBCalls)
	assert.False(t, client.IsClosed)
}

func TestMockClient_Query(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("Query without custom func", func(t *testing.T) {
		var dest []string
		err := client.Query(ctx, &dest, "SELECT * FROM users", 1, "test")

		assert.NoError(t, err)
		assert.Equal(t, 1, len(client.QueryCalls))
		assert.Equal(t, "SELECT * FROM users", client.QueryCalls[0].Query)
		assert.Equal(t, []interface{}{1, "test"}, client.QueryCalls[0].Args)
	})

	t.Run("Query with custom func", func(t *testing.T) {
		client.Reset()
		expectedErr := fmt.Errorf("custom error")
		client.QueryFunc = func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
			return expectedErr
		}

		var dest []string
		err := client.Query(ctx, &dest, "SELECT * FROM orders")

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, len(client.QueryCalls))
		assert.Equal(t, expectedErr, client.QueryCalls[0].Error)
	})

	t.Run("Query when closed", func(t *testing.T) {
		client.Reset()
		client.Close()

		var dest []string
		err := client.Query(ctx, &dest, "SELECT * FROM users")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client is closed")
	})
}

func TestMockClient_QueryRow(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("QueryRow without custom func", func(t *testing.T) {
		var dest string
		err := client.QueryRow(ctx, &dest, "SELECT name FROM users WHERE id = ?", 1)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(client.QueryCalls))
		assert.Equal(t, "SELECT name FROM users WHERE id = ?", client.QueryCalls[0].Query)
		assert.Equal(t, []interface{}{1}, client.QueryCalls[0].Args)
	})

	t.Run("QueryRow with custom func", func(t *testing.T) {
		client.Reset()
		expectedErr := fmt.Errorf("not found")
		client.QueryRowFunc = func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
			return expectedErr
		}

		var dest string
		err := client.QueryRow(ctx, &dest, "SELECT name FROM users WHERE id = ?", 999)

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, len(client.QueryCalls))
	})

	t.Run("QueryRow when closed", func(t *testing.T) {
		client.Reset()
		client.Close()

		var dest string
		err := client.QueryRow(ctx, &dest, "SELECT name FROM users WHERE id = ?", 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client is closed")
	})
}

func TestMockClient_Exec(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("Exec without custom func", func(t *testing.T) {
		err := client.Exec(ctx, "DELETE FROM users WHERE id = ?", 1)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(client.ExecCalls))
		assert.Equal(t, "DELETE FROM users WHERE id = ?", client.ExecCalls[0].Query)
		assert.Equal(t, []interface{}{1}, client.ExecCalls[0].Args)
	})

	t.Run("Exec with custom func", func(t *testing.T) {
		client.Reset()
		expectedErr := fmt.Errorf("constraint violation")
		client.ExecFunc = func(ctx context.Context, query string, args ...interface{}) error {
			return expectedErr
		}

		err := client.Exec(ctx, "INSERT INTO users (name) VALUES (?)", "John")

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, len(client.ExecCalls))
		assert.Equal(t, expectedErr, client.ExecCalls[0].Error)
	})

	t.Run("Exec when closed", func(t *testing.T) {
		client.Reset()
		client.Close()

		err := client.Exec(ctx, "DELETE FROM users")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client is closed")
	})
}

func TestMockClient_Ping(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("Ping without custom func", func(t *testing.T) {
		err := client.Ping(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 1, client.PingCalls)
	})

	t.Run("Ping with custom func", func(t *testing.T) {
		client.Reset()
		expectedErr := fmt.Errorf("connection lost")
		client.PingFunc = func(ctx context.Context) error {
			return expectedErr
		}

		err := client.Ping(ctx)

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, client.PingCalls)
	})

	t.Run("Multiple Pings", func(t *testing.T) {
		client.Reset()
		client.Ping(ctx)
		client.Ping(ctx)
		client.Ping(ctx)

		assert.Equal(t, 3, client.PingCalls)
	})

	t.Run("Ping when closed", func(t *testing.T) {
		client.Reset()
		client.Close()

		err := client.Ping(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "client is closed")
	})
}

func TestMockClient_Close(t *testing.T) {
	client := NewMockClient()

	t.Run("Close without custom func", func(t *testing.T) {
		err := client.Close()

		assert.NoError(t, err)
		assert.Equal(t, 1, client.CloseCalls)
		assert.True(t, client.IsClosed)
	})

	t.Run("Close with custom func", func(t *testing.T) {
		client.Reset()
		expectedErr := fmt.Errorf("close error")
		client.CloseFunc = func() error {
			return expectedErr
		}

		err := client.Close()

		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, client.CloseCalls)
		assert.True(t, client.IsClosed)
	})

	t.Run("Multiple Closes", func(t *testing.T) {
		client.Reset()
		client.Close()
		client.Close()

		assert.Equal(t, 2, client.CloseCalls)
	})
}

func TestMockClient_GetDB(t *testing.T) {
	client := NewMockClient()

	t.Run("GetDB without custom func", func(t *testing.T) {
		db := client.GetDB()

		assert.Nil(t, db)
		assert.Equal(t, 1, client.GetDBCalls)
	})

	t.Run("GetDB with custom func", func(t *testing.T) {
		client.Reset()
		expectedDB := &sql.DB{}
		client.GetDBFunc = func() *sql.DB {
			return expectedDB
		}

		db := client.GetDB()

		assert.Equal(t, expectedDB, db)
		assert.Equal(t, 1, client.GetDBCalls)
	})

	t.Run("Multiple GetDB calls", func(t *testing.T) {
		client.Reset()
		client.GetDB()
		client.GetDB()
		client.GetDB()

		assert.Equal(t, 3, client.GetDBCalls)
	})
}

func TestMockClient_Reset(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Populate with calls
	client.Query(ctx, nil, "SELECT * FROM users")
	client.Exec(ctx, "DELETE FROM sessions")
	client.Ping(ctx)
	client.Close()
	client.GetDB()

	assert.Equal(t, 1, len(client.QueryCalls))
	assert.Equal(t, 1, len(client.ExecCalls))
	assert.Equal(t, 1, client.PingCalls)
	assert.Equal(t, 1, client.CloseCalls)
	assert.Equal(t, 1, client.GetDBCalls)
	assert.True(t, client.IsClosed)

	// Reset
	client.Reset()

	assert.Equal(t, 0, len(client.QueryCalls))
	assert.Equal(t, 0, len(client.ExecCalls))
	assert.Equal(t, 0, client.PingCalls)
	assert.Equal(t, 0, client.CloseCalls)
	assert.Equal(t, 0, client.GetDBCalls)
	assert.False(t, client.IsClosed)
}

func TestMockClient_GetQueryCallCount(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	assert.Equal(t, 0, client.GetQueryCallCount())

	var dest []string
	client.Query(ctx, &dest, "SELECT * FROM users")
	assert.Equal(t, 1, client.GetQueryCallCount())

	client.QueryRow(ctx, &dest, "SELECT name FROM users WHERE id = ?", 1)
	assert.Equal(t, 2, client.GetQueryCallCount())

	client.Reset()
	assert.Equal(t, 0, client.GetQueryCallCount())
}

func TestMockClient_GetExecCallCount(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	assert.Equal(t, 0, client.GetExecCallCount())

	client.Exec(ctx, "DELETE FROM users")
	assert.Equal(t, 1, client.GetExecCallCount())

	client.Exec(ctx, "UPDATE users SET active = ?", 1)
	assert.Equal(t, 2, client.GetExecCallCount())

	client.Reset()
	assert.Equal(t, 0, client.GetExecCallCount())
}

func TestMockClient_GetLastQuery(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("No queries", func(t *testing.T) {
		assert.Equal(t, "", client.GetLastQuery())
	})

	t.Run("With queries", func(t *testing.T) {
		var dest []string
		client.Query(ctx, &dest, "SELECT * FROM users")
		assert.Equal(t, "SELECT * FROM users", client.GetLastQuery())

		client.Query(ctx, &dest, "SELECT * FROM orders")
		assert.Equal(t, "SELECT * FROM orders", client.GetLastQuery())
	})

	t.Run("After reset", func(t *testing.T) {
		client.Reset()
		assert.Equal(t, "", client.GetLastQuery())
	})
}

func TestMockClient_GetLastExec(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	t.Run("No execs", func(t *testing.T) {
		assert.Equal(t, "", client.GetLastExec())
	})

	t.Run("With execs", func(t *testing.T) {
		client.Exec(ctx, "DELETE FROM users")
		assert.Equal(t, "DELETE FROM users", client.GetLastExec())

		client.Exec(ctx, "UPDATE orders SET status = ?", "shipped")
		assert.Equal(t, "UPDATE orders SET status = ?", client.GetLastExec())
	})

	t.Run("After reset", func(t *testing.T) {
		client.Reset()
		assert.Equal(t, "", client.GetLastExec())
	})
}

func TestMockClient_SetQueryError(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()
	expectedErr := fmt.Errorf("query error")

	client.SetQueryError(expectedErr)

	var dest []string
	err := client.Query(ctx, &dest, "SELECT * FROM users")

	assert.Equal(t, expectedErr, err)
}

func TestMockClient_SetExecError(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()
	expectedErr := fmt.Errorf("exec error")

	client.SetExecError(expectedErr)

	err := client.Exec(ctx, "DELETE FROM users")

	assert.Equal(t, expectedErr, err)
}

func TestMockClient_SetPingError(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()
	expectedErr := fmt.Errorf("ping error")

	client.SetPingError(expectedErr)

	err := client.Ping(ctx)

	assert.Equal(t, expectedErr, err)
}

func TestMockClient_ThreadSafety(t *testing.T) {
	client := NewMockClient()
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			var dest []string
			client.Query(ctx, &dest, fmt.Sprintf("SELECT * FROM table%d", id))
			client.Exec(ctx, fmt.Sprintf("DELETE FROM table%d", id))
			client.Ping(ctx)
			client.GetDB()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts
	assert.Equal(t, 10, client.GetQueryCallCount())
	assert.Equal(t, 10, client.GetExecCallCount())
	assert.Equal(t, 10, client.PingCalls)
	assert.Equal(t, 10, client.GetDBCalls)
}

func TestMockClient_InterfaceCompliance(t *testing.T) {
	// This test ensures MockClient implements SQLServerClient interface
	var _ SQLServerClient = (*MockClient)(nil)
}
