import { render, screen, fireEvent, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MeterDetail } from './MeterDetail'
import { getMeterReadings, listAnomalies } from '../api'
import type { Anomaly, Reading } from '../types'

vi.mock('../api')

// recharts' ResponsiveContainer needs a real ResizeObserver/layout, neither
// of which jsdom provides — the chart's own rendering isn't this
// component's concern, so it's swapped for a lightweight stub.
vi.mock('./ConsumptionChart', () => ({
  ConsumptionChart: () => <div data-testid="chart-stub" />,
}))

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

function makeAnomaly(overrides: Partial<Anomaly> & { meter_id: string }): Anomaly {
  return {
    id: 1,
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

beforeEach(() => {
  vi.mocked(getMeterReadings).mockResolvedValue([makeReading(), makeReading({ timestamp: '2026-09-01T01:00:00Z' })])
  vi.mocked(listAnomalies).mockResolvedValue([
    makeAnomaly({ meter_id: 'M-109' }),
    makeAnomaly({ meter_id: 'M-999' }), // belongs to a different meter — must be filtered out
  ])
})

describe('MeterDetail', () => {
  it('loads and shows the readings count and only this meter\'s anomalies', async () => {
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={() => {}} />)

    expect(await screen.findByText(/Consumo horario — M-109/)).toBeTruthy()
    expect(await screen.findByText(/2 lecturas horarias/)).toBeTruthy()
    expect(screen.getByText(/Consumo muy por encima del baseline/)).toBeTruthy()
  })

  it('shows an error state when the fetch fails', async () => {
    vi.mocked(getMeterReadings).mockRejectedValue(new Error('boom'))
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/No se pudo cargar el medidor/)).toBeTruthy()
  })

  it('shows an empty state when the meter has no anomalies', async () => {
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/Sin anomalías detectadas para este medidor/)).toBeTruthy()
  })

  it('calls onBack when the back link is clicked', async () => {
    const onBack = vi.fn()
    render(<MeterDetail meterId="M-109" onBack={onBack} onSelectAnomaly={() => {}} />)
    await screen.findByText(/Consumo horario/)
    fireEvent.click(screen.getByText(/Volver a medidores/))
    expect(onBack).toHaveBeenCalled()
  })

  it('calls onSelectAnomaly when an anomaly card is clicked', async () => {
    const onSelectAnomaly = vi.fn()
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={onSelectAnomaly} />)
    const card = await screen.findByText(/Consumo muy por encima del baseline/)
    fireEvent.click(card)
    expect(onSelectAnomaly).toHaveBeenCalled()
  })

  it('calls onSelectAnomaly when the "Ver detalle" link inside a card is clicked', async () => {
    const onSelectAnomaly = vi.fn()
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={onSelectAnomaly} />)
    const link = await screen.findByText('Ver detalle')
    fireEvent.click(link)
    expect(onSelectAnomaly).toHaveBeenCalledTimes(1)
  })

  it('shows the summary card (estado, consumo, baseline, variación, eléctricas) from the worst anomaly', async () => {
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={() => {}} />)
    await screen.findByText(/Consumo horario/)

    const summary = screen.getByText('Resumen — M-109').closest('.card') as HTMLElement
    expect(within(summary).getByText('Anomalía real')).toBeTruthy()
    expect(within(summary).getByText('10.00 kWh')).toBeTruthy() // consumo actual
    expect(within(summary).getByText('100.00 kWh')).toBeTruthy() // baseline (de la evidencia)
    expect(within(summary).getByText('-90.0%')).toBeTruthy() // variación
    expect(within(summary).getByText('220.0 V')).toBeTruthy()
    expect(within(summary).getByText('5.00 A')).toBeTruthy()
    expect(within(summary).getByText('0.95')).toBeTruthy()
  })

  it('shows Estado "Normal" and falls back to the median as baseline when there are no anomalies', async () => {
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={() => {}} />)
    await screen.findByText(/Sin anomalías detectadas/)

    const summary = screen.getByText('Resumen — M-109').closest('.card') as HTMLElement
    expect(within(summary).getByText('Normal')).toBeTruthy()
    expect(within(summary).getAllByText('10.00 kWh')).toHaveLength(2) // consumo actual === baseline (mediana)
    expect(within(summary).getByText('0.0%')).toBeTruthy()
  })

  it('shows placeholder dashes in the summary when there are no readings and no anomalies', async () => {
    vi.mocked(getMeterReadings).mockResolvedValue([])
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<MeterDetail meterId="M-109" onBack={() => {}} onSelectAnomaly={() => {}} />)
    await screen.findByText(/Sin anomalías detectadas/)

    const summary = screen.getByText('Resumen — M-109').closest('.card') as HTMLElement
    expect(within(summary).getByText('Normal')).toBeTruthy()
    // consumo actual, baseline, variación, voltaje, corriente, factor de potencia
    expect(within(summary).getAllByText('—')).toHaveLength(6)
  })
})
