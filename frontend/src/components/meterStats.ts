// Pure helpers behind the "Detalle de medidor" summary card: consumo
// actual, baseline, variación, y las variables eléctricas (voltaje,
// corriente, factor de potencia) que pide la sección 7 de la prueba
// técnica. Kept free of React/DOM so it's cheap to unit-test in full.
import type { Anomaly, Reading } from '../types'
import { isWorseAnomaly } from './meters'

export interface MeterStats {
  currentConsumptionKwh: number | null
  baselineKwh: number | null
  variationPct: number | null
  voltageV: number | null
  currentA: number | null
  powerFactor: number | null
  worstAnomaly: Anomaly | null
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

function worstOf(anomalies: Anomaly[]): Anomaly | null {
  return anomalies.reduce<Anomaly | null>(
    (acc, a) => (acc === null || isWorseAnomaly(a, acc) ? a : acc),
    null,
  )
}

// readings is assumed sorted ascending by timestamp (as the API returns
// them), so the last entry is the most recent reading.
export function computeMeterStats(readings: Reading[], anomalies: Anomaly[]): MeterStats {
  const latest = readings.length > 0 ? readings[readings.length - 1] : null
  const worstAnomaly = worstOf(anomalies)

  const currentConsumptionKwh = latest ? latest.consumption_kwh : null
  const baselineKwh = worstAnomaly ? worstAnomaly.evidence.baseline_median_kwh : medianConsumption(readings)

  const variationPct =
    currentConsumptionKwh !== null && baselineKwh !== null && baselineKwh !== 0
      ? ((currentConsumptionKwh - baselineKwh) / baselineKwh) * 100
      : null

  return {
    currentConsumptionKwh,
    baselineKwh,
    variationPct,
    voltageV: latest ? latest.voltage_v : null,
    currentA: latest ? latest.current_a : null,
    powerFactor: latest ? latest.power_factor : null,
    worstAnomaly,
  }
}
