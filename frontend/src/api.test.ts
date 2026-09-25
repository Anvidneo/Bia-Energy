import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  getHealth,
  getDashboardSummary,
  listMeters,
  getMeter,
  getMeterReadings,
  runAnalysis,
  getAnalysis,
  listAnomalies,
  getAnomaly,
} from './api'

function mockFetchOnce(response: Partial<Response> & { jsonBody?: unknown }) {
  const { jsonBody, ...rest } = response
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    statusText: 'OK',
    json: async () => jsonBody,
    ...rest,
  } as Response)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('request() path safety', () => {
  it('rejects a path that would percent-encode unsafe characters', async () => {
    const fetchMock = mockFetchOnce({ jsonBody: {} })
    await expect(getMeter('a b')).rejects.toThrow(/refusing to fetch unsafe path/)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('allows a plain encoded id and hits the right URL', async () => {
    const fetchMock = mockFetchOnce({ jsonBody: { id: 'm-1' } })
    await getMeter('m-1')
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/meters/m-1', undefined)
  })
})

describe('request() happy path', () => {
  it('returns parsed JSON on success', async () => {
    mockFetchOnce({ jsonBody: { status: 'ok', service: 'bia-energy' } })
    await expect(getHealth()).resolves.toEqual({ status: 'ok', service: 'bia-energy' })
  })

  it('covers every simple GET endpoint', async () => {
    mockFetchOnce({ jsonBody: [] })
    await expect(listMeters()).resolves.toEqual([])
    mockFetchOnce({ jsonBody: [] })
    await expect(listAnomalies()).resolves.toEqual([])
    mockFetchOnce({ jsonBody: {} })
    await expect(getDashboardSummary()).resolves.toEqual({})
    mockFetchOnce({ jsonBody: [] })
    await expect(getMeterReadings('m-1')).resolves.toEqual([])
    mockFetchOnce({ jsonBody: {} })
    await expect(getAnalysis('a-1')).resolves.toEqual({})
    mockFetchOnce({ jsonBody: {} })
    await expect(getAnomaly(42)).resolves.toEqual({})
  })

  it('POSTs for runAnalysis', async () => {
    const fetchMock = mockFetchOnce({ jsonBody: { analysisId: 'a-1' } })
    await expect(runAnalysis()).resolves.toEqual({ analysisId: 'a-1' })
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/ai/analyze', { method: 'POST' })
  })
})

describe('request() error handling', () => {
  it('surfaces the backend-provided error message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: async () => ({ error: 'meter not found' }),
      } as Response),
    )
    await expect(getMeter('missing')).rejects.toThrow('meter not found')
  })

  it('falls back to statusText when the error body is not JSON', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => {
          throw new Error('not json')
        },
      } as unknown as Response),
    )
    await expect(getMeter('boom')).rejects.toThrow('Internal Server Error')
  })
})
