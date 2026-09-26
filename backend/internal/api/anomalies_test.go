package api

import (
	"fmt"
	"net/http"
	"testing"

	"bia-energy/backend/internal/models"
)

func TestListAnomaliesEmpty(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/anomalies")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	anomalies := decodeJSON[[]models.Anomaly](t, rec)
	if len(anomalies) != 0 {
		t.Errorf("got %d anomalies, want 0", len(anomalies))
	}
}

func TestListAnomaliesReturnsSeededRows(t *testing.T) {
	conn := testDB(t)
	insertAnomaly(t, conn, "M-101")
	insertAnomaly(t, conn, "M-102")
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/anomalies")
	anomalies := decodeJSON[[]models.Anomaly](t, rec)
	if len(anomalies) != 2 {
		t.Fatalf("got %d anomalies, want 2", len(anomalies))
	}
	for _, a := range anomalies {
		if !a.Anomaly || a.Type != models.TypeRealAnomaly || a.Severity != models.SeverityHigh {
			t.Errorf("unexpected anomaly shape: %+v", a)
		}
	}
}

func TestGetAnomalyFound(t *testing.T) {
	conn := testDB(t)
	id := insertAnomaly(t, conn, "M-101")
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, fmt.Sprintf("/anomalies/%d", id))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	a := decodeJSON[models.Anomaly](t, rec)
	if a.ID != id || a.MeterID != "M-101" {
		t.Errorf("got %+v, want id=%d meter_id=M-101", a, id)
	}
}

func TestGetAnomalyNotFound(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/anomalies/999999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decodeJSON[map[string]string](t, rec)
	if body["error"] != "anomaly not found" {
		t.Errorf("error = %q, want \"anomaly not found\"", body["error"])
	}
}

func TestGetAnomalyInvalidIDIsServerError(t *testing.T) {
	// The handler casts the path param to ::bigint in SQL; a non-numeric id
	// fails that cast rather than matching zero rows.
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/anomalies/not-a-number")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 for a non-numeric id", rec.Code)
	}
}
