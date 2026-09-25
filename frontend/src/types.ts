// Shared TypeScript mirror of the Go structs in backend/internal/models.
// Field names match the JSON tags exactly (snake_case, as the Go backend
// encodes them) so no mapping layer is needed between fetch and render.

export interface Meter {
  id: string
}

export interface Reading {
  meter_id: string
  timestamp: string // RFC3339
  consumption_kwh: number
  voltage_v: number
  current_a: number
  power_factor: number
}

export interface AppEvent {
  meter_id: string
  event_timestamp: string
  event_type: 'OPERATIONAL_CHANGE' | 'SCHEDULED_OUTAGE' | 'DATA_QUALITY' | 'UNKNOWN' | string
  description: string
}

export type AnomalyType =
  | 'NORMAL'
  | 'REAL_ANOMALY'
  | 'FALSE_POSITIVE'
  | 'EXPLAINABLE_ANOMALY'
  | 'DATA_QUALITY'

export type Severity = 'HIGH' | 'MEDIUM' | 'LOW'

export interface Evidence {
  baseline_median_kwh: number
  observed_kwh: number
  deviation_pct: number
  z_score: number
  consecutive_hours: number
  expected_kwh_from_electrical: number
}

export interface Anomaly {
  id: number
  meter_id: string
  detected_at: string
  anomaly: boolean
  type: AnomalyType
  severity: Severity
  confidence: number
  reason: string
  recommended_action: string
  related_event_type?: string
  related_event_at?: string
  evidence: Evidence
}

export type AnalysisStatus = 'pending' | 'processing' | 'done' | 'error'

export interface AnalysisResult {
  id: string
  status: AnalysisStatus
  stage: string
  started_at: string
  finished_at?: string
  anomalies?: Anomaly[]
  error?: string
}

export interface DashboardSummary {
  total_meters: number
  active_anomalies: number
  last_analysis_at?: string
}

// Mirrors backend/internal/api/ai.go's pipelineStages — shown as the visual
// pipeline while an analysis run is in progress.
export const PIPELINE_STAGES = [
  'Lecturas',
  'Baseline',
  'Deteccion',
  'Correlacion',
  'Eventos',
  'Explicacion',
  'Recomendacion',
] as const
