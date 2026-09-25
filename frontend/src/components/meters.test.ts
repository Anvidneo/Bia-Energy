import { describe, expect, it } from 'vitest'
import type { Anomaly, Meter, Reading } from '../types'
import { buildMeterRows, filterMeterRows, sortMeterRows } from './meters'

function meter(id: string): Meter {
  return { id }
}

function anomaly(overrides: Partial<Anomaly> & { meter_id: string }): Anomaly {
  return {
    id: Math.floor(Math.random() * 100000),
    detected_at: '2026-09-01T00:00:00Z',
    anomaly: true,
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    confidence: 0.9,
    reason: 'test',
    recommended_action: 'test',
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

const METERS: Meter[] = [meter('M-101'), meter('M-102'), meter('M-103')]

describe('buildMeterRows', () => {
  it('reports zero anomalies for a meter with none', () => {
    const rows = buildMeterRows(METERS, [])
    expect(rows).toEqual([
      { id: 'M-101', anomalyCount: 0, lastType: null, lastSeverity: null, lastDetectedAt: null, consumptionKwh: null, variationPct: null },
      { id: 'M-102', anomalyCount: 0, lastType: null, lastSeverity: null, lastDetectedAt: null, consumptionKwh: null, variationPct: null },
      { id: 'M-103', anomalyCount: 0, lastType: null, lastSeverity: null, lastDetectedAt: null, consumptionKwh: null, variationPct: null },
    ])
  })

  it('defaults consumptionKwh/variationPct to null when no readingsByMeter map is passed', () => {
    const [row] = buildMeterRows([meter('M-101')], [])
    expect(row.consumptionKwh).toBe(null)
    expect(row.variationPct).toBe(null)
  })

  it('fills consumptionKwh/variationPct from the readingsByMeter map when provided', () => {
    const readings: Reading[] = [
      { meter_id: 'M-101', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 8, voltage_v: 220, current_a: 5, power_factor: 0.95 },
      { meter_id: 'M-101', timestamp: '2026-09-01T01:00:00Z', consumption_kwh: 12, voltage_v: 220, current_a: 5, power_factor: 0.95 },
    ]
    const [row] = buildMeterRows([meter('M-101')], [], { 'M-101': readings })
    expect(row.consumptionKwh).toBe(12) // latest reading
    expect(row.variationPct).toBeCloseTo(20) // vs. median(8, 12) = 10
  })

  it('prefers the worst anomaly\'s baseline over the readings median for variationPct', () => {
    const readings: Reading[] = [
      { meter_id: 'M-101', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 110, voltage_v: 220, current_a: 5, power_factor: 0.95 },
    ]
    const anomalies = [anomaly({ meter_id: 'M-101', evidence: { ...anomaly({ meter_id: 'M-101' }).evidence, baseline_median_kwh: 55 } })]
    const [row] = buildMeterRows([meter('M-101')], anomalies, { 'M-101': readings })
    expect(row.consumptionKwh).toBe(110)
    expect(row.variationPct).toBeCloseTo(100) // (110-55)/55 * 100
  })

  it('picks the highest-severity anomaly to represent the meter', () => {
    const anomalies = [
      anomaly({ meter_id: 'M-101', severity: 'LOW', type: 'FALSE_POSITIVE', detected_at: '2026-09-02T00:00:00Z' }),
      anomaly({ meter_id: 'M-101', severity: 'HIGH', type: 'REAL_ANOMALY', detected_at: '2026-09-01T00:00:00Z' }),
    ]
    const [row] = buildMeterRows([meter('M-101')], anomalies)
    expect(row.anomalyCount).toBe(2)
    expect(row.lastSeverity).toBe('HIGH')
    expect(row.lastType).toBe('REAL_ANOMALY')
  })

  it('breaks a same-severity tie by the most recent detection', () => {
    const anomalies = [
      anomaly({ meter_id: 'M-101', severity: 'MEDIUM', type: 'EXPLAINABLE_ANOMALY', detected_at: '2026-09-01T00:00:00Z' }),
      anomaly({ meter_id: 'M-101', severity: 'MEDIUM', type: 'DATA_QUALITY', detected_at: '2026-09-05T00:00:00Z' }),
    ]
    const [row] = buildMeterRows([meter('M-101')], anomalies)
    expect(row.lastType).toBe('DATA_QUALITY')
    expect(row.lastDetectedAt).toBe('2026-09-05T00:00:00Z')
  })

  it('only counts anomalies belonging to that meter', () => {
    const anomalies = [anomaly({ meter_id: 'M-102' })]
    const rows = buildMeterRows(METERS, anomalies)
    expect(rows.find((r) => r.id === 'M-101')?.anomalyCount).toBe(0)
    expect(rows.find((r) => r.id === 'M-102')?.anomalyCount).toBe(1)
  })
})

describe('filterMeterRows', () => {
  const rows = buildMeterRows(METERS, [anomaly({ meter_id: 'M-102' })])

  it('matches by id substring, case-insensitively', () => {
    expect(filterMeterRows(rows, '102', 'all').map((r) => r.id)).toEqual(['M-102'])
    expect(filterMeterRows(rows, 'm-102', 'all').map((r) => r.id)).toEqual(['M-102'])
  })

  it('returns everything for an empty query', () => {
    expect(filterMeterRows(rows, '', 'all')).toHaveLength(3)
  })

  it('filters to only meters with anomalies', () => {
    expect(filterMeterRows(rows, '', 'anomaly').map((r) => r.id)).toEqual(['M-102'])
  })

  it('filters to only meters without anomalies', () => {
    expect(filterMeterRows(rows, '', 'normal').map((r) => r.id)).toEqual(['M-101', 'M-103'])
  })

  it('combines a query with a status filter', () => {
    expect(filterMeterRows(rows, '10', 'normal').map((r) => r.id)).toEqual(['M-101', 'M-103'])
    expect(filterMeterRows(rows, '102', 'normal')).toEqual([])
  })
})

describe('sortMeterRows', () => {
  const rows = buildMeterRows(
    [meter('M-103'), meter('M-101'), meter('M-102')],
    [
      anomaly({ meter_id: 'M-101', severity: 'LOW', detected_at: '2026-09-01T00:00:00Z' }),
      anomaly({ meter_id: 'M-101', severity: 'LOW', detected_at: '2026-09-03T00:00:00Z' }),
      anomaly({ meter_id: 'M-102', severity: 'HIGH', detected_at: '2026-09-02T00:00:00Z' }),
    ],
    {
      'M-101': [{ meter_id: 'M-101', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 30, voltage_v: 220, current_a: 5, power_factor: 0.95 }],
      'M-102': [{ meter_id: 'M-102', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 10, voltage_v: 220, current_a: 5, power_factor: 0.95 }],
      'M-103': [{ meter_id: 'M-103', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 20, voltage_v: 220, current_a: 5, power_factor: 0.95 }],
    },
  )

  it('sorts by id, numerically aware, ascending and descending', () => {
    expect(sortMeterRows(rows, 'id', 'asc').map((r) => r.id)).toEqual(['M-101', 'M-102', 'M-103'])
    expect(sortMeterRows(rows, 'id', 'desc').map((r) => r.id)).toEqual(['M-103', 'M-102', 'M-101'])
  })

  it('sorts by anomaly count', () => {
    expect(sortMeterRows(rows, 'anomalyCount', 'desc').map((r) => r.id)).toEqual(['M-101', 'M-102', 'M-103'])
  })

  it('sorts by severity, treating "no anomaly" as the lowest rank', () => {
    expect(sortMeterRows(rows, 'severity', 'desc').map((r) => r.id)).toEqual(['M-102', 'M-101', 'M-103'])
  })

  it('sorts by last detection time, treating "never" as the earliest', () => {
    expect(sortMeterRows(rows, 'lastDetectedAt', 'desc').map((r) => r.id)).toEqual(['M-101', 'M-102', 'M-103'])
  })

  it('does not mutate the input array', () => {
    const copy = [...rows]
    sortMeterRows(rows, 'id', 'desc')
    expect(rows).toEqual(copy)
  })

  it('falls back to a stable no-op comparison for an unrecognized sort key', () => {
    const before = rows.map((r) => r.id)
    // @ts-expect-error exercising the switch's defensive default branch
    expect(sortMeterRows(rows, 'bogus', 'asc').map((r) => r.id)).toEqual(before)
  })

  it('sorts by consumption', () => {
    // M-101=30, M-102=10, M-103=20 (see the readingsByMeter map above)
    expect(sortMeterRows(rows, 'consumption', 'asc').map((r) => r.id)).toEqual(['M-102', 'M-103', 'M-101'])
    expect(sortMeterRows(rows, 'consumption', 'desc').map((r) => r.id)).toEqual(['M-101', 'M-103', 'M-102'])
  })

  it('sorts by variation, treating a meter with no baseline as 0', () => {
    // M-101 has anomalies (baseline from evidence: 100) -> consumo 30 => variación -70%
    // M-102 has an anomaly too (baseline 100) -> consumo 10 => variación -90%
    // M-103 has no anomaly -> baseline = median of its own single reading (20) => variación 0%
    expect(sortMeterRows(rows, 'variation', 'asc').map((r) => r.id)).toEqual(['M-102', 'M-101', 'M-103'])
  })

  it('treats a meter with no consumo/variación data (no readings at all) as 0 when sorting', () => {
    // M-101 has readings (consumo 30, variación -70%); M-102 has none at all
    // (absent from the readingsByMeter map) -> both null -> compared as 0.
    const partialRows = buildMeterRows([meter('M-101'), meter('M-102')], [], {
      'M-101': [{ meter_id: 'M-101', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 30, voltage_v: 220, current_a: 5, power_factor: 0.95 }],
    })
    expect(sortMeterRows(partialRows, 'consumption', 'asc').map((r) => r.id)).toEqual(['M-102', 'M-101'])
    // Variación: M-101 has a single reading with no anomaly, so baseline =
    // median = itself => 0%; M-102 has no data at all => null => also
    // treated as 0. Both compare equal, so the stable sort keeps input order.
    expect(sortMeterRows(partialRows, 'variation', 'asc').map((r) => r.id)).toEqual(['M-101', 'M-102'])
  })
})
