# AGENTS.md — Uso de IA en el desarrollo de este proyecto

Este documento es una divulgación transparente para quien revise esta prueba técnica:
**Bia Energy fue construido con la asistencia de Claude Code / Claude (Cowork)**,
la herramienta de desarrollo agéntico de Anthropic, bajo la dirección y revisión
directa de **Juan David Botero Cabrera (Anvid)**, autor y responsable final de
todo el código, las decisiones de arquitectura y lo que se entrega.

## Qué significa esto en la práctica

- **Autoría y responsabilidad**: todos los commits del repositorio están firmados
  como `Anvidneo <Botero1400@gmail.com>`. Cada cambio generado con asistencia de
  Claude fue revisado, entendido y aprobado antes de ser comiteado y pusheado —
  Claude no tiene credenciales de push a este repositorio; cada `git push` lo
  ejecutó Anvid manualmente, típicamente después de correr `go build`/`go test`
  localmente para confirmar que el cambio compila y pasa.
- **Qué se hizo con asistencia de IA**: una parte significativa del código
  (backend en Go, frontend en React/TypeScript, motor de detección, capas
  opcionales de LLM/Firebase, pipeline de CI/CD, y la documentación —
  `README.md` y `docs/SRS.md`) se escribió de forma colaborativa con Claude
  Code, mediante iteración: Anvid describía el requisito o el problema, Claude
  proponía o implementaba el cambio, y Anvid lo validaba (build, tests locales,
  revisión de logs de producción en Render/GitHub Actions/SonarCloud) antes de
  aceptarlo.
- **Ejemplos concretos de esta prueba**: diagnóstico y arreglo de una cadena de
  fallos de CI/CD (dependencias de `go.mod`, versión del toolchain de Go,
  generación de Swagger, un hallazgo de SonarCloud sobre instalación no
  verificada de dependencias en el Dockerfile); implementación de los
  explicadores `RuleExplainer`/`LLMExplainer` y del publicador de alertas a
  Firebase; una corrección de seguridad (la API key de Gemini viajaba en la
  URL y quedaba expuesta en logs de error — se movió a un header) y la
  adición de reintentos con backoff para errores transitorios de la API de
  Gemini (429/503); y mantenimiento del SRS (`docs/SRS.md`) para que quede
  sincronizado con el comportamiento real del sistema.
- **Lo que NO hizo la IA**: no tomó decisiones de negocio ni de alcance por su
  cuenta, no manejó credenciales reales (API keys, tokens) en ningún momento
  — esas siempre las ingresó Anvid directamente en Render/GitHub/Google AI
  Studio —, y no hizo push ni deploy de nada sin que Anvid lo ejecutara y
  verificara.

## Por qué se documenta esto

El objetivo es que quien evalúe esta prueba tenga el contexto completo: el
código refleja cómo Anvid trabaja hoy como desarrollador —usando herramientas
de IA como un acelerador dentro de un flujo donde el criterio técnico, la
revisión y la decisión final siguen siendo humanas—, no una entrega generada
sin supervisión.
