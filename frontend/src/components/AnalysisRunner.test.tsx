import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AnalysisRunner } from './AnalysisRunner'
import { runAnalysis, getAnalysis } from '../api'
import type { AnalysisResult } from '../types'

vi.mock('../api')
vi.mock('./AnomaliesTable', () => ({
  AnomaliesTable: ({ anomalies }: { anomalies?: unknown[] }) => (
    <div data-testid="anomalies-stub">{anomalies?.length ?? 0}</div>
  ),
}))

// The component polls every 400ms with real timers here (fake timers don't
// interleave cleanly with its async setInterval callback), so these tests
// wait on real wall-clock time — still well under vitest's default timeout.
const POLL_MS = 400

function makeResult(overrides: Partial<AnalysisResult> = {}): AnalysisResult {
  return {
    id: 'run-1',
    status: 'pending',
    stage: 'Lecturas',
    started_at: '2026-09-01T00:00:00Z',
    ...overrides,
  }
}

const startButton = () => screen.getByRole('button', { name: /Ejecutar análisis IA|Ejecutando…/ })

describe('AnalysisRunner', () => {
  it('shows the idle button before anything runs', () => {
    render(<AnalysisRunner onSelectAnomaly={() => {}} />)
    expect(startButton().textContent).toContain('Ejecutar análisis IA')
  })

  it('starts an analysis, polls until done, and shows the pipeline/result', async () => {
    vi.mocked(runAnalysis).mockResolvedValue({ analysisId: 'run-1' })
    vi.mocked(getAnalysis)
      .mockResolvedValueOnce(makeResult({ status: 'processing', stage: 'Baseline' }))
      .mockResolvedValueOnce(makeResult({ status: 'done', stage: 'Recomendacion', anomalies: [{}, {}] as never }))

    render(<AnalysisRunner onSelectAnomaly={() => {}} />)
    fireEvent.click(startButton())

    await waitFor(() => expect(screen.getByText('Baseline')).toBeTruthy(), { timeout: POLL_MS * 4 })
    expect(startButton().textContent).toContain('Ejecutando…')

    await waitFor(
      () => expect(screen.getByText(/Análisis completo — 2 anomalía\(s\) detectada\(s\)/)).toBeTruthy(),
      { timeout: POLL_MS * 4 },
    )
    expect(screen.getByTestId('anomalies-stub').textContent).toBe('2')
    expect(startButton().textContent).toContain('Ejecutar análisis IA') // re-enabled/reset text
  })

  it('shows an error when starting the analysis fails', async () => {
    vi.mocked(runAnalysis).mockRejectedValue(new Error('no se pudo iniciar'))
    render(<AnalysisRunner onSelectAnomaly={() => {}} />)
    fireEvent.click(startButton())
    expect(await screen.findByText('Error: no se pudo iniciar')).toBeTruthy()
  })

  it('shows an error and stops polling when a poll request fails', async () => {
    vi.mocked(runAnalysis).mockResolvedValue({ analysisId: 'run-2' })
    vi.mocked(getAnalysis).mockRejectedValue(new Error('perdió la conexión'))

    render(<AnalysisRunner onSelectAnomaly={() => {}} />)
    fireEvent.click(startButton())
    expect(await screen.findByText('Error: perdió la conexión', {}, { timeout: POLL_MS * 4 })).toBeTruthy()
  })

  it('shows the failure message when the backend reports status "error"', async () => {
    vi.mocked(runAnalysis).mockResolvedValue({ analysisId: 'run-3' })
    vi.mocked(getAnalysis).mockResolvedValue(makeResult({ status: 'error', error: 'dataset inválido' }))

    render(<AnalysisRunner onSelectAnomaly={() => {}} />)
    fireEvent.click(startButton())
    expect(
      await screen.findByText(/El análisis falló: dataset inválido/, {}, { timeout: POLL_MS * 4 }),
    ).toBeTruthy()
  })

  it('clears its polling interval on unmount', async () => {
    vi.mocked(runAnalysis).mockResolvedValue({ analysisId: 'run-4' })
    vi.mocked(getAnalysis).mockResolvedValue(makeResult({ status: 'processing' }))

    const { unmount } = render(<AnalysisRunner onSelectAnomaly={() => {}} />)
    fireEvent.click(startButton())
    await waitFor(() => expect(getAnalysis).toHaveBeenCalled(), { timeout: POLL_MS * 4 })

    const callsBeforeUnmount = vi.mocked(getAnalysis).mock.calls.length
    unmount()
    await new Promise((resolve) => setTimeout(resolve, POLL_MS * 3))
    expect(vi.mocked(getAnalysis).mock.calls.length).toBe(callsBeforeUnmount)
  })
})
