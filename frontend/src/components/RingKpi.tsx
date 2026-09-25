// Ring/donut SVG KPI, matching the mockup's hand-rolled chart (r=22,
// circumference 138.2..., stroke-dashoffset computed from the percentage).
const RADIUS = 22
const CIRCUMFERENCE = 2 * Math.PI * RADIUS // ≈138.2

interface Props {
  label: string
  value: number | string
  sub: string
  percent: number // 0-100
  trackColor: string
  color: string
}

export function RingKpi({ label, value, sub, percent, trackColor, color }: Props) {
  const clamped = Math.max(0, Math.min(100, percent))
  const offset = CIRCUMFERENCE * (1 - clamped / 100)

  return (
    <div className="kpi-card">
      <div className="kpi-text">
        <span className="kpi-label">{label}</span>
        <span className="kpi-value">{value}</span>
        <span className="kpi-sub">{sub}</span>
      </div>
      <svg width="56" height="56" viewBox="0 0 56 56" style={{ flex: '0 0 auto' }}>
        <circle cx="28" cy="28" r={RADIUS} fill="none" stroke={trackColor} strokeWidth="7" />
        <circle
          cx="28" cy="28" r={RADIUS} fill="none" stroke={color} strokeWidth="7"
          strokeLinecap="round"
          strokeDasharray={CIRCUMFERENCE}
          strokeDashoffset={offset}
          transform="rotate(-90 28 28)"
        />
        <text x="28" y="32" textAnchor="middle" fontSize="12" fontWeight="700" fill={color}>
          {Math.round(clamped)}%
        </text>
      </svg>
    </div>
  )
}
