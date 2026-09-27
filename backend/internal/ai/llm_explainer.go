package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"bia-energy/backend/internal/models"
)

// completer is the minimal seam LLMExplainer needs from an LLM client —
// deliberately tiny so tests can inject a fake instead of making a real
// network call.
type completer interface {
	complete(ctx context.Context, prompt string) (string, error)
}

// LLMExplainer rephrases what the deterministic detection engine already
// decided into more natural language via an LLM. It NEVER reclassifies an
// anomaly or invents numbers: the prompt cites detection's own Evidence,
// and the model is asked only for prose. Any failure — network, timeout,
// an unparseable response — falls back to Fallback (RuleExplainer in
// production) so an anomaly is never left without Reason/RecommendedAction,
// and the POST /ai/analyze pipeline never fails because of the LLM.
type LLMExplainer struct {
	Client   completer
	Fallback Explainer
	Timeout  time.Duration
}

// NewLLMExplainer builds an LLMExplainer backed by Gemini, falling back to
// RuleExplainer on any failure.
//
// Timeout is generous (20s, vs. Gemini's own docs suggesting responses
// usually land well under 10s) because geminiClient already retries 429s
// and 503s internally with backoff, and Explain runs inside the
// background POST /ai/analyze pipeline (see internal/api/ai.go), never on
// an HTTP request goroutine — so there's no user-facing latency budget to
// protect here, only "don't hang forever if Gemini is completely down".
func NewLLMExplainer(apiKey, model string) LLMExplainer {
	return LLMExplainer{
		Client:   newGeminiClient(apiKey, model),
		Fallback: RuleExplainer{},
		Timeout:  20 * time.Second,
	}
}

func (e LLMExplainer) Explain(a *models.Anomaly) {
	fallback := e.Fallback
	if fallback == nil {
		fallback = RuleExplainer{}
	}

	// Compute the rule-based text first: it's the fallback, and it also
	// gives the prompt an evidence-grounded starting point to rephrase
	// instead of asking the model to frame the numbers on its own.
	fallback.Explain(a)

	if e.Client == nil {
		return
	}

	timeout := e.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	text, err := e.Client.complete(ctx, buildPrompt(a))
	if err != nil {
		log.Printf("ai: LLM explain failed for %s, using rule-based text: %v", a.MeterID, err)
		return
	}

	reason, action, ok := parseCompletion(text)
	if !ok {
		log.Printf("ai: LLM explain returned an unparseable response for %s, using rule-based text", a.MeterID)
		return
	}
	a.Reason = reason
	a.RecommendedAction = action
}

// buildPrompt cites only numbers detection already computed (a.Evidence)
// plus the rule-based Reason as a reference — the model is asked to
// rephrase, never to reclassify or invent figures.
func buildPrompt(a *models.Anomaly) string {
	var b strings.Builder
	b.WriteString("Eres un asistente que redacta, en español y en tono claro y profesional, ")
	b.WriteString("la explicación de una anomalía de consumo eléctrico ya clasificada por un sistema determinístico. ")
	b.WriteString("No cambies la clasificación ni inventes cifras: usa únicamente los datos de abajo. ")
	b.WriteString("Responde SOLO con un JSON de la forma {\"reason\": \"...\", \"recommended_action\": \"...\"}, sin texto adicional ni bloques de código.\n\n")
	fmt.Fprintf(&b, "Medidor: %s\n", a.MeterID)
	fmt.Fprintf(&b, "Tipo: %s\n", a.Type)
	fmt.Fprintf(&b, "Severidad: %s\n", a.Severity)
	fmt.Fprintf(&b, "Confianza: %.2f\n", a.Confidence)
	if a.RelatedEventType != "" {
		fmt.Fprintf(&b, "Evento relacionado: %s\n", a.RelatedEventType)
	}
	fmt.Fprintf(&b, "Consumo observado: %.1f kWh\n", a.Evidence.ObservedKWh)
	fmt.Fprintf(&b, "Baseline esperado: %.1f kWh\n", a.Evidence.BaselineMedianKWh)
	fmt.Fprintf(&b, "Desviación: %.1f%%\n", a.Evidence.DeviationPct)
	fmt.Fprintf(&b, "Horas consecutivas: %d\n", a.Evidence.ConsecutiveHours)
	fmt.Fprintf(&b, "Explicación de referencia (basada en reglas; puedes reformularla): %s\n", a.Reason)
	return b.String()
}

// completionPayload is the shape the prompt asks the model for.
type completionPayload struct {
	Reason            string `json:"reason"`
	RecommendedAction string `json:"recommended_action"`
}

// parseCompletion extracts {reason, recommended_action} from the model's
// raw text response. ok is false whenever the text isn't usable — the
// caller must then keep the rule-based text already set.
func parseCompletion(text string) (reason, action string, ok bool) {
	text = extractJSONObject(text)
	var payload completionPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return "", "", false
	}
	payload.Reason = strings.TrimSpace(payload.Reason)
	payload.RecommendedAction = strings.TrimSpace(payload.RecommendedAction)
	if payload.Reason == "" || payload.RecommendedAction == "" {
		return "", "", false
	}
	return payload.Reason, payload.RecommendedAction, true
}

// extractJSONObject trims anything before the first '{' and after the
// last '}', since models sometimes wrap JSON in prose or a markdown fence
// despite being asked not to.
func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return s
	}
	return s[start : end+1]
}
