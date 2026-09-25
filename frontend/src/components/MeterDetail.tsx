import { useEffect, useState } from 'react'
import type { Anomaly, Reading } from '../types'
import { getMeterReadings, listAnomalies } from '../api'
import { ConsumptionChart } from './ConsumptionChart'
import { pillClass, typeLabel, formatTime } from './severity'

interface Props {
  meterId: string
  onBack: () => void
  onSelectAnomaly: (a: Anomaly) => void
}

export function MeterDetail({ meterId, onBack, onSelectAnomaly }: Props) {
  const [readings, setReadings] = useState<Reading[] | null>(null)
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Keyed by meterId in the parent (see App.tsx), so this remounts — and
  // readings/anomalies/error reset to their initial values — every time the
  // selected meter changes, instead of resetting state imperatively here.
  useEffect(() => {
    let cancelled = false
    Promise.all([getMeterReadings(meterId), listAnomalies()])
      .then(([readingData, anomalyData]) => {
        if (cancelled) return
        setReadings(readingData)
        setAnomalies(anomalyData.filter((a) => a.meter_id === meterId))
      })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
  }, [meterId])

  const anomalyTimestamps = new Set((anomalies ?? []).map((a) => a.detected_at))

  return (
    <>
      <button type="button" className="back-link" onClick={onBack}>&larr; Volver a medidores</button>

      {error && <p className="empty-state">No se pudo cargar el medidor: {error}</p>}

      {!error && (
        <>
          <div className="card">
            <h3>Consumo horario — {meterId}</h3>
            {readings ? (
              <ConsumptionChart readings={readings} anomalyTimestamps={anomalyTimestamps} height={260} gradientId="areaGradMeter" />
            ) : (
              <p className="empty-state">Cargando lecturas…</p>
            )}
            <p className="caption">{readings?.length ?? 0} lecturas horarias · puntos rojos = anomalía detectada en esa hora</p>
          </div>

          <div className="card">
            <h3>Anomalías de {meterId}</h3>
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
        </>
      )}
    </>
  )
}
