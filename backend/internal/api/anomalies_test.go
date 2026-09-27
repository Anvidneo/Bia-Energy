package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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

// TestListAnomaliesOrderedByPriority pins GET /anomalies' actual ordering
// against the enunciado's own section 11 example (HIGH, HIGH, MEDIUM, LOW)
// and the "Prioriza M-109" grading criterion — a meter detected earlier but
// more severe must still come first.
func TestListAnomaliesOrderedByPriority(t *testing.T) {
	conn := testDB(t)
	now := time.Now()
	low := insertAnomalyDetailed(t, conn, "M-106", "LOW", 0.60, now.Add(-1*time.Hour))
	m109 := insertAnomalyDetailed(t, conn, "M-109", "HIGH", 0.92, now.Add(-3*time.Hour))
	medium := insertAnomalyDetailed(t, conn, "M-104", "MEDIUM", 0.99, now.Add(-2*time.Hour))
	otherHigh := insertAnomalyDetailed(t, conn, "M-112", "HIGH", 0.80, now)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/anomalies")
	anomalies := decodeJSON[[]models.Anomaly](t, rec)
	if len(anomalies) != 4 {
		t.Fatalf("got %d anomalies, want 4", len(anomalies))
	}

	got := []int64{anomalies[0].ID, anomalies[1].ID, anomalies[2].ID, anomalies[3].ID}
	want := []int64{m109, otherHigh, medium, low}
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Fatalf("got order %v, want %v (HIGH by confidence, then MEDIUM, then LOW — never plain detected_at)", got, want)
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
