package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bia-energy/backend/internal/models"
)

// fakeCompleter lets tests control exactly what LLMExplainer sees back
// from "the LLM" without any network call.
type fakeCompleter struct {
	resp  string
	err   error
	sleep time.Duration
}

func (f fakeCompleter) complete(ctx context.Context, prompt string) (string, error) {
	if f.sleep > 0 {
		select {
		case <-time.After(f.sleep):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.resp, f.err
}

func sampleAnomaly() *models.Anomaly {
	return &models.Anomaly{
		MeterID:    "M-109",
		Type:       models.TypeRealAnomaly,
		Severity:   models.SeverityHigh,
		Confidence: 0.99,
		Evidence: models.Evidence{
			BaselineMedianKWh: 52.6,
			ObservedKWh:       110.3,
			DeviationPct:      109.8,
			ConsecutiveHours:  58,
		},
	}
}

func TestLLMExplainerUsesLLMTextWhenWellFormed(t *testing.T) {
	e := LLMExplainer{
		Client:   fakeCompleter{resp: `{"reason": "Consumo muy por encima de lo normal.", "recommended_action": "Revisar el medidor ya."}`},
		Fallback: RuleExplainer{},
		Timeout:  time.Second,
	}
	a := sampleAnomaly()
	e.Explain(a)

	if a.Reason != "Consumo muy por encima de lo normal." {
		t.Errorf("Reason = %q, want the LLM text", a.Reason)
	}
	if a.RecommendedAction != "Revisar el medidor ya." {
		t.Errorf("RecommendedAction = %q, want the LLM text", a.RecommendedAction)
	}
}

func TestLLMExplainerFallsBackOnClientError(t *testing.T) {
	want := sampleAnomaly()
	RuleExplainer{}.Explain(want)

	e := LLMExplainer{
		Client:   fakeCompleter{err: errors.New("network down")},
		Fallback: RuleExplainer{},
		Timeout:  time.Second,
	}
	a := sampleAnomaly()
	e.Explain(a)

	if a.Reason != want.Reason || a.RecommendedAction != want.RecommendedAction {
		t.Errorf("expected the rule-based fallback text (%q/%q), got Reason=%q Action=%q",
			want.Reason, want.RecommendedAction, a.Reason, a.RecommendedAction)
	}
}

func TestLLMExplainerFallsBackOnUnparseableResponse(t *testing.T) {
	e := LLMExplainer{
		Client:   fakeCompleter{resp: "not json at all"},
		Fallback: RuleExplainer{},
		Timeout:  time.Second,
	}
	a := sampleAnomaly()
	e.Explain(a)

	if a.Reason == "" || a.RecommendedAction == "" {
		t.Fatal("fallback should have left Reason/RecommendedAction populated")
	}
	if strings.Contains(a.Reason, "not json") {
		t.Error("unparseable LLM text leaked into Reason")
	}
}

func TestLLMExplainerFallsBackOnEmptyFields(t *testing.T) {
	e := LLMExplainer{
		Client:   fakeCompleter{resp: `{"reason": "", "recommended_action": ""}`},
		Fallback: RuleExplainer{},
		Timeout:  time.Second,
	}
	a := sampleAnomaly()
	ruleReason := a.Reason
	e.Explain(a)
	// RuleExplainer.Explain was already called as part of Explain's own
	// fallback-first step, so Reason should be the rule text either way —
	// this asserts it wasn't blanked out by the empty LLM fields.
	_ = ruleReason
	if a.Reason == "" || a.RecommendedAction == "" {
		t.Error("empty LLM fields must not blank out the rule-based fallback")
	}
}

func TestLLMExplainerFallsBackOnTimeout(t *testing.T) {
	e := LLMExplainer{
		Client:   fakeCompleter{resp: `{"reason":"too late","recommended_action":"too late"}`, sleep: 50 * time.Millisecond},
		Fallback: RuleExplainer{},
		Timeout:  5 * time.Millisecond,
	}
	a := sampleAnomaly()
	e.Explain(a)

	if a.Reason == "too late" {
		t.Error("a response arriving after the timeout must not be used")
	}
	if a.Reason == "" || a.RecommendedAction == "" {
		t.Error("timeout must still leave the rule-based fallback in place")
	}
}

func TestLLMExplainerDefaultsTimeoutWhenUnset(t *testing.T) {
	// Timeout left at zero value: Explain must not hang forever or panic,
	// and should still complete using the default internal timeout.
	e := LLMExplainer{
		Client:   fakeCompleter{resp: `{"reason":"ok","recommended_action":"ok"}`},
		Fallback: RuleExplainer{},
	}
	a := sampleAnomaly()
	e.Explain(a)
	if a.Reason != "ok" {
		t.Errorf("Reason = %q, want ok", a.Reason)
	}
}

func TestLLMExplainerDefaultsFallbackWhenNil(t *testing.T) {
	e := LLMExplainer{
		Client:  fakeCompleter{err: errors.New("down")},
		Timeout: time.Second,
		// Fallback intentionally left nil.
	}
	a := sampleAnomaly()
	e.Explain(a)
	if a.Reason == "" || a.RecommendedAction == "" {
		t.Error("a nil Fallback should default to RuleExplainer, not leave Reason/RecommendedAction empty")
	}
}

func TestLLMExplainerNilClientUsesFallbackOnly(t *testing.T) {
	e := LLMExplainer{Fallback: RuleExplainer{}}
	a := sampleAnomaly()
	e.Explain(a)
	if a.Reason == "" || a.RecommendedAction == "" {
		t.Error("a nil Client should still leave the rule-based text in place")
	}
}

func TestNewLLMExplainerWiresGeminiClient(t *testing.T) {
	e := NewLLMExplainer("key", "model")
	if e.Client == nil {
		t.Fatal("Client should not be nil")
	}
	if e.Fallback == nil {
		t.Fatal("Fallback should not be nil")
	}
	if e.Timeout <= 0 {
		t.Error("Timeout should have a positive default")
	}
}

func TestBuildPromptCitesEvidenceNotJustLabels(t *testing.T) {
	a := sampleAnomaly()
	RuleExplainer{}.Explain(a)
	prompt := buildPrompt(a)

	for _, want := range []string{"M-109", "REAL_ANOMALY", "HIGH", "110.3", "52.6", "109.8"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildPromptIncludesRelatedEventWhenPresent(t *testing.T) {
	a := sampleAnomaly()
	a.Type = models.TypeExplainableAnomaly
	a.RelatedEventType = models.EventOperationalChange
	RuleExplainer{}.Explain(a)
	prompt := buildPrompt(a)
	if !strings.Contains(prompt, models.EventOperationalChange) {
		t.Error("prompt should mention the related event type when set")
	}
}

func TestParseCompletionRejectsMissingFields(t *testing.T) {
	if _, _, ok := parseCompletion(`{"reason": "only reason"}`); ok {
		t.Error("expected parseCompletion to fail without recommended_action")
	}
}

func TestParseCompletionHandlesMarkdownFence(t *testing.T) {
	text := "```json\n{\"reason\": \"r\", \"recommended_action\": \"a\"}\n```"
	reason, action, ok := parseCompletion(text)
	if !ok {
		t.Fatal("expected parseCompletion to strip surrounding markdown fence")
	}
	if reason != "r" || action != "a" {
		t.Errorf("reason=%q action=%q, want r/a", reason, action)
	}
}

func TestExtractJSONObjectNoBraces(t *testing.T) {
	if got := extractJSONObject("no braces here"); got != "no braces here" {
		t.Errorf("extractJSONObject returned %q, want the input unchanged", got)
	}
}
