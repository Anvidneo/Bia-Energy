import { useEffect, useState } from 'react'
import type { Anomaly } from '../types'
import { listAnomalies } from '../api'
import { useMediaQuery, MOBILE_BREAKPOINT } from '../hooks'
import { pillClass, typeLabel, formatTime } from './severity'

interface Props {
  onSelect: (a: Anomaly) => void
  limit?: number
  anomalies?: Anomaly[] // pass pre-fetched list (e.g. from Dashboard) to skip its own fetch
}

export function AnomaliesTable({ onSelect, limit, anomalies: provided }: Props) {
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(provided ?? null)
  const [error, setError] = useState<string | null>(null)
  const isMobile = useMediaQuery(MOBILE_BREAKPOINT)

  useEffect(() => {
    if (provided) {
      setAnomalies(provided)
      return
    }
    let cancelled = false
    listAnomalies()
      .then((data) => { if (!cancelled) setAnomalies(data) })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
  }, [provided])

  if (error) return <p className="empty-state">No se pudieron cargar las anomalías: {error}</p>
  if (!anomalies) return <p className="empty-state">Cargando…</p>
  if (anomalies.length === 0) return <p className="empty-state">Sin anomalías registradas. Ejecuta un análisis desde la pestaña Análisis.</p>

  const rows = limit ? anomalies.slice(0, limit) : anomalies

  if (isMobile) {
    return (
      <div className="anomaly-cards">
        {rows.map((a) => (
          <div key={a.id} className="anomaly-card" onClick={() => onSelect(a)} role="button" tabIndex={0}>
            <div className="row-top">
              <span className="meter-id">{a.meter_id}</span>
              <span className={pillClass(a)}>{typeLabel(a.type)}</span>
            </div>
            <span className="type-line">{a.reason}</span>
            <div className="row-bottom">
              <span className="meta">{Math.round(a.confidence * 100)}% · {formatTime(a.detected_at)}</span>
              <a href="#" onClick={(e) => { e.preventDefault(); onSelect(a) }}>Ver detalle</a>
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <table className="anomalies-table">
      <thead>
        <tr>
          <th>MEDIDOR</th>
          <th>TIPO</th>
          <th>SEVERIDAD</th>
          <th>CONFIANZA</th>
          <th>HORA</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {rows.map((a) => (
          <tr key={a.id} className="clickable" onClick={() => onSelect(a)}>
            <td className="meter-cell">{a.meter_id}</td>
            <td>{a.reason}</td>
            <td><span className={pillClass(a)}>{typeLabel(a.type)}</span></td>
            <td>{Math.round(a.confidence * 100)}%</td>
            <td className="muted">{formatTime(a.detected_at)}</td>
            <td style={{ textAlign: 'right' }}>
              <a href="#" onClick={(e) => { e.stopPropagation(); e.preventDefault(); onSelect(a) }}>Ver detalle</a>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
