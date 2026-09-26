package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"

	"bia-energy/backend/internal/ai"
	"bia-energy/backend/internal/db"
)

// testDBURL is set up once by TestMain, in a throwaway database dedicated
// to this package's test run — never the same database internal/seed's or
// internal/db's own tests use. go test runs different packages' tests in
// parallel, and every test here TRUNCATEs shared tables between cases, so
// sharing a database across packages would make them flaky by stepping on
// each other's rows.
var testDBURL string

func TestMain(m *testing.M) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		// No Postgres available (e.g. running locally without
		// `docker compose up postgres`): every testDB(t) call below will
		// see testDBURL == "" and skip itself.
		os.Exit(m.Run())
	}

	u, err := url.Parse(base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing TEST_DATABASE_URL: %v\n", err)
		os.Exit(1)
	}

	dbName := fmt.Sprintf("bia_energy_test_api_%d", time.Now().UnixNano())
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
// applies the real schema, and truncates every table so each test starts
// from a clean slate. Tests in this package run sequentially (none call
// t.Parallel()), so truncating between them is safe.
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

// testRouter builds a real router (chi.Router, no mocking) backed by conn,
// exactly like cmd/api/main.go does.
func testRouter(conn *sql.DB) http.Handler {
	return NewRouter(&Deps{DB: conn, Explainer: ai.RuleExplainer{}})
}

// doRequest fires method/path against router and returns the recorded
// response.
func doRequest(router http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decoding JSON response %s: %v", rec.Body.String(), err)
	}
	return v
}

// insertMeter, insertReading and insertAnomaly seed rows directly via SQL
// so each handler test can set up exactly the data it needs without going
// through the seed package.
func insertMeter(t *testing.T, conn *sql.DB, id string) {
	t.Helper()
	if _, err := conn.Exec(`INSERT INTO meters (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, id); err != nil {
		t.Fatalf("inserting meter %s: %v", id, err)
	}
}

func insertReading(t *testing.T, conn *sql.DB, meterID, ts string, kwh, voltage, current, pf float64) {
	t.Helper()
	insertMeter(t, conn, meterID)
	if _, err := conn.Exec(
		`INSERT INTO readings (meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor)
		 VALUES ($1,$2::timestamptz,$3,$4,$5,$6)`,
		meterID, ts, kwh, voltage, current, pf,
	); err != nil {
		t.Fatalf("inserting reading for %s: %v", meterID, err)
	}
}

func insertAnomaly(t *testing.T, conn *sql.DB, meterID string) int64 {
	t.Helper()
	insertMeter(t, conn, meterID)
	var id int64
	err := conn.QueryRow(`
		INSERT INTO anomalies
			(meter_id, detected_at, type, severity, confidence, reason, recommended_action, evidence_json, analysis_id)
		VALUES ($1, now(), 'REAL_ANOMALY', 'HIGH', 0.9, 'test reason', 'test action', '{}', 'an_test')
		RETURNING id`, meterID).Scan(&id)
	if err != nil {
		t.Fatalf("inserting anomaly for %s: %v", meterID, err)
	}
	return id
}
