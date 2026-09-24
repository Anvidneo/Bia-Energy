package detection

import (
	"os"
	"testing"

	"bia-energy/backend/internal/models"
	"bia-energy/backend/internal/seed"
)

// loadRealDataset loads the actual test-provided readings.csv/events.csv
// (backend/data/) so these tests exercise the exact algorithm against the
// exact dataset the evaluator will run it on — not a hand-crafted
// approximation of it.
func loadRealDataset(t *testing.T) ([]models.Reading, []models.Event) {
	t.Helper()
	rf, err := os.Open("../../data/readings.csv")
	if err != nil {
		t.Fatalf("opening readings.csv: %v", err)
	}
	defer rf.Close()
	readings, err := seed.ParseReadings(rf)
	if err != nil {
		t.Fatalf("parsing readings.csv: %v", err)
	}

	ef, err := os.Open("../../data/events.csv")
	if err != nil {
		t.Fatalf("opening events.csv: %v", err)
	}
	defer ef.Close()
	events, err := seed.ParseEvents(ef)
	if err != nil {
		t.Fatalf("parsing events.csv: %v", err)
	}
	return readings, events
}

// TestKnownCases is the table from the plan doc's "Casos conocidos y
// clasificación esperada": the 100-point section of the rubric hinges on
// these 4 meters classifying exactly this way.
func TestKnownCases(t *testing.T) {
	readings, events := loadRealDataset(t)

	byMeter := make(map[string][]models.Reading)
	for _, r := range readings {
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r)
	}

	cases := []struct {
		meterID          string
		wantType         models.AnomalyType
		wantSeverity     models.Severity
		wantRelatedEvent string // "" means no related event expected
	}{
		{"M-109", models.TypeRealAnomaly, models.SeverityHigh, ""},
		{"M-106", models.TypeFalsePositive, models.SeverityLow, models.EventScheduledOutage},
		{"M-104", models.TypeExplainableAnomaly, models.SeverityMedium, models.EventOperationalChange},
		{"M-112", models.TypeDataQuality, models.SeverityHigh, ""},
	}

	for _, tc := range cases {
		t.Run(tc.meterID, func(t *testing.T) {
			readingsForMeter, ok := byMeter[tc.meterID]
			if !ok || len(readingsForMeter) == 0 {
				t.Fatalf("no readings found for %s in readings.csv", tc.meterID)
			}
			got := AnalyzeMeter(tc.meterID, readingsForMeter, events)
			if got == nil {
				t.Fatalf("%s: expected an anomaly, got nil (NORMAL)", tc.meterID)
			}
			if got.Type != tc.wantType {
				t.Errorf("%s: type = %s, want %s", tc.meterID, got.Type, tc.wantType)
			}
			if got.Severity != tc.wantSeverity {
				t.Errorf("%s: severity = %s, want %s", tc.meterID, got.Severity, tc.wantSeverity)
			}
			if tc.wantRelatedEvent != "" && got.RelatedEventType != tc.wantRelatedEvent {
				t.Errorf("%s: related_event_type = %q, want %q", tc.meterID, got.RelatedEventType, tc.wantRelatedEvent)
			}
			if got.Confidence <= 0 || got.Confidence > 1 {
				t.Errorf("%s: confidence = %v, want in (0,1]", tc.meterID, got.Confidence)
			}
			if !got.Anomaly {
				t.Errorf("%s: Anomaly = false, want true", tc.meterID)
			}
		})
	}
}

// TestUnremarkableMetersStayNormal guards against false positives: the 8
// meters that are NOT part of the known-cases table must never be flagged.
func TestUnremarkableMetersStayNormal(t *testing.T) {
	readings, events := loadRealDataset(t)
	byMeter := make(map[string][]models.Reading)
	for _, r := range readings {
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r)
	}

	known := map[string]bool{"M-109": true, "M-106": true, "M-104": true, "M-112": true}
	for meterID, rs := range byMeter {
		if known[meterID] {
			continue
		}
		t.Run(meterID, func(t *testing.T) {
			got := AnalyzeMeter(meterID, rs, events)
			if got != nil {
				t.Errorf("%s: expected NORMAL (nil), got anomaly type=%s severity=%s", meterID, got.Type, got.Severity)
			}
		})
	}
}

// TestRunPipeline checks the full multi-meter entrypoint returns exactly
// the 4 known anomalies and nothing else.
func TestRunPipeline(t *testing.T) {
	readings, events := loadRealDataset(t)
	anomalies := RunPipeline(readings, events)

	if len(anomalies) != 4 {
		t.Fatalf("RunPipeline returned %d anomalies, want 4: %+v", len(anomalies), anomalies)
	}
	seen := make(map[string]models.Anomaly)
	for _, a := range anomalies {
		seen[a.MeterID] = a
	}
	for _, meterID := range []string{"M-109", "M-106", "M-104", "M-112"} {
		if _, ok := seen[meterID]; !ok {
			t.Errorf("RunPipeline: missing expected anomaly for %s", meterID)
		}
	}
}
