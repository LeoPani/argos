# ── Build stage ──────────────────────────────────────────────────────────────
# Use latest stable Go — go.mod declares go 1.26.x via toolchain directive.
FROM golang:alpine AS builder

WORKDIR /app

# Cache dependencies first (layer-efficient)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binaries — no libc dependency at runtime.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /argos-api     ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /argos-worker  ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /argos-migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /argos-seed    ./cmd/seed

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /argos-api      /app/argos-api
COPY --from=builder /argos-worker   /app/argos-worker
COPY --from=builder /argos-migrate  /app/argos-migrate
COPY --from=builder /argos-seed     /app/argos-seed
COPY migrations/                    /app/migrations/

ENV API_ADDR=:8080 \
    LOG_FORMAT=json \
    LOG_LEVEL=info

EXPOSE 8080

# Run migrations on boot, then start API. Override the entrypoint to run
# worker / seed in separate Fly.io machines or Railway services.
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["/app/argos-api"]
