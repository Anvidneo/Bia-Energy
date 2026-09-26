package api

import (
	"database/sql"
	"net/http"

	"bia-energy/backend/internal/models"
)

// handleDashboardSummary returns the aggregate KPIs the dashboard's top
// row shows.
//
// @Summary      Resumen del dashboard
// @Description  KPIs agregados: total de medidores, anomalías activas y fecha del último análisis.
// @Tags         dashboard
// @Produce      json
// @Success      200  {object}  models.DashboardSummary
// @Failure      500  {object}  ErrorResponse
// @Router       /dashboard/summary [get]
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
