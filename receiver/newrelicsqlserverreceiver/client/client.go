// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"database/sql"
)

// SQLServerClient defines the interface for SQL Server database operations
// This abstraction enables testing with mock clients and decouples scrapers from direct DB access
type SQLServerClient interface {
	// Query executes a query and scans results into the provided destination
	// dest should be a pointer to a slice of structs with db tags
	Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// QueryRow executes a query expected to return at most one row
	QueryRow(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// Exec executes a query without returning any rows
	Exec(ctx context.Context, query string, args ...interface{}) error

	// Ping verifies the connection to the database is still alive
	Ping(ctx context.Context) error

	// Close closes the database connection
	Close() error

	// GetDB returns the underlying database connection (for advanced operations)
	GetDB() *sql.DB
}
