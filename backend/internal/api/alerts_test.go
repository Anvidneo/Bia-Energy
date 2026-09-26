package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"bia-energy/backend/internal/ai"
	"bia-energy/backend/internal/models"
	"bia-energy/backend/internal/seed"
)

// fakePublisher records every call it receives instead of talking to a
// real Firestore project, per the design doc's testing plan: it exists to
// verify which anomalies qualify (only HIGH) and that a publisher error
// never propagates out of publishCriticalAlerts.
type fakePublisher struct {
	mu    sync.Mutex
	calls []models.Anomaly
	err   error
}

func (f *fakePublisher) PublishCriticalAlert(ctx context.Context, a models.Anomaly, analysisID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, a)
	return f.err
}

func (f *fakePublisher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestPublishCriticalAlertsOnlyPublishesHighSeverity(t *testing.T) {
	pub := &fakePublisher{}
	d := &Deps{AlertPublisher: pub}

	anomalies := []models.Anomaly{
		{MeterID: "M-1", Severity: models.SeverityHigh},
		{MeterID: "M-2", Severity: models.SeverityMedium},
		{MeterID: "M-3", Severity: models.SeverityLow},
		{MeterID: "M-4", Severity: models.SeverityHigh},
	}

	d.publishCriticalAlerts(context.Background(), "an_test", anomalies)

	if pub.callCount() != 2 {
		t.Fatalf("callCount = %d, want 2 (only HIGH severity)", pub.callCount())
	}
	for _, a := range pub.calls {
		if a.Severity != models.SeverityHigh {
			t.Errorf("published a non-HIGH anomaly: %+v", a)
		}
	}
}

func TestPublishCriticalAlertsIsNoOpWithoutPublisher(t *testing.T) {
	d := &Deps{} // AlertPublisher left nil, as it is unless USE_FIREBASE_ALERTS is on
	anomalies := []models.Anomaly{{MeterID: "M-1", Severity: models.SeverityHigh}}

	// Must not panic and must not require an AlertPublisher.
	d.publishCriticalAlerts(context.Background(), "an_test", anomalies)
}

func TestPublishCriticalAlertsSwallowsPublisherErrors(t *testing.T) {
	pub := &fakePublisher{err: errors.New("firestore unavailable")}
	d := &Deps{AlertPublisher: pub}
	anomalies := []models.Anomaly{
		{MeterID: "M-1", Severity: models.SeverityHigh},
		{MeterID: "M-2", Severity: models.SeverityHigh},
	}

	// The point of this test: an error from every single call must not
	// panic, return an error, or stop earlier anomalies' calls from firing.
	d.publishCriticalAlerts(context.Background(), "an_test", anomalies)

	if pub.callCount() != 2 {
		t.Errorf("callCount = %d, want 2 (a publisher error must not stop the loop)", pub.callCount())
	}
}

func TestPublishCriticalAlertsEmptyBatch(t *testing.T) {
	pub := &fakePublisher{}
	d := &Deps{AlertPublisher: pub}
	d.publishCriticalAlerts(context.Background(), "an_test", nil)
	if pub.callCount() != 0 {
		t.Errorf("callCount = %d, want 0 for an empty batch", pub.callCount())
	}
}

// TestRunAnalysisPublishesCriticalAlertsEndToEnd drives the real seed
// dataset (data/readings.csv + data/events.csv) through POST /ai/analyze
// so it exercises the actual wiring in runAnalysis, not just
// publishCriticalAlerts in isolation. The dataset is known (from the
// enunciado and internal/detection's own tests) to classify M-109 as a
// HIGH-severity REAL_ANOMALY.
func TestRunAnalysisPublishesCriticalAlertsEndToEnd(t *testing.T) {
	conn := testDB(t)
	if err := seed.Load(conn, "../../data/readings.csv", "../../data/events.csv"); err != nil {
		t.Fatalf("seeding real dataset: %v", err)
	}

	pub := &fakePublisher{}
	router := NewRouter(&Deps{DB: conn, Explainer: ai.RuleExplainer{}, AlertPublisher: pub})

	rec := doRequest(router, http.MethodPost, "/ai/analyze")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST /ai/analyze status = %d, want 202", rec.Code)
	}
	analysisID := decodeJSON[map[string]string](t, rec)["analysisId"]

	waitForAnalysis(t, router, analysisID)

	if pub.callCount() == 0 {
		t.Fatal("expected at least one critical alert to be published for the real dataset's known HIGH anomalies")
	}
	for _, a := range pub.calls {
		if a.Severity != models.SeverityHigh {
			t.Errorf("published a non-HIGH anomaly end-to-end: %+v", a)
		}
	}
}
