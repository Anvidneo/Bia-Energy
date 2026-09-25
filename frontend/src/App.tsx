import { useState } from 'react'
import './styles.css'
import { Layout, type Screen } from './components/Layout'
import { Login } from './components/Login'
import { Dashboard } from './components/Dashboard'
import { MeterDetail } from './components/MeterDetail'
import { MetersTable } from './components/MetersTable'
import { AnomaliesTable } from './components/AnomaliesTable'
import { AnomalyDetail } from './components/AnomalyDetail'
import { AnalysisRunner } from './components/AnalysisRunner'
import { useTheme } from './hooks'
import { extractMeterIdFromQuery } from './search'
import type { Anomaly } from './types'

// Simple state-based screen switching — no router library, per the plan's
// own decision (the dataset/app is small enough that back/forward and deep
// links aren't worth a second new dependency alongside the Go backend).
const AUTH_STORAGE_KEY = 'bia-auth'

function readStoredAuth(): boolean {
  try {
    return sessionStorage.getItem(AUTH_STORAGE_KEY) === '1'
  } catch {
    // sessionStorage unavailable (e.g. private browsing) — just ask again
    return false
  }
}

function App() {
  const [authenticated, setAuthenticated] = useState<boolean>(readStoredAuth)
  const [screen, setScreen] = useState<Screen>('dashboard')
  const [meterId, setMeterId] = useState<string | null>(null)
  const [selectedAnomaly, setSelectedAnomaly] = useState<Anomaly | null>(null)
  const [returnScreen, setReturnScreen] = useState<Screen>('dashboard')
  const [theme, toggleTheme] = useTheme()

  const handleLogin = () => {
    setAuthenticated(true)
    try {
      sessionStorage.setItem(AUTH_STORAGE_KEY, '1')
    } catch {
      // best-effort persistence only — session still stays logged in in memory
    }
  }

  const handleLogout = () => {
    setAuthenticated(false)
    try {
      sessionStorage.removeItem(AUTH_STORAGE_KEY)
    } catch {
      // ignore
    }
  }

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
    // Landing on Medidores from the sidebar always starts at the table, even
    // if a specific meter was open before — only goToMeter (search, "view
    // meter" links) should jump straight to a meter's detail.
    if (s === 'meters') setMeterId(null)
    setScreen(s)
  }

  // Quick search: the topbar box only understands meter ids for now
  // (e.g. "M-110", "110", "m-110") — it jumps straight to that meter's
  // detail view, reusing the same navigation goToMeter already does for
  // "view meter" links coming from an anomaly.
  const handleSearch = (query: string) => {
    const meterId = extractMeterIdFromQuery(query)
    if (!meterId) return
    goToMeter(meterId)
  }

  if (!authenticated) {
    return <Login onLogin={handleLogin} />
  }

  return (
    <Layout
      screen={screen}
      onNavigate={navigate}
      theme={theme}
      onToggleTheme={toggleTheme}
      onSearch={handleSearch}
      onLogout={handleLogout}
    >
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
          {screen === 'meters' && (
            meterId
              ? <MeterDetail key={meterId} meterId={meterId} onBack={() => setMeterId(null)} onSelectAnomaly={openAnomaly} />
              : <MetersTable onSelectMeter={setMeterId} />
          )}
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
