// Shared TypeScript mirror of the Go structs in backend/internal/models.
// Fields are placeholders for today's skeleton — to be filled in once the
// Go structs are defined (Friday) so the two stay in sync.

export interface Meter {
  id: string
  // TODO: fill in once internal/models.Meter is defined (Friday)
}

export interface Reading {
  meterId: string
  timestamp: string
  // TODO: fill in once internal/models.Reading is defined (Friday)
}

export interface AnomalyEvent {
  id: string
  // TODO: fill in once internal/models.Event is defined (Friday)
}

export interface Anomaly {
  id: string
  meterId: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  confidence: number
  reason: string
  recommendedAction: string
  // TODO: fill in once internal/models.Anomaly is defined (Friday)
}

export interface AnalysisResult {
  id: string
  status: 'pending' | 'processing' | 'done' | 'error'
  anomalies: Anomaly[]
  // TODO: fill in once GET /ai/analysis/:id shape is finalized (Friday)
}
