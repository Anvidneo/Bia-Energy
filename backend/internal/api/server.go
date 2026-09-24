// Package api holds the HTTP handlers and router (chi) for the Bia Energy
// backend — one file per resource, mounted here.
package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"bia-energy/backend/internal/ai"
)

// Deps are the dependencies every handler needs. Kept as one small struct
// (rather than package-level globals) so tests can construct their own.
type Deps struct {
	DB        *sql.DB
	Explainer ai.Explainer

	mu       sync.Mutex
	analyses map[string]*analysisState
}

// NewRouter builds the chi router with all 8 endpoints from the plan doc.
func NewRouter(d *Deps) http.Handler {
	if d.analyses == nil {
		d.analyses = make(map[string]*analysisState)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "bia-energy-api"})
	})

	r.Get("/dashboard/summary", d.handleDashboardSummary)

	r.Get("/meters", d.handleListMeters)
	r.Get("/meters/{id}", d.handleGetMeter)
	r.Get("/meters/{id}/readings", d.handleGetMeterReadings)

	r.Post("/ai/analyze", d.handlePostAnalyze)
	r.Get("/ai/analysis/{id}", d.handleGetAnalysis)

	r.Get("/anomalies", d.handleListAnomalies)
	r.Get("/anomalies/{id}", d.handleGetAnomaly)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
