package api

import (
	"database/sql"
	"net/http"

	"bia-energy/backend/internal/models"
)

func (d *Deps) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	var summary models.DashboardSummary

	if err := d.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM meters`).Scan(&summary.TotalMeters); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := d.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM anomalies`).Scan(&summary.ActiveAnomalies); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var lastAnalysis sql.NullTime
	if err := d.DB.QueryRowContext(r.Context(), `SELECT max(created_at) FROM anomalies`).Scan(&lastAnalysis); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if lastAnalysis.Valid {
		summary.LastAnalysisAt = &lastAnalysis.Time
	}

	writeJSON(w, http.StatusOK, summary)
}
