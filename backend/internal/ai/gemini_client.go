package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// geminiClient calls Gemini's generateContent REST endpoint directly over
// net/http. A single request/response pair needs no SDK, so this adds no
// new dependency to go.mod.
type geminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string // overridable in tests; production default set below
}

// newGeminiClient builds a client for the given API key and model. An
// empty model falls back to a current free-tier Gemini model.
func newGeminiClient(apiKey, model string) *geminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &geminiClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
		baseURL:    "https://generativelanguage.googleapis.com/v1beta",
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
func (c *geminiClient) complete(ctx context.Context, prompt string) (string, error) {
	reqBody, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	})
	if err != nil {
		return "", fmt.Errorf("gemini: encoding request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("gemini: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decoding response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: response had no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
