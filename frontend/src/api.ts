// One typed fetch per backend endpoint (backend/internal/api). All 8 routes
// from the plan doc, plus /health.

import type { Meter, Reading, Anomaly, AnalysisResult, DashboardSummary } from './types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

// Every call site below passes a literal path or one built from
// encodeURIComponent(...), so this can never actually fail — it's a
// guard against a future call site smuggling a full URL (e.g.
// "https://evil.example") through a dynamic segment and redirecting the
// request away from our own API host.
const SAFE_PATH = /^\/[A-Za-z0-9\-._~/]*$/

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  if (!SAFE_PATH.test(path)) {
    throw new Error(`refusing to fetch unsafe path: ${path}`)
  }
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
  return request<Anomaly>(`/anomalies/${encodeURIComponent(id)}`)
}
