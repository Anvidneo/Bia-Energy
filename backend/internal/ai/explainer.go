// Package ai is the explanation layer. It NEVER classifies — the
// deterministic internal/detection engine already decided type/severity —
// it only turns that decision plus its evidence into natural-language
// text. Two implementations share the Explainer interface so an
// LLM-backed one can be swapped in later (USE_LLM_EXPLAINER) without
// touching detection at all.
package ai

import (
	"fmt"

	"bia-energy/backend/internal/models"
)

// Explainer fills in an anomaly's Reason and RecommendedAction from what
// the detection engine already computed.
type Explainer interface {
	Explain(a *models.Anomaly)
}

// RuleExplainer is the default, deterministic implementation: template
// strings that cite the real numbers from a.Evidence. Zero cost, zero
// latency, 100% reproducible — which is what keeps this well suited for a
// demo and for tests that need stable output.
type RuleExplainer struct{}

func (RuleExplainer) Explain(a *models.Anomaly) {
	switch a.Type {
	case models.TypeRealAnomaly:
		a.Reason = fmt.Sprintf(
			"Consumo %.1f%% por encima del baseline horario (%.1f kWh observado vs %.1f kWh esperado), sostenido durante %d horas consecutivas, sin ningún evento operativo conocido que lo explique.",
			a.Evidence.DeviationPct, a.Evidence.ObservedKWh, a.Evidence.BaselineMedianKWh, a.Evidence.ConsecutiveHours,
		)
		a.RecommendedAction = "Investigar el medidor y la instalación asociada; priorizar como caso urgente."

	case models.TypeFalsePositive:
		a.Reason = fmt.Sprintf(
			"Caída de consumo (%.1f%% respecto al baseline, %.1f kWh observado vs %.1f kWh esperado) coincide en tiempo con un evento %s registrado para este medidor.",
			a.Evidence.DeviationPct, a.Evidence.ObservedKWh, a.Evidence.BaselineMedianKWh, a.RelatedEventType,
		)
		a.RecommendedAction = "No escalar: la desviación se explica por la salida programada."

	case models.TypeExplainableAnomaly:
		a.Reason = fmt.Sprintf(
			"Consumo %.1f%% por encima del baseline (%.1f kWh observado vs %.1f kWh esperado), sostenido durante %d horas, coincide en tiempo con un evento %s registrado para este medidor.",
			a.Evidence.DeviationPct, a.Evidence.ObservedKWh, a.Evidence.BaselineMedianKWh, a.Evidence.ConsecutiveHours, a.RelatedEventType,
		)
		a.RecommendedAction = "Validar que el cambio operativo reportado justifique el nuevo nivel de consumo."

	case models.TypeDataQuality:
		a.Reason = fmt.Sprintf(
			"El consumo reportado (%.1f kWh) no reconcilia con el estimado a partir de voltaje x corriente x factor de potencia (%.1f kWh) mientras el consumo se mantiene cerca de su baseline; se detectaron %d lecturas inconsistentes/intermitentes.",
			a.Evidence.ObservedKWh, a.Evidence.ExpectedKWhFromElectrical, a.Evidence.ConsecutiveHours,
		)
		a.RecommendedAction = "Validar el medidor y su instrumentación antes de confiar en sus lecturas."

	default:
		a.Reason = "Sin evidencia suficiente para explicar este caso."
		a.RecommendedAction = "Revisar manualmente."
	}
}
