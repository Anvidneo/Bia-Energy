import { useState, type ReactNode, type ReactElement } from 'react'
import {
  IconBolt, IconDashboard, IconMeters, IconAnomalies, IconAnalysis, IconReports,
  IconSearch, IconSun, IconMoon, IconBell, IconMenu, IconClose,
} from '../icons'
import type { Theme } from '../hooks'

export type Screen = 'dashboard' | 'meters' | 'anomalies' | 'analysis' | 'reports'

const NAV_ITEMS: { screen: Screen; label: string; icon: (props: { size?: number }) => ReactElement }[] = [
  { screen: 'dashboard', label: 'Dashboard', icon: IconDashboard },
  { screen: 'meters', label: 'Medidores', icon: IconMeters },
  { screen: 'anomalies', label: 'Anomalías', icon: IconAnomalies },
  { screen: 'analysis', label: 'Análisis', icon: IconAnalysis },
  { screen: 'reports', label: 'Reportes', icon: IconReports },
]

const TITLES: Record<Screen, string> = {
  dashboard: 'Dashboard',
  meters: 'Medidores',
  anomalies: 'Anomalías',
  analysis: 'Análisis',
  reports: 'Reportes',
}

interface Props {
  screen: Screen
  onNavigate: (s: Screen) => void
  theme: Theme
  onToggleTheme: () => void
  onSearch?: (query: string) => void
  onLogout?: () => void
  children: ReactNode
}

export function Layout({ screen, onNavigate, theme, onToggleTheme, onSearch, onLogout, children }: Props) {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const [searchValue, setSearchValue] = useState('')

  const navButtons = (variant: 'rail' | 'panel') =>
    NAV_ITEMS.map(({ screen: s, label, icon: Icon }) => (
      <button
        key={s}
        aria-label={label}
        aria-current={screen === s ? 'page' : undefined}
        className={`nav-btn${screen === s ? ' active' : ''}`}
        onClick={() => {
          onNavigate(s)
          setMobileNavOpen(false)
        }}
      >
        <Icon size={variant === 'rail' ? 22 : 18} />
        {variant === 'panel' && <span>{label}</span>}
      </button>
    ))

  return (
    <div className="app-shell">
      <nav aria-label="Navegación principal" className="sidebar">
        <div className="brand-mark" aria-hidden="true"><IconBolt /></div>
        <div style={{ height: 8, flex: '0 0 auto' }} />
        {navButtons('rail')}
      </nav>

      {mobileNavOpen && (
        <>
          {/* Native <button> instead of a clickable <div> so the backdrop is
              keyboard/focus accessible without needing manual role/key handling.
              It's a sibling of the panel (not a wrapper) so real <button> nav
              items inside the panel are never nested inside this button. */}
          <button
            type="button"
            className="mobile-nav-overlay"
            aria-label="Cerrar menú"
            onClick={() => setMobileNavOpen(false)}
          />
          <div className="mobile-nav-panel">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
              <div className="brand-mark" aria-hidden="true"><IconBolt /></div>
              <button className="mobile-icon-btn" aria-label="Cerrar menú" onClick={() => setMobileNavOpen(false)}>
                <IconClose />
              </button>
            </div>
            {navButtons('panel')}
          </div>
        </>
      )}

      <div className="main-column">
        <header className="mobile-header">
          <div className="brand-group">
            <button aria-label="Abrir menú" className="mobile-icon-btn" onClick={() => setMobileNavOpen(true)}>
              <IconMenu />
            </button>
            <div className="brand-mark" aria-hidden="true" style={{ width: 32, height: 32 }}><IconBolt size={14} /></div>
            <span className="brand-name">Bia Energy</span>
          </div>
          <div className="actions">
            <button
              aria-label={theme === 'light' ? 'Cambiar a tema oscuro' : 'Cambiar a tema claro'}
              aria-pressed={theme === 'dark'}
              className="mobile-icon-btn"
              style={{ color: 'var(--accent)' }}
              onClick={onToggleTheme}
            >
              {theme === 'light' ? <IconSun /> : <IconMoon />}
            </button>
            <button aria-label="Notificaciones" className="mobile-icon-btn">
              <IconBell size={15} />
              <span aria-hidden="true" className="badge-dot" style={{ top: 7, right: 8, width: 7, height: 7 }} />
            </button>
          </div>
        </header>

        <header className="topbar">
          <h1>{TITLES[screen]}</h1>
          <form
            role="search"
            className="search-form"
            onSubmit={(e) => {
              e.preventDefault()
              onSearch?.(searchValue)
            }}
          >
            <IconSearch />
            <label htmlFor="search-desktop" className="sr-only">Buscar</label>
            <input
              id="search-desktop"
              type="text"
              placeholder="Buscar medidor, anomalía..."
              value={searchValue}
              onChange={(e) => setSearchValue(e.target.value)}
            />
          </form>
          <div className="topbar-actions">
            <div role="group" aria-label="Cambiar tema" className="theme-toggle">
              <button aria-label="Tema claro" aria-pressed={theme === 'light'} onClick={() => theme !== 'light' && onToggleTheme()}>
                <IconSun />
              </button>
              <button aria-label="Tema oscuro" aria-pressed={theme === 'dark'} onClick={() => theme !== 'dark' && onToggleTheme()}>
                <IconMoon />
              </button>
            </div>
            <button aria-label="Notificaciones" className="icon-btn">
              <IconBell />
              <span aria-hidden="true" className="badge-dot" />
            </button>
            {onLogout ? (
              <button type="button" aria-label="Cerrar sesión" title="Cerrar sesión" className="avatar" onClick={onLogout}>
                {/* Real user identity intentionally left generic — no real auth backend in this MVP */}
                BE
              </button>
            ) : (
              <div aria-label="Cuenta" className="avatar">
                BE
              </div>
            )}
          </div>
        </header>

        <main className="content">{children}</main>
      </div>
    </div>
  )
}
