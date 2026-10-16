# Sentinel AI Gateway — Local Operations Runbook

**Date**: 2026-10-15  
**Version**: 1.0.0  
**Target Environment**: Local Development & Evaluation (Docker & Host Binaries)  

---

## 1. Quick Start Commands

### 1.1 Start Infrastructure and Core Services

Start all 7 core containers (PostgreSQL, Redis, NATS, Control API, Gateway v2, Worker, and Next.js Web App):

```bash
docker compose up -d
```

Verify container status:
```bash
docker compose ps
```
All containers should display state `Up` or `Up (healthy)`.

---

## 2. Database Migrations & Deterministic Seeding

### 2.1 Apply Database Migrations
Migrations run automatically on control plane boot, or can be run manually:
```bash
cd backend
go run ./cmd/migrate
```

### 2.2 Seed Deterministic Local Data
Populate the database with the complete deterministic test organization, employees, provider accounts, credentials, policies, guardrails, and telemetry events:
```bash
cd backend
go run ./cmd/seed
```
*Note: `seed.sql` is completely idempotent using `ON CONFLICT DO UPDATE / NOTHING`.*

---

## 3. Service Endpoints & Health Checks

| Service | Port | Health Check URL | Description |
|---|---|---|---|
| **Control Plane API** | `8081` | `http://localhost:8081/healthz` | Platform administration, policies, auth |
| **Gateway v2** | `8084` | `http://localhost:8084/healthz` | High-throughput AI proxy & routing |
| **Async Worker** | `8082` | `http://localhost:8082/healthz` | Telemetry consumer & alert evaluator |
| **Web Dashboard** | `3000` | `http://localhost:3000` | Next.js 16 Executive Control Center |
| **PostgreSQL** | `5432` | `pg_isready -h localhost -p 5432` | Primary transactional and usage ledger |
| **Redis** | `6379` | `redis-cli -p 6379 ping` | Distributed rate limits & pub/sub cache |
| **NATS Bus** | `4222` | `curl -s http://localhost:8222/varz` | JetStream event streaming broker |

Verify all service health endpoints via curl:
```bash
curl -f http://localhost:8081/healthz
curl -f http://localhost:8084/healthz
curl -f http://localhost:8082/healthz
```

---

## 4. Test Suite Execution

### 4.1 Backend Quality Gates
Run all unit tests across the Go codebase:
```bash
cd backend
go test ./...
```

Run race condition detection tests:
```bash
cd backend
go test -race ./internal/...
```

### 4.2 End-to-End Acceptance Journey (21 Steps)
Run the automated 21-step acceptance journey covering the complete lifecycle:
```bash
cd backend
go run ./cmd/e2e
```

### 4.3 Local Load & Performance Benchmark
Execute the local load testing suite (mixed throughput, streaming TTFT, memory growth, query health):
```bash
cd backend
go run ./cmd/perf
```

### 4.4 Frontend Quality Gates
```bash
cd app
npx tsc --noEmit          # Type check (0 errors)
npm run lint              # ESLint check
npm test                  # Unit tests (24 passed)
npm run build             # Next.js 16 production build
```

---

## 5. Troubleshooting & Operational Tasks

### 5.1 Inspecting Container Logs
Follow logs for specific services:
```bash
docker compose logs -f control-api
docker compose logs -f gateway-v2
docker compose logs -f worker
docker compose logs -f app
```

### 5.2 Restarting a Single Service
```bash
docker compose restart gateway-v2
docker compose restart worker
```

### 5.3 Rebuilding Local Images After Changes
```bash
# Rebuild backend services
docker build -t ai-gateway-backend:dev -f backend/Dockerfile .
docker compose up -d control-api gateway-v2 worker

# Rebuild frontend web service
docker build -t ai-gateway-web:dev -f app/Dockerfile .
docker compose up -d app
```

### 5.4 Database Inspection & Reset
Direct psql session into local database:
```bash
docker compose exec postgres psql -U gateway -d ai_gateway
```

Full wipe and clean restart:
```bash
docker compose down -v
docker compose up -d
cd backend && go run ./cmd/migrate && go run ./cmd/seed
```
