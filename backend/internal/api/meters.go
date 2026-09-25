package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"bia-energy/backend/internal/models"
)

func (d *Deps) handleListMeters(w http.ResponseWriter, r *http.Request) {
	rows, err := d.DB.QueryContext(r.Context(), `SELECT id FROM meters ORDER BY id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = rows.Close() }()

	meters := []models.Meter{}
	for rows.Next() {
		var m models.Meter
		if err := rows.Scan(&m.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		meters = append(meters, m)
	}
	writeJSON(w, http.StatusOK, meters)
}

func (d *Deps) handleGetMeter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m models.Meter
	err := d.DB.QueryRowContext(r.Context(), `SELECT id FROM meters WHERE id = $1`, id).Scan(&m.ID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "meter not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (d *Deps) handleGetMeterReadings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := d.DB.QueryContext(r.Context(),
		`SELECT meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor
		 FROM readings WHERE meter_id = $1 ORDER BY ts`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = rows.Close() }()

	readings := []models.Reading{}
	for rows.Next() {
		var rd models.Reading
		if err := rows.Scan(&rd.MeterID, &rd.Timestamp, &rd.ConsumptionKWh, &rd.VoltageV, &rd.CurrentA, &rd.PowerFactor); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		readings = append(readings, rd)
	}
	if len(readings) == 0 {
		writeError(w, http.StatusNotFound, "meter not found or has no readings")
		return
	}
	writeJSON(w, http.StatusOK, readings)
}
