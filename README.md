# Bia Energy — AI Energy Management Platform

Plataforma de gestión energética con detección de anomalías: ingiere lecturas horarias de medidores eléctricos, corre un motor de detección determinístico (estadística robusta, sin LLM) y explica cada hallazgo en lenguaje natural con una acción recomendada. Ciclo completo: **Datos → Análisis → Anomalía → Explicación → Priorización → Acción**.

- **Backend**: Go 1.24 + chi + Postgres — ver [`backend/README.md`](backend/README.md)
- **Frontend**: React 19 + TypeScript + Vite — ver [`frontend/README.md`](frontend/README.md)
- **Repo**: [github.com/Anvidneo/Bia-Energy](https://github.com/Anvidneo/Bia-Energy)

## Despliegue en vivo

| Servicio | URL |
|---|---|
| Frontend | https://bia-energy.juan-botero.dev |
| Backend / API | https://bia-energy-backend.onrender.com |
| Documentación (Swagger) | https://bia-energy-backend.onrender.com/swagger/index.html |

CI/CD despliega automáticamente a ambos en cada push a `master` que pase tests + cobertura + Quality Gate de SonarCloud (ver sección CI/CD más abajo).

> **Login**: la pantalla de inicio de sesión es únicamente visual (ver [`frontend/src/components/Login.tsx`](frontend/src/components/Login.tsx)) — no hay backend de autenticación en el alcance de esta prueba. Cualquier correo/contraseña con formato válido (no vacíos) deja entrar al dashboard; el objetivo es que la app se sienta como un producto real y no una colección de pantallas sueltas.

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
| `USE_LLM_EXPLAINER` | Activa `ai.LLMExplainer` (Gemini) en vez de `RuleExplainer`; requiere `LLM_API_KEY` | `false` |
| `LLM_API_KEY` | API key de Gemini (free tier, [Google AI Studio](https://aistudio.google.com/apikey)) | — |
| `LLM_MODEL` | Modelo de Gemini a usar | `gemini-2.5-flash` |
| `USE_FIREBASE_ALERTS` | Publica cada anomalía `HIGH` a Firestore vía `firebase.Publisher`; requiere `FIREBASE_PROJECT_ID`/`FIREBASE_CREDENTIALS_JSON` | `false` |
| `FIREBASE_PROJECT_ID` | ID del proyecto de Firebase | — |
| `FIREBASE_CREDENTIALS_JSON` | Contenido completo (una sola línea) del JSON de una service account con permiso de escritura en Firestore | — |
| `VITE_API_URL` | URL del backend que consume el frontend | `http://localhost:8080` |

Ambos flags están apagados por defecto y son aditivos: si están en `true` pero faltan sus credenciales, el backend loguea una advertencia al arrancar y sigue funcionando sin ellos (`RuleExplainer` / sin alertas) en vez de fallar. Las credenciales reales nunca se commitean — van solo en `.env` local y en las variables de entorno de Render.

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

Login (cualquier correo/contraseña — ver nota en "Despliegue en vivo") → Dashboard → Medidor M-109 → Ejecutar análisis IA → ver la anomalía detectada → abrir su detalle (explicación + acción recomendada).

## Alcance — qué se construyó y qué no

Construido y funcional: los 9 endpoints, el motor de detección completo, el explicador basado en reglas, el frontend completo (login, dashboard con KPIs y gráfica de serie temporal, tabla de medidores, detalle de medidor con resumen eléctrico, tabla y detalle de anomalías, ejecución de análisis con pipeline visual), documentación interactiva de la API vía Swagger generado dinámicamente desde el código (`swaggo/swag`, ver [`backend/README.md`](backend/README.md#documentación-interactiva-swagger)), un SRS formal (IEEE 830) en [`docs/SRS.md`](docs/SRS.md), CI/CD con deploy automático a Render (backend) y Vercel (frontend), y cobertura de tests real en ambos lados (no solo del código nuevo).

Construido como extensión opcional, apagada por defecto (ninguna es requerida por el enunciado): `ai.LLMExplainer` (`internal/ai/llm_explainer.go` + `gemini_client.go`, gateado por `USE_LLM_EXPLAINER`) rephrasea el `Reason`/`RecommendedAction` del explicador de reglas vía Gemini sin nunca reclasificar ni inventar cifras — cualquier falla (red, timeout, respuesta no parseable) cae de vuelta a `RuleExplainer` para esa anomalía puntual; y `firebase.Publisher` (`internal/firebase`, gateado por `USE_FIREBASE_ALERTS`) publica cada anomalía `HIGH` a la colección `critical_alerts` de Firestore justo después de que `persistAnomalies` confirma en la base de datos — cualquier error de Firebase se loguea y se descarta, nunca interrumpe el pipeline. Ambos usan el SDK/API oficiales, no lógica de auth hecha a mano.

## Testing

```
cd backend && go test ./...
cd frontend && npm run test:coverage
```

Ver el detalle de cada suite en `backend/README.md` y `frontend/README.md`.
