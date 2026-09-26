package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"bia-energy/backend/internal/models"
)

func TestGetAnalysisNotFound(t *testing.T) {
	conn := testDB(t)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodGet, "/ai/analysis/an_doesnotexist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decodeJSON[map[string]string](t, rec)
	if body["error"] != "analysis not found" {
		t.Errorf("error = %q, want \"analysis not found\"", body["error"])
	}
}

// TestPostAnalyzeRunsPipelineToCompletion is the wiring-level test for the
// whole POST /ai/analyze -> GET /ai/analysis/:id flow: it does NOT assert
// which anomalies get detected (internal/detection's own tests already
// cover that exhaustively against the real dataset) — only that a run
// started here actually reaches "done", clears its readings/events from
// the DB correctly, and persists whatever it finds.
func TestPostAnalyzeRunsPipelineToCompletion(t *testing.T) {
	conn := testDB(t)
	insertReading(t, conn, "M-101", "2026-01-01 00:00:00", 12.5, 220.1, 5.2, 0.95)
	insertReading(t, conn, "M-101", "2026-01-01 01:00:00", 12.8, 220.0, 5.3, 0.95)
	router := testRouter(conn)

	rec := doRequest(router, http.MethodPost, "/ai/analyze")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /ai/analyze status = %d, want 202", rec.Code)
	}
	posted := decodeJSON[map[string]string](t, rec)
	analysisID := posted["analysisId"]
	if analysisID == "" {
		t.Fatal("expected a non-empty analysisId")
	}

	deadline := time.Now().Add(5 * time.Second)
	var result models.AnalysisResult
	for {
		pollRec := doRequest(router, http.MethodGet, "/ai/analysis/"+analysisID)
		if pollRec.Code != http.StatusOK {
			t.Fatalf("GET /ai/analysis/%s status = %d, want 200", analysisID, pollRec.Code)
		}
		if err := json.Unmarshal(pollRec.Body.Bytes(), &result); err != nil {
			t.Fatalf("decoding analysis result: %v", err)
		}
		if result.Status == models.AnalysisDone || result.Status == models.AnalysisError {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("analysis %s did not finish within 5s (last status: %s)", analysisID, result.Status)
		}
		time.Sleep(50 * time.Millisecond)
	}

	if result.Status != models.AnalysisDone {
		t.Fatalf("final status = %s, want done (error: %s)", result.Status, result.Error)
	}
	if result.FinishedAt == nil {
		t.Error("FinishedAt is nil for a done analysis")
	}

	// persistAnomalies should have replaced the anomalies table's contents
	// with this run's results (possibly empty — two readings are nowhere
	// near enough to build a meaningful hourly baseline).
	listRec := doRequest(router, http.MethodGet, "/anomalies")
	anomalies := decodeJSON[[]models.Anomaly](t, listRec)
	if len(anomalies) != len(result.Anomalies) {
		t.Errorf("GET /anomalies returned %d rows, but the analysis result carried %d", len(anomalies), len(result.Anomalies))
	}
	for _, a := range result.Anomalies {
		if a.Reason == "" || a.RecommendedAction == "" {
			t.Errorf("anomaly %+v was not explained (empty Reason/RecommendedAction)", a)
		}
	}
}

func TestPostAnalyzeFailsWhenDBUnavailable(t *testing.T) {
	conn := testDB(t)
	// Close the connection before the pipeline can use it, forcing
	// loadReadingsAndEvents to fail and exercising the "error" status path.
	_ = conn.Close()
	router := testRouter(conn)

	rec := doRequest(router, http.MethodPost, "/ai/analyze")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /ai/analyze status = %d, want 202 (failure surfaces via polling)", rec.Code)
	}
	analysisID := decodeJSON[map[string]string](t, rec)["analysisId"]

	deadline := time.Now().Add(5 * time.Second)
	var result models.AnalysisResult
	for {
		pollRec := doRequest(router, http.MethodGet, "/ai/analysis/"+analysisID)
		_ = json.Unmarshal(pollRec.Body.Bytes(), &result)
		if result.Status == models.AnalysisDone || result.Status == models.AnalysisError {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("analysis %s did not settle within 5s", analysisID)
		}
		time.Sleep(50 * time.Millisecond)
	}

	if result.Status != models.AnalysisError {
		t.Fatalf("status = %s, want error (closed DB connection)", result.Status)
	}
	if result.Error == "" {
		t.Error("expected a non-empty Error message")
	}
}
