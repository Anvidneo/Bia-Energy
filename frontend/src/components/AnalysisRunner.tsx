import { useEffect, useRef, useState } from 'react'
import type { AnalysisResult, Anomaly } from '../types'
import { PIPELINE_STAGES } from '../types'
import { runAnalysis, getAnalysis } from '../api'
import { IconPlay, IconCheck } from '../icons'
import { AnomaliesTable } from './AnomaliesTable'

interface Props {
  onSelectAnomaly: (a: Anomaly) => void
}

const POLL_INTERVAL_MS = 400

export function AnalysisRunner({ onSelectAnomaly }: Props) {
  const [result, setResult] = useState<AnalysisResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const timerRef = useRef<number | null>(null)

  useEffect(() => () => {
    if (timerRef.current) window.clearInterval(timerRef.current)
  }, [])

  const start = async () => {
    setError(null)
    setResult(null)
    try {
      const { analysisId } = await runAnalysis()
      poll(analysisId)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const poll = (id: string) => {
    if (timerRef.current) window.clearInterval(timerRef.current)
    timerRef.current = window.setInterval(async () => {
      try {
        const r = await getAnalysis(id)
        setResult(r)
        if (r.status === 'done' || r.status === 'error') {
          if (timerRef.current) window.clearInterval(timerRef.current)
        }
      } catch (e: any) {
        setError(e.message)
        if (timerRef.current) window.clearInterval(timerRef.current)
      }
    }, POLL_INTERVAL_MS)
  }

  const isRunning = result?.status === 'processing' || result?.status === 'pending'
  const currentStageIndex = result ? PIPELINE_STAGES.indexOf(result.stage as (typeof PIPELINE_STAGES)[number]) : -1

  return (
    <>
      <div className="card">
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
          <div>
            <h3>Ejecutar análisis IA</h3>
            <p className="caption" style={{ marginTop: 4 }}>
              Corre el motor de detección determinístico sobre todas las lecturas y eventos cargados.
            </p>
          </div>
          <button className="btn btn-primary" onClick={start} disabled={isRunning}>
            <IconPlay /> {isRunning ? 'Ejecutando…' : 'Ejecutar análisis IA'}
          </button>
        </div>

        {error && <p className="empty-state">Error: {error}</p>}

        {result && (
          <div className="pipeline" style={{ marginTop: 16 }}>
            {PIPELINE_STAGES.map((stage, i) => {
              const done = result.status === 'done' || i < currentStageIndex
              const active = i === currentStageIndex && result.status === 'processing'
              return (
                <div key={stage} className={`pipeline-stage${done ? ' done' : ''}${active ? ' active' : ''}`}>
                  <span className="stage-dot">{done ? <IconCheck /> : i + 1}</span>
                  {stage}
                </div>
              )
            })}
          </div>
        )}

        {result?.status === 'error' && (
          <p className="empty-state" style={{ marginTop: 12 }}>El análisis falló: {result.error}</p>
        )}

        {result?.status === 'done' && (
          <p className="caption" style={{ marginTop: 12 }}>
            Análisis completo — {result.anomalies?.length ?? 0} anomalía(s) detectada(s).
          </p>
        )}
      </div>

      {result?.status === 'done' && (
        <div className="card">
          <h3>Resultados</h3>
          <AnomaliesTable anomalies={result.anomalies ?? []} onSelect={onSelectAnomaly} />
        </div>
      )}
    </>
  )
}
