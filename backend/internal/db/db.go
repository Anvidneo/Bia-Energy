// Package db owns the Postgres connection and applies schema.sql
// (CREATE TABLE IF NOT EXISTS) on startup — no migration tool for this
// MVP, see the plan doc's "Decisiones técnicas".
package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/lib/pq"
)

//go:embed schema.sql
var schemaSQL string

// Connect opens a Postgres connection pool and verifies it with a ping.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return db, nil
}

// ApplySchema runs schema.sql (idempotent: CREATE TABLE/INDEX IF NOT
// EXISTS) against db.
func ApplySchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}
	return nil
}
