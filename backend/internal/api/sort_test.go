package api

import (
	"testing"
	"time"

	"bia-energy/backend/internal/models"
)

func mkAnomaly(id int64, severity models.Severity, confidence float64, detectedAt time.Time) models.Anomaly {
	return models.Anomaly{ID: id, Severity: severity, Confidence: confidence, DetectedAt: detectedAt}
}

func TestSortAnomaliesByPrioritySeverityWins(t *testing.T) {
	now := time.Now()
	// Lower severity detected later than a higher-severity one — severity
	// must still win, exactly the "Prioriza M-109" scenario (M-109 is HIGH
	// but might not be the most recently detected row).
	low := mkAnomaly(1, models.SeverityLow, 0.9, now)
	high := mkAnomaly(2, models.SeverityHigh, 0.5, now.Add(-time.Hour))
	anomalies := []models.Anomaly{low, high}

	sortAnomaliesByPriority(anomalies)

	if anomalies[0].ID != high.ID || anomalies[1].ID != low.ID {
		t.Fatalf("got order %+v, want HIGH (id=%d) before LOW (id=%d)", anomalies, high.ID, low.ID)
	}
}

func TestSortAnomaliesByPriorityConfidenceBreaksSeverityTie(t *testing.T) {
	now := time.Now()
	lowConfidence := mkAnomaly(1, models.SeverityHigh, 0.4, now)
	highConfidence := mkAnomaly(2, models.SeverityHigh, 0.95, now.Add(-time.Hour))
	anomalies := []models.Anomaly{lowConfidence, highConfidence}

	sortAnomaliesByPriority(anomalies)

	if anomalies[0].ID != highConfidence.ID || anomalies[1].ID != lowConfidence.ID {
		t.Fatalf("got order %+v, want higher-confidence anomaly first", anomalies)
	}
}

func TestSortAnomaliesByPriorityRecencyBreaksRemainingTie(t *testing.T) {
	older := mkAnomaly(1, models.SeverityMedium, 0.7, time.Now().Add(-24*time.Hour))
	newer := mkAnomaly(2, models.SeverityMedium, 0.7, time.Now())
	anomalies := []models.Anomaly{older, newer}

	sortAnomaliesByPriority(anomalies)

	if anomalies[0].ID != newer.ID || anomalies[1].ID != older.ID {
		t.Fatalf("got order %+v, want the more recent anomaly first when severity and confidence tie", anomalies)
	}
}

func TestSortAnomaliesByPriorityMatchesEnunciadoExample(t *testing.T) {
	// Mirrors section 11's example table order: HIGH, HIGH, MEDIUM, LOW —
	// seeded here in a scrambled, purely-chronological order to prove the
	// sort — not insertion order — produces it.
	now := time.Now()
	m109 := mkAnomaly(109, models.SeverityHigh, 0.92, now.Add(-3*time.Hour))
	otherHigh := mkAnomaly(2, models.SeverityHigh, 0.80, now)
	medium := mkAnomaly(3, models.SeverityMedium, 0.99, now.Add(-1*time.Hour))
	low := mkAnomaly(4, models.SeverityLow, 0.60, now.Add(-2*time.Hour))
	anomalies := []models.Anomaly{low, medium, otherHigh, m109}

	sortAnomaliesByPriority(anomalies)

	gotOrder := []int64{anomalies[0].ID, anomalies[1].ID, anomalies[2].ID, anomalies[3].ID}
	wantOrder := []int64{m109.ID, otherHigh.ID, medium.ID, low.ID}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("got order %v, want %v (HIGH by confidence, then MEDIUM, then LOW)", gotOrder, wantOrder)
		}
	}
}
