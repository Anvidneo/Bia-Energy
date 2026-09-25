import { describe, expect, it } from 'vitest'
import type { Anomaly } from '../types'
import {
  pillClass, typeLabel, severityLabel, dotColorVar, tintColorVar,
  formatTime, formatRelative, meterStatusBadge,
} from './severity'

type Classifiable = Pick<Anomaly, 'type' | 'severity'>

const FALSE_POSITIVE: Classifiable = { type: 'FALSE_POSITIVE', severity: 'LOW' }
const DATA_QUALITY: Classifiable = { type: 'DATA_QUALITY', severity: 'HIGH' }
const REAL_HIGH: Classifiable = { type: 'REAL_ANOMALY', severity: 'HIGH' }
const REAL_MEDIUM: Classifiable = { type: 'REAL_ANOMALY', severity: 'MEDIUM' }
const REAL_LOW: Classifiable = { type: 'REAL_ANOMALY', severity: 'LOW' }

describe('pillClass', () => {
  it('treats a false positive as resolved/good regardless of severity', () => {
    expect(pillClass(FALSE_POSITIVE)).toBe('pill pill-good')
  })
  it('treats a data-quality issue as high, regardless of severity', () => {
    expect(pillClass(DATA_QUALITY)).toBe('pill pill-high')
  })
  it('maps HIGH severity to the critical pill', () => {
    expect(pillClass(REAL_HIGH)).toBe('pill pill-crit')
  })
  it('maps MEDIUM severity to the medium pill', () => {
    expect(pillClass(REAL_MEDIUM)).toBe('pill pill-med')
  })
  it('falls back to good for LOW severity', () => {
    expect(pillClass(REAL_LOW)).toBe('pill pill-good')
  })
})

describe('typeLabel', () => {
  it('labels every known anomaly type in Spanish', () => {
    expect(typeLabel('REAL_ANOMALY')).toBe('Anomalía real')
    expect(typeLabel('FALSE_POSITIVE')).toBe('Falso positivo')
    expect(typeLabel('EXPLAINABLE_ANOMALY')).toBe('Explicada por evento')
    expect(typeLabel('DATA_QUALITY')).toBe('Calidad de datos')
  })
  it('falls back to the raw type for an unmapped value', () => {
    expect(typeLabel('NORMAL')).toBe('NORMAL')
  })
})

describe('severityLabel', () => {
  it('labels every severity in Spanish', () => {
    expect(severityLabel('HIGH')).toBe('Alta')
    expect(severityLabel('MEDIUM')).toBe('Media')
    expect(severityLabel('LOW')).toBe('Baja')
  })
  it('falls back to the raw value for an unrecognized severity', () => {
    // @ts-expect-error exercising the switch's defensive default branch
    expect(severityLabel('UNKNOWN')).toBe('UNKNOWN')
  })
})

describe('dotColorVar / tintColorVar', () => {
  it('mirror pillClass\'s classification for the dot color', () => {
    expect(dotColorVar(FALSE_POSITIVE)).toBe('var(--good)')
    expect(dotColorVar(DATA_QUALITY)).toBe('var(--high)')
    expect(dotColorVar(REAL_HIGH)).toBe('var(--crit)')
    expect(dotColorVar(REAL_MEDIUM)).toBe('var(--med)')
    expect(dotColorVar(REAL_LOW)).toBe('var(--good)')
  })
  it('mirror pillClass\'s classification for the tint color', () => {
    expect(tintColorVar(FALSE_POSITIVE)).toBe('var(--good-tint)')
    expect(tintColorVar(DATA_QUALITY)).toBe('var(--high-tint)')
    expect(tintColorVar(REAL_HIGH)).toBe('var(--crit-tint)')
    expect(tintColorVar(REAL_MEDIUM)).toBe('var(--med-tint)')
    expect(tintColorVar(REAL_LOW)).toBe('var(--good-tint)')
  })
})

describe('formatTime', () => {
  it('formats an ISO timestamp as a non-empty display string', () => {
    const result = formatTime('2026-09-01T12:30:00Z')
    expect(typeof result).toBe('string')
    expect(result.length).toBeGreaterThan(0)
  })
})

describe('formatRelative', () => {
  it('reports "just now" for a timestamp seconds ago', () => {
    const iso = new Date(Date.now() - 5_000).toISOString()
    expect(formatRelative(iso)).toBe('Justo ahora')
  })
  it('reports minutes for a timestamp under an hour ago', () => {
    const iso = new Date(Date.now() - 5 * 60_000).toISOString()
    expect(formatRelative(iso)).toBe('Hace 5 min')
  })
  it('reports hours for a timestamp under a day ago', () => {
    const iso = new Date(Date.now() - 3 * 3_600_000).toISOString()
    expect(formatRelative(iso)).toBe('Hace 3 h')
  })
  it('reports days for a timestamp a day or more ago', () => {
    const iso = new Date(Date.now() - 2 * 86_400_000).toISOString()
    expect(formatRelative(iso)).toBe('Hace 2 d')
  })
})

describe('meterStatusBadge', () => {
  it('returns the good/Normal badge when there is no anomaly', () => {
    expect(meterStatusBadge(null)).toEqual({ className: 'pill pill-good', label: 'Normal' })
  })
  it('delegates to pillClass/typeLabel for a real anomaly', () => {
    expect(meterStatusBadge(REAL_HIGH)).toEqual({ className: 'pill pill-crit', label: 'Anomalía real' })
  })
  it('delegates to pillClass/typeLabel for a false positive', () => {
    expect(meterStatusBadge(FALSE_POSITIVE)).toEqual({ className: 'pill pill-good', label: 'Falso positivo' })
  })
})
