// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewSQLClient(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	logger := zap.NewNop()

	client := NewSQLClient(sqlxDB, logger)

	assert.NotNil(t, client)
	assert.Equal(t, sqlxDB, client.db)
	assert.Equal(t, logger, client.logger)
}

func TestSQLClient_Query(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	logger := zap.NewNop()
	client := NewSQLClient(sqlxDB, logger)

	tests := []struct {
		name    string
		query   string
		args    []interface{}
		setup   func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:  "Query without args",
			query: "SELECT * FROM users",
			args:  nil,
			setup: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(1, "John").
					AddRow(2, "Jane")
				m.ExpectQuery("SELECT \\* FROM users").WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "Query with args",
			query: "SELECT * FROM users WHERE id = ?",
			args:  []interface{}{1},
			setup: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(1, "John")
				m.ExpectQuery("SELECT \\* FROM users WHERE id = \\?").
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "Query error",
			query: "SELECT * FROM users",
			args:  nil,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT \\* FROM users").
					WillReturnError(fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(mock)

			type user struct {
				ID   int    `db:"id"`
				Name string `db:"name"`
			}
			var users []user

			err := client.Query(context.Background(), &users, tt.query, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSQLClient_QueryRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	logger := zap.NewNop()
	client := NewSQLClient(sqlxDB, logger)

	tests := []struct {
		name    string
		query   string
		args    []interface{}
		setup   func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:  "QueryRow without args",
			query: "SELECT COUNT(*) as count FROM users",
			args:  nil,
			setup: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
				m.ExpectQuery("SELECT COUNT\\(\\*\\) as count FROM users").WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "QueryRow with args",
			query: "SELECT * FROM users WHERE id = ?",
			args:  []interface{}{1},
			setup: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "John")
				m.ExpectQuery("SELECT \\* FROM users WHERE id = \\?").
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "QueryRow error",
			query: "SELECT * FROM users WHERE id = ?",
			args:  []interface{}{999},
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT \\* FROM users WHERE id = \\?").
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(mock)

			type result struct {
				ID    int    `db:"id"`
				Name  string `db:"name"`
				Count int    `db:"count"`
			}
			var res result

			err := client.QueryRow(context.Background(), &res, tt.query, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSQLClient_Exec(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	logger := zap.NewNop()
	client := NewSQLClient(sqlxDB, logger)

	tests := []struct {
		name    string
		query   string
		args    []interface{}
		setup   func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:  "Exec without args",
			query: "DELETE FROM sessions WHERE expired = 1",
			args:  nil,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM sessions WHERE expired = 1").
					WillReturnResult(sqlmock.NewResult(0, 5))
			},
			wantErr: false,
		},
		{
			name:  "Exec with args",
			query: "UPDATE users SET status = ? WHERE id = ?",
			args:  []interface{}{"active", 1},
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE users SET status = \\? WHERE id = \\?").
					WithArgs("active", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:  "Exec error",
			query: "INSERT INTO users (name) VALUES (?)",
			args:  []interface{}{"John"},
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO users \\(name\\) VALUES \\(\\?\\)").
					WithArgs("John").
					WillReturnError(fmt.Errorf("constraint violation"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(mock)

			err := client.Exec(context.Background(), tt.query, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSQLClient_Ping(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	logger := zap.NewNop()
	client := NewSQLClient(sqlxDB, logger)

	tests := []struct {
		name    string
		setup   func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "Successful ping",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectPing()
			},
			wantErr: false,
		},
		{
			name: "Failed ping",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectPing().WillReturnError(fmt.Errorf("connection lost"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(mock)

			err := client.Ping(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSQLClient_Close(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *SQLClient
		wantErr bool
	}{
		{
			name: "Close successful",
			setup: func() *SQLClient {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				mock.ExpectClose()
				sqlxDB := sqlx.NewDb(db, "sqlmock")
				return NewSQLClient(sqlxDB, zap.NewNop())
			},
			wantErr: false,
		},
		{
			name: "Close with nil db",
			setup: func() *SQLClient {
				return &SQLClient{
					db:     nil,
					logger: zap.NewNop(),
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setup()
			err := client.Close()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLClient_GetDB(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *SQLClient
		expect func(*testing.T, *sql.DB)
	}{
		{
			name: "GetDB returns underlying DB",
			setup: func() *SQLClient {
				db, _, err := sqlmock.New()
				require.NoError(t, err)
				sqlxDB := sqlx.NewDb(db, "sqlmock")
				return NewSQLClient(sqlxDB, zap.NewNop())
			},
			expect: func(t *testing.T, db *sql.DB) {
				assert.NotNil(t, db)
			},
		},
		{
			name: "GetDB with nil db",
			setup: func() *SQLClient {
				return &SQLClient{
					db:     nil,
					logger: zap.NewNop(),
				}
			},
			expect: func(t *testing.T, db *sql.DB) {
				assert.Nil(t, db)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setup()
			db := client.GetDB()
			tt.expect(t, db)
		})
	}
}

func TestSQLClient_ContextCancellation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	logger := zap.NewNop()
	client := NewSQLClient(sqlxDB, logger)

	t.Run("Query respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		var result []struct{}
		err := client.Query(ctx, &result, "SELECT * FROM users")
		assert.Error(t, err)
	})

	t.Run("Exec respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := client.Exec(ctx, "DELETE FROM users")
		assert.Error(t, err)
	})

	t.Run("Ping respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := client.Ping(ctx)
		assert.Error(t, err)
	})

	// No expectations should have been set on mock
	assert.NoError(t, mock.ExpectationsWereMet())
}
