# syntax=docker/dockerfile:1

# Multi-stage build producing a single image with all six backend binaries
# (gateway, gateway-v2, control-api, worker, migrate, seed). Compose services
# select their binary via `command` + `AIGW_SERVER_ADDR`.
#
# Context MUST be the repo root (docker-compose.yml sets context: ../..) so the
# root .dockerignore and the backend module are both available.

FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache module downloads first (go.sum committed).
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# -o /out/ with multiple main packages: one binary per package dir (entry name).
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S -G app app \
    && mkdir -p /data && chown app:app /data

COPY --from=build /out/ /app/

# Non-root; AIGW_SECRETS_FILE defaults to /data/secrets.json (writable above).
USER app
EXPOSE 8080

# No ENTRYPOINT: compose sets command: ["/app/<binary>"].