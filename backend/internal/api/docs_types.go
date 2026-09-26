package api

// This file exists purely to give swaggo (swag init) a concrete Go type
// to point @Success/@Failure annotations at for responses that the real
// handlers build as inline map/struct literals (writeJSON/writeError in
// server.go). Nothing here is used at runtime — it only shapes the
// generated OpenAPI spec so it matches what the handlers actually return.

// ErrorResponse is the JSON body writeError sends on every non-2xx
// response: {"error": "..."}.
type ErrorResponse struct {
	Error string `json:"error"`
}

// AnalyzeAccepted is the JSON body handlePostAnalyze sends on 202.
type AnalyzeAccepted struct {
	AnalysisID string `json:"analysisId"`
}

// HealthResponse is the JSON body GET /health sends on 200.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}
