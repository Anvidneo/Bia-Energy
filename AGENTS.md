# AGENTS.md

Guía para cualquier agente de IA (Claude Code, Cursor, Codex, etc.) que trabaje
o revise este repositorio. Si sos un agente entrando por primera vez, leé esto
antes de tocar código.

## Qué es este proyecto

**Bia Energy**: plataforma de gestión energética con detección de anomalías.
Ingiere lecturas horarias de medidores eléctricos (`data/readings.csv`,
`data/events.csv`), corre un motor de detección **100% determinístico**
(estadística robusta: mediana + MAD por hora del día, z-score modificado —
nunca un LLM) y expone los resultados vía una API REST consumida por un
dashboard React. Especificación completa de requisitos en
[`docs/SRS.md`](docs/SRS.md) (formato IEEE 830) — es la fuente de verdad sobre
alcance y comportamiento esperado; si vas a cambiar algo, verificá ahí primero
si hay un RF/RNF que lo cubra, y actualizalo si tu cambio lo vuelve obsoleto
(ver el caso RF-17/RNF-03 en el historial de commits como ejemplo).

## Estructura

```
backend/    Go 1.26 + chi + Postgres — API REST, motor de detección, explicadores
frontend/   React 19 + TypeScript + Vite — SPA sin router (una vista raíz, navegación por estado)
docs/SRS.md Especificación de requisitos (IEEE 830) — fuente de verdad de alcance
.github/    Workflow único de CI/CD (test + cobertura + SonarCloud + deploy)
```

Dentro de `backend/internal/`: `detection` (motor, determinístico), `ai`
(explicadores — `RuleExplainer` por defecto, `LLMExplainer` opcional vía
Gemini), `firebase` (alertas críticas opcionales), `api` (handlers HTTP),
`db`, `config`, `seed`, `models`. Cada paquete con lógica de negocio tiene su
`_test.go` al lado — no hay un paquete de tests centralizado.

## Cómo correrlo

```
docker compose up --build        # backend :8080, frontend :5173, Postgres incluido
```

Sin Docker: `cd backend && go run ./cmd/api` (con `DATABASE_URL` propio) y
`cd frontend && npm install && npm run dev`. Detalle completo en
[`backend/README.md`](backend/README.md) y [`frontend/README.md`](frontend/README.md).

## Cómo testear (obligatorio antes de commitear)

```
cd backend  && go build ./... && go test ./... -cover
cd frontend && npm run test        # o npm run test:coverage
```

**Regla del proyecto: cobertura real, no solo del código nuevo.** Ningún
paquete con lógica de negocio no trivial debe quedar en 0%. Si agregás una
función o rama nueva, agregale su test en el mismo cambio — no lo dejes para
después.

## Convenciones

- **Commits**: Conventional Commits, reforzado con commitlint + husky
  (`commitlint.config.cjs`, `.husky/commit-msg`). Formato: `tipo(scope):
  descripción` corta — sin cuerpo largo salvo que agregue información real.
- **Credenciales**: nunca en código ni en el repo. Todo vía variables de
  entorno (`.env`, ver [`.env.example`](.env.example)); en producción, vía las
  variables de entorno de Render/Vercel. Si una llamada externa falla y el
  error puede incluir la URL de la petición, verificá que la credencial no
  viaje en el query string (ver el fix de `internal/ai/gemini_client.go` como
  precedente — la API key de Gemini se movió de `?key=` a un header
  `x-goog-api-key` después de que una key apareciera en logs de error).
- **Capacidades opcionales** (`USE_LLM_EXPLAINER`, `USE_FIREBASE_ALERTS`):
  apagadas por defecto, aditivas — su ausencia o falla nunca debe romper el
  flujo principal ni propagar como error 5xx. Cualquier código nuevo en estas
  capas debe mantener esa degradación elegante (ver `RF-17`/`RNF-06`/`RNF-07`
  en el SRS).
- **CI/CD** (`.github/workflows/ci-cd.yml`): `test-backend` + `test-frontend`
  → `sonarcloud` (bloquea el deploy si el Quality Gate no pasa) →
  `deploy-backend` (Render) + `deploy-frontend` (Vercel), solo en push a
  `master`. Un cambio que rompa cualquiera de estos jobs no debe mergearse.

## Despliegue en vivo

| Servicio | URL |
|---|---|
| Frontend | https://bia-energy.juan-botero.dev |
| Backend / API | https://bia-energy-backend.onrender.com |
| Swagger | https://bia-energy-backend.onrender.com/swagger/index.html |

El login del frontend es únicamente visual (no hay backend de autenticación
en el alcance de esta prueba) — ver [`frontend/src/components/Login.tsx`](frontend/src/components/Login.tsx).
Cualquier correo/contraseña con formato válido deja entrar.

## Sobre el uso de IA en este proyecto

Este proyecto fue construido con la asistencia de **Claude Code / Claude
(Cowork)**, bajo la dirección y revisión directa de **Juan David Botero
Cabrera (Anvid)**, autor y responsable final de todo el código. Todos los
commits están firmados como `Anvidneo <Botero1400@gmail.com>`; cada cambio
generado con asistencia de IA fue revisado, compilado y testeado localmente
por Anvid antes de comitearse, y todo `git push`/deploy lo ejecutó él
manualmente — Claude no tiene credenciales de push a este repositorio ni
maneja API keys o tokens reales en ningún momento.

Ejemplos concretos de trabajo asistido en esta prueba: diagnóstico y arreglo
de una cadena de fallos de CI/CD (dependencias de `go.mod`, versión del
toolchain de Go, generación de Swagger, un hallazgo de SonarCloud sobre
instalación no verificada de dependencias en el Dockerfile); implementación
de `RuleExplainer`/`LLMExplainer` y del publicador de alertas a Firebase; la
corrección de seguridad de la API key mencionada arriba y la adición de
reintentos con backoff para errores transitorios de la API de Gemini
(429/503); y mantenimiento de `docs/SRS.md` para que quede sincronizado con
el comportamiento real del sistema.
