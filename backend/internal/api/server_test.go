package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	router := NewRouter(&Deps{})
	rec := doRequest(router, http.MethodGet, "/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := decodeJSON[map[string]string](t, rec)
	if body["status"] != "ok" || body["service"] != "bia-energy-api" {
		t.Errorf("unexpected /health body: %+v", body)
	}
}

func TestCorsMiddlewareSetsHeadersAndHandlesPreflight(t *testing.T) {
	router := NewRouter(&Deps{})

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing/wrong Access-Control-Allow-Origin header: %v", rec.Header())
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing Access-Control-Allow-Methods header")
	}
}

func TestNewRouterInitializesAnalysesMapWhenNil(t *testing.T) {
	d := &Deps{}
	NewRouter(d)
	if d.analyses == nil {
		t.Error("NewRouter should initialize a nil analyses map")
	}
}
