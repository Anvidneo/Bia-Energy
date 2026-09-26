package firebase

import (
	"context"
	"testing"
	"time"

	"bia-energy/backend/internal/models"
)

// A real Firestore project is out of scope for CI (per the design doc),
// so these tests cover everything that doesn't require one: the pure
// document-shaping logic, and the nil-safety guards that let a Publisher
// be used as a no-op zero value.

func sampleAlertAnomaly() models.Anomaly {
	detected := time.Date(2026, 9, 21, 19, 0, 0, 0, time.UTC)
	return models.Anomaly{
		ID:                42,
		MeterID:           "M-109",
		DetectedAt:        detected,
		Type:              models.TypeRealAnomaly,
		Severity:          models.SeverityHigh,
		Confidence:        0.99,
		Reason:            "Consumo muy por encima del baseline.",
		RecommendedAction: "Investigar de inmediato.",
	}
}

func TestAlertDocIDUsesDatabaseID(t *testing.T) {
	a := sampleAlertAnomaly()
	if got := alertDocID(a); got != "42" {
		t.Errorf("alertDocID = %q, want 42", got)
	}
}

func TestAlertDocumentFields(t *testing.T) {
	a := sampleAlertAnomaly()
	doc := alertDocument(a, "an_abc123")

	want := map[string]interface{}{
		"meter_id":           "M-109",
		"type":               "REAL_ANOMALY",
		"severity":           "HIGH",
		"confidence":         0.99,
		"reason":             a.Reason,
		"recommended_action": a.RecommendedAction,
		"detected_at":        a.DetectedAt,
		"analysis_id":        "an_abc123",
	}
	for k, v := range want {
		if doc[k] != v {
			t.Errorf("doc[%q] = %v, want %v", k, doc[k], v)
		}
	}
	if _, ok := doc["published_at"]; !ok {
		t.Error("doc should include a published_at timestamp")
	}
}

func TestPublishCriticalAlertOnNilPublisher(t *testing.T) {
	var p *Publisher
	if err := p.PublishCriticalAlert(context.Background(), sampleAlertAnomaly(), "an_x"); err == nil {
		t.Error("expected an error from a nil Publisher, not a panic or nil error")
	}
}

func TestPublishCriticalAlertOnUninitializedPublisher(t *testing.T) {
	p := &Publisher{}
	if err := p.PublishCriticalAlert(context.Background(), sampleAlertAnomaly(), "an_x"); err == nil {
		t.Error("expected an error from a Publisher with no client")
	}
}

func TestCloseOnNilPublisher(t *testing.T) {
	var p *Publisher
	if err := p.Close(); err != nil {
		t.Errorf("Close() on a nil Publisher should be a no-op, got %v", err)
	}
}

func TestCloseOnUninitializedPublisher(t *testing.T) {
	p := &Publisher{}
	if err := p.Close(); err != nil {
		t.Errorf("Close() with no client should be a no-op, got %v", err)
	}
}
