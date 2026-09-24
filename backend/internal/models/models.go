// Package models defines the shared data contract used across seed,
// detection, ai and api: Meter, Reading, Event and Anomaly.
package models

import "time"

// Meter is an electric meter identified by its readings.csv meter_id
// (e.g. "M-109").
type Meter struct {
	ID string `json:"id"`
}

// Reading is one hourly measurement from readings.csv.
type Reading struct {
	MeterID        string    `json:"meter_id"`
	Timestamp      time.Time `json:"timestamp"`
	ConsumptionKWh float64   `json:"consumption_kwh"`
	VoltageV       float64   `json:"voltage_v"`
	CurrentA       float64   `json:"current_a"`
	PowerFactor    float64   `json:"power_factor"`
}

// Event is a known operational event from events.csv that can explain an
// otherwise-anomalous reading.
type Event struct {
	MeterID        string    `json:"meter_id"`
	EventTimestamp time.Time `json:"event_timestamp"`
	EventType      string    `json:"event_type"` // OPERATIONAL_CHANGE | SCHEDULED_OUTAGE | DATA_QUALITY | UNKNOWN | ...
	Description    string    `json:"description"`
}

// Known event types (events.csv). UNKNOWN means "no known event", not a
// literal CSV value the pipeline needs to special-case.
const (
	EventOperationalChange = "OPERATIONAL_CHANGE"
	EventScheduledOutage   = "SCHEDULED_OUTAGE"
	EventDataQuality       = "DATA_QUALITY"
	EventUnknown           = "UNKNOWN"
)

// AnomalyType is the classification produced by the deterministic
// detection engine (internal/detection) — never by an LLM.
type AnomalyType string

const (
	TypeNormal             AnomalyType = "NORMAL"
	TypeRealAnomaly        AnomalyType = "REAL_ANOMALY"
	TypeFalsePositive      AnomalyType = "FALSE_POSITIVE"
	TypeExplainableAnomaly AnomalyType = "EXPLAINABLE_ANOMALY"
	TypeDataQuality        AnomalyType = "DATA_QUALITY"
)

// Severity mirrors the enunciado's HIGH/MEDIUM/LOW scale.
type Severity string

const (
	SeverityHigh   Severity = "HIGH"
	SeverityMedium Severity = "MEDIUM"
	SeverityLow    Severity = "LOW"
)

// Evidence carries the raw numbers the detection engine computed, so
// ai.RuleExplainer can cite them instead of inventing language, and so the
// anomaly-detail screen has something concrete to show.
type Evidence struct {
	BaselineMedianKWh         float64 `json:"baseline_median_kwh"`
	ObservedKWh               float64 `json:"observed_kwh"`
	DeviationPct              float64 `json:"deviation_pct"` // (observed-baseline)/baseline * 100
	ZScore                    float64 `json:"z_score"`       // modified z-score, 0.6745*(x-median)/MAD
	ConsecutiveHours          int     `json:"consecutive_hours"`
	ExpectedKWhFromElectrical float64 `json:"expected_kwh_from_electrical"` // voltage*current*power_factor derived estimate
}

// Anomaly is the unit persisted and returned by the API. Its JSON shape
// matches the contract in the enunciado (section 10):
//
//	{ "meter_id", "anomaly", "type", "severity", "confidence", "reason", "recommended_action" }
type Anomaly struct {
	ID                int64       `json:"id"`
	MeterID           string      `json:"meter_id"`
	DetectedAt        time.Time   `json:"detected_at"`
	Anomaly           bool        `json:"anomaly"`
	Type              AnomalyType `json:"type"`
	Severity          Severity    `json:"severity"`
	Confidence        float64     `json:"confidence"`
	Reason            string      `json:"reason"`
	RecommendedAction string      `json:"recommended_action"`
	RelatedEventType  string      `json:"related_event_type,omitempty"`
	RelatedEventAt    *time.Time  `json:"related_event_at,omitempty"`
	Evidence          Evidence    `json:"evidence"`
}

// AnalysisStatus is the state of a POST /ai/analyze run, polled via
// GET /ai/analysis/:id.
type AnalysisStatus string

const (
	AnalysisPending    AnalysisStatus = "pending"
	AnalysisProcessing AnalysisStatus = "processing"
	AnalysisDone       AnalysisStatus = "done"
	AnalysisError      AnalysisStatus = "error"
)

// AnalysisResult tracks one run of the detection+explanation pipeline.
// Stage names match the visual pipeline the frontend shows (enunciado
// section 13): Lecturas -> Baseline -> Deteccion -> Correlacion -> Eventos
// -> Explicacion -> Recomendacion.
type AnalysisResult struct {
	ID         string         `json:"id"`
	Status     AnalysisStatus `json:"status"`
	Stage      string         `json:"stage"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Anomalies  []Anomaly      `json:"anomalies,omitempty"`
	Error      string         `json:"error,omitempty"`
}

// DashboardSummary backs GET /dashboard/summary.
type DashboardSummary struct {
	TotalMeters     int        `json:"total_meters"`
	ActiveAnomalies int        `json:"active_anomalies"`
	LastAnalysisAt  *time.Time `json:"last_analysis_at,omitempty"`
}
