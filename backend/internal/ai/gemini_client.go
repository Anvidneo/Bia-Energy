package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// geminiClient calls Gemini's generateContent REST endpoint directly over
// net/http. A single request/response pair needs no SDK, so this adds no
// new dependency to go.mod.
type geminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string // overridable in tests; production default set below

	// maxAttempts/backoff control the retry policy for transient failures
	// (429 quota, 503 overloaded, and network errors while ctx still has
	// time left). Zero means "use the production default" — see
	// complete(). Overridable in tests to keep them fast.
	maxAttempts int
	backoff     time.Duration
}

// newGeminiClient builds a client for the given API key and model. An
// empty model falls back to a current free-tier Gemini model.
func newGeminiClient(apiKey, model string) *geminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &geminiClient{
		apiKey:      apiKey,
		model:       model,
		httpClient:  &http.Client{},
		baseURL:     "https://generativelanguage.googleapis.com/v1beta",
		maxAttempts: 3,
		backoff:     300 * time.Millisecond,
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

// complete sends prompt as a single-turn request and returns the first
// candidate's text. It satisfies the completer interface LLMExplainer
// depends on.
//
// Transient failures (429 quota exhaustion, 503 "model overloaded", and a
// network error while the caller's context still has time left) get a
// short jittered-backoff retry instead of failing the whole anomaly
// explanation on the first hiccup. Anything else (bad model name, invalid
// key, a malformed response) returns immediately — retrying wouldn't fix
// it.
func (c *geminiClient) complete(ctx context.Context, prompt string) (string, error) {
	reqBody, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	})
	if err != nil {
		return "", fmt.Errorf("gemini: encoding request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, c.model)

	maxAttempts := c.maxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	backoff := c.backoff
	if backoff <= 0 {
		backoff = 300 * time.Millisecond
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			wait := backoff*time.Duration(1<<uint(attempt-2)) + time.Duration(rand.Int64N(int64(backoff)))
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return "", fmt.Errorf("gemini: %w", ctx.Err())
			}
		}

		text, retryable, err := c.doRequest(ctx, url, reqBody)
		if err == nil {
			return text, nil
		}
		lastErr = err
		if !retryable {
			return "", err
		}
	}
	return "", fmt.Errorf("gemini: giving up after %d attempts: %w", maxAttempts, lastErr)
}

// doRequest performs a single attempt against the API. retryable tells
// complete whether another attempt is worth making.
func (c *geminiClient) doRequest(ctx context.Context, url string, reqBody []byte) (text string, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", false, fmt.Errorf("gemini: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// The API key travels as a header, never in the URL: a request error
	// (timeout, connection failure) gets logged together with the request
	// URL, and a key in the query string would end up in plaintext in
	// application logs.
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// A network-level failure is worth one retry, unless the caller's
		// own context is what killed it — retrying then would just fail
		// the same way immediately.
		return "", ctx.Err() == nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("gemini: reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// 429 (quota) and 503 (overloaded) are transient; anything else
		// (bad model name, invalid key, malformed request) won't fix
		// itself on retry.
		retryable = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable
		return "", retryable, fmt.Errorf("gemini: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", false, fmt.Errorf("gemini: decoding response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", false, fmt.Errorf("gemini: response had no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, false, nil
}
