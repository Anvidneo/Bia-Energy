package db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestMissingDatabaseName(t *testing.T) {
	t.Run("does not exist error", func(t *testing.T) {
		err := &pq.Error{Code: "3D000", Message: `database "bia_energy" does not exist`}
		name, ok := missingDatabaseName(err)
		if !ok || name != "bia_energy" {
			t.Fatalf("got (%q, %v), want (bia_energy, true)", name, ok)
		}
	})

	t.Run("unrelated pq error", func(t *testing.T) {
		err := &pq.Error{Code: "42501", Message: "permission denied"}
		if _, ok := missingDatabaseName(err); ok {
			t.Fatal("expected ok=false for a non-3D000 error")
		}
	})

	t.Run("non-pq error", func(t *testing.T) {
		if _, ok := missingDatabaseName(errors.New("boom")); ok {
			t.Fatal("expected ok=false for a plain error")
		}
	})
}

func TestValidPostgresIdentifier(t *testing.T) {
	cases := map[string]bool{
		"bia_energy":         true,
		"ai_job_search_db":   true,
		"_leading_underscore": true,
		"":                   false,
		"1starts_with_digit": false,
		"has-hyphen":         false,
		"has space":          false,
		`bia"; DROP TABLE x`: false,
	}
	for name, want := range cases {
		if got := validPostgresIdentifier.MatchString(name); got != want {
			t.Errorf("validPostgresIdentifier(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestCreateDatabaseIfPossibleRejectsUnexpectedName(t *testing.T) {
	// Exercised without a live Postgres: an invalid name is rejected before
	// any connection is attempted.
	err := createDatabaseIfPossible("postgres://user:pass@localhost/postgres", `bad"name`)
	if err == nil {
		t.Fatal("expected an error for a non-identifier database name")
	}
}

func TestCreateDatabaseIfPossibleRejectsUnparseableDSN(t *testing.T) {
	err := createDatabaseIfPossible("://not-a-url", "validname")
	if err == nil {
		t.Fatal("expected an error for an unparseable dsn")
	}
}

// testDatabaseURL returns the base connection string for integration tests
// against a real Postgres server, or skips the test if none is available.
// CI sets TEST_DATABASE_URL against a service container; locally, run
// `docker compose up postgres` and export it yourself to opt in.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping test that needs a real Postgres")
	}
	return dsn
}

// TestConnectAutoCreatesMissingDatabase is the integration test for the
// whole feature db.go exists for: pointing Connect at a database that
// doesn't exist yet should create it and come up successfully, with no
// external psql/admin step required.
func TestConnectAutoCreatesMissingDatabase(t *testing.T) {
	base := testDatabaseURL(t)

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parsing TEST_DATABASE_URL: %v", err)
	}
	dbName := fmt.Sprintf("bia_energy_test_%d", time.Now().UnixNano())
	u.Path = "/" + dbName
	dsn := u.String()

	t.Cleanup(func() {
		admin, err := sql.Open("postgres", base)
		if err != nil {
			return
		}
		defer func() { _ = admin.Close() }()
		_, _ = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", pq.QuoteIdentifier(dbName)))
	})

	conn, err := Connect(dsn)
	if err != nil {
		t.Fatalf("Connect() to a not-yet-existing database: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := ApplySchema(conn); err != nil {
		t.Fatalf("ApplySchema: %v", err)
	}

	// Connecting again should now succeed on the very first ping — nothing
	// left to auto-create.
	conn2, err := Connect(dsn)
	if err != nil {
		t.Fatalf("Connect() to an already-existing database: %v", err)
	}
	defer func() { _ = conn2.Close() }()
}

func TestConnectFailsForUnrelatedPingError(t *testing.T) {
	base := testDatabaseURL(t)

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parsing TEST_DATABASE_URL: %v", err)
	}
	// A bad password produces a real ping error that isn't "database does
	// not exist" (28P01, not 3D000) — Connect must surface it as-is rather
	// than attempting to auto-create anything.
	u.User = url.UserPassword(u.User.Username(), "definitely-the-wrong-password")
	dsn := u.String()

	if _, err := Connect(dsn); err == nil {
		t.Fatal("expected Connect to fail for a wrong password")
	}
}

func TestMissingDatabaseNameUnrecognizedMessageShape(t *testing.T) {
	// Same SQLSTATE as the real case, but a message shape Postgres has
	// never actually sent — missingDatabaseName must not guess a name out
	// of it.
	err := &pq.Error{Code: "3D000", Message: "something unexpected happened"}
	if _, ok := missingDatabaseName(err); ok {
		t.Fatal("expected ok=false when the message doesn't name a database")
	}
}

func TestCreateDatabaseIfPossibleSurfacesExecError(t *testing.T) {
	base := testDatabaseURL(t)

	dbName := fmt.Sprintf("bia_energy_test_dup_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		admin, err := sql.Open("postgres", base)
		if err != nil {
			return
		}
		defer func() { _ = admin.Close() }()
		_, _ = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", pq.QuoteIdentifier(dbName)))
	})

	// First call creates it for real; the second must surface Postgres'
	// "database already exists" error instead of swallowing it.
	if err := createDatabaseIfPossible(base, dbName); err != nil {
		t.Fatalf("first createDatabaseIfPossible: %v", err)
	}
	if err := createDatabaseIfPossible(base, dbName); err == nil {
		t.Fatal("expected an error when the database already exists")
	}
}

func TestApplySchemaSurfacesExecError(t *testing.T) {
	base := testDatabaseURL(t)
	conn, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatalf("opening connection: %v", err)
	}
	_ = conn.Close() // closed before use, so any Exec on it fails

	if err := ApplySchema(conn); err == nil {
		t.Fatal("expected an error when applying schema on a closed connection")
	}
}
