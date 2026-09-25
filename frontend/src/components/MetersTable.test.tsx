import { render, screen, fireEvent, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MetersTable } from './MetersTable'
import { listMeters, listAnomalies } from '../api'
import { useMediaQuery } from '../hooks'
import type { Anomaly, Meter } from '../types'

vi.mock('../api')
vi.mock('../hooks', () => ({
  useMediaQuery: vi.fn(() => false),
  MOBILE_BREAKPOINT: '(max-width: 860px)',
}))

function makeAnomaly(overrides: Partial<Anomaly> & { meter_id: string }): Anomaly {
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

const METERS: Meter[] = [{ id: 'M-101' }, { id: 'M-102' }, { id: 'M-103' }]
const ANOMALIES: Anomaly[] = [
  makeAnomaly({ meter_id: 'M-102', severity: 'HIGH', type: 'REAL_ANOMALY', detected_at: '2026-09-05T00:00:00Z' }),
]

beforeEach(() => {
  vi.mocked(listMeters).mockResolvedValue(METERS)
  vi.mocked(listAnomalies).mockResolvedValue(ANOMALIES)
  vi.mocked(useMediaQuery).mockReturnValue(false)
})

describe('MetersTable (desktop)', () => {
  it('shows a loading state, then all fetched meters', async () => {
    render(<MetersTable onSelectMeter={() => {}} />)
    expect(screen.getByText(/Cargando medidores/)).toBeTruthy()
    expect(await screen.findByText('M-101')).toBeTruthy()
    expect(screen.getByText('M-102')).toBeTruthy()
    expect(screen.getByText('M-103')).toBeTruthy()
  })

  it('shows an error state when the fetch fails', async () => {
    vi.mocked(listMeters).mockRejectedValue(new Error('boom'))
    render(<MetersTable onSelectMeter={() => {}} />)
    expect(await screen.findByText(/No se pudieron cargar los medidores/)).toBeTruthy()
  })

  it('filters rows by the search box', async () => {
    render(<MetersTable onSelectMeter={() => {}} />)
    await screen.findByText('M-101')
    fireEvent.change(screen.getByPlaceholderText(/Buscar por ID/), { target: { value: '102' } })
    expect(screen.queryByText('M-101')).toBeNull()
    expect(screen.getByText('M-102')).toBeTruthy()
  })

  it('filters rows with the status chips', async () => {
    render(<MetersTable onSelectMeter={() => {}} />)
    await screen.findByText('M-101')

    fireEvent.click(screen.getByText('Con anomalías'))
    expect(screen.queryByText('M-101')).toBeNull()
    expect(screen.getByText('M-102')).toBeTruthy()

    fireEvent.click(screen.getByText('Normal'))
    expect(screen.getByText('M-101')).toBeTruthy()
    expect(screen.queryByText('M-102')).toBeNull()

    fireEvent.click(screen.getByText('Todos'))
    expect(screen.getByText('M-101')).toBeTruthy()
    expect(screen.getByText('M-102')).toBeTruthy()
  })

  it('shows an empty state when nothing matches the search', async () => {
    render(<MetersTable onSelectMeter={() => {}} />)
    await screen.findByText('M-101')
    fireEvent.change(screen.getByPlaceholderText(/Buscar por ID/), { target: { value: 'ZZZ' } })
    expect(await screen.findByText(/Ningún medidor coincide/)).toBeTruthy()
  })

  it('sorts ascending by id by default, and flips on a second header click', async () => {
    render(<MetersTable onSelectMeter={() => {}} />)
    await screen.findByText('M-101')

    const bodyRow = (index: number) => screen.getAllByRole('row').slice(1)[index]
    expect(within(bodyRow(0)).getByText('M-101')).toBeTruthy()

    fireEvent.click(screen.getByText('MEDIDOR'))
    expect(within(bodyRow(0)).getByText('M-103')).toBeTruthy()
  })

  it('sorts by a different column when its header is clicked', async () => {
    render(<MetersTable onSelectMeter={() => {}} />)
    await screen.findByText('M-101')

    fireEvent.click(screen.getByText('ANOMALÍAS'))
    const bodyRow = (index: number) => screen.getAllByRole('row').slice(1)[index]
    // Ascending by anomaly count: the two 0-count meters first, M-102 last.
    expect(within(bodyRow(2)).getByText('M-102')).toBeTruthy()
  })

  it('calls onSelectMeter when a row is clicked', async () => {
    const onSelectMeter = vi.fn()
    render(<MetersTable onSelectMeter={onSelectMeter} />)
    await screen.findByText('M-101')
    fireEvent.click(screen.getByText('M-101'))
    expect(onSelectMeter).toHaveBeenCalledWith('M-101')
  })
})

describe('MetersTable (mobile)', () => {
  it('renders stacked cards instead of a table', async () => {
    vi.mocked(useMediaQuery).mockReturnValue(true)
    const onSelectMeter = vi.fn()
    render(<MetersTable onSelectMeter={onSelectMeter} />)

    expect(await screen.findByText('M-101')).toBeTruthy()
    expect(screen.queryByRole('table')).toBeNull()

    fireEvent.click(screen.getAllByText('Ver detalle')[0])
    expect(onSelectMeter).toHaveBeenCalled()
  })
})
