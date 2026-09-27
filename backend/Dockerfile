# --- build stage ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# server.go blank-imports the generated Swagger docs package (see
# internal/api/server.go) — it's never committed (see .gitignore), so it
# must be generated here before the build, same as in CI (ci-cd.yml).
# swag is a go.mod `tool` dependency (see go.mod's `tool` block) — `go tool`
# resolves and verifies it against go.sum instead of an ad-hoc `go install
# ...@version`, which SonarCloud flags (docker:S7031) as an unpredictable,
# unverified dependency install.
RUN go tool swag init -g cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# --- runtime stage ---
FROM alpine:3.20
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=build /out/api /usr/local/bin/api
# seed.Load reads these via the relative paths "data/readings.csv" /
# "data/events.csv" (see cmd/api/main.go) — must ship with the image since
# there's no volume mount in a Render/Docker deploy.
COPY --from=build /src/data ./data
USER appuser
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
