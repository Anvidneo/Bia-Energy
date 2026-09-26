package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGeminiClientCompleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("api key query param = %q, want test-key", r.URL.Query().Get("key"))
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

	c := &geminiClient{apiKey: "test-key", model: "gemini-2.5-flash", httpClient: server.Client(), baseURL: server.URL}
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
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL}
	_, err := c.complete(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected an error for a non-200 status")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %v, want it to mention the status code", err)
	}
}

func TestGeminiClientCompleteMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL}
	if _, err := c.complete(context.Background(), "hello"); err == nil {
		t.Fatal("expected a decoding error for malformed JSON")
	}
}

func TestGeminiClientCompleteNoCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}))
	defer server.Close()

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL}
	if _, err := c.complete(context.Background(), "hello"); err == nil {
		t.Fatal("expected an error when the response has no candidates")
	}
}

func TestGeminiClientCompleteRequestFailure(t *testing.T) {
	// baseURL with no listener: the HTTP request itself fails.
	c := &geminiClient{apiKey: "k", model: "m", httpClient: &http.Client{Timeout: time.Second}, baseURL: "http://127.0.0.1:1"}
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

	c := &geminiClient{apiKey: "k", model: "m", httpClient: server.Client(), baseURL: server.URL}
	if _, err := c.complete(ctx, "hello"); err == nil {
		t.Fatal("expected a context-deadline error")
	}
}
