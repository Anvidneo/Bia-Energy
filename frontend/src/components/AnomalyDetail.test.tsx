import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AnomalyDetail } from './AnomalyDetail'
import { getAnomaly } from '../api'
import type { Anomaly } from '../types'

vi.mock('../api')

function makeAnomaly(overrides: Partial<Anomaly> = {}): Anomaly {
  return {
    id: 42,
    meter_id: 'M-109',
    detected_at: '2026-09-01T12:00:00Z',
    anomaly: true,
    type: 'REAL_ANOMALY',
    severity: 'HIGH',
    confidence: 0.95,
    reason: 'Consumo 103.7% por encima del baseline',
    recommended_action: 'Inspeccionar línea productiva nueva',
    evidence: {
      baseline_median_kwh: 1070,
      observed_kwh: 2180,
      deviation_pct: 103.7,
      z_score: 4.2,
      consecutive_hours: 6,
      expected_kwh_from_electrical: 2100,
    },
    ...overrides,
  }
}

describe('AnomalyDetail', () => {
  it('renders immediately from the initial prop, without fetching', () => {
    const initial = makeAnomaly()
    render(<AnomalyDetail anomalyId={42} initial={initial} onBack={() => {}} onViewMeter={() => {}} />)
    expect(screen.getByText(/M-109/)).toBeTruthy()
    expect(screen.getByText(/Consumo 103.7% por encima del baseline/)).toBeTruthy()
    expect(getAnomaly).not.toHaveBeenCalled()
  })

  it('fetches the anomaly when no initial value is given', async () => {
    vi.mocked(getAnomaly).mockResolvedValue(makeAnomaly({ id: 7, meter_id: 'M-107' }))
    render(<AnomalyDetail anomalyId={7} onBack={() => {}} onViewMeter={() => {}} />)
    expect(screen.getByText(/Cargando/)).toBeTruthy()
    expect(await screen.findByText(/M-107/)).toBeTruthy()
  })

  it('re-fetches when the initial value is for a different anomaly id', async () => {
    vi.mocked(getAnomaly).mockResolvedValue(makeAnomaly({ id: 9, meter_id: 'M-109-fresh' }))
    render(<AnomalyDetail anomalyId={9} initial={makeAnomaly({ id: 1, meter_id: 'stale' })} onBack={() => {}} onViewMeter={() => {}} />)
    expect(await screen.findByText(/M-109-fresh/)).toBeTruthy()
  })

  it('shows an error state when the fetch fails', async () => {
    vi.mocked(getAnomaly).mockRejectedValue(new Error('boom'))
    render(<AnomalyDetail anomalyId={7} onBack={() => {}} onViewMeter={() => {}} />)
    expect(await screen.findByText(/No se pudo cargar la anomalía/)).toBeTruthy()
  })

  it('calls onBack when the back button is clicked', () => {
    const onBack = vi.fn()
    render(<AnomalyDetail anomalyId={42} initial={makeAnomaly()} onBack={onBack} onViewMeter={() => {}} />)
    fireEvent.click(screen.getByText('Volver'))
    expect(onBack).toHaveBeenCalled()
  })

  it('calls onViewMeter with the meter id when "ver medidor" is clicked', () => {
    const onViewMeter = vi.fn()
    render(<AnomalyDetail anomalyId={42} initial={makeAnomaly()} onBack={() => {}} onViewMeter={onViewMeter} />)
    fireEvent.click(screen.getByText('ver medidor'))
    expect(onViewMeter).toHaveBeenCalledWith('M-109')
  })

  it('shows the electrical estimate field only for data-quality anomalies', () => {
    render(<AnomalyDetail anomalyId={42} initial={makeAnomaly({ type: 'DATA_QUALITY' })} onBack={() => {}} onViewMeter={() => {}} />)
    expect(screen.getByText('Estimado eléctrico (V×I×PF)')).toBeTruthy()
  })

  it('hides the electrical estimate field for non-data-quality anomalies', () => {
    render(<AnomalyDetail anomalyId={42} initial={makeAnomaly({ type: 'REAL_ANOMALY' })} onBack={() => {}} onViewMeter={() => {}} />)
    expect(screen.queryByText('Estimado eléctrico (V×I×PF)')).toBeNull()
  })

  it('shows the related event line, with its timestamp, when present', () => {
    render(
      <AnomalyDetail
        anomalyId={42}
        initial={makeAnomaly({ related_event_type: 'OPERATIONAL_CHANGE', related_event_at: '2026-09-01T08:00:00Z' })}
        onBack={() => {}}
        onViewMeter={() => {}}
      />,
    )
    expect(screen.getByText(/Evento relacionado: OPERATIONAL_CHANGE/)).toBeTruthy()
  })

  it('omits the related event line when there is none', () => {
    render(<AnomalyDetail anomalyId={42} initial={makeAnomaly({ related_event_type: undefined })} onBack={() => {}} onViewMeter={() => {}} />)
    expect(screen.queryByText(/Evento relacionado/)).toBeNull()
  })
})
