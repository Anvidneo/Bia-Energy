import { render, screen, fireEvent, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Dashboard } from './Dashboard'
import { getDashboardSummary, listAnomalies, getMeterReadings, listMeters } from '../api'
import type { Anomaly, DashboardSummary, Meter, Reading } from '../types'

vi.mock('../api')

function makeAnomaly(overrides: Partial<Anomaly> & { id: number; meter_id: string }): Anomaly {
  return {
    detected_at: '2026-09-01T00:00:00Z',
    anomaly: true,
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    confidence: 0.9,
    reason: 'test reason',
    recommended_action: 'test action',
    evidence: {
      baseline_median_kwh: 100, observed_kwh: 200, deviation_pct: 100,
      z_score: 3, consecutive_hours: 4, expected_kwh_from_electrical: 190,
    },
    ...overrides,
  }
}

const SUMMARY: DashboardSummary = { total_meters: 12, active_anomalies: 2 }
const METERS: Meter[] = [{ id: 'M-101' }, { id: 'M-102' }]
const READINGS: Reading[] = [
  { meter_id: 'M-109', timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 10, voltage_v: 220, current_a: 5, power_factor: 0.95 },
]

beforeEach(() => {
  vi.mocked(listMeters).mockResolvedValue(METERS)
  vi.mocked(getMeterReadings).mockResolvedValue(READINGS)
})

describe('Dashboard', () => {
  it('shows an error state when the initial fetch fails', async () => {
    vi.mocked(getDashboardSummary).mockRejectedValue(new Error('boom'))
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/No se pudo cargar el dashboard/)).toBeTruthy()
  })

  it('renders KPIs computed from the summary and anomalies', async () => {
    vi.mocked(getDashboardSummary).mockResolvedValue(SUMMARY)
    vi.mocked(listAnomalies).mockResolvedValue([
      makeAnomaly({ id: 1, meter_id: 'M-101', severity: 'HIGH', type: 'REAL_ANOMALY' }),
      makeAnomaly({ id: 2, meter_id: 'M-102', severity: 'HIGH', type: 'FALSE_POSITIVE' }), // not "critical"
      makeAnomaly({ id: 3, meter_id: 'M-103', severity: 'LOW', type: 'DATA_QUALITY' }),
    ])

    render(<Dashboard onSelectAnomaly={() => {}} />)

    // total_meters (KPI 1), active anomalies = anomalies.length = 3 (KPI 2)
    expect(await screen.findByText('12')).toBeTruthy()
    expect(screen.getByText('3')).toBeTruthy()
    // critical = HIGH severity, excluding FALSE_POSITIVE => just anomaly #1
    const kpiGrid = document.querySelector('.kpi-grid') as HTMLElement
    expect(within(kpiGrid).getByText('Alertas críticas').closest('.kpi-card')?.textContent).toContain('1')
    // data quality = type === DATA_QUALITY => just anomaly #3 (scoped to the
    // KPI grid: the same "Calidad de datos" label also appears as an anomaly
    // type in the feed/table further down the page)
    expect(within(kpiGrid).getByText('Calidad de datos').closest('.kpi-card')?.textContent).toContain('1')
    // Consumo total: suma de las lecturas de los 2 medidores (METERS),
    // cada uno resuelto por el mock global de getMeterReadings a un único
    // reading de 10 kWh => 20 kWh en total.
    await screen.findByText('20.00 kWh')
    expect(within(kpiGrid).getByText('Consumo total').closest('.kpi-card')?.textContent).toContain('20.00 kWh')
    // Confianza IA: promedio de confidence entre las 3 anomalías (todas 0.9) => 90%
    expect(within(kpiGrid).getByText('Confianza IA').closest('.kpi-card')?.textContent).toContain('90%')
  })

  it('shows a dash for Confianza IA when there are no anomalies yet', async () => {
    vi.mocked(getDashboardSummary).mockResolvedValue(SUMMARY)
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    await screen.findByText(/Sin anomalías todavía/)
    const kpiGrid = document.querySelector('.kpi-grid') as HTMLElement
    expect(within(kpiGrid).getByText('Confianza IA').closest('.kpi-card')?.textContent).toContain('—')
  })

  it('shows "Aún sin análisis ejecutado" when there is no last_analysis_at', async () => {
    vi.mocked(getDashboardSummary).mockResolvedValue({ total_meters: 0, active_anomalies: 0 })
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    expect(await screen.findByText('Aún sin análisis ejecutado')).toBeTruthy()
  })

  it('shows a relative last-analysis time when one is present', async () => {
    vi.mocked(getDashboardSummary).mockResolvedValue({
      total_meters: 0, active_anomalies: 0, last_analysis_at: new Date(Date.now() - 5 * 60_000).toISOString(),
    })
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/Último análisis: Hace 5 min/)).toBeTruthy()
  })

  it('picks the chart meter from the first anomaly, and shows "—" when there are none and no meters', async () => {
    vi.mocked(listMeters).mockResolvedValue([])
    vi.mocked(getDashboardSummary).mockResolvedValue(SUMMARY)
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/Consumo — — \(últimas 24h\)/)).toBeTruthy()
    expect(getMeterReadings).not.toHaveBeenCalled()
  })

  it('falls back to the first meter for the chart when there are no anomalies', async () => {
    vi.mocked(getDashboardSummary).mockResolvedValue(SUMMARY)
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/Consumo — M-101 \(últimas 24h\)/)).toBeTruthy()
    expect(getMeterReadings).toHaveBeenCalledWith('M-101')
  })

  it('shows an empty state for the recent-anomalies feed when there are none', async () => {
    vi.mocked(getDashboardSummary).mockResolvedValue(SUMMARY)
    vi.mocked(listAnomalies).mockResolvedValue([])
    render(<Dashboard onSelectAnomaly={() => {}} />)
    expect(await screen.findByText(/Sin anomalías todavía/)).toBeTruthy()
  })

  it('calls onSelectAnomaly when a recent-anomaly feed item is clicked', async () => {
    const onSelectAnomaly = vi.fn()
    const anomaly = makeAnomaly({ id: 5, meter_id: 'M-105' })
    vi.mocked(getDashboardSummary).mockResolvedValue(SUMMARY)
    vi.mocked(listAnomalies).mockResolvedValue([anomaly])

    render(<Dashboard onSelectAnomaly={onSelectAnomaly} />)
    // Scoped to .feed-title: the meter id also shows up in the chart heading
    // and, once loaded, the anomalies table below.
    const feedTitle = await screen.findByText(/M-105/, { selector: '.feed-title' })
    fireEvent.click(feedTitle.closest('.feed-item') as HTMLElement)
    expect(onSelectAnomaly).toHaveBeenCalledWith(anomaly)
  })
})
