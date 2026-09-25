import { useEffect, useMemo, useState } from 'react'
import type { Anomaly, Meter } from '../types'
import { listMeters, listAnomalies } from '../api'
import { useMediaQuery, MOBILE_BREAKPOINT } from '../hooks'
import { IconSearch } from '../icons'
import { pillClass, typeLabel, formatTime } from './severity'
import {
  buildMeterRows, filterMeterRows, sortMeterRows,
  type MeterRow, type SortKey, type SortDirection, type StatusFilter,
} from './meters'

interface Props {
  onSelectMeter: (id: string) => void
}

const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'Todos' },
  { value: 'anomaly', label: 'Con anomalías' },
  { value: 'normal', label: 'Normal' },
]

const COLUMNS: { key: SortKey; label: string }[] = [
  { key: 'id', label: 'MEDIDOR' },
  { key: 'anomalyCount', label: 'ANOMALÍAS' },
  { key: 'severity', label: 'ESTADO' },
  { key: 'lastDetectedAt', label: 'ÚLTIMA DETECCIÓN' },
]

function MeterStatusPill({ row }: { row: MeterRow }) {
  if (row.anomalyCount > 0 && row.lastType && row.lastSeverity) {
    const meta = { type: row.lastType, severity: row.lastSeverity }
    return <span className={pillClass(meta)}>{typeLabel(row.lastType)}</span>
  }
  return <span className="pill pill-good">Normal</span>
}

export function MetersTable({ onSelectMeter }: Props) {
  const [meters, setMeters] = useState<Meter[] | null>(null)
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [sortBy, setSortBy] = useState<SortKey>('id')
  const [direction, setDirection] = useState<SortDirection>('asc')
  const isMobile = useMediaQuery(MOBILE_BREAKPOINT)

  useEffect(() => {
    let cancelled = false
    Promise.all([listMeters(), listAnomalies()])
      .then(([meterData, anomalyData]) => {
        if (cancelled) return
        setMeters(meterData)
        setAnomalies(anomalyData)
      })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
  }, [])

  const rows = useMemo(() => {
    if (!meters || !anomalies) return null
    const built = buildMeterRows(meters, anomalies)
    const filtered = filterMeterRows(built, query, status)
    return sortMeterRows(filtered, sortBy, direction)
  }, [meters, anomalies, query, status, sortBy, direction])

  const toggleSort = (key: SortKey) => {
    if (key === sortBy) {
      setDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortBy(key)
      setDirection('asc')
    }
  }

  if (error) return <p className="empty-state">No se pudieron cargar los medidores: {error}</p>
  if (!rows) return <p className="empty-state">Cargando medidores…</p>

  return (
    <div className="card">
      <h3>Medidores</h3>

      <div className="table-toolbar">
        <form role="search" className="search-form" onSubmit={(e) => e.preventDefault()}>
          <IconSearch />
          <label htmlFor="meters-search" className="sr-only">Buscar medidor</label>
          <input
            id="meters-search"
            type="text"
            placeholder="Buscar por ID de medidor…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </form>
        <div className="chip-row">
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.value}
              type="button"
              className={`chip${status === f.value ? ' active' : ''}`}
              onClick={() => setStatus(f.value)}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {rows.length === 0 && <p className="empty-state">Ningún medidor coincide con la búsqueda.</p>}

      {rows.length > 0 && (isMobile ? (
        <div className="entity-cards">
          {rows.map((row) => (
            <div key={row.id} className="entity-card" onClick={() => onSelectMeter(row.id)} role="button" tabIndex={0}>
              <div className="row-top">
                <span className="meter-id">{row.id}</span>
                <MeterStatusPill row={row} />
              </div>
              <span className="type-line">{row.anomalyCount} anomalía(s) detectada(s)</span>
              <div className="row-bottom">
                <span className="meta">{row.lastDetectedAt ? formatTime(row.lastDetectedAt) : 'Sin eventos'}</span>
                <a href="#" onClick={(e) => { e.preventDefault(); onSelectMeter(row.id) }}>Ver detalle</a>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
              {COLUMNS.map((col) => (
                <th key={col.key} scope="col">
                  <button type="button" className="sort-btn" onClick={() => toggleSort(col.key)}>
                    {col.label}
                    {sortBy === col.key && <span className="sort-arrow">{direction === 'asc' ? '↑' : '↓'}</span>}
                  </button>
                </th>
              ))}
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.id} className="clickable" onClick={() => onSelectMeter(row.id)}>
                <td className="meter-cell">{row.id}</td>
                <td>{row.anomalyCount}</td>
                <td><MeterStatusPill row={row} /></td>
                <td className="muted">{row.lastDetectedAt ? formatTime(row.lastDetectedAt) : '—'}</td>
                <td style={{ textAlign: 'right' }}>
                  <a href="#" onClick={(e) => { e.stopPropagation(); e.preventDefault(); onSelectMeter(row.id) }}>Ver detalle</a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ))}
    </div>
  )
}
