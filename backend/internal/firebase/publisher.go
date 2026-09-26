// Package firebase is an optional, isolated client that publishes
// critical-severity anomaly alerts to Firestore. Any error here must never
// interrupt the main request flow — internal/api swallows and logs
// whatever PublishCriticalAlert returns instead of failing an analysis run.
package firebase

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"bia-energy/backend/internal/models"
)

// criticalAlertsCollection is the Firestore collection every HIGH-severity
// anomaly is written to.
const criticalAlertsCollection = "critical_alerts"

// Publisher writes critical-severity anomaly alerts to Firestore using the
// official Firebase Admin SDK (rather than hand-signing OAuth JWTs).
type Publisher struct {
	client *firestore.Client
}

// NewPublisher builds a Publisher for the given Firebase project, using a
// service account's JSON key (the raw contents of FIREBASE_CREDENTIALS_JSON,
// never a file path — nothing here reads credentials from disk).
func NewPublisher(ctx context.Context, projectID, credentialsJSON string) (*Publisher, error) {
	app, err := firebase.NewApp(ctx,
		&firebase.Config{ProjectID: projectID},
		option.WithCredentialsJSON([]byte(credentialsJSON)),
	)
	if err != nil {
		return nil, fmt.Errorf("firebase: initializing app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase: creating firestore client: %w", err)
	}

	return &Publisher{client: client}, nil
}

// Close releases the underlying Firestore client's connections. Safe to
// call on a nil Publisher.
func (p *Publisher) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}

// PublishCriticalAlert writes one anomaly to the critical_alerts
// collection, keyed by its database id so re-running an analysis
// overwrites the same alert instead of duplicating it. The caller
// (internal/api) is responsible for only calling this for
// Severity == HIGH and for swallowing the returned error.
func (p *Publisher) PublishCriticalAlert(ctx context.Context, a models.Anomaly, analysisID string) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("firebase: publisher not initialized")
	}

	docID := alertDocID(a)
	_, err := p.client.Collection(criticalAlertsCollection).Doc(docID).Set(ctx, alertDocument(a, analysisID))
	if err != nil {
		return fmt.Errorf("firebase: publishing alert for %s: %w", a.MeterID, err)
	}
	return nil
}

// alertDocID keys a Firestore document by the anomaly's database id, so
// re-running an analysis overwrites the same alert instead of duplicating
// it. Factored out (and alertDocument below) purely so this mapping is
// unit-testable without a real Firestore client.
func alertDocID(a models.Anomaly) string {
	return fmt.Sprintf("%d", a.ID)
}

// alertDocument builds the Firestore document fields for one critical
// alert, per the design doc: meter_id, type, severity, confidence, reason,
// recommended_action, detected_at and analysis_id, plus a published_at
// timestamp for observability.
func alertDocument(a models.Anomaly, analysisID string) map[string]interface{} {
	return map[string]interface{}{
		"meter_id":           a.MeterID,
		"type":               string(a.Type),
		"severity":           string(a.Severity),
		"confidence":         a.Confidence,
		"reason":             a.Reason,
		"recommended_action": a.RecommendedAction,
		"detected_at":        a.DetectedAt,
		"analysis_id":        analysisID,
		"published_at":       time.Now().UTC(),
	}
}
