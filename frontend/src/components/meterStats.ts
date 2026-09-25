// Pure helpers behind the "Detalle de medidor" summary card: consumo
// actual, baseline, variación, y las variables eléctricas (voltaje,
// corriente, factor de potencia) que pide la sección 7 de la prueba
// técnica. Kept free of React/DOM so it's cheap to unit-test in full.
//
// The consumo/baseline/variación math itself lives in meters.ts
// (computeConsumptionSummary) so the Medidores table and this detail card
// can never disagree on the numbers — re-exported here for convenience and
// so existing imports/tests don't need to know it moved.
import type { Anomaly, Reading } from '../types'
import { isWorseAnomaly, computeConsumptionSummary, medianConsumption } from './meters'

export { medianConsumption, computeConsumptionSummary }

export interface MeterStats {
  currentConsumptionKwh: number | null
  baselineKwh: number | null
  variationPct: number | null
  voltageV: number | null
  currentA: number | null
  powerFactor: number | null
  worstAnomaly: Anomaly | null
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
  const summary = computeConsumptionSummary(readings, worstAnomaly)

  return {
    ...summary,
    voltageV: latest ? latest.voltage_v : null,
    currentA: latest ? latest.current_a : null,
    powerFactor: latest ? latest.power_factor : null,
    worstAnomaly,
  }
}
