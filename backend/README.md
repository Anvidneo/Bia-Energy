# Backend — Bia Energy API

Go 1.24 + [chi](https://github.com/go-chi/chi) + Postgres. Motor de detección de anomalías 100% determinístico (estadística, sin LLM) y capa de explicación en lenguaje natural. Ver la arquitectura general en el [README raíz](../README.md).

## Estructura

```
cmd/api/            entrypoint: config → conexión DB → schema → seed → router
internal/config/    lee variables de entorno (PORT, DATABASE_URL, USE_LLM_EXPLAINER)
internal/db/        conexión Postgres (con auto-creación de la DB si no existe) + schema.sql
internal/seed/      parseo de readings.csv/events.csv e inserción idempotente
internal/models/    contrato de datos compartido (Meter, Reading, Event, Anomaly, ...)
internal/detection/ motor de detección determinístico (baseline.go + engine.go)
internal/ai/        capa de explicación (Reason + RecommendedAction a partir de la Evidence)
internal/api/       handlers HTTP (chi) — un archivo por recurso
internal/firebase/  stub sin implementar — stretch goal, ver "Alcance" en el README raíz
data/                readings.csv, events.csv (dataset del enunciado)
```

## Correr localmente

```
# Requiere un Postgres accesible en DATABASE_URL (ver .env.example en la raíz)
go run ./cmd/api
```

Al arrancar: conecta a Postgres (creando la base si no existe todavía), aplica `schema.sql` (idempotente, `CREATE TABLE IF NOT EXISTS`) y siembra `data/readings.csv`/`data/events.csv` solo si las tablas están vacías — seguro de re-ejecutar en cada restart de `docker compose`.

## API

Base URL por defecto: `http://localhost:8080`. Todas las respuestas son JSON; CORS abierto (`*`) para simplificar el demo.

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/health` | Liveness check |
| GET | `/dashboard/summary` | Total de medidores, anomalías activas, fecha del último análisis |
| GET | `/meters` | Lista de medidores |
| GET | `/meters/{id}` | Un medidor |
| GET | `/meters/{id}/readings` | Lecturas horarias de un medidor (404 si no existe o no tiene lecturas) |
| POST | `/ai/analyze` | Dispara una corrida del motor de detección + explicación (asíncrona, devuelve `analysisId`) |
| GET | `/ai/analysis/{id}` | Estado/resultado de una corrida (`pending`/`processing`/`done`/`error`), con el stage visual actual |
| GET | `/anomalies` | Todas las anomalías detectadas (de la última corrida — ver nota abajo) |
| GET | `/anomalies/{id}` | Una anomalía, con su `evidence` completa |

`POST /ai/analyze` corre en goroutine y avanza por 7 stages visuales (Lecturas → Baseline → Detección → Correlación → Eventos → Explicación → Recomendación) que el frontend muestra en tiempo real vía polling a `GET /ai/analysis/{id}`. Al terminar, **reemplaza** el contenido de la tabla `anomalies` con los resultados de esa corrida (modelo "la última corrida gana" — suficiente para este MVP de dataset estático; un sistema real guardaría historial por `analysisId`).

Forma de una `Anomaly` (sección 10 del enunciado):

```json
{
  "id": 1, "meter_id": "M-109", "detected_at": "...",
  "anomaly": true, "type": "REAL_ANOMALY", "severity": "HIGH", "confidence": 0.87,
  "reason": "...", "recommended_action": "...",
  "related_event_type": "OPERATIONAL_CHANGE", "related_event_at": "...",
  "evidence": {
    "baseline_median_kwh": 12.3, "observed_kwh": 34.5, "deviation_pct": 180.5,
    "z_score": 4.1, "consecutive_hours": 5, "expected_kwh_from_electrical": 12.8
  }
}
```

## Motor de detección

`internal/detection` — 100% reglas + estadística robusta (mediana + MAD, nunca media/desvest, para no ser sensible a outliers), reproducible y cubierto por tests unitarios contra el dataset real. Umbrales calibrados para clasificar correctamente los 4 casos conocidos sin falsos positivos en los 8 medidores normales.

1. **Baseline horario**: por cada hora del día (0-23), mediana y MAD de `consumption_kwh` a través de todos los días del medidor.
2. **Modified z-score**: `0.6745 × (x - mediana) / MAD` por lectura, contra el baseline de su propia hora.
3. **Señal de calidad de datos** (se evalúa primero): si el ratio `consumption_kwh / (voltage_v × current_a × power_factor / 1000)` tiene un z-score > 5.0 mientras el consumo en sí sigue cerca de su baseline (`|z_consumo| ≤ 3.5`), la lectura es "eléctricamente inconsistente". ≥3 lecturas así (no necesariamente consecutivas) → `DATA_QUALITY`, severidad `HIGH`.
4. **Señal de desviación de consumo**: lecturas con `|z_consumo| > 3.5` (y que no sean ya de calidad de datos) se agrupan; el episodio consecutivo más largo debe durar ≥3 horas para reportarse (filtra picos aislados de una hora). Si no hay ninguno, el medidor es `NORMAL` (no se reporta).
5. **Correlación con eventos**: el inicio del episodio se busca en `events.csv` dentro de una ventana de ±24h.
   - Evento `SCHEDULED_OUTAGE` cercano → `FALSE_POSITIVE`, severidad `LOW`.
   - Evento `OPERATIONAL_CHANGE` cercano → `EXPLAINABLE_ANOMALY`, severidad `MEDIUM`.
   - Sin evento (o uno que no aplica) → `REAL_ANOMALY`, severidad `HIGH`.
6. **Confianza**: `0.5 + 0.49 × clamp((|z| - umbral) / umbral, 0, 1)`, tope 0.99 — más lejos del umbral, más confianza.

Casos conocidos codificados en `internal/detection/engine_test.go::TestKnownCases`: M-104 → `EXPLAINABLE_ANOMALY`/Medium, M-106 → `FALSE_POSITIVE`/Low, M-109 → `REAL_ANOMALY`/High, M-112 → `DATA_QUALITY`/High.

## Explicación (`internal/ai`)

`Explainer` es una interfaz con una sola implementación real: `RuleExplainer`, que genera `Reason`/`RecommendedAction` con templates que citan las cifras de `Evidence` — determinístico, sin costo, sin latencia, reproducible en tests. El código deja el punto de extensión para un `LLMExplainer` (gateado por `USE_LLM_EXPLAINER`) pero ese explicador nunca se construyó: es un stretch goal explícito, no una funcionalidad rota.

## Modelo de datos

`schema.sql` (aplicado con `CREATE TABLE IF NOT EXISTS`, sin herramienta de migraciones — MVP):

- `meters(id)`
- `readings(id, meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor)` — único por `(meter_id, ts)`
- `events(id, meter_id, event_ts, event_type, description)` — único por `(meter_id, event_ts, event_type)`
- `anomalies(id, meter_id, detected_at, type, severity, confidence, reason, recommended_action, related_event_type, related_event_at, evidence_json, analysis_id, created_at)`

## Testing

```
go test ./...          # todos los paquetes
go test ./... -cover    # con resumen de cobertura
```

- `internal/detection/engine_test.go` — `TestKnownCases` (los 4 casos del enunciado), `TestUnremarkableMetersStayNormal`, `TestRunPipeline`.
- `internal/db/db_test.go` — test de integración contra Postgres real; se activa solo si `TEST_DATABASE_URL` está definida (así en CI, ver `.github/workflows/ci-cd.yml`) y se saltea localmente si no hay Postgres a mano.

## Deploy

Desplegado automáticamente a [Render](https://render.com) en cada push a `master` que pasa CI + Quality Gate de SonarCloud (ver el README raíz). La URL de producción vive en el dashboard de Render, no en este repo.
