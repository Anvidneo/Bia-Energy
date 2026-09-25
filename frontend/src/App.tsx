import { useState } from 'react'
import './styles.css'
import { Layout, type Screen } from './components/Layout'
import { Dashboard } from './components/Dashboard'
import { MeterDetail } from './components/MeterDetail'
import { AnomaliesTable } from './components/AnomaliesTable'
import { AnomalyDetail } from './components/AnomalyDetail'
import { AnalysisRunner } from './components/AnalysisRunner'
import { useTheme } from './hooks'
import type { Anomaly } from './types'

// Simple state-based screen switching — no router library, per the plan's
// own decision (the dataset/app is small enough that back/forward and deep
// links aren't worth a second new dependency alongside the Go backend).
function App() {
  const [screen, setScreen] = useState<Screen>('dashboard')
  const [meterId, setMeterId] = useState<string | null>(null)
  const [selectedAnomaly, setSelectedAnomaly] = useState<Anomaly | null>(null)
  const [returnScreen, setReturnScreen] = useState<Screen>('dashboard')
  const [theme, toggleTheme] = useTheme()

  const openAnomaly = (a: Anomaly) => {
    setReturnScreen(screen)
    setSelectedAnomaly(a)
  }

  const closeAnomaly = () => setSelectedAnomaly(null)

  const goToMeter = (id: string) => {
    setSelectedAnomaly(null)
    setMeterId(id)
    setScreen('meters')
  }

  const navigate = (s: Screen) => {
    setSelectedAnomaly(null)
    setScreen(s)
  }

  return (
    <Layout screen={screen} onNavigate={navigate} theme={theme} onToggleTheme={toggleTheme}>
      {selectedAnomaly ? (
        <AnomalyDetail
          anomalyId={selectedAnomaly.id}
          initial={selectedAnomaly}
          onBack={() => {
            closeAnomaly()
            setScreen(returnScreen)
          }}
          onViewMeter={goToMeter}
        />
      ) : (
        <>
          {screen === 'dashboard' && <Dashboard onSelectAnomaly={openAnomaly} />}
          {screen === 'meters' && <MeterDetail initialMeterId={meterId} onSelectAnomaly={openAnomaly} />}
          {screen === 'anomalies' && (
            <div className="card">
              <h3>Todas las anomalías</h3>
              <AnomaliesTable onSelect={openAnomaly} />
            </div>
          )}
          {screen === 'analysis' && <AnalysisRunner onSelectAnomaly={openAnomaly} />}
          {screen === 'reports' && (
            <div className="card">
              <h3>Reportes</h3>
              <p className="empty-state">Próximamente — no forma parte del alcance actual del MVP.</p>
            </div>
          )}
        </>
      )}
    </Layout>
  )
}

export default App
