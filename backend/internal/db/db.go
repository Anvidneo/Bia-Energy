// Package db owns the Postgres connection and applies schema.sql
// (CREATE TABLE IF NOT EXISTS) on startup — no migration tool for this
// MVP, see the plan doc's "Decisiones técnicas".
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"regexp"

	"github.com/lib/pq"
)

//go:embed schema.sql
var schemaSQL string

var missingDBPattern = regexp.MustCompile(`database "([^"]+)" does not exist`)

// validPostgresIdentifier matches a plain, unquoted-safe Postgres
// identifier: it must start with a letter or underscore and contain only
// letters, digits, and underscores. dbName always comes from Postgres'
// own error message (see missingDatabaseName), which can only name a
// database that exists on the server and therefore was itself created
// under these same rules — but createDatabaseIfPossible re-validates it
// here anyway before formatting it into SQL, rather than relying solely
// on pq.QuoteIdentifier, since CREATE DATABASE's target can't be a bind
// parameter in any Postgres driver (identifiers aren't parameterizable).
var validPostgresIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Connect opens a Postgres connection pool and verifies it with a ping.
//
// If the target database named in dsn doesn't exist yet, Connect creates
// it automatically via a short-lived maintenance connection to the same
// server's default "postgres" database, then retries. This lets a second
// app share one Postgres instance (e.g. a Render free-tier instance,
// which only allows one per account) using nothing more than its own
// DATABASE_URL — no manual "CREATE DATABASE" step, which would otherwise
// require external psql access some deployments don't have.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if pingErr := db.Ping(); pingErr != nil {
		dbName, missing := missingDatabaseName(pingErr)
		if !missing {
			_ = db.Close()
			return nil, fmt.Errorf("pinging database: %w", pingErr)
		}
		if createErr := createDatabaseIfPossible(dsn, dbName); createErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("pinging database: %w (auto-create also failed: %v)", pingErr, createErr)
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("pinging database after auto-create: %w", err)
		}
	}
	return db, nil
}

// missingDatabaseName reports whether err is Postgres' "database does not
// exist" error (SQLSTATE 3D000) and, if so, the database name it named.
func missingDatabaseName(err error) (string, bool) {
	pqErr, ok := err.(*pq.Error)
	if !ok || pqErr.Code != "3D000" {
		return "", false
	}
	m := missingDBPattern.FindStringSubmatch(pqErr.Message)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// createDatabaseIfPossible connects to the "postgres" maintenance database
// on the same server as dsn and issues CREATE DATABASE for dbName.
func createDatabaseIfPossible(dsn, dbName string) error {
	if !validPostgresIdentifier.MatchString(dbName) {
		return fmt.Errorf("refusing to auto-create database with unexpected name %q", dbName)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("dsn is not a URL, cannot auto-create: %w", err)
	}
	u.Path = "/postgres"
	admin, err := sql.Open("postgres", u.String())
	if err != nil {
		return fmt.Errorf("opening maintenance connection: %w", err)
	}
	defer func() { _ = admin.Close() }()

	// dbName is validated above and pq.QuoteIdentifier double-quotes it,
	// so this can't be used to inject arbitrary SQL.
	if _, err := admin.Exec(fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(dbName))); err != nil {
		return fmt.Errorf("creating database %s: %w", dbName, err)
	}
	return nil
}

// ApplySchema runs schema.sql (idempotent: CREATE TABLE/INDEX IF NOT
// EXISTS) against db.
func ApplySchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}
	return nil
}
