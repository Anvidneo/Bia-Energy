import { useEffect, useState } from 'react'
import type { Anomaly, Reading } from '../types'
import { getMeterReadings, listAnomalies } from '../api'
import { ConsumptionChart } from './ConsumptionChart'
import { pillClass, typeLabel, formatTime, meterStatusBadge, formatKwh, formatPct } from './severity'
import { computeMeterStats } from './meterStats'

interface Props {
  meterId: string
  onBack: () => void
  onSelectAnomaly: (a: Anomaly) => void
}

// Keyed by meterId in the parent (see App.tsx), so this remounts — and
// readings/anomalies/error reset to their initial values — every time the
// selected meter changes, instead of resetting state imperatively here.
export function MeterDetail({ meterId, onBack, onSelectAnomaly }: Props) {
  const [readings, setReadings] = useState<Reading[] | null>(null)
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(null)
  const [error, setError] = useState<string | null>(null)

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
  // Section 7 de la prueba: "Debe mostrar consumo actual, baseline,
  // variación, estado e histórico. Idealmente también voltaje, corriente y
  // factor de potencia." — el histórico es la gráfica de abajo; este
  // resumen cubre el resto a partir de la última lectura + la peor
  // anomalía activa del medidor.
  const stats = readings && anomalies ? computeMeterStats(readings, anomalies) : null
  const statusBadge = meterStatusBadge(stats?.worstAnomaly ?? null)

  return (
    <>
      <button type="button" className="back-link" onClick={onBack}>&larr; Volver a medidores</button>

      {error && <p className="empty-state">No se pudo cargar el medidor: {error}</p>}

      {!error && (
        <>
          <div className="card">
            <h3>Resumen — {meterId}</h3>
            {stats ? (
              <div className="evidence-grid">
                <div className="evidence-item">
                  <span className="label">Estado</span>
                  <span className={statusBadge.className}>{statusBadge.label}</span>
                </div>
                <div className="evidence-item">
                  <span className="label">Consumo actual</span>
                  <span className="value">{formatKwh(stats.currentConsumptionKwh)}</span>
                </div>
                <div className="evidence-item">
                  <span className="label">Baseline aproximado</span>
                  <span className="value">{formatKwh(stats.baselineKwh)}</span>
                </div>
                <div className="evidence-item">
                  <span className="label">Variación</span>
                  <span className="value">{formatPct(stats.variationPct)}</span>
                </div>
                <div className="evidence-item">
                  <span className="label">Voltaje</span>
                  <span className="value">{stats.voltageV !== null ? `${stats.voltageV.toFixed(1)} V` : '—'}</span>
                </div>
                <div className="evidence-item">
                  <span className="label">Corriente</span>
                  <span className="value">{stats.currentA !== null ? `${stats.currentA.toFixed(2)} A` : '—'}</span>
                </div>
                <div className="evidence-item">
                  <span className="label">Factor de potencia</span>
                  <span className="value">{stats.powerFactor !== null ? stats.powerFactor.toFixed(2) : '—'}</span>
                </div>
              </div>
            ) : (
              <p className="empty-state">Cargando resumen…</p>
            )}
          </div>

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
                      <a href="#" onClick={(e) => { e.stopPropagation(); e.preventDefault(); onSelectAnomaly(a) }}>Ver detalle</a>
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
