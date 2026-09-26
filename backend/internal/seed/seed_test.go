package seed

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lib/pq"

	"bia-energy/backend/internal/db"
)

// testDBURL is set up once by TestMain, in a throwaway database dedicated
// to this package's test run — never the same database internal/api's or
// internal/db's own tests use, since go test runs different packages'
// tests in parallel and these tests TRUNCATE shared tables between cases.
var testDBURL string

func TestMain(m *testing.M) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		os.Exit(m.Run())
	}

	u, err := url.Parse(base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing TEST_DATABASE_URL: %v\n", err)
		os.Exit(1)
	}

	dbName := fmt.Sprintf("bia_energy_test_seed_%d", time.Now().UnixNano())
	admin, err := sql.Open("postgres", base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening maintenance connection: %v\n", err)
		os.Exit(1)
	}
	if _, err := admin.Exec(fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(dbName))); err != nil {
		fmt.Fprintf(os.Stderr, "creating test database %s: %v\n", dbName, err)
		os.Exit(1)
	}
	_ = admin.Close()

	dbURL := *u
	dbURL.Path = "/" + dbName
	testDBURL = dbURL.String()

	code := m.Run()

	cleanup, err := sql.Open("postgres", base)
	if err == nil {
		_, _ = cleanup.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", pq.QuoteIdentifier(dbName)))
		_ = cleanup.Close()
	}
	os.Exit(code)
}

// testDB opens a connection to this package's dedicated test database,
// applies the real schema, and wipes any leftover rows so each test starts
// clean. Tests in this package run sequentially (none call t.Parallel()),
// so truncating between them is safe.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping test that needs a real Postgres")
	}
	conn, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	if err := db.ApplySchema(conn); err != nil {
		t.Fatalf("applying schema: %v", err)
	}
	if _, err := conn.Exec(`TRUNCATE anomalies, events, readings, meters RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncating tables: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func writeTempCSV(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

func TestLoadInsertsReadingsAndEvents(t *testing.T) {
	conn := testDB(t)

	readingsCSV := "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor\n" +
		"M-101,2026-01-01 00:00:00,12.5,220.1,5.2,0.95\n" +
		"M-101,2026-01-01 01:00:00,12.8,220.0,5.3,0.95\n"
	eventsCSV := "meter_id,event_timestamp,event_type,description\n" +
		"M-101,2026-01-01 00:30:00,OPERATIONAL_CHANGE,Cambio de turno\n"

	readingsPath := writeTempCSV(t, "readings.csv", readingsCSV)
	eventsPath := writeTempCSV(t, "events.csv", eventsCSV)

	if err := Load(conn, readingsPath, eventsPath); err != nil {
		t.Fatalf("Load: %v", err)
	}

	var meterCount, readingCount, eventCount int
	_ = conn.QueryRow(`SELECT count(*) FROM meters`).Scan(&meterCount)
	_ = conn.QueryRow(`SELECT count(*) FROM readings`).Scan(&readingCount)
	_ = conn.QueryRow(`SELECT count(*) FROM events`).Scan(&eventCount)

	if meterCount != 1 {
		t.Errorf("meterCount = %d, want 1", meterCount)
	}
	if readingCount != 2 {
		t.Errorf("readingCount = %d, want 2", readingCount)
	}
	if eventCount != 1 {
		t.Errorf("eventCount = %d, want 1", eventCount)
	}

	// Load is meant to be safe to call again on every startup: once the
	// tables have rows, it must skip re-inserting rather than duplicating
	// or erroring on the UNIQUE constraints.
	if err := Load(conn, readingsPath, eventsPath); err != nil {
		t.Fatalf("second Load call: %v", err)
	}
	_ = conn.QueryRow(`SELECT count(*) FROM readings`).Scan(&readingCount)
	if readingCount != 2 {
		t.Errorf("readingCount after second Load = %d, want still 2 (already-seeded skip)", readingCount)
	}
}

func TestLoadMissingReadingsFile(t *testing.T) {
	conn := testDB(t)
	eventsPath := writeTempCSV(t, "events.csv", "meter_id,event_timestamp,event_type,description\n")
	if err := Load(conn, "/nonexistent/readings.csv", eventsPath); err == nil {
		t.Fatal("expected an error for a missing readings file")
	}
}

func TestLoadMissingEventsFile(t *testing.T) {
	conn := testDB(t)
	readingsPath := writeTempCSV(t, "readings.csv",
		"meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor\n"+
			"M-101,2026-01-01 00:00:00,12.5,220.1,5.2,0.95\n")
	if err := Load(conn, readingsPath, "/nonexistent/events.csv"); err == nil {
		t.Fatal("expected an error for a missing events file")
	}
}

func TestLoadInvalidReadingsCSV(t *testing.T) {
	conn := testDB(t)
	readingsPath := writeTempCSV(t, "readings.csv", "meter_id,timestamp\nM-101,2026-01-01 00:00:00\n")
	eventsPath := writeTempCSV(t, "events.csv", "meter_id,event_timestamp,event_type,description\n")
	if err := Load(conn, readingsPath, eventsPath); err == nil {
		t.Fatal("expected an error for readings.csv missing required columns")
	}
}
