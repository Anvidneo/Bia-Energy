// Pure helpers behind the Medidores table: turning (meters, anomalies,
// readings) into display rows, then filtering and sorting them. Kept free
// of React/DOM so it's cheap to unit-test in full.
import type { Anomaly, AnomalyType, Meter, Reading, Severity } from '../types'

export interface MeterRow {
  id: string
  anomalyCount: number
  lastType: AnomalyType | null
  lastSeverity: Severity | null
  lastDetectedAt: string | null
  // Sección 6 de la prueba ("Gestión de medidores"): Consumo y Variación
  // son columnas de la tabla de ejemplo. null mientras las lecturas del
  // medidor todavía no han llegado (o si no hay ninguna).
  consumptionKwh: number | null
  variationPct: number | null
}

export type StatusFilter = 'all' | 'anomaly' | 'normal'
export type SortKey = 'id' | 'anomalyCount' | 'severity' | 'lastDetectedAt' | 'consumption' | 'variation'
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

// Median consumption across all available readings — used as a fallback
// baseline when the meter has no anomaly (and therefore no
// evidence.baseline_median_kwh from the real detection engine to borrow).
export function medianConsumption(readings: Reading[]): number | null {
  if (readings.length === 0) return null
  const sorted = [...readings.map((r) => r.consumption_kwh)].sort((a, b) => a - b)
  const mid = Math.floor(sorted.length / 2)
  return sorted.length % 2 !== 0 ? sorted[mid] : (sorted[mid - 1] + sorted[mid]) / 2
}

export interface ConsumptionSummary {
  currentConsumptionKwh: number | null
  baselineKwh: number | null
  variationPct: number | null
}

// readings is assumed sorted ascending by timestamp (as the API returns
// them), so the last entry is the most recent reading. Shared by the
// Medidores table (per-row Consumo/Variación) and the meter detail summary
// card (see meterStats.ts) so both agree on the same numbers.
export function computeConsumptionSummary(readings: Reading[], worstAnomaly: Anomaly | null): ConsumptionSummary {
  const latest = readings.length > 0 ? readings[readings.length - 1] : null
  const currentConsumptionKwh = latest ? latest.consumption_kwh : null
  const baselineKwh = worstAnomaly ? worstAnomaly.evidence.baseline_median_kwh : medianConsumption(readings)
  const variationPct =
    currentConsumptionKwh !== null && baselineKwh !== null && baselineKwh !== 0
      ? ((currentConsumptionKwh - baselineKwh) / baselineKwh) * 100
      : null
  return { currentConsumptionKwh, baselineKwh, variationPct }
}

export function buildMeterRows(
  meters: Meter[],
  anomalies: Anomaly[],
  readingsByMeter: Record<string, Reading[]> = {},
): MeterRow[] {
  return meters.map((meter) => {
    const meterAnomalies = anomalies.filter((a) => a.meter_id === meter.id)
    const worst = meterAnomalies.reduce<Anomaly | null>(
      (acc, a) => (acc === null || isWorseAnomaly(a, acc) ? a : acc),
      null,
    )
    const summary = computeConsumptionSummary(readingsByMeter[meter.id] ?? [], worst)
    return {
      id: meter.id,
      anomalyCount: meterAnomalies.length,
      lastType: worst?.type ?? null,
      lastSeverity: worst?.severity ?? null,
      lastDetectedAt: worst?.detected_at ?? null,
      consumptionKwh: summary.currentConsumptionKwh,
      variationPct: summary.variationPct,
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
    case 'consumption':
      return (a.consumptionKwh ?? 0) - (b.consumptionKwh ?? 0)
    case 'variation':
      return (a.variationPct ?? 0) - (b.variationPct ?? 0)
    default:
      return 0
  }
}

export function sortMeterRows(rows: MeterRow[], sortBy: SortKey, direction: SortDirection): MeterRow[] {
  const sorted = [...rows].sort((a, b) => compareRows(a, b, sortBy))
  return direction === 'asc' ? sorted : sorted.reverse()
}
