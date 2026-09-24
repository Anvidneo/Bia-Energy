# Bia Energy — AI Energy Management Platform

Esqueleto del repo (jueves). Backend en Go (`backend/`), frontend en React
+ Vite (`frontend/`), orquestado con `docker-compose.yml`. Todavía sin
lógica de negocio — eso arranca el viernes (motor de detección + capa de
explicación + endpoints).

## Correr localmente

```
docker compose up --build
```

- Backend: http://localhost:8080/health
- Frontend: http://localhost:5173
