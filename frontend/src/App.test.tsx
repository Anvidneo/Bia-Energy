import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import App from './App'

// App.tsx is pure wiring: which screen shows, which prop closures get built
// and passed down, and how the "meters" screen splits between the table and
// a specific meter. Every child is stubbed to a couple of buttons/markers so
// these tests exercise exactly that wiring, not the children's own behavior
// (each of which has its own dedicated test file already).
vi.mock('./components/Layout', () => ({
  Layout: ({ screen, onNavigate, onSearch, children }: any) => (
    <div>
      <div data-testid="current-screen">{screen}</div>
      <button onClick={() => onNavigate('dashboard')}>nav-dashboard</button>
      <button onClick={() => onNavigate('meters')}>nav-meters</button>
      <button onClick={() => onNavigate('anomalies')}>nav-anomalies</button>
      <button onClick={() => onNavigate('analysis')}>nav-analysis</button>
      <button onClick={() => onNavigate('reports')}>nav-reports</button>
      <button onClick={() => onSearch?.('110')}>trigger-search</button>
      <button onClick={() => onSearch?.('no digits here')}>trigger-search-invalid</button>
      {children}
    </div>
  ),
}))

vi.mock('./components/Dashboard', () => ({
  Dashboard: ({ onSelectAnomaly }: any) => (
    <button onClick={() => onSelectAnomaly({ id: 1, meter_id: 'M-101' })}>dashboard-select-anomaly</button>
  ),
}))

vi.mock('./components/MetersTable', () => ({
  MetersTable: ({ onSelectMeter }: any) => (
    <button onClick={() => onSelectMeter('M-102')}>meters-table-select</button>
  ),
}))

vi.mock('./components/MeterDetail', () => ({
  MeterDetail: ({ meterId, onBack }: any) => (
    <div>
      <div data-testid="meter-detail">{meterId}</div>
      <button onClick={onBack}>meter-detail-back</button>
    </div>
  ),
}))

vi.mock('./components/AnomaliesTable', () => ({
  AnomaliesTable: ({ onSelect }: any) => (
    <button onClick={() => onSelect({ id: 3, meter_id: 'M-103' })}>anomalies-table-select</button>
  ),
}))

vi.mock('./components/AnomalyDetail', () => ({
  AnomalyDetail: ({ anomalyId, onBack, onViewMeter }: any) => (
    <div>
      <div data-testid="anomaly-detail">{anomalyId}</div>
      <button onClick={onBack}>anomaly-detail-back</button>
      <button onClick={() => onViewMeter('M-999')}>anomaly-detail-view-meter</button>
    </div>
  ),
}))

vi.mock('./components/AnalysisRunner', () => ({
  AnalysisRunner: () => <div>analysis-runner</div>,
}))

vi.mock('./hooks', () => ({
  useTheme: () => ['light', vi.fn()],
}))

describe('App', () => {
  it('starts on the dashboard', () => {
    render(<App />)
    expect(screen.getByText('dashboard-select-anomaly')).toBeTruthy()
  })

  it('shows the meters table, then a meter\'s detail once one is picked', () => {
    render(<App />)
    fireEvent.click(screen.getByText('nav-meters'))
    expect(screen.getByText('meters-table-select')).toBeTruthy()

    fireEvent.click(screen.getByText('meters-table-select'))
    expect(screen.getByTestId('meter-detail').textContent).toBe('M-102')

    fireEvent.click(screen.getByText('meter-detail-back'))
    expect(screen.getByText('meters-table-select')).toBeTruthy()
  })

  it('resets to the meters table when navigating back to it from the sidebar', () => {
    render(<App />)
    fireEvent.click(screen.getByText('nav-meters'))
    fireEvent.click(screen.getByText('meters-table-select'))
    expect(screen.getByTestId('meter-detail')).toBeTruthy()

    fireEvent.click(screen.getByText('nav-dashboard'))
    fireEvent.click(screen.getByText('nav-meters'))
    expect(screen.getByText('meters-table-select')).toBeTruthy()
    expect(screen.queryByTestId('meter-detail')).toBeNull()
  })

  it('jumps straight to a meter\'s detail from the topbar search', () => {
    render(<App />)
    fireEvent.click(screen.getByText('trigger-search'))
    expect(screen.getByTestId('current-screen').textContent).toBe('meters')
    expect(screen.getByTestId('meter-detail').textContent).toBe('M-110')
  })

  it('ignores a search query with no digits', () => {
    render(<App />)
    fireEvent.click(screen.getByText('trigger-search-invalid'))
    expect(screen.getByTestId('current-screen').textContent).toBe('dashboard')
  })

  it('opens an anomaly from the dashboard and returns to the dashboard on back', () => {
    render(<App />)
    fireEvent.click(screen.getByText('dashboard-select-anomaly'))
    expect(screen.getByTestId('anomaly-detail').textContent).toBe('1')

    fireEvent.click(screen.getByText('anomaly-detail-back'))
    expect(screen.getByText('dashboard-select-anomaly')).toBeTruthy()
  })

  it('returns to the anomalies screen (not the dashboard) when that\'s where the anomaly was opened from', () => {
    render(<App />)
    fireEvent.click(screen.getByText('nav-anomalies'))
    fireEvent.click(screen.getByText('anomalies-table-select'))
    expect(screen.getByTestId('anomaly-detail').textContent).toBe('3')

    fireEvent.click(screen.getByText('anomaly-detail-back'))
    expect(screen.getByText('anomalies-table-select')).toBeTruthy()
  })

  it('jumps to a meter\'s detail from within an open anomaly, closing the anomaly view', () => {
    render(<App />)
    fireEvent.click(screen.getByText('dashboard-select-anomaly'))
    fireEvent.click(screen.getByText('anomaly-detail-view-meter'))
    expect(screen.queryByTestId('anomaly-detail')).toBeNull()
    expect(screen.getByTestId('current-screen').textContent).toBe('meters')
    expect(screen.getByTestId('meter-detail').textContent).toBe('M-999')
  })

  it('renders the analysis and reports screens', () => {
    render(<App />)
    fireEvent.click(screen.getByText('nav-analysis'))
    expect(screen.getByText('analysis-runner')).toBeTruthy()

    fireEvent.click(screen.getByText('nav-reports'))
    expect(screen.getByText(/Próximamente/)).toBeTruthy()
  })
})
