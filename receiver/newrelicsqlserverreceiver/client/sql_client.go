// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// SQLClient implements SQLServerClient interface by wrapping sqlx.DB
// This provides a consistent interface for database operations while maintaining existing behavior
type SQLClient struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewSQLClient creates a new SQL client wrapping the provided sqlx.DB connection
func NewSQLClient(db *sqlx.DB, logger *zap.Logger) *SQLClient {
	return &SQLClient{
		db:     db,
		logger: logger,
	}
}

// Query executes a query and scans results into the provided destination
// dest should be a pointer to a slice of structs with db tags
func (c *SQLClient) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	c.logger.Debug("Running query", zap.String("query", query), zap.Any("args", args))

	if len(args) > 0 {
		return c.db.SelectContext(ctx, dest, query, args...)
	}
	return c.db.SelectContext(ctx, dest, query)
}

// QueryRow executes a query expected to return at most one row
// dest should be a pointer to a struct or variables for scanning
func (c *SQLClient) QueryRow(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	c.logger.Debug("Running single row query", zap.String("query", query), zap.Any("args", args))

	if len(args) > 0 {
		return c.db.GetContext(ctx, dest, query, args...)
	}
	return c.db.GetContext(ctx, dest, query)
}

// Exec executes a query without returning any rows
// Useful for INSERT, UPDATE, DELETE, or DDL statements
func (c *SQLClient) Exec(ctx context.Context, query string, args ...interface{}) error {
	c.logger.Debug("Executing query", zap.String("query", query), zap.Any("args", args))

	var err error
	if len(args) > 0 {
		_, err = c.db.ExecContext(ctx, query, args...)
	} else {
		_, err = c.db.ExecContext(ctx, query)
	}
	return err
}

// Ping verifies the connection to the database is still alive
func (c *SQLClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close closes the database connection
func (c *SQLClient) Close() error {
	if c.db == nil {
		return nil
	}

	if err := c.db.Close(); err != nil {
		c.logger.Warn("Unable to close SQL Connection", zap.Error(err))
		return err
	}
	return nil
}

// GetDB returns the underlying database connection for advanced operations
// This allows access to the raw *sql.DB when needed for compatibility
func (c *SQLClient) GetDB() *sql.DB {
	if c.db == nil {
		return nil
	}
	return c.db.DB
}
