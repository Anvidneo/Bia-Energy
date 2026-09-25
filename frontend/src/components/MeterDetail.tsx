import { useEffect, useState } from 'react'
import type { Anomaly, Meter, Reading } from '../types'
import { listMeters, getMeterReadings, listAnomalies } from '../api'
import { ConsumptionChart } from './ConsumptionChart'
import { pillClass, typeLabel, formatTime } from './severity'

interface Props {
  initialMeterId?: string | null
  onSelectAnomaly: (a: Anomaly) => void
}

export function MeterDetail({ initialMeterId, onSelectAnomaly }: Props) {
  const [meters, setMeters] = useState<Meter[] | null>(null)
  const [selected, setSelected] = useState<string | null>(initialMeterId ?? null)
  const [readings, setReadings] = useState<Reading[] | null>(null)
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    listMeters()
      .then((data) => {
        if (cancelled) return
        setMeters(data)
        if (!selected && data.length > 0) setSelected(data[0].id)
      })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!selected) return
    let cancelled = false
    setReadings(null)
    Promise.all([getMeterReadings(selected), listAnomalies()])
      .then(([readingData, anomalyData]) => {
        if (cancelled) return
        setReadings(readingData)
        setAnomalies(anomalyData.filter((a) => a.meter_id === selected))
      })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
  }, [selected])

  if (error) return <p className="empty-state">No se pudo cargar el medidor: {error}</p>

  const anomalyTimestamps = new Set((anomalies ?? []).map((a) => a.detected_at))

  return (
    <>
      <div className="card">
        <h3>Seleccionar medidor</h3>
        {meters ? (
          <div className="chip-row">
            {meters.map((m) => (
              <button
                key={m.id}
                className={`chip${selected === m.id ? ' active' : ''}`}
                onClick={() => setSelected(m.id)}
              >
                {m.id}
              </button>
            ))}
          </div>
        ) : (
          <p className="empty-state">Cargando medidores…</p>
        )}
      </div>

      {selected && (
        <div className="card">
          <h3>Consumo horario — {selected}</h3>
          {readings ? (
            <ConsumptionChart readings={readings} anomalyTimestamps={anomalyTimestamps} height={260} gradientId="areaGradMeter" />
          ) : (
            <p className="empty-state">Cargando lecturas…</p>
          )}
          <p className="caption">{readings?.length ?? 0} lecturas horarias · puntos rojos = anomalía detectada en esa hora</p>
        </div>
      )}

      {selected && (
        <div className="card">
          <h3>Anomalías de {selected}</h3>
          {anomalies === null && <p className="empty-state">Cargando…</p>}
          {anomalies !== null && anomalies.length === 0 && (
            <p className="empty-state">Sin anomalías detectadas para este medidor.</p>
          )}
          {anomalies && anomalies.length > 0 && (
            <div className="anomaly-cards">
              {anomalies.map((a) => (
                <div key={a.id} className="anomaly-card" onClick={() => onSelectAnomaly(a)} role="button" tabIndex={0}>
                  <div className="row-top">
                    <span className="meter-id">{formatTime(a.detected_at)}</span>
                    <span className={pillClass(a)}>{typeLabel(a.type)}</span>
                  </div>
                  <span className="type-line">{a.reason}</span>
                  <div className="row-bottom">
                    <span className="meta">Confianza {Math.round(a.confidence * 100)}%</span>
                    <a href="#" onClick={(e) => { e.preventDefault(); onSelectAnomaly(a) }}>Ver detalle</a>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </>
  )
}
