package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"bia-energy/backend/internal/detection"
	"bia-energy/backend/internal/models"
)

// analysisState is the in-memory bookkeeping for one POST /ai/analyze run.
// It's ephemeral on purpose (resets on restart): the durable output of a
// run is the rows it writes to the anomalies table, not this progress
// tracker, which exists only so the frontend can poll and show the
// pipeline stages from enunciado section 13.
type analysisState struct {
	result models.AnalysisResult
}

// pipelineStages mirrors the frontend's visual pipeline: Lecturas ->
// Baseline -> Deteccion -> Correlacion -> Eventos -> Explicacion ->
// Recomendacion. The dataset is tiny enough that the real computation
// finishes in well under a second, so each stage gets a short artificial
// pause purely so the UI has something to show.
var pipelineStages = []string{
	"Lecturas", "Baseline", "Deteccion", "Correlacion", "Eventos", "Explicacion", "Recomendacion",
}

func (d *Deps) handlePostAnalyze(w http.ResponseWriter, r *http.Request) {
	id := newAnalysisID()
	now := time.Now().UTC()

	d.mu.Lock()
	d.analyses[id] = &analysisState{result: models.AnalysisResult{
		ID:        id,
		Status:    models.AnalysisProcessing,
		Stage:     pipelineStages[0],
		StartedAt: now,
	}}
	d.mu.Unlock()

	go d.runAnalysis(id)

	writeJSON(w, http.StatusAccepted, map[string]string{"analysisId": id})
}

func (d *Deps) handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	d.mu.Lock()
	st, ok := d.analyses[id]
	var result models.AnalysisResult
	if ok {
		result = st.result
	}
	d.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "analysis not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *Deps) runAnalysis(id string) {
	ctx := context.Background()
	setStage := func(stage string) {
		d.mu.Lock()
		if st, ok := d.analyses[id]; ok {
			st.result.Stage = stage
		}
		d.mu.Unlock()
		time.Sleep(300 * time.Millisecond)
	}

	setStage(pipelineStages[0]) // Lecturas
	readings, events, err := d.loadReadingsAndEvents(ctx)
	if err != nil {
		d.failAnalysis(id, err)
		return
	}

	setStage(pipelineStages[1]) // Baseline
	setStage(pipelineStages[2]) // Deteccion
	anomalies := detection.RunPipeline(readings, events)

	setStage(pipelineStages[3]) // Correlacion
	setStage(pipelineStages[4]) // Eventos

	setStage(pipelineStages[5]) // Explicacion
	for i := range anomalies {
		d.Explainer.Explain(&anomalies[i])
	}

	setStage(pipelineStages[6]) // Recomendacion
	if err := d.persistAnomalies(ctx, id, anomalies); err != nil {
		d.failAnalysis(id, err)
		return
	}

	finished := time.Now().UTC()
	d.mu.Lock()
	if st, ok := d.analyses[id]; ok {
		st.result.Status = models.AnalysisDone
		st.result.FinishedAt = &finished
		st.result.Anomalies = anomalies
	}
	d.mu.Unlock()
}

func (d *Deps) failAnalysis(id string, cause error) {
	d.mu.Lock()
	if st, ok := d.analyses[id]; ok {
		st.result.Status = models.AnalysisError
		st.result.Error = cause.Error()
	}
	d.mu.Unlock()
}

func (d *Deps) loadReadingsAndEvents(ctx context.Context) ([]models.Reading, []models.Event, error) {
	readingRows, err := d.DB.QueryContext(ctx,
		`SELECT meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor FROM readings ORDER BY meter_id, ts`)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = readingRows.Close() }()

	var readings []models.Reading
	for readingRows.Next() {
		var rd models.Reading
		if err := readingRows.Scan(&rd.MeterID, &rd.Timestamp, &rd.ConsumptionKWh, &rd.VoltageV, &rd.CurrentA, &rd.PowerFactor); err != nil {
			return nil, nil, err
		}
		readings = append(readings, rd)
	}

	eventRows, err := d.DB.QueryContext(ctx, `SELECT meter_id, event_ts, event_type, description FROM events`)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = eventRows.Close() }()

	var events []models.Event
	for eventRows.Next() {
		var e models.Event
		if err := eventRows.Scan(&e.MeterID, &e.EventTimestamp, &e.EventType, &e.Description); err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}

	return readings, events, nil
}

// persistAnomalies replaces the anomalies table's contents with the latest
// analysis run's results. Simple "latest run wins" model — good enough for
// this MVP's static dataset; a real system would keep history per
// analysisId instead of truncating.
// persistAnomalies replaces the anomalies table's contents with the latest
// analysis run's results, and writes each row's DB-assigned id back into
// anomalies (by index) so the copy kept in analysisState.result matches
// what GET /anomalies will later return. Simple "latest run wins" model —
// good enough for this MVP's static dataset; a real system would keep
// history per analysisId instead of truncating.
func (d *Deps) persistAnomalies(ctx context.Context, analysisID string, anomalies []models.Anomaly) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM anomalies`); err != nil {
		return err
	}

	for i := range anomalies {
		a := &anomalies[i]
		evidenceJSON, err := json.Marshal(a.Evidence)
		if err != nil {
			return err
		}
		var relatedType interface{}
		if a.RelatedEventType != "" {
			relatedType = a.RelatedEventType
		}
		var relatedAt interface{}
		if a.RelatedEventAt != nil {
			relatedAt = *a.RelatedEventAt
		}
		err = tx.QueryRowContext(ctx, `
			INSERT INTO anomalies
				(meter_id, detected_at, type, severity, confidence, reason, recommended_action,
				 related_event_type, related_event_at, evidence_json, analysis_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			RETURNING id`,
			a.MeterID, a.DetectedAt, a.Type, a.Severity, a.Confidence, a.Reason, a.RecommendedAction,
			relatedType, relatedAt, string(evidenceJSON), analysisID,
		).Scan(&a.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func newAnalysisID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "an_" + hex.EncodeToString(b)
}
