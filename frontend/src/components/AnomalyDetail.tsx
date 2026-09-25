import { useEffect, useState } from 'react'
import type { Anomaly } from '../types'
import { getAnomaly } from '../api'
import { IconBack } from '../icons'
import { pillClass, typeLabel, severityLabel, formatTime } from './severity'

interface Props {
  anomalyId: number
  initial?: Anomaly
  onBack: () => void
  onViewMeter: (meterId: string) => void
}

export function AnomalyDetail({ anomalyId, initial, onBack, onViewMeter }: Props) {
  const [anomaly, setAnomaly] = useState<Anomaly | null>(initial ?? null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (initial && initial.id === anomalyId) return
    let cancelled = false
    getAnomaly(anomalyId)
      .then((a) => { if (!cancelled) setAnomaly(a) })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
  }, [anomalyId, initial])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <button className="btn btn-secondary" style={{ alignSelf: 'flex-start' }} onClick={onBack}>
        <IconBack /> Volver
      </button>

      {error && <p className="empty-state">No se pudo cargar la anomalía: {error}</p>}
      {!error && !anomaly && <p className="empty-state">Cargando…</p>}

      {anomaly && (
        <>
          <div className="card">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
              <div>
                <h3 style={{ fontSize: 18 }}>
                  {anomaly.meter_id} ·{' '}
                  <a href="#" onClick={(e) => { e.preventDefault(); onViewMeter(anomaly.meter_id) }}>
                    ver medidor
                  </a>
                </h3>
                <p className="caption" style={{ marginTop: 4 }}>Detectada {formatTime(anomaly.detected_at)}</p>
              </div>
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <span className={pillClass(anomaly)}>{typeLabel(anomaly.type)}</span>
                <span className="caption">Severidad: {severityLabel(anomaly.severity)}</span>
              </div>
            </div>

            <div style={{ marginTop: 16 }}>
              <div className="evidence-item" style={{ background: 'transparent', padding: 0 }}>
                <span className="label">Razón</span>
                <span style={{ fontSize: 14 }}>{anomaly.reason}</span>
              </div>
            </div>

            <div style={{ marginTop: 12 }}>
              <div className="evidence-item" style={{ background: 'transparent', padding: 0 }}>
                <span className="label">Acción recomendada</span>
                <span style={{ fontSize: 14 }}>{anomaly.recommended_action}</span>
              </div>
            </div>

            {anomaly.related_event_type && (
              <p className="caption" style={{ marginTop: 12 }}>
                Evento relacionado: {anomaly.related_event_type}
                {anomaly.related_event_at ? ` · ${formatTime(anomaly.related_event_at)}` : ''}
              </p>
            )}
          </div>

          <div className="card">
            <h3>Evidencia estadística</h3>
            <div className="evidence-grid">
              <div className="evidence-item">
                <span className="label">Confianza</span>
                <span className="value">{Math.round(anomaly.confidence * 100)}%</span>
              </div>
              <div className="evidence-item">
                <span className="label">Consumo observado</span>
                <span className="value">{anomaly.evidence.observed_kwh.toFixed(2)} kWh</span>
              </div>
              <div className="evidence-item">
                <span className="label">Baseline (mediana)</span>
                <span className="value">{anomaly.evidence.baseline_median_kwh.toFixed(2)} kWh</span>
              </div>
              <div className="evidence-item">
                <span className="label">Desviación</span>
                <span className="value">{anomaly.evidence.deviation_pct.toFixed(1)}%</span>
              </div>
              <div className="evidence-item">
                <span className="label">Z-score</span>
                <span className="value">{anomaly.evidence.z_score.toFixed(2)}</span>
              </div>
              <div className="evidence-item">
                <span className="label">Horas consecutivas</span>
                <span className="value">{anomaly.evidence.consecutive_hours}</span>
              </div>
              {anomaly.type === 'DATA_QUALITY' && (
                <div className="evidence-item">
                  <span className="label">Estimado eléctrico (V×I×PF)</span>
                  <span className="value">{anomaly.evidence.expected_kwh_from_electrical.toFixed(2)} kWh</span>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
