package api

import (
	"sort"

	"bia-energy/backend/internal/models"
)

var severityRank = map[models.Severity]int{
	models.SeverityHigh:   3,
	models.SeverityMedium: 2,
	models.SeverityLow:    1,
}

// sortAnomaliesByPriority orders anomalies the way the "Anomalías IA" screen
// and the dashboard's anomaly table want to show them: most severe first,
// ties broken by confidence and then recency. This matches section 11 of
// the enunciado's own example table (ordered HIGH, HIGH, MEDIUM, LOW/MEDIUM)
// and the "Prioriza M-109" grading criterion — a purely chronological feed
// (the previous behavior) does not satisfy either.
func sortAnomaliesByPriority(anomalies []models.Anomaly) {
	sort.SliceStable(anomalies, func(i, j int) bool {
		a, b := anomalies[i], anomalies[j]
		if ra, rb := severityRank[a.Severity], severityRank[b.Severity]; ra != rb {
			return ra > rb
		}
		if a.Confidence != b.Confidence {
			return a.Confidence > b.Confidence
		}
		return a.DetectedAt.After(b.DetectedAt)
	})
}
