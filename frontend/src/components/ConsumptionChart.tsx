import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Dot } from 'recharts'
import type { Reading } from '../types'

interface Props {
  readings: Reading[]
  anomalyTimestamps?: Set<string> // ISO timestamps to mark as anomalous points
  height?: number
  gradientId: string
}

function formatHour(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('es-CO', { hour: '2-digit', minute: '2-digit' })
}

export function ConsumptionChart({ readings, anomalyTimestamps, height = 170, gradientId }: Props) {
  const data = readings.map((r) => ({
    ts: r.timestamp,
    label: formatHour(r.timestamp),
    kwh: r.consumption_kwh,
    isAnomaly: anomalyTimestamps?.has(r.timestamp) ?? false,
  }))

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: -20 }}>
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--area-gradient-from)" stopOpacity="var(--area-gradient-from-opacity)" />
            <stop offset="100%" stopColor="var(--area-gradient-from)" stopOpacity="var(--area-gradient-to-opacity)" />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
        <XAxis
          dataKey="label"
          tick={{ fontSize: 10, fill: 'var(--text-2)' }}
          axisLine={{ stroke: 'var(--border)' }}
          tickLine={false}
          minTickGap={30}
        />
        <YAxis tick={{ fontSize: 10, fill: 'var(--text-2)' }} axisLine={false} tickLine={false} width={40} />
        <Tooltip
          formatter={(value: number) => [`${value.toFixed(2)} kWh`, 'Consumo']}
          labelFormatter={(label) => `Hora: ${label}`}
          contentStyle={{
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            fontSize: 12,
            color: 'var(--text-1)',
          }}
        />
        <Area
          type="monotone"
          dataKey="kwh"
          stroke="var(--accent)"
          strokeWidth={2.5}
          fill={`url(#${gradientId})`}
          dot={(props: any) =>
            props.payload.isAnomaly ? (
              <Dot key={props.key} cx={props.cx} cy={props.cy} r={5} fill="var(--crit)" stroke="none" />
            ) : (
              <g key={props.key} />
            )
          }
          activeDot={{ r: 5, fill: 'var(--accent)' }}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
