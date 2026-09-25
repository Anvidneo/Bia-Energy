import { describe, expect, it } from 'vitest'
import type { Anomaly, Meter } from '../types'
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
      { id: 'M-101', anomalyCount: 0, lastType: null, lastSeverity: null, lastDetectedAt: null },
      { id: 'M-102', anomalyCount: 0, lastType: null, lastSeverity: null, lastDetectedAt: null },
      { id: 'M-103', anomalyCount: 0, lastType: null, lastSeverity: null, lastDetectedAt: null },
    ])
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
})
