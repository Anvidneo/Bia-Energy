import { useState } from 'react'
import './styles.css'
import { Dashboard } from './components/Dashboard'
import { MeterDetail } from './components/MeterDetail'
import { AnomaliesTable } from './components/AnomaliesTable'

// Simple routing between Dashboard, meter detail and anomalies screens.
// No router library yet — matches the "don't add a second new thing while
// the backend is already new territory (Go)" decision. Placeholder for
// today's skeleton — wired for real on Saturday.
type Screen = 'dashboard' | 'meter' | 'anomalies'

function App() {
  const [screen, setScreen] = useState<Screen>('dashboard')

  return (
    <div>
      <nav>
        <button onClick={() => setScreen('dashboard')}>Dashboard</button>
        <button onClick={() => setScreen('meter')}>Meter detail</button>
        <button onClick={() => setScreen('anomalies')}>Anomalías</button>
      </nav>
      {screen === 'dashboard' && <Dashboard />}
      {screen === 'meter' && <MeterDetail />}
      {screen === 'anomalies' && <AnomaliesTable />}
    </div>
  )
}

export default App
