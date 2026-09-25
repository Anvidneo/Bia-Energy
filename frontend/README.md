# Frontend — Bia Energy

React 19 + TypeScript + Vite. Consume la API del backend Go (ver [`../backend/README.md`](../backend/README.md)) y presenta el flujo completo del enunciado: Login → Dashboard → Medidores → Ejecutar análisis IA → Anomalía → Explicación → Acción. Ver la arquitectura general en el [README raíz](../README.md).

## Correr localmente

```
npm install
npm run dev       # http://localhost:5173, apunta a VITE_API_URL (ver .env.example en la raíz)
npm run build     # tsc -b && vite build
npm run test:coverage
npm run lint
```

## Estructura

```
src/
  api.ts                 un fetch tipado por endpoint del backend (9 rutas)
  App.tsx                gate de login + cambio de pantalla por estado (sin router)
  hooks.ts                useTheme (persistido en localStorage), useMediaQuery (breakpoint mobile)
  search.ts               parseo del buscador del topbar ("M-110", "110" → id de medidor)
  types.ts                tipos compartidos con el contrato del backend
  components/
    Login.tsx              formulario de acceso (gate de sesión)
    Layout.tsx              sidebar + topbar (búsqueda, tema, logout) + navegación
    Dashboard.tsx           KPIs + gráfica de serie temporal + anomalías recientes
    MetersTable.tsx         tabla de medidores (ordenable/filtrable)
    MeterDetail.tsx         resumen eléctrico de un medidor + su historial de anomalías
    ConsumptionChart.tsx    gráfica de área (recharts) de consumo horario
    RingKpi.tsx             tarjeta de KPI tipo anillo de progreso
    AnomaliesTable.tsx      tabla completa de anomalías
    AnomalyDetail.tsx       detalle de una anomalía: evidencia, explicación, acción recomendada
    AnalysisRunner.tsx      dispara POST /ai/analyze y hace polling del pipeline visual
    meters.ts / meterStats.ts  lógica pura compartida (consumo, baseline, variación %) — sin duplicar entre Dashboard/MetersTable/MeterDetail
    severity.ts             formato y estilos compartidos (badges, colores, formatKwh/formatPct)
```

## Pantallas

- **Login**: correo + contraseña (validación de campos no vacíos, sin backend de auth real — gate de demo). La sesión se guarda en `sessionStorage` (`bia-auth`); persiste al recargar, se limpia al cerrar sesión.
- **Dashboard**: KPIs (total de medidores, anomalías activas, consumo total del periodo, confianza promedio de la IA), gráfica de serie temporal (últimas 24h del medidor con la anomalía más reciente, o el primer medidor si no hay ninguna) y la tabla de anomalías recientes.
- **Medidores**: tabla ordenable/filtrable con conteo de anomalías, última severidad/tipo, consumo actual y variación % vs. baseline. Click en un medidor abre su detalle.
- **Detalle de medidor**: tarjeta de resumen (estado, consumo actual, baseline aproximado, variación %, voltaje, corriente, factor de potencia) + gráfica de consumo + historial de anomalías de ese medidor.
- **Anomalías**: tabla completa, cada fila abre el detalle.
- **Detalle de anomalía**: evidencia cruda del motor de detección, explicación en lenguaje natural y acción recomendada (`RuleExplainer` del backend), con link de vuelta al medidor.
- **Ejecutar análisis IA**: dispara `POST /ai/analyze` y hace polling de `GET /ai/analysis/{id}`, mostrando el pipeline visual de 7 etapas (Lecturas → Baseline → Detección → Correlación → Eventos → Explicación → Recomendación) que el backend reporta.

## Decisiones de diseño

- **Sin router**: cambio de pantalla por estado de React (`App.tsx`). El dataset y el alcance del MVP son pequeños — deep links y back/forward no justifican una dependencia adicional junto al backend en Go.
- **Cálculos de consumo/variación compartidos**: `meters.ts` es el módulo base sin dependencias; `meterStats.ts` lo importa y reexporta. Así Dashboard, MetersTable y MeterDetail nunca pueden mostrar números distintos para el mismo medidor.
- **Tema claro/oscuro**: persistido en `localStorage`, con fallback a `prefers-color-scheme`.

## Testing

```
npm run test            # vitest run
npm run test:coverage   # con cobertura v8
```

Cobertura completa del código de producción (componentes, lógica pura, hooks, API client), no solo del código nuevo en cada cambio — cada archivo `*.ts(x)` en `src/` tiene su `*.test.ts(x)` correspondiente. Se usa `@testing-library/react` + `jsdom`; `oxlint` para linting.

## Deploy

Desplegado automáticamente a [Vercel](https://vercel.com) en cada push a `master` que pasa CI + Quality Gate de SonarCloud (ver el README raíz). La URL de producción vive en el dashboard de Vercel, no en este repo.
