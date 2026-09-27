# Especificación de Requisitos de Software (SRS)
## Bia Energy — Plataforma de Gestión Energética con Detección de Anomalías

**Versión:** 1.2
**Fecha:** 2026-09-27
**Estándar de referencia:** IEEE 830-1998

---

## 1. Introducción

### 1.1 Propósito

Este documento especifica los requisitos funcionales y no funcionales del sistema **Bia Energy**, una plataforma de gestión energética que ingiere lecturas horarias de medidores eléctricos, ejecuta un motor de detección de anomalías determinístico y presenta los resultados a través de un dashboard web, con explicaciones en lenguaje natural y alertas críticas opcionales.

El documento está dirigido a: el equipo de desarrollo (como referencia de contrato funcional), evaluadores de la prueba técnica, y cualquier persona que necesite entender el alcance exacto del sistema sin leer el código fuente.

### 1.2 Alcance

El sistema, identificado como **Bia Energy**, comprende:

- Un **backend** (Go) que expone una API REST de 9 endpoints, corre el motor de detección de anomalías sobre un dataset de lecturas y eventos, genera explicaciones en lenguaje natural (basadas en reglas o, opcionalmente, mediante un LLM) y puede publicar alertas críticas a un servicio externo (Firebase Firestore).
- Un **frontend** (React + TypeScript) que consume esa API: login, dashboard con KPIs y series temporales, listado y detalle de medidores, listado y detalle de anomalías, y ejecución interactiva del pipeline de análisis.
- Infraestructura de **CI/CD** que valida (tests + cobertura + quality gate) y despliega automáticamente ambos componentes.

Fuera de alcance: autenticación multi-usuario con roles, persistencia histórica de corridas de análisis (el sistema mantiene solo la última corrida), ingestión de datos en tiempo real desde medidores físicos (el dataset es estático, cargado desde CSV), y facturación o integración con sistemas de nómina energética.

### 1.3 Definiciones, acrónimos y abreviaturas

| Término | Definición |
|---|---|
| **Anomalía** | Desviación del consumo de un medidor respecto a su patrón horario esperado (baseline), clasificada por el motor de detección. |
| **Baseline** | Consumo horario esperado de un medidor, calculado como la mediana de sus lecturas históricas para esa hora del día. |
| **MAD** | Median Absolute Deviation — medida robusta de dispersión estadística usada para normalizar desviaciones sin ser sensible a outliers. |
| **Z-score modificado** | `0.6745 × (x − mediana) / MAD`; cuantifica cuántas "desviaciones típicas robustas" se aleja una lectura de su baseline. |
| **Evidencia (Evidence)** | Conjunto de cifras crudas calculadas por el motor de detección (consumo observado, baseline, desviación %, z-score, horas consecutivas) que respaldan la clasificación de una anomalía. |
| **Explicador (Explainer)** | Componente que traduce el `Type`/`Severity`/`Evidence` de una anomalía a texto en lenguaje natural (`Reason`, `RecommendedAction`). |
| **SRS** | Software Requirements Specification (este documento). |
| **RF** | Requisito Funcional. |
| **RNF** | Requisito No Funcional. |
| **LLM** | Large Language Model — modelo de lenguaje usado opcionalmente para redactar explicaciones. |
| **CSV** | Comma-Separated Values — formato del dataset de entrada (`readings.csv`, `events.csv`). |

### 1.4 Referencias

- `README.md` — arquitectura general, diagrama, CI/CD, alcance.
- `backend/README.md` — estructura del backend, motor de detección, endpoints, capas de explicación y alertas.
- `frontend/README.md` — estructura del frontend, decisiones de diseño.
- `.env.example` — variables de entorno documentadas.
- IEEE Std 830-1998, *IEEE Recommended Practice for Software Requirements Specifications*.

### 1.5 Resumen del documento

La sección 2 describe el sistema a alto nivel (perspectiva, funciones, usuarios, restricciones). La sección 3 detalla los requisitos específicos: interfaces externas, requisitos funcionales por endpoint/capacidad, y requisitos no funcionales. La sección 4 contiene apéndices con el modelo de datos y la matriz de trazabilidad.

---

## 2. Descripción general

### 2.1 Perspectiva del producto

Bia Energy es un sistema independiente (no se integra con un ERP o SCADA externo en esta versión). Su ciclo de vida funcional es:

```
Datos (CSV) → Análisis (detección + explicación) → Anomalía → Priorización (severidad) → Acción (recomendación / alerta)
```

El backend es stateless respecto a las corridas de análisis (guarda solo la última) y stateful respecto a los datos base (medidores, lecturas, eventos, anomalías) en Postgres. El frontend es una SPA que consume la API vía HTTP/JSON.

### 2.2 Funciones del producto (resumen)

1. Servir lecturas y eventos desde un dataset CSV precargado.
2. Calcular un baseline horario robusto por medidor y detectar desviaciones sostenidas.
3. Clasificar cada desviación en `REAL_ANOMALY`, `FALSE_POSITIVE`, `EXPLAINABLE_ANOMALY` o `DATA_QUALITY`, cruzando con eventos operativos conocidos.
4. Asignar severidad (`HIGH`/`MEDIUM`/`LOW`) y confianza a cada anomalía.
5. Generar una explicación en lenguaje natural y una acción recomendada por anomalía (basada en reglas, o vía LLM si está habilitado).
6. Publicar alertas críticas a un canal externo (Firestore) cuando la severidad es `HIGH` (si está habilitado).
7. Exponer los resultados vía API REST y visualizarlos en un dashboard web con KPIs, gráfica de serie temporal, tablas y vistas de detalle.
8. Permitir disparar y monitorear (polling) una corrida de análisis desde la UI, con progreso visual por etapas.

### 2.3 Características de los usuarios

| Tipo de usuario | Descripción | Nivel técnico esperado |
|---|---|---|
| Analista de operaciones energéticas | Usa el dashboard para monitorear medidores, revisar anomalías y decidir acciones. | No técnico; interactúa solo vía UI web. |
| Evaluador / revisor técnico | Revisa código, documentación, cobertura de tests y despliegue. | Técnico (desarrollador de software). |

No hay diferenciación de roles/permisos dentro de la aplicación en esta versión: existe una pantalla de login, pero no hay un modelo de autorización granular.

### 2.4 Restricciones

- El motor de detección es **100% determinístico** (estadística clásica) por requisito explícito del enunciado: no puede depender de un LLM para clasificar.
- El dataset de lecturas/eventos es estático (`data/readings.csv`, `data/events.csv`), cargado una vez al iniciar el backend si las tablas están vacías.
- Backend: Go 1.24, Postgres, framework de ruteo `chi`.
- Frontend: React + TypeScript, Vite, sin librería de ruteo (una sola vista raíz con navegación por estado).
- Despliegue: Render (backend) y Vercel (frontend), vía GitHub Actions.
- Las credenciales de terceros (LLM, Firebase) nunca se manejan en texto plano en el repositorio ni en el código: solo vía variables de entorno.

### 2.5 Supuestos y dependencias

- Se asume que el dataset provisto (`readings.csv`/`events.csv`) es representativo y no requiere limpieza adicional más allá de la validación de parseo ya implementada.
- El explicador LLM y las alertas de Firebase son **capacidades opcionales, apagadas por defecto**: el sistema debe funcionar correctamente sin ellas (RNF-06).
- Se asume disponibilidad de un proveedor de LLM externo (Gemini) y de un proyecto de Firebase cuando esas capacidades se activan; su indisponibilidad no debe afectar el flujo principal (RF-14, RF-17).

---

## 3. Requisitos específicos

### 3.1 Requisitos de interfaces externas

#### 3.1.1 Interfaces de usuario

- **UI-01**: El sistema debe proveer una pantalla de login como puerta de entrada al dashboard.
- **UI-02**: El dashboard debe mostrar, sin necesidad de navegación adicional: KPIs agregados (medidores monitoreados, consumo total, anomalías activas, alertas críticas, confianza de IA, calidad de datos), una gráfica de consumo por medidor (últimas 24h) y un listado de anomalías recientes.
- **UI-03**: Debe existir una vista de detalle de medidor con resumen eléctrico y baseline.
- **UI-04**: Debe existir una vista de detalle de anomalía con la explicación completa y la acción recomendada.
- **UI-05**: La ejecución de un análisis debe mostrar el progreso visual por etapas (Lecturas → Baseline → Detección → Correlación → Eventos → Explicación → Recomendación).
- **UI-06**: La interfaz debe ser utilizable en viewport móvil (responsive), sin bugs de zoom no deseado ni elementos visuales superpuestos.

#### 3.1.2 Interfaces de hardware

No aplica — el sistema no interactúa directamente con medidores físicos; consume datos ya digitalizados en formato CSV.

#### 3.1.3 Interfaces de software

- **SI-01**: El backend debe exponer una API REST sobre HTTP/1.1, con payloads JSON (`Content-Type: application/json`).
- **SI-02**: El backend debe conectarse a una base de datos PostgreSQL vía el driver `lib/pq`.
- **SI-03**: Cuando `USE_LLM_EXPLAINER=true`, el backend debe comunicarse con la API REST de Google Gemini (`generativelanguage.googleapis.com`) vía HTTPS.
- **SI-04**: Cuando `USE_FIREBASE_ALERTS=true`, el backend debe comunicarse con Google Cloud Firestore vía el SDK oficial de Firebase Admin (Go).
- **SI-05**: El frontend debe consumir la API del backend vía `fetch`, configurable mediante `VITE_API_URL`.
- **SI-06**: El sistema debe exponer su documentación de API en formato OpenAPI/Swagger, generada dinámicamente desde el código fuente (ver RF-19).

#### 3.1.4 Interfaces de comunicación

- **CI-01**: Todas las comunicaciones externas (Gemini, Firestore) deben usar HTTPS/TLS.
- **CI-02**: El backend debe habilitar CORS (`Access-Control-Allow-Origin: *`) para permitir que el frontend, servido desde otro dominio (Vercel), consuma la API (Render).

### 3.2 Requisitos funcionales

Cada requisito funcional está identificado con un ID único, trazable a un endpoint o componente concreto (ver matriz de trazabilidad, Apéndice B).

#### Ingesta y datos base

- **RF-01**: El sistema debe parsear `readings.csv` y `events.csv`, validando tipos de dato (timestamps, numéricos) y reportando errores de formato sin corromper datos ya insertados.
- **RF-02**: La carga de datos debe ser idempotente: reiniciar el backend con las tablas ya pobladas no debe duplicar filas.
- **RF-03**: El sistema debe exponer `GET /health` para verificación de disponibilidad del servicio.
- **RF-04**: El sistema debe exponer `GET /meters` (listado de medidores) y `GET /meters/{id}` (detalle de un medidor).
- **RF-05**: El sistema debe exponer `GET /meters/{id}/readings` con las lecturas históricas de un medidor.
- **RF-06**: El sistema debe exponer `GET /dashboard/summary` con los KPIs agregados que consume la pantalla principal.

#### Motor de detección

- **RF-07**: El sistema debe calcular, para cada medidor y cada hora del día, un baseline robusto (mediana de consumo histórico para esa hora).
- **RF-08**: El sistema debe calcular el z-score modificado (`0.6745 × (x − mediana) / MAD`) de cada lectura respecto a su baseline horario.
- **RF-09**: El sistema debe considerar una desviación como "episodio" solo cuando es sostenida por al menos 3 horas consecutivas (`MinConsecutiveDeviationHours`), para evitar falsos positivos por picos aislados.
- **RF-10**: El sistema debe detectar inconsistencias de calidad de datos comparando el consumo reportado contra el estimado por voltaje × corriente × factor de potencia, usando un umbral de z-score de ratio (`RatioZThreshold = 5.0`).
- **RF-11**: El sistema debe cruzar cada episodio de desviación con `events.csv` para clasificarlo como:
  - `REAL_ANOMALY` — sin evento operativo que lo explique.
  - `FALSE_POSITIVE` — coincide con un `SCHEDULED_OUTAGE`.
  - `EXPLAINABLE_ANOMALY` — coincide con un `OPERATIONAL_CHANGE`.
  - `DATA_QUALITY` — inconsistencia eléctrica sostenida, independiente de eventos.
- **RF-12**: El sistema debe asignar una severidad (`HIGH`/`MEDIUM`/`LOW`) y un nivel de confianza (0–1) a cada anomalía detectada.
- **RF-13**: El motor de detección debe ser 100% determinístico: la misma entrada debe producir siempre la misma clasificación (sin uso de LLM en esta capa).

#### Explicación en lenguaje natural

- **RF-14**: El sistema debe generar, para cada anomalía, un `Reason` (explicación) y un `RecommendedAction` (acción recomendada) en español, citando las cifras de la evidencia (nunca inventando números).
- **RF-15**: Por defecto, la explicación debe generarse mediante plantillas determinísticas (`RuleExplainer`), sin dependencia de servicios externos.
- **RF-16**: Cuando `USE_LLM_EXPLAINER=true` y hay una `LLM_API_KEY` configurada, el sistema debe usar un explicador basado en LLM (`LLMExplainer`, Gemini) para reformular el texto de forma más natural, sin cambiar la clasificación ni las cifras.
- **RF-17**: Si la llamada al LLM falla, excede el timeout (20s por defecto, con hasta 3 intentos y backoff exponencial ante errores transitorios — 429/503) o devuelve una respuesta no interpretable, el sistema debe usar automáticamente la explicación basada en reglas para esa anomalía puntual, sin interrumpir el resto del pipeline.

#### Alertas críticas

- **RF-18**: Cuando `USE_FIREBASE_ALERTS=true` y hay credenciales de Firebase configuradas, el sistema debe publicar cada anomalía de severidad `HIGH` como un documento en la colección `critical_alerts` de Firestore, inmediatamente después de persistirla en la base de datos.
- **RF-18b**: Un error al publicar en Firebase debe registrarse en el log y descartarse; nunca debe propagarse como error HTTP ni interrumpir el análisis en curso.

#### Pipeline de análisis (orquestación)

- **RF-19**: El sistema debe exponer `POST /ai/analyze` para iniciar una corrida de análisis de forma asíncrona, devolviendo un `analysisId` inmediatamente (HTTP 202).
- **RF-20**: El sistema debe exponer `GET /ai/analysis/{id}` para consultar (polling) el estado de una corrida (`pending`/`processing`/`done`/`error`) y su etapa actual.
- **RF-21**: Al completarse, una corrida debe reemplazar el contenido de la tabla de anomalías con los resultados de esa corrida ("last run wins").
- **RF-22**: El sistema debe exponer `GET /anomalies` (listado) y `GET /anomalies/{id}` (detalle) sobre la última corrida persistida.
- **RF-22b**: El listado de `GET /anomalies` debe priorizarse por severidad (`HIGH` > `MEDIUM` > `LOW`), usando confianza y luego fecha de detección como desempate — no debe ser un orden puramente cronológico. El panel "Anomalías recientes" del dashboard (UI-02) es la única excepción intencional: al ser un feed de actividad reciente, ordena explícitamente por fecha de detección en el frontend, independiente del orden que entregue la API.

#### Documentación de API

- **RF-23**: El sistema debe generar y exponer documentación interactiva de su API (Swagger/OpenAPI) directamente desde anotaciones en el código fuente, de modo que se mantenga sincronizada con los handlers reales sin edición manual de un spec separado.
- **RF-24**: La documentación Swagger debe ser accesible vía navegador en una ruta dedicada del backend (`/swagger/index.html`) y debe listar los 9 endpoints con sus parámetros, cuerpos de petición/respuesta y códigos de estado posibles.

### 3.3 Requisitos de rendimiento

- **RNF-01**: Una corrida completa de análisis sobre el dataset provisto (≈4000 lecturas) debe finalizar en menos de 5 segundos en condiciones normales (excluyendo la latencia de un LLM externo, si está habilitado).
- **RNF-02**: Los endpoints de solo lectura (`GET /meters`, `GET /anomalies`, `GET /dashboard/summary`, etc.) deben responder en menos de 500ms bajo carga normal (dataset de referencia, sin concurrencia significativa).
- **RNF-03**: El timeout de la llamada al LLM externo (incluyendo reintentos) no debe exceder 20 segundos por anomalía. Como la explicación corre en el pipeline asíncrono de `POST /ai/analyze` (RF-19), este presupuesto no afecta la latencia percibida por el usuario, que solo hace polling del estado.

### 3.4 Restricciones de diseño

- **RNF-04**: El motor de detección no debe usar ningún componente probabilístico o basado en LLM; debe ser auditable y reproducible.
- **RNF-05**: Las credenciales de terceros (API keys, JSON de service account) deben leerse exclusivamente de variables de entorno; ninguna debe aparecer en el código fuente ni en el control de versiones.
- **RNF-06**: Las funcionalidades de LLM y Firebase deben ser aditivas y estar apagadas por defecto (`false`); su ausencia o mala configuración no debe impedir que el resto del sistema funcione (degradación elegante).

### 3.5 Atributos de calidad del sistema

- **RNF-07 (Confiabilidad)**: Un fallo en un componente opcional (LLM, Firebase) nunca debe provocar un error 5xx en la API ni detener una corrida de análisis en curso.
- **RNF-08 (Disponibilidad)**: El backend debe recuperarse automáticamente de una caída de conexión a base de datos sin intervención manual (reintento de conexión al reiniciar el proceso).
- **RNF-09 (Seguridad)**: Ninguna clave, token o credencial debe registrarse en logs en texto plano.
- **RNF-10 (Mantenibilidad)**: El código debe mantener cobertura de tests real (no solo de código nuevo) en todos los paquetes del backend que contienen lógica de negocio no trivial.
- **RNF-11 (Portabilidad)**: El backend debe ejecutarse de forma idéntica en local (docker-compose) y en el entorno de despliegue (Render), sin cambios de código entre ambos.
- **RNF-12 (Usabilidad)**: La interfaz debe funcionar correctamente tanto en desktop como en dispositivos móviles, sin bugs de layout conocidos (zoom no deseado, elementos superpuestos).

---

## 4. Apéndices

### Apéndice A — Modelo de datos (resumen)

| Entidad | Campos principales |
|---|---|
| `Meter` | `id` |
| `Reading` | `meter_id`, `timestamp`, `consumption_kwh`, `voltage_v`, `current_a`, `power_factor` |
| `Event` | `meter_id`, `event_timestamp`, `event_type` (`OPERATIONAL_CHANGE`/`SCHEDULED_OUTAGE`/`DATA_QUALITY`/`UNKNOWN`), `description` |
| `Evidence` | `baseline_median_kwh`, `observed_kwh`, `deviation_pct`, `z_score`, `consecutive_hours`, `expected_kwh_from_electrical` |
| `Anomaly` | `id`, `meter_id`, `detected_at`, `type`, `severity`, `confidence`, `reason`, `recommended_action`, `related_event_type`, `related_event_at`, `evidence` |
| `AnalysisResult` | `id`, `status`, `stage`, `started_at`, `finished_at`, `error`, `anomalies[]` |

### Apéndice B — Matriz de trazabilidad (endpoint → requisitos)

| Endpoint | Requisitos funcionales |
|---|---|
| `GET /health` | RF-03 |
| `GET /dashboard/summary` | RF-06 |
| `GET /meters` | RF-04 |
| `GET /meters/{id}` | RF-04 |
| `GET /meters/{id}/readings` | RF-05 |
| `POST /ai/analyze` | RF-07 a RF-21 (dispara todo el pipeline) |
| `GET /ai/analysis/{id}` | RF-20 |
| `GET /anomalies` | RF-22 |
| `GET /anomalies/{id}` | RF-22 |
| `/swagger/index.html` | RF-23, RF-24 |

### Apéndice C — Historial de revisiones

| Versión | Fecha | Cambios |
|---|---|---|
| 1.0 | 2026-09-26 | Versión inicial, cubre el sistema tal como está implementado (incluye LLM explainer y alertas Firebase como capacidades opcionales). |
| 1.1 | 2026-09-27 | Actualiza RF-17/RNF-03: timeout del LLM subido de 8s a 20s con reintentos (backoff exponencial ante 429/503), sin impacto en la latencia percibida por correr en el pipeline asíncrono. |
| 1.2 | 2026-09-27 | Agrega RF-22b: `GET /anomalies` pasa de orden cronológico a priorizado por severidad/confianza/fecha; el panel "Anomalías recientes" del dashboard mantiene explícitamente orden cronológico como excepción documentada. |
