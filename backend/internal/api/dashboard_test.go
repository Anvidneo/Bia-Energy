package api

import (
	"net/http"
	"testing"

	"bia-energy/backend/internal/models"
)

func TestDashboardSummaryEmpty(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/dashboard/summary")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	summary := decodeJSON[models.DashboardSummary](t, rec)
	if summary.TotalMeters != 0 || summary.ActiveAnomalies != 0 || summary.LastAnalysisAt != nil {
		t.Errorf("unexpected summary for an empty database: %+v", summary)
	}
}

func TestDashboardSummaryWithData(t *testing.T) {
	conn := testDB(t)
	insertMeter(t, conn, "M-101")
	insertMeter(t, conn, "M-102")
	insertAnomaly(t, conn, "M-101")
	insertAnomaly(t, conn, "M-102")
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/dashboard/summary")
	summary := decodeJSON[models.DashboardSummary](t, rec)

	if summary.TotalMeters != 2 {
		t.Errorf("TotalMeters = %d, want 2", summary.TotalMeters)
	}
	if summary.ActiveAnomalies != 2 {
		t.Errorf("ActiveAnomalies = %d, want 2", summary.ActiveAnomalies)
	}
	if summary.LastAnalysisAt == nil {
		t.Error("LastAnalysisAt = nil, want a timestamp once anomalies exist")
	}
}
