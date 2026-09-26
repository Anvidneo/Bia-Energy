package ai

import (
	"strings"
	"testing"

	"bia-energy/backend/internal/models"
)

func TestRuleExplainerRealAnomaly(t *testing.T) {
	a := &models.Anomaly{
		Type: models.TypeRealAnomaly,
		Evidence: models.Evidence{
			DeviationPct: 180.5, ObservedKWh: 34.5, BaselineMedianKWh: 12.3, ConsecutiveHours: 5,
		},
	}
	RuleExplainer{}.Explain(a)

	if a.Reason == "" {
		t.Fatal("expected a non-empty Reason")
	}
	for _, want := range []string{"180.5%", "34.5 kWh", "12.3 kWh", "5 horas"} {
		if !strings.Contains(a.Reason, want) {
			t.Errorf("Reason %q does not mention %q", a.Reason, want)
		}
	}
	if !strings.Contains(a.RecommendedAction, "urgente") {
		t.Errorf("RecommendedAction %q does not read as urgent", a.RecommendedAction)
	}
}

func TestRuleExplainerFalsePositive(t *testing.T) {
	a := &models.Anomaly{
		Type:             models.TypeFalsePositive,
		RelatedEventType: models.EventScheduledOutage,
		Evidence: models.Evidence{
			DeviationPct: -95.0, ObservedKWh: 0.5, BaselineMedianKWh: 10.0,
		},
	}
	RuleExplainer{}.Explain(a)

	if !strings.Contains(a.Reason, models.EventScheduledOutage) {
		t.Errorf("Reason %q does not cite the related event type", a.Reason)
	}
	if !strings.Contains(a.RecommendedAction, "No escalar") {
		t.Errorf("RecommendedAction %q should say not to escalate", a.RecommendedAction)
	}
}

func TestRuleExplainerExplainableAnomaly(t *testing.T) {
	a := &models.Anomaly{
		Type:             models.TypeExplainableAnomaly,
		RelatedEventType: models.EventOperationalChange,
		Evidence: models.Evidence{
			DeviationPct: 60.0, ObservedKWh: 20.0, BaselineMedianKWh: 12.5, ConsecutiveHours: 4,
		},
	}
	RuleExplainer{}.Explain(a)

	if !strings.Contains(a.Reason, models.EventOperationalChange) {
		t.Errorf("Reason %q does not cite the related event type", a.Reason)
	}
	if !strings.Contains(a.RecommendedAction, "Validar") {
		t.Errorf("RecommendedAction %q should ask to validate the operational change", a.RecommendedAction)
	}
}

func TestRuleExplainerDataQuality(t *testing.T) {
	a := &models.Anomaly{
		Type: models.TypeDataQuality,
		Evidence: models.Evidence{
			ObservedKWh: 40.0, ExpectedKWhFromElectrical: 12.0, ConsecutiveHours: 3,
		},
	}
	RuleExplainer{}.Explain(a)

	for _, want := range []string{"40.0 kWh", "12.0 kWh", "3 lecturas"} {
		if !strings.Contains(a.Reason, want) {
			t.Errorf("Reason %q does not mention %q", a.Reason, want)
		}
	}
	if !strings.Contains(a.RecommendedAction, "instrumentación") {
		t.Errorf("RecommendedAction %q should mention checking the instrumentation", a.RecommendedAction)
	}
}

func TestRuleExplainerUnknownTypeFallsBackToDefault(t *testing.T) {
	a := &models.Anomaly{Type: models.TypeNormal}
	RuleExplainer{}.Explain(a)

	if a.Reason != "Sin evidencia suficiente para explicar este caso." {
		t.Errorf("Reason = %q, want the default fallback text", a.Reason)
	}
	if a.RecommendedAction != "Revisar manualmente." {
		t.Errorf("RecommendedAction = %q, want the default fallback text", a.RecommendedAction)
	}
}

func TestRuleExplainerImplementsExplainer(t *testing.T) {
	var _ Explainer = RuleExplainer{}
}
