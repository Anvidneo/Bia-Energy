# Bia Energy — AI Energy Management Platform

Plataforma de gestión energética con detección de anomalías: ingiere lecturas horarias de medidores eléctricos, corre un motor de detección determinístico (estadística robusta, sin LLM) y explica cada hallazgo en lenguaje natural con una acción recomendada. Ciclo completo: **Datos → Análisis → Anomalía → Explicación → Priorización → Acción**.

- **Backend**: Go 1.24 + chi + Postgres — ver [`backend/README.md`](backend/README.md)
- **Frontend**: React 19 + TypeScript + Vite — ver [`frontend/README.md`](frontend/README.md)
- **Repo**: [github.com/Anvidneo/Bia-Energy](https://github.com/Anvidneo/Bia-Energy)

## Arquitectura

```
┌─────────────┐        HTTP/JSON         ┌──────────────┐        SQL        ┌────────────┐
│  Frontend   │ ───────────────────────▶ │   Backend    │ ─────────────────▶│  Postgres  │
│ React+Vite  │ ◀─────────────────────── │  Go + chi    │ ◀──────────────── │            │
└─────────────┘                          └──────────────┘                   └────────────┘
                                                 │
                                                 ▼
                                     internal/detection (motor)
                                     internal/ai (explicación)
```

El backend seedea `readings.csv`/`events.csv` en Postgres al arrancar (solo si las tablas están vacías), expone una API REST de 9 endpoints, y corre el motor de detección bajo demanda vía `POST /ai/analyze`. El frontend es una SPA sin router (cambio de pantalla por estado de React — dataset pequeño, no justifica una dependencia extra) que consume esa API.

## Correr localmente

### Con Docker (recomendado)

```
docker compose up --build
```

- Backend: http://localhost:8080/health
- Frontend: http://localhost:5173

### Sin Docker

```
# Postgres propio, luego:
cd backend && cp ../.env.example .env   # ajustar DATABASE_URL si aplica
go run ./cmd/api

# En otra terminal:
cd frontend && npm install && npm run dev
```

Detalles de cada servicio (endpoints, variables, testing) en sus propios README.

## Variables de entorno

Ver [`.env.example`](.env.example). Las relevantes hoy:

| Variable | Descripción | Default |
|---|---|---|
| `PORT` | Puerto HTTP del backend | `8080` |
| `DATABASE_URL` | DSN de Postgres | `postgres://bia:bia@localhost:5432/bia_energy?sslmode=disable` |
| `USE_LLM_EXPLAINER` | Activa el explicador LLM (stretch goal, no implementado — cae a `RuleExplainer` si se pone en `true`) | `false` |
| `VITE_API_URL` | URL del backend que consume el frontend | `http://localhost:8080` |

`LLM_API_KEY` y `FIREBASE_PROJECT_ID`/`FIREBASE_CREDENTIALS_JSON` están comentadas en `.env.example`: son para dos stretch goals documentados pero nunca construidos (ver "Alcance" abajo).

## CI/CD

`.github/workflows/ci-cd.yml`, en cada push/PR a `master`:

1. **test-backend** — `go build ./...` + `go test ./...` contra un Postgres de servicio.
2. **test-frontend** — `npm run test` (vitest) + `npm run build`.
3. **sonarcloud** — genera cobertura de ambos lados (`go test -coverprofile`, `vitest --coverage`), reescribe los paths del LCOV del frontend para que SonarCloud los resuelva desde la raíz del repo, corre el scan y **bloquea el deploy si el Quality Gate no pasa**.
4. **deploy-backend** (solo push a `master`, tras los tres jobs anteriores) — dispara el deploy hook de Render y hace polling de su API hasta que el servicio queda `live` (o falla el job si el deploy falla).
5. **deploy-frontend** (mismo trigger) — build y deploy con el CLI oficial de Vercel (`vercel pull` → `vercel build` → `vercel deploy --prebuilt`).

Commits: Conventional Commits reforzado con commitlint + husky (`.husky/commit-msg`) — ver `commitlint.config.cjs`.

## Motor de detección (resumen)

Ver el detalle completo en [`backend/README.md`](backend/README.md#motor-de-detección). En corto: cada medidor se evalúa con un baseline horario (mediana + MAD por hora del día) y un modified z-score; una desviación sostenida ≥3 horas es un "episodio", que se cruza con `events.csv` para decidir si es `REAL_ANOMALY`, `FALSE_POSITIVE` o `EXPLAINABLE_ANOMALY`; una inconsistencia eléctrica (consumo vs. voltaje×corriente×factor de potencia) sostenida es `DATA_QUALITY`. Los 4 casos conocidos del dataset (M-104, M-106, M-109, M-112) están codificados como test en `backend/internal/detection/engine_test.go`.

## Flujo de demo (5-10 min)

Login → Dashboard → Medidor M-109 → Ejecutar análisis IA → ver la anomalía detectada → abrir su detalle (explicación + acción recomendada).

## Alcance — qué se construyó y qué no

Construido y funcional: los 9 endpoints, el motor de detección completo, el explicador basado en reglas, el frontend completo (login, dashboard con KPIs y gráfica de serie temporal, tabla de medidores, detalle de medidor con resumen eléctrico, tabla y detalle de anomalías, ejecución de análisis con pipeline visual), CI/CD con deploy automático a Render (backend) y Vercel (frontend), y cobertura de tests real en ambos lados (no solo del código nuevo).

Dejado como stretch goal explícito, sin implementar (documentado en el propio código, no son bugs): un explicador basado en LLM (`internal/ai`, gateado por `USE_LLM_EXPLAINER`) y una integración de alertas a Firebase (`internal/firebase`, package vacío salvo su doc comment). Ninguno de los dos es requerido por el enunciado.

## Testing

```
cd backend && go test ./...
cd frontend && npm run test:coverage
```

Ver el detalle de cada suite en `backend/README.md` y `frontend/README.md`.
