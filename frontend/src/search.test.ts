import { describe, expect, it } from 'vitest'
import { extractMeterIdFromQuery } from './search'

describe('extractMeterIdFromQuery', () => {
  it('normalizes a well-formed meter id', () => {
    expect(extractMeterIdFromQuery('M-110')).toBe('M-110')
  })

  it('normalizes a lowercase id with no separator', () => {
    expect(extractMeterIdFromQuery('m110')).toBe('M-110')
  })

  it('extracts digits from surrounding whitespace/text', () => {
    expect(extractMeterIdFromQuery('  buscar 110  ')).toBe('M-110')
  })

  it('returns null when the query has no digits', () => {
    expect(extractMeterIdFromQuery('anomalias')).toBeNull()
  })

  it('returns null for an empty query', () => {
    expect(extractMeterIdFromQuery('')).toBeNull()
  })
})
