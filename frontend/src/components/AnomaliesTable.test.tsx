import { render, screen, fireEvent, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AnomaliesTable } from './AnomaliesTable'
import { listAnomalies } from '../api'
import { useMediaQuery } from '../hooks'
import type { Anomaly } from '../types'

vi.mock('../api')
vi.mock('../hooks', () => ({
  useMediaQuery: vi.fn(() => false),
  MOBILE_BREAKPOINT: '(max-width: 860px)',
}))

function makeAnomaly(overrides: Partial<Anomaly> & { id: number; meter_id: string }): Anomaly {
  return {
    detected_at: '2026-09-01T00:00:00Z',
    anomaly: true,
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    confidence: 0.87,
    reason: 'Consumo fuera de rango',
    recommended_action: 'Revisar línea',
    evidence: {
      baseline_median_kwh: 100, observed_kwh: 200, deviation_pct: 100,
      z_score: 3, consecutive_hours: 4, expected_kwh_from_electrical: 190,
    },
    ...overrides,
  }
}

const ANOMALIES: Anomaly[] = [
  makeAnomaly({ id: 1, meter_id: 'M-101' }),
  makeAnomaly({ id: 2, meter_id: 'M-102' }),
  makeAnomaly({ id: 3, meter_id: 'M-103' }),
]

beforeEach(() => {
  vi.mocked(useMediaQuery).mockReturnValue(false)
})

describe('AnomaliesTable (desktop)', () => {
  it('renders a pre-fetched list without calling the API', () => {
    render(<AnomaliesTable anomalies={ANOMALIES} onSelect={() => {}} />)
    expect(screen.getByText('M-101')).toBeTruthy()
    expect(listAnomalies).not.toHaveBeenCalled()
  })

  it('fetches its own list when none is provided, showing a loading state first', async () => {
    vi.mocked(listAnomalies).mockResolvedValue(ANOMALIES)
    render(<AnomaliesTable onSelect={() => {}} />)
    expect(screen.getByText(/Cargando/)).toBeTruthy()
    expect(await screen.findByText('M-101')).toBeTruthy()
  })

  it('shows an error state when the fetch fails', async () => {
    vi.mocked(listAnomalies).mockRejectedValue(new Error('boom'))
    render(<AnomaliesTable onSelect={() => {}} />)
    expect(await screen.findByText(/No se pudieron cargar las anomalías/)).toBeTruthy()
  })

  it('shows an empty state when there are no anomalies', () => {
    render(<AnomaliesTable anomalies={[]} onSelect={() => {}} />)
    expect(screen.getByText(/Sin anomalías registradas/)).toBeTruthy()
  })

  it('respects the limit prop', () => {
    render(<AnomaliesTable anomalies={ANOMALIES} onSelect={() => {}} limit={2} />)
    expect(screen.getAllByRole('row')).toHaveLength(3) // header + 2 rows
  })

  it('calls onSelect when a row is clicked', () => {
    const onSelect = vi.fn()
    render(<AnomaliesTable anomalies={ANOMALIES} onSelect={onSelect} />)
    fireEvent.click(screen.getByText('M-101'))
    expect(onSelect).toHaveBeenCalledWith(ANOMALIES[0])
  })

  it('calls onSelect from the "Ver detalle" link without double-firing the row handler', () => {
    const onSelect = vi.fn()
    render(<AnomaliesTable anomalies={ANOMALIES} onSelect={onSelect} />)
    const row = screen.getAllByRole('row')[1]
    fireEvent.click(within(row).getByText('Ver detalle'))
    expect(onSelect).toHaveBeenCalledTimes(1)
  })
})

describe('AnomaliesTable (mobile)', () => {
  beforeEach(() => {
    vi.mocked(useMediaQuery).mockReturnValue(true)
  })

  it('renders stacked cards and calls onSelect when one is clicked', () => {
    const onSelect = vi.fn()
    render(<AnomaliesTable anomalies={ANOMALIES} onSelect={onSelect} />)
    expect(screen.queryByRole('table')).toBeNull()
    fireEvent.click(screen.getByText('M-101'))
    expect(onSelect).toHaveBeenCalledWith(ANOMALIES[0])
  })

  it('calls onSelect from a card\'s "Ver detalle" link', () => {
    const onSelect = vi.fn()
    render(<AnomaliesTable anomalies={ANOMALIES} onSelect={onSelect} />)
    fireEvent.click(screen.getAllByText('Ver detalle')[0])
    expect(onSelect).toHaveBeenCalledWith(ANOMALIES[0])
  })
})
