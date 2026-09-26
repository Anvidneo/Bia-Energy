package api

import (
	"context"
	"log"

	"bia-energy/backend/internal/models"
)

// CriticalAlertPublisher pushes a HIGH-severity anomaly to an external
// alerting channel (firebase.Publisher in production; a fake in tests).
// It is small and defined here — not in package firebase — so this
// package doesn't need to import firebase's SDK dependencies at all;
// firebase.Publisher satisfies this interface structurally.
type CriticalAlertPublisher interface {
	PublishCriticalAlert(ctx context.Context, a models.Anomaly, analysisID string) error
}

// publishCriticalAlerts fires PublishCriticalAlert for every HIGH-severity
// anomaly in the batch. d.AlertPublisher is nil unless USE_FIREBASE_ALERTS
// is on, so this is a no-op by default. Any error is logged and dropped —
// per the design doc, Firebase must never fail an analysis run or its HTTP
// response.
func (d *Deps) publishCriticalAlerts(ctx context.Context, analysisID string, anomalies []models.Anomaly) {
	if d.AlertPublisher == nil {
		return
	}
	for _, a := range anomalies {
		if a.Severity != models.SeverityHigh {
			continue
		}
		if err := d.AlertPublisher.PublishCriticalAlert(ctx, a, analysisID); err != nil {
			log.Printf("firebase: publishing critical alert for %s: %v", a.MeterID, err)
		}
	}
}
