package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"bia-energy/backend/internal/models"
)

func scanAnomaly(row interface {
	Scan(dest ...interface{}) error
}) (models.Anomaly, error) {
	var a models.Anomaly
	var relatedType sql.NullString
	var relatedAt sql.NullTime
	var evidenceJSON string
	err := row.Scan(
		&a.ID, &a.MeterID, &a.DetectedAt, &a.Type, &a.Severity, &a.Confidence,
		&a.Reason, &a.RecommendedAction, &relatedType, &relatedAt, &evidenceJSON,
	)
	if err != nil {
		return a, err
	}
	a.Anomaly = true
	if relatedType.Valid {
		a.RelatedEventType = relatedType.String
	}
	if relatedAt.Valid {
		a.RelatedEventAt = &relatedAt.Time
	}
	_ = json.Unmarshal([]byte(evidenceJSON), &a.Evidence)
	return a, nil
}

// handleListAnomalies lists every anomaly from the latest persisted
// analysis run.
//
// @Summary      Listar anomalías
// @Description  Anomalías de la última corrida de análisis persistida, más reciente primero.
// @Tags         anomalies
// @Produce      json
// @Success      200  {array}   models.Anomaly
// @Failure      500  {object}  ErrorResponse
// @Router       /anomalies [get]
func (d *Deps) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	rows, err := d.DB.QueryContext(r.Context(), `
		SELECT id, meter_id, detected_at, type, severity, confidence,
		       reason, recommended_action, related_event_type, related_event_at, evidence_json
		FROM anomalies ORDER BY detected_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = rows.Close() }()

	anomalies := []models.Anomaly{}
	for rows.Next() {
		a, err := scanAnomaly(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		anomalies = append(anomalies, a)
	}
	writeJSON(w, http.StatusOK, anomalies)
}

// handleGetAnomaly returns one anomaly by its database id.
//
// @Summary      Obtener una anomalía
// @Description  Detalle completo de una anomalía, incluida su evidencia.
// @Tags         anomalies
// @Produce      json
// @Param        id   path      integer  true  "id numérico de la anomalía"
// @Success      200  {object}  models.Anomaly
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /anomalies/{id} [get]
func (d *Deps) handleGetAnomaly(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row := d.DB.QueryRowContext(r.Context(), `
		SELECT id, meter_id, detected_at, type, severity, confidence,
		       reason, recommended_action, related_event_type, related_event_at, evidence_json
		FROM anomalies WHERE id = $1::bigint`, id)

	a, err := scanAnomaly(row)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "anomaly not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}
