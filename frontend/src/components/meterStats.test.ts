import { describe, expect, it } from 'vitest'
import { computeMeterStats, medianConsumption } from './meterStats'
import type { Anomaly, Reading } from '../types'

function makeReading(overrides: Partial<Reading> = {}): Reading {
  return {
    meter_id: 'M-109',
    timestamp: '2026-09-01T00:00:00Z',
    consumption_kwh: 10,
    voltage_v: 220,
    current_a: 5,
    power_factor: 0.95,
    ...overrides,
  }
}

function makeAnomaly(overrides: Partial<Anomaly> = {}): Anomaly {
  return {
    id: 1,
    meter_id: 'M-109',
    detected_at: '2026-09-01T00:00:00Z',
    anomaly: true,
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    confidence: 0.9,
    reason: 'Consumo muy por encima del baseline',
    recommended_action: 'Revisar línea productiva',
    evidence: {
      baseline_median_kwh: 100,
      observed_kwh: 200,
      deviation_pct: 100,
      z_score: 3,
      consecutive_hours: 4,
      expected_kwh_from_electrical: 190,
    },
    ...overrides,
  }
}

describe('medianConsumption', () => {
  it('returns null for an empty list', () => {
    expect(medianConsumption([])).toBe(null)
  })

  it('returns the single value for one reading', () => {
    expect(medianConsumption([makeReading({ consumption_kwh: 7 })])).toBe(7)
  })

  it('returns the middle value for an odd count', () => {
    const readings = [1, 5, 3].map((consumption_kwh) => makeReading({ consumption_kwh }))
    expect(medianConsumption(readings)).toBe(3)
  })

  it('averages the two middle values for an even count', () => {
    const readings = [1, 2, 3, 4].map((consumption_kwh) => makeReading({ consumption_kwh }))
    expect(medianConsumption(readings)).toBe(2.5)
  })
})

describe('computeMeterStats', () => {
  it('returns all nulls (and no worst anomaly) when there is no data at all', () => {
    const stats = computeMeterStats([], [])
    expect(stats).toEqual({
      currentConsumptionKwh: null,
      baselineKwh: null,
      variationPct: null,
      voltageV: null,
      currentA: null,
      powerFactor: null,
      worstAnomaly: null,
    })
  })

  it('uses the median of readings as baseline when the meter has no anomalies', () => {
    const readings = [
      makeReading({ timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 8 }),
      makeReading({ timestamp: '2026-09-01T01:00:00Z', consumption_kwh: 12 }),
    ]
    const stats = computeMeterStats(readings, [])
    expect(stats.currentConsumptionKwh).toBe(12) // latest reading
    expect(stats.baselineKwh).toBe(10) // median(8, 12)
    expect(stats.variationPct).toBeCloseTo(20) // (12-10)/10 * 100
    expect(stats.voltageV).toBe(220)
    expect(stats.currentA).toBe(5)
    expect(stats.powerFactor).toBe(0.95)
    expect(stats.worstAnomaly).toBe(null)
  })

  it('prefers the worst anomaly\'s baseline_median_kwh over the readings median', () => {
    const readings = [makeReading({ consumption_kwh: 110 })]
    const anomaly = makeAnomaly({ evidence: { ...makeAnomaly().evidence, baseline_median_kwh: 52.6 } })
    const stats = computeMeterStats(readings, [anomaly])
    expect(stats.baselineKwh).toBe(52.6)
    expect(stats.worstAnomaly).toBe(anomaly)
  })

  it('picks the most severe anomaly among several as the worst one', () => {
    const low = makeAnomaly({ id: 1, severity: 'LOW', detected_at: '2026-09-01T02:00:00Z' })
    const high = makeAnomaly({ id: 2, severity: 'HIGH', detected_at: '2026-09-01T00:00:00Z' })
    const stats = computeMeterStats([makeReading()], [low, high])
    expect(stats.worstAnomaly?.id).toBe(2)
  })

  it('keeps the accumulated worst anomaly when a later one in the list is not worse', () => {
    const high = makeAnomaly({ id: 1, severity: 'HIGH', detected_at: '2026-09-01T00:00:00Z' })
    const low = makeAnomaly({ id: 2, severity: 'LOW', detected_at: '2026-09-01T01:00:00Z' })
    const medium = makeAnomaly({ id: 3, severity: 'MEDIUM', detected_at: '2026-09-01T02:00:00Z' })
    const stats = computeMeterStats([makeReading()], [high, low, medium])
    expect(stats.worstAnomaly?.id).toBe(1)
  })

  it('does not divide by zero when the baseline is 0', () => {
    const anomaly = makeAnomaly({ evidence: { ...makeAnomaly().evidence, baseline_median_kwh: 0 } })
    const stats = computeMeterStats([makeReading({ consumption_kwh: 5 })], [anomaly])
    expect(stats.baselineKwh).toBe(0)
    expect(stats.variationPct).toBe(null)
  })
})
