// One typed fetch per backend endpoint (backend/internal/api).
// Today's skeleton only wires the base client + health check; the other 7
// endpoints get filled in once internal/api ships (Friday).

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, init)
  if (!res.ok) {
    throw new Error(`API ${path} failed: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export function getHealth() {
  return request<{ status: string; service: string }>('/health')
}

// TODO (Friday), once internal/api defines the real routes:
// export function getDashboardSummary() { ... }      // GET /dashboard/summary
// export function getMeterReadings(meterId: string)  // GET /meters/:id/readings
// export function runAnalysis(meterId: string)       // POST /ai/analysis
// export function getAnalysis(analysisId: string)    // GET /ai/analysis/:id
// export function listAnomalies()                    // GET /anomalies
// export function getAnomaly(id: string)              // GET /anomalies/:id
// export function listMeters()                        // GET /meters
