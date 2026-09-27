package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewGeminiClientDefaultsModel(t *testing.T) {
	c := newGeminiClient("key", "")
	if c.model != "gemini-2.5-flash" {
		t.Errorf("model = %q, want the default", c.model)
	}
	c2 := newGeminiClient("key", "gemini-custom")
	if c2.model != "gemini-custom" {
		t.Errorf("model = %q, want gemini-custom", c2.model)
	}
}

func TestNewGeminiClientSetsRetryDefaults(t *testing.T) {
	c := newGeminiClient("key", "")
	if c.maxAttempts <= 0 {
		t.Error("maxAttempts should have a positive production default")
	}
	if c.backoff <= 0 {
		t.Error("backoff should have a positive production default")
	}
}

func TestGeminiClientCompleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("key"); got != "" {
			t.Errorf("api key leaked into the URL query string: %q", got)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Errorf("x-goog-api-key header = %q, want test-key", got)
		}
		var body geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if len(body.Contents) != 1 || body.Contents[0].Parts[0].Text != "hello" {
			t.Errorf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"world"}]}}]}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "test-key", model: "gemini-2.5-flash", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 1}
	text, err := c.complete(context.Background(), "hello")
	if err != nil {
		t.Fatalf("complete() error = %v", err)
	}
	if text != "world" {
		t.Errorf("text = %q, want world", text)
	}
}

func TestGeminiClientCompleteNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 1}
	_, err := c.complete(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected an error for a non-200 status")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error = %v, want it to mention the status code", err)
	}
}

func TestGeminiClientCompleteMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 1}
	if _, err := c.complete(context.Background(), "hello"); err == nil {
		t.Fatal("expected a decoding error for malformed JSON")
	}
}

func TestGeminiClientCompleteNoCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 1}
	if _, err := c.complete(context.Background(), "hello"); err == nil {
		t.Fatal("expected an error when the response has no candidates")
	}
}

func TestGeminiClientCompleteRequestFailure(t *testing.T) {
	// baseURL with no listener: the HTTP request itself fails.
	c := &geminiClient{apiKey: "k", model: "m", httpClient: &http.Client{Timeout: time.Second}, baseURL: "http://127.0.0.1:1", maxAttempts: 1}
	if _, err := c.complete(context.Background(), "hello"); err == nil {
		t.Fatal("expected a request error when the server is unreachable")
	}
}

func TestGeminiClientCompleteContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"late"}]}}]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 1}
	if _, err := c.complete(ctx, "hello"); err == nil {
		t.Fatal("expected a context-deadline error")
	}
}

func TestGeminiClientRetriesOn503ThenSucceeds(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":503,"message":"overloaded","status":"UNAVAILABLE"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 3, backoff: time.Millisecond}
	text, err := c.complete(context.Background(), "hello")
	if err != nil {
		t.Fatalf("complete() error = %v, want it to succeed after retrying", err)
	}
	if text != "ok" {
		t.Errorf("text = %q, want ok", text)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server got %d calls, want exactly 2 (one 503, one success)", got)
	}
}

func TestGeminiClientRetriesOn429ThenSucceeds(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 3, backoff: time.Millisecond}
	if _, err := c.complete(context.Background(), "hello"); err != nil {
		t.Fatalf("complete() error = %v, want it to succeed after retrying", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server got %d calls, want exactly 2 (one 429, one success)", got)
	}
}

func TestGeminiClientGivesUpAfterMaxAttempts(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":503,"message":"overloaded","status":"UNAVAILABLE"}}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 3, backoff: time.Millisecond}
	_, err := c.complete(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected an error once every attempt is exhausted")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error = %v, want it to still mention the underlying 503", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("server got %d calls, want exactly maxAttempts (3)", got)
	}
}

func TestGeminiClientDoesNotRetryNonTransientStatus(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"message":"model not found","status":"NOT_FOUND"}}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 3, backoff: time.Millisecond}
	_, err := c.complete(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected an error for a 404")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server got %d calls, want exactly 1 — a 404 is not retryable", got)
	}
}

func TestGeminiClientStopsRetryingWhenContextExpires(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":503,"message":"overloaded","status":"UNAVAILABLE"}}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// A backoff far longer than the context's own deadline: complete must
	// give up waiting via ctx.Done() rather than sleeping through it.
	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL, maxAttempts: 5, backoff: time.Second}
	start := time.Now()
	if _, err := c.complete(ctx, "hello"); err == nil {
		t.Fatal("expected an error once the context expires mid-retry")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("complete() took %v, want it to stop waiting once ctx expired", elapsed)
	}
}
