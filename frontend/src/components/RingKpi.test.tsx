import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { RingKpi } from './RingKpi'

describe('RingKpi', () => {
  it('renders the label, value, sub-text and rounded percent', () => {
    render(
      <RingKpi label="Medidores" value={12} sub="Con datos cargados" percent={42.6} trackColor="#eee" color="#333" />,
    )
    expect(screen.getByText('Medidores')).toBeTruthy()
    expect(screen.getByText('12')).toBeTruthy()
    expect(screen.getByText('Con datos cargados')).toBeTruthy()
    expect(screen.getByText('43%')).toBeTruthy()
  })

  it('clamps a percent above 100 down to a full ring', () => {
    const { container } = render(
      <RingKpi label="x" value={1} sub="y" percent={150} trackColor="#eee" color="#333" />,
    )
    expect(screen.getByText('100%')).toBeTruthy()
    const progressCircle = container.querySelectorAll('circle')[1]
    expect(progressCircle.getAttribute('stroke-dashoffset')).toBe('0')
  })

  it('clamps a negative percent up to zero', () => {
    render(<RingKpi label="x" value={1} sub="y" percent={-20} trackColor="#eee" color="#333" />)
    expect(screen.getByText('0%')).toBeTruthy()
  })

  it('hides the percent text when showPercentLabel is false, keeping the ring full', () => {
    const { container } = render(
      <RingKpi label="Consumo total" value="20.00 kWh" sub="Periodo completo" percent={100} showPercentLabel={false} trackColor="#eee" color="#333" />,
    )
    expect(screen.queryByText('100%')).toBeNull()
    const progressCircle = container.querySelectorAll('circle')[1]
    expect(progressCircle.getAttribute('stroke-dashoffset')).toBe('0')
  })
})
