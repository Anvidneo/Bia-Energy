// One typed fetch per backend endpoint (backend/internal/api). All 8 routes
// from the plan doc, plus /health.

import type { Meter, Reading, Anomaly, AnalysisResult, DashboardSummary } from './types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, init)
  if (!res.ok) {
    let detail = res.statusText
    try {
      const body = await res.json()
      if (body?.error) detail = body.error
    } catch {
      // ignore — not all error responses are JSON
    }
    throw new Error(`${path}: ${res.status} ${detail}`)
  }
  return res.json() as Promise<T>
}

export function getHealth() {
  return request<{ status: string; service: string }>('/health')
}

export function getDashboardSummary() {
  return request<DashboardSummary>('/dashboard/summary')
}

export function listMeters() {
  return request<Meter[]>('/meters')
}

export function getMeter(meterId: string) {
  return request<Meter>(`/meters/${encodeURIComponent(meterId)}`)
}

export function getMeterReadings(meterId: string) {
  return request<Reading[]>(`/meters/${encodeURIComponent(meterId)}/readings`)
}

export function runAnalysis() {
  return request<{ analysisId: string }>('/ai/analyze', { method: 'POST' })
}

export function getAnalysis(analysisId: string) {
  return request<AnalysisResult>(`/ai/analysis/${encodeURIComponent(analysisId)}`)
}

export function listAnomalies() {
  return request<Anomaly[]>('/anomalies')
}

export function getAnomaly(id: number | string) {
  return request<Anomaly>(`/anomalies/${id}`)
}
