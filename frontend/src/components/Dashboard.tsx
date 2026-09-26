import { useEffect, useState } from 'react'
import type { Anomaly, DashboardSummary, Reading } from '../types'
import { getDashboardSummary, listAnomalies, getMeterReadings, listMeters } from '../api'
import { RingKpi } from './RingKpi'
import { AnomaliesTable } from './AnomaliesTable'
import { ConsumptionChart } from './ConsumptionChart'
import { dotColorVar, tintColorVar, typeLabel, formatRelative, formatKwh } from './severity'

interface Props {
  onSelectAnomaly: (a: Anomaly) => void
}

export function Dashboard({ onSelectAnomaly }: Props) {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [anomalies, setAnomalies] = useState<Anomaly[] | null>(null)
  const [chartMeter, setChartMeter] = useState<string | null>(null)
  const [readings, setReadings] = useState<Reading[] | null>(null)
  const [totalConsumptionKwh, setTotalConsumptionKwh] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([getDashboardSummary(), listAnomalies(), listMeters()])
      .then(([summaryData, anomalyData, meters]) => {
        if (cancelled) return
        setSummary(summaryData)
        setAnomalies(anomalyData)
        const meterId = anomalyData[0]?.meter_id ?? meters[0]?.id ?? null
        setChartMeter(meterId)

        const chartPromise = meterId
          ? getMeterReadings(meterId).then((r) => { if (!cancelled) setReadings(r.slice(-24)) })
          : Promise.resolve()

        // Consumo total del periodo (sección 5 de la prueba): suma de todas
        // las lecturas de todos los medidores. Se pide aparte de la gráfica
        // porque necesita el histórico completo, no solo las últimas 24h.
        const totalPromise = Promise.all(meters.map((m) => getMeterReadings(m.id))).then((all) => {
          if (cancelled) return
          setTotalConsumptionKwh(all.flat().reduce((sum, r) => sum + r.consumption_kwh, 0))
        })

        return Promise.all([chartPromise, totalPromise])
      })
      .catch((e) => { if (!cancelled) setError(e.message) })
    return () => { cancelled = true }
  }, [])

  if (error) return <p className="empty-state">No se pudo cargar el dashboard: {error}</p>

  const totalMeters = summary?.total_meters ?? 0
  const activeAnomalies = anomalies?.length ?? summary?.active_anomalies ?? 0
  const criticalCount = anomalies?.filter((a) => a.severity === 'HIGH' && a.type !== 'FALSE_POSITIVE').length ?? 0
  const dataQualityCount = anomalies?.filter((a) => a.type === 'DATA_QUALITY').length ?? 0
  // Confianza IA (sección 5): métrica agregada — promedio de confianza entre
  // todas las anomalías detectadas.
  const avgConfidence = anomalies && anomalies.length > 0
    ? anomalies.reduce((sum, a) => sum + a.confidence, 0) / anomalies.length
    : null

  const pct = (count: number) => (totalMeters > 0 ? Math.min(100, (count / totalMeters) * 100) : 0)

  const anomalyTimestamps = new Set(
    (anomalies ?? []).filter((a) => a.meter_id === chartMeter).map((a) => a.detected_at),
  )

  return (
    <>
      <section aria-label="Resumen" className="hero">
        <div>
          <h2>Monitoreo de red eléctrica</h2>
          <p>Detección de anomalías y calidad de datos en tiempo real</p>
        </div>
        <div className="hero-actions">
          <div className="date-chip">
            {summary?.last_analysis_at
              ? `Último análisis: ${formatRelative(summary.last_analysis_at)}`
              : 'Aún sin análisis ejecutado'}
          </div>
        </div>
      </section>

      <div className="kpi-grid">
        <RingKpi
          label="Medidores monitoreados"
          value={totalMeters}
          sub="Con datos cargados"
          percent={totalMeters > 0 ? 100 : 0}
          showPercentLabel={false}
          trackColor="#DCEBFC"
          color="var(--accent)"
        />
        <RingKpi
          label="Consumo total"
          value={formatKwh(totalConsumptionKwh)}
          sub="Periodo completo"
          percent={100}
          showPercentLabel={false}
          trackColor="#DCEBFC"
          color="var(--accent)"
        />
        <RingKpi
          label="Anomalías activas"
          value={activeAnomalies}
          sub={`En ${totalMeters} medidores`}
          percent={pct(activeAnomalies)}
          trackColor="var(--med-tint)"
          color="var(--med)"
        />
        <RingKpi
          label="Alertas críticas"
          value={criticalCount}
          sub="Atención inmediata"
          percent={pct(criticalCount)}
          trackColor="var(--crit-tint)"
          color="var(--crit)"
        />
        <RingKpi
          label="Confianza IA"
          value={avgConfidence !== null ? `${Math.round(avgConfidence * 100)}%` : '—'}
          sub="Métrica agregada"
          percent={avgConfidence !== null ? avgConfidence * 100 : 0}
          trackColor="var(--good-tint)"
          color="var(--good)"
        />
        <RingKpi
          label="Calidad de datos"
          value={dataQualityCount}
          sub="Discrepancias sensor"
          percent={pct(dataQualityCount)}
          trackColor="var(--high-tint)"
          color="var(--high)"
        />
      </div>

      <div className="two-col">
        <div className="card">
          <h3>Consumo — {chartMeter ?? '—'} (últimas 24h)</h3>
          {readings ? (
            <ConsumptionChart readings={readings} anomalyTimestamps={anomalyTimestamps} gradientId="areaGradDashboard" />
          ) : (
            <p className="empty-state">Cargando…</p>
          )}
          <p className="caption">Puntos marcados: desviación sostenida &gt; 3.5σ respecto al patrón horario</p>
        </div>

        <div className="card">
          <h3>Anomalías recientes</h3>
          {(anomalies ?? []).slice(0, 4).map((a) => (
            <div key={a.id} className="feed-item" onClick={() => onSelectAnomaly(a)} role="button" tabIndex={0} style={{ cursor: 'pointer' }}>
              <div className="feed-icon" style={{ background: tintColorVar(a) }}>
                <span className="dot" style={{ background: dotColorVar(a) }} />
              </div>
              <div className="feed-text">
                <div className="feed-title">{a.meter_id} · {typeLabel(a.type)}</div>
                <div className="feed-time">{formatRelative(a.detected_at)}</div>
              </div>
            </div>
          ))}
          {anomalies && anomalies.length === 0 && (
            <p className="empty-state">Sin anomalías todavía. Ejecuta un análisis desde la pestaña Análisis.</p>
          )}
        </div>
      </div>

      <div className="card">
        <h3>Anomalías detectadas</h3>
        <AnomaliesTable anomalies={anomalies ?? undefined} onSelect={onSelectAnomaly} limit={5} />
      </div>
    </>
  )
}
