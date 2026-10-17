# Sentinel AI Gateway / AI Governance — Local Setup & Developer Guide

## 1. Prerequisites

Before running Sentinel locally, ensure the host machine has the following installed:
- **Go**: 1.25+ (`go version`)
- **Node.js**: 20+ (`node -v`) & **npm**: 10+ (`npm -v`)
- **Docker**: 24+ & **Docker Compose**: v2+ (`docker compose version`)
- **Make**: GNU Make 4+ (`make --version`)
- **Git** & **curl**

---

## 2. Environment Configuration

The monorepo uses two environment files for local execution:

### 2.1 Compose Environment (`deploy/compose/.env`)
Copy template if not already present:
```bash
cp deploy/compose/.env.example deploy/compose/.env
```
Contains:
```ini
AIGW_ENV=development
AIGW_ALLOW_INSECURE_DEV=true
POSTGRES_USER=gateway
POSTGRES_PASSWORD=gateway
POSTGRES_DB=ai_gateway
REDIS_PASSWORD=change-me
AUTH_HMAC_SIGNING_KEY=change-me-change-me-change-me-change-me
```

### 2.2 Backend Process Environment (`backend/.env`)
Copy from template if not present (never overwritten):
```bash
cp backend/.env.example backend/.env
```
Contains:
```ini
AIGW_ENV=development
AIGW_ALLOW_INSECURE_DEV=true
AIGW_DATABASE_URL=postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable
AIGW_REDIS_ADDR=localhost:6379
AIGW_REDIS_PASSWORD=change-me
AIGW_AUTH_HMAC_SIGNING_KEY=change-me-change-me-change-me-change-me
```

> [!IMPORTANT]
> **Safety Guardrails**: If `AIGW_ENV` is set to `production` or `staging`, or if `AIGW_ALLOW_INSECURE_DEV` is `false`, any known placeholder secret (`change-me`, `password`, `secret`) and dev conveniences (`AUTH_DEV_LOGIN_ENABLED`, insecure cookies, disabled CSRF) will be rejected at startup.

---

## 3. Local Service Lifecycle Management (`scripts/dev-ctl.sh` & Makefile)

The lifecycle script `scripts/dev-ctl.sh` provides safe, non-destructive defaults:

```bash
# 1. Start all services in topological order and await /readyz 200 OK
make dev-start       # or: bash scripts/dev-ctl.sh start

# 2. Inspect checkout commit, host URLs, container status, and /readyz health
make dev-status      # or: bash scripts/dev-ctl.sh status

# 3. View streaming logs
dev-logs             # or: bash scripts/dev-ctl.sh logs

# 4. Gracefully stop services (retaining all container state)
make dev-stop        # or: bash scripts/dev-ctl.sh stop

# 5. Remove containers while preserving volumes (pgdata, redisdata, natsdata, secrets)
make dev-down        # or: bash scripts/dev-ctl.sh down

# 6. Destructive reset (wipes volumes and re-migrates; REQUIRES explicit confirmation)
make dev-reset       # or: bash scripts/dev-ctl.sh reset [--confirm]
```

### Service Endpoints
| Service | Local URL | Description |
|---|---|---|
| **Web Dashboard** | `http://localhost:3000` | Next.js 16 App Router UI shell |
| **Control Plane API** | `http://localhost:8081` | Management, Auth, Policies, Analytics |
| **Gateway Data Plane (v2)**| `http://localhost:8084` | High-performance streaming proxy |
| **Background Worker** | `http://localhost:8082` | Telemetry consumer & rollups |
| **PostgreSQL** | `localhost:5432` | DB: `ai_gateway`, User: `gateway` |
| **Redis** | `localhost:6379` | Cache, Rate Limits, Invalidation |
| **NATS JetStream** | `localhost:4222` / `8222` | Event bus & monitoring dashboard |

---

## 4. Database Migrations & Seeding

```bash
# Apply pending Goose migrations (00001 -> 00032)
make db-migrate

# Seed demo records (organizations, users, sample providers)
make dev-seed
```

---

## 5. Running Processes Locally (Hybrid Mode)

If you wish to develop backend or frontend code directly on your host machine while keeping infrastructure (Postgres, Redis, NATS) in Docker:

1. Start only infrastructure containers:
```bash
docker compose -f deploy/compose/docker-compose.yml up -d postgres redis nats
```
2. Start the backend services in separate terminal windows:
```bash
# Terminal 1: Control API (port 8081)
make dev-control-api

# Terminal 2: Gateway-v2 (port 8084)
make dev-gateway

# Terminal 3: Worker (port 8082)
make dev-worker
```
3. Start the Next.js frontend:
```bash
cd app
npm run dev
# Dashboard available on http://localhost:3000
```

---

## 6. Running Tests & Release Verification

Sentinel provides a single, deterministic entry point that verifies all 15 check gates in order:

```bash
# Run complete verification suite
make verify
# or:
./scripts/verify-local.sh
```

### Running Individual Test Suites
```bash
# Go unit tests across all packages
cd backend && go test -count=1 ./...

# Go race detection suite
cd backend && go test -race -count=1 ./internal/interfaces/controlapi ./internal/interfaces/gatewayv2 ./internal/interfaces/worker

# PostgreSQL repository integration tests
cd backend && go test -count=1 ./internal/infrastructure/postgres/repositories/...

# Frontend unit tests
cd app && npm test

# Frontend type check & lint
cd app && npx tsc --noEmit && npm run lint

# End-to-End User Journeys (A through G)
cd backend && go run ./cmd/e2e/main.go

# Local Performance & Resilience validation
cd backend && go run ./cmd/perf/main.go
```

---

## 7. Troubleshooting

- **Port Conflict (`bind: address already in use`):**
  Check which process is listening on the conflicting port:
  ```bash
  sudo lsof -i :8081 -i :8084 -i :5432 -i :6379 -i :3000
  ```
- **Postgres Migrations Fail with Connection Refused:**
  Ensure container is healthy:
  ```bash
  docker exec ai-gateway-postgres-1 pg_isready -U gateway -d ai_gateway
  ```
- **Gateway Requests Return 401 with Dummy Key:**
  Ensure the gateway image was rebuilt with the offline mock provider update (`make dev-up`).
- **Data Persistence:**
  To reset the database entirely to a fresh state (warning: destroys test data):
  ```bash
  docker compose -f deploy/compose/docker-compose.yml down -v
  make dev-up
  ```
