# --- build stage ---
FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
# npm ci (not install) locks resolved versions to package-lock.json, and
# --ignore-scripts stops any dependency's install/postinstall lifecycle
# scripts from running arbitrary code during this build.
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# --- runtime stage: static files served by nginx, running as a non-root user ---
FROM nginxinc/nginx-unprivileged:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
