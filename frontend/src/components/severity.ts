import type { Anomaly, AnomalyType, Severity } from '../types'

// Maps the backend's classification to the mockup's pill styles/colors.
// FALSE_POSITIVE reads as "resolved" (good/green) since it means the
// system correctly explained the deviation away.
export function pillClass(a: Pick<Anomaly, 'type' | 'severity'>): string {
  if (a.type === 'FALSE_POSITIVE') return 'pill pill-good'
  if (a.type === 'DATA_QUALITY') return 'pill pill-high'
  if (a.severity === 'HIGH') return 'pill pill-crit'
  if (a.severity === 'MEDIUM') return 'pill pill-med'
  return 'pill pill-good'
}

export function typeLabel(type: AnomalyType): string {
  switch (type) {
    case 'REAL_ANOMALY': return 'Anomalía real'
    case 'FALSE_POSITIVE': return 'Falso positivo'
    case 'EXPLAINABLE_ANOMALY': return 'Explicada por evento'
    case 'DATA_QUALITY': return 'Calidad de datos'
    default: return type
  }
}

// Single source of truth for "Normal vs. this anomaly's pill" so the
// Medidores table and the meter detail card never drift apart.
export function meterStatusBadge(meta: Pick<Anomaly, 'type' | 'severity'> | null): { className: string; label: string } {
  if (!meta) return { className: 'pill pill-good', label: 'Normal' }
  return { className: pillClass(meta), label: typeLabel(meta.type) }
}

export function severityLabel(s: Severity): string {
  switch (s) {
    case 'HIGH': return 'Alta'
    case 'MEDIUM': return 'Media'
    case 'LOW': return 'Baja'
    default: return s
  }
}

export function dotColorVar(a: Pick<Anomaly, 'type' | 'severity'>): string {
  if (a.type === 'FALSE_POSITIVE') return 'var(--good)'
  if (a.type === 'DATA_QUALITY') return 'var(--high)'
  if (a.severity === 'HIGH') return 'var(--crit)'
  if (a.severity === 'MEDIUM') return 'var(--med)'
  return 'var(--good)'
}

export function tintColorVar(a: Pick<Anomaly, 'type' | 'severity'>): string {
  if (a.type === 'FALSE_POSITIVE') return 'var(--good-tint)'
  if (a.type === 'DATA_QUALITY') return 'var(--high-tint)'
  if (a.severity === 'HIGH') return 'var(--crit-tint)'
  if (a.severity === 'MEDIUM') return 'var(--med-tint)'
  return 'var(--good-tint)'
}

// Compartido por la tabla de Medidores y el resumen del detalle de medidor
// para que ambos formateen consumo/variación exactamente igual.
export function formatKwh(value: number | null): string {
  return value === null ? '—' : `${value.toFixed(2)} kWh`
}

export function formatPct(value: number | null): string {
  if (value === null) return '—'
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(1)}%`
}

export function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('es-CO', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

export function formatRelative(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime()
  const mins = Math.round(diffMs / 60000)
  if (mins < 1) return 'Justo ahora'
  if (mins < 60) return `Hace ${mins} min`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `Hace ${hours} h`
  const days = Math.round(hours / 24)
  return `Hace ${days} d`
}
