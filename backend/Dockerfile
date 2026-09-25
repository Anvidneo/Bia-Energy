# --- build stage ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
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
