import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ConsumptionChart } from './ConsumptionChart'
import type { Reading } from '../types'

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

describe('ConsumptionChart', () => {
  it('renders without crashing for a normal set of readings', () => {
    const readings = [
      makeReading({ timestamp: '2026-09-01T00:00:00Z', consumption_kwh: 10 }),
      makeReading({ timestamp: '2026-09-01T01:00:00Z', consumption_kwh: 12 }),
      makeReading({ timestamp: '2026-09-01T02:00:00Z', consumption_kwh: 200 }),
    ]
    const { container } = render(
      <ConsumptionChart
        readings={readings}
        anomalyTimestamps={new Set(['2026-09-01T02:00:00Z'])}
        gradientId="areaGradTest"
      />,
    )
    expect(container.querySelector('.recharts-responsive-container')).toBeTruthy()
  })

  it('renders with an empty reading list and no anomaly set', () => {
    const { container } = render(<ConsumptionChart readings={[]} gradientId="areaGradEmpty" />)
    expect(container.querySelector('.recharts-responsive-container')).toBeTruthy()
  })
})
