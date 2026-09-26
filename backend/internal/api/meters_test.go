package api

import (
	"net/http"
	"testing"

	"bia-energy/backend/internal/models"
)

func TestListMetersEmpty(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	meters := decodeJSON[[]models.Meter](t, rec)
	if len(meters) != 0 {
		t.Errorf("got %d meters, want 0", len(meters))
	}
}

func TestListMetersReturnsSeededMetersInOrder(t *testing.T) {
	conn := testDB(t)
	insertMeter(t, conn, "M-102")
	insertMeter(t, conn, "M-101")
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters")
	meters := decodeJSON[[]models.Meter](t, rec)
	if len(meters) != 2 || meters[0].ID != "M-101" || meters[1].ID != "M-102" {
		t.Errorf("got %+v, want [M-101, M-102] (ORDER BY id)", meters)
	}
}

func TestGetMeterFound(t *testing.T) {
	conn := testDB(t)
	insertMeter(t, conn, "M-101")
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters/M-101")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	meter := decodeJSON[models.Meter](t, rec)
	if meter.ID != "M-101" {
		t.Errorf("got meter %+v, want M-101", meter)
	}
}

func TestGetMeterNotFound(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters/M-999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decodeJSON[map[string]string](t, rec)
	if body["error"] != "meter not found" {
		t.Errorf("error = %q, want \"meter not found\"", body["error"])
	}
}

func TestGetMeterReadingsFound(t *testing.T) {
	conn := testDB(t)
	insertReading(t, conn, "M-101", "2026-01-01 00:00:00", 12.5, 220.1, 5.2, 0.95)
	insertReading(t, conn, "M-101", "2026-01-01 01:00:00", 12.8, 220.0, 5.3, 0.95)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters/M-101/readings")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	readings := decodeJSON[[]models.Reading](t, rec)
	if len(readings) != 2 {
		t.Fatalf("got %d readings, want 2", len(readings))
	}
	if readings[0].ConsumptionKWh != 12.5 || readings[1].ConsumptionKWh != 12.8 {
		t.Errorf("unexpected readings order/values: %+v", readings)
	}
}

func TestGetMeterReadingsNotFound(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters/M-999/readings")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decodeJSON[map[string]string](t, rec)
	if body["error"] != "meter not found or has no readings" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestGetMeterReadingsMeterExistsButHasNone(t *testing.T) {
	conn := testDB(t)
	insertMeter(t, conn, "M-101")
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/meters/M-101/readings")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (meter with zero readings)", rec.Code)
	}
}
