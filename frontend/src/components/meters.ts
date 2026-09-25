// Pure helpers behind the Medidores table: turning (meters, anomalies) into
// display rows, then filtering and sorting them. Kept free of React/DOM so
// it's cheap to unit-test in full.
import type { Anomaly, AnomalyType, Meter, Severity } from '../types'

export interface MeterRow {
  id: string
  anomalyCount: number
  lastType: AnomalyType | null
  lastSeverity: Severity | null
  lastDetectedAt: string | null
}

export type StatusFilter = 'all' | 'anomaly' | 'normal'
export type SortKey = 'id' | 'anomalyCount' | 'severity' | 'lastDetectedAt'
export type SortDirection = 'asc' | 'desc'

const SEVERITY_RANK: Record<Severity, number> = { LOW: 1, MEDIUM: 2, HIGH: 3 }

// Which of two anomalies for the same meter should represent it in the
// table: the more severe one wins, ties go to whichever fired more recently.
export function isWorseAnomaly(a: Anomaly, b: Anomaly): boolean {
  const rankA = SEVERITY_RANK[a.severity]
  const rankB = SEVERITY_RANK[b.severity]
  if (rankA !== rankB) return rankA > rankB
  return new Date(a.detected_at).getTime() > new Date(b.detected_at).getTime()
}

export function buildMeterRows(meters: Meter[], anomalies: Anomaly[]): MeterRow[] {
  return meters.map((meter) => {
    const meterAnomalies = anomalies.filter((a) => a.meter_id === meter.id)
    const worst = meterAnomalies.reduce<Anomaly | null>(
      (acc, a) => (acc === null || isWorseAnomaly(a, acc) ? a : acc),
      null,
    )
    return {
      id: meter.id,
      anomalyCount: meterAnomalies.length,
      lastType: worst?.type ?? null,
      lastSeverity: worst?.severity ?? null,
      lastDetectedAt: worst?.detected_at ?? null,
    }
  })
}

export function filterMeterRows(rows: MeterRow[], query: string, status: StatusFilter): MeterRow[] {
  const normalizedQuery = query.trim().toUpperCase()
  return rows.filter((row) => {
    if (normalizedQuery && !row.id.toUpperCase().includes(normalizedQuery)) return false
    if (status === 'anomaly' && row.anomalyCount === 0) return false
    if (status === 'normal' && row.anomalyCount > 0) return false
    return true
  })
}

function compareRows(a: MeterRow, b: MeterRow, sortBy: SortKey): number {
  switch (sortBy) {
    case 'id':
      return a.id.localeCompare(b.id, undefined, { numeric: true })
    case 'anomalyCount':
      return a.anomalyCount - b.anomalyCount
    case 'severity':
      return (a.lastSeverity ? SEVERITY_RANK[a.lastSeverity] : 0) - (b.lastSeverity ? SEVERITY_RANK[b.lastSeverity] : 0)
    case 'lastDetectedAt':
      return (a.lastDetectedAt ? new Date(a.lastDetectedAt).getTime() : 0) -
        (b.lastDetectedAt ? new Date(b.lastDetectedAt).getTime() : 0)
    default:
      return 0
  }
}

export function sortMeterRows(rows: MeterRow[], sortBy: SortKey, direction: SortDirection): MeterRow[] {
  const sorted = [...rows].sort((a, b) => compareRows(a, b, sortBy))
  return direction === 'asc' ? sorted : sorted.reverse()
}
