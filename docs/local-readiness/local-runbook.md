# Sentinel AI Gateway — Local Operations Runbook

## Overview and Purpose

This runbook provides complete operational procedures for configuring, launching, verifying, and recovering the Sentinel AI Gateway in a local development environment.

---

## 1. Environment Architecture & Endpoints

| Service | Port | Functional Readiness Probe | Purpose |
|---|---|---|---|
| **Web Dashboard** | `3000` | `http://localhost:3000` | Next.js 16 Executive Control Center & Developer UI |
| **Control Plane API** | `8081` | `http://localhost:8081/readyz` | Org management, auth, credentials, policies, billing |
| **Async Worker** | `8082` | `http://localhost:8082/readyz` | Telemetry processing, rollups, alerts, retention |
| **Gateway v2 Proxy** | `8084` | `http://localhost:8084/readyz` | High-throughput streaming AI proxy & rate limiting |
| **PostgreSQL** | `5432` | `pg_isready -h localhost -p 5432` | Relational ledger, FORCE RLS, audit logs, usage data |
| **Redis** | `6379` | `redis-cli -p 6379 ping` | Rate limiting sliding window, pub/sub invalidation |
| **NATS Event Bus** | `4222` / `8222` | `http://localhost:8222/healthz` | JetStream message broker for async telemetry pipeline |

---

## 2. Service Lifecycle Management

Use [`scripts/dev-ctl.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/dev-ctl.sh) for consistent lifecycle operations.

### 2.1 Start All Services
```bash
./scripts/dev-ctl.sh start
```
*Note: Uses `--no-build` with pre-built local Docker images to avoid network timeout interruptions.*

### 2.2 Stop All Services
```bash
./scripts/dev-ctl.sh stop
```

### 2.3 Check Service Status & Readiness
```bash
./scripts/dev-ctl.sh status
```

### 2.4 Controlled Local Data Reset
To wipe container state and re-seed from scratch:
```bash
./scripts/dev-ctl.sh reset
```

---

## 3. Database Migrations & Seeding

### 3.1 Apply PostgreSQL Migrations
The control plane automatically validates and applies schema migrations on startup. To run migrations manually:
```bash
cd backend
go run ./cmd/migrate
```

### 3.2 Deterministic Local Data Seeding
To populate realistic test data (organizations, 3 test users, teams, projects, provider accounts, policies, guardrails, and telemetry):
```bash
cd backend
go run ./cmd/seed
```
*Note: Seeding is idempotent and safe to re-run.*

---

## 4. Verification and Quality Gates

### 4.1 Master Release-Readiness Script
Run the 15-point verification suite checking configurations, services, migrations, unit tests, race detector, frontend lint, typecheck, SDK tests, CLI tests, and E2E acceptance:
```bash
./scripts/verify-local.sh
```

### 4.2 End-to-End Acceptance Journey
Execute the automated 21-step acceptance journey covering user signup, credential issuance, streaming proxying, guardrail rejection, and usage verification:
```bash
cd backend
go run ./cmd/e2e
```

### 4.3 Performance and Latency Benchmarks
Run local load tests to measure throughput, TTFT, goroutine leak checks, and query latency:
```bash
cd backend
go run ./cmd/perf
```

### 4.4 Frontend Quality Checks
```bash
cd app
npm run lint              # ESLint checks
npm run typecheck         # TypeScript strict type checking
npm test                  # Client unit tests (24 passed)
npm run build             # Production bundle build
```

---

## 5. Client Operations: SDK & CLI

### 5.1 Sentinel CLI (`aigw`)

Install or build the CLI:
```bash
cd packages/cli
go build -o /tmp/aigw ./main.go
```

Authenticate and inspect resources:
```bash
# Login against local control plane
/tmp/aigw auth login --api-url http://localhost:8081

# List accessible projects
/tmp/aigw projects list

# List permitted models
/tmp/aigw models list

# Make an interactive completion request through Gateway v2
/tmp/aigw request --model mock-gemini-2.5-flash --prompt "Explain idempotency in one sentence."

# Make a streaming request
/tmp/aigw request --model mock-gemini-2.5-flash --stream --prompt "List 3 benefits of AI governance."
```

### 5.2 Sentinel SDK (`@ai-governance/api-client`)

Initialize client with individual user gateway credential:
```typescript
import { GatewayClient } from '@ai-governance/api-client';

const client = new GatewayClient({
  baseUrl: 'http://localhost:8084',
  apiKey: 'gw_live_user_test_credential',
});

// Non-streaming completion
const response = await client.chatCompletion({
  model: 'mock-gemini-2.5-flash',
  messages: [{ role: 'user', content: 'Hello Sentinel' }],
});

// Streaming completion
const stream = await client.streamCompletion({
  model: 'mock-gemini-2.5-flash',
  messages: [{ role: 'user', content: 'Stream test' }],
});
for await (const chunk of stream) {
  process.stdout.write(chunk.content);
}
```

---

## 6. Failure Modes, Recovery & Architecture Semantics

### 6.1 Fail-Closed vs. Fail-Open Architecture

| Boundary | Behavior | Reason |
|---|---|---|
| **Authentication & Credentials** | **Fail-Closed** | Requests with invalid, missing, or revoked credentials immediately return `401 Unauthorized`. |
| **Policies & Model Access** | **Fail-Closed** | Requests violating policy rules return `403 Forbidden` without reaching upstream provider. |
| **Guardrails (PII / Injection)** | **Fail-Closed** | Requests triggering guardrail violations return `403 Forbidden` and block upstream forwarding. |
| **Spend & Token Budgets** | **Fail-Closed** | Requests exceeding allocated budgets return `429 Too Many Requests`. |
| **Telemetry & Async Logging** | **Fail-Open (Best-Effort)** | Inability to write non-critical audit log records does not drop live client responses; events are spooled locally. |

### 6.2 Redis Outage & Recovery
- **Symptoms**: Redis down or connection refused on `localhost:6379`.
- **System Impact**: Rate limiting falls back to local in-memory sliding windows. Policy invalidation pub/sub operates on in-memory TTL bounds (max 5s cache staleness).
- **Recovery Procedure**:
  ```bash
  docker compose restart redis
  # Wait for health
  redis-cli -p 6379 ping
  ```

### 6.3 NATS Event Bus Outage & Recovery
- **Symptoms**: Worker logs show reconnection attempts; Gateway logs show event bus publish warnings.
- **System Impact**: Gateway v2 spools usage telemetry events in memory and to disk buffer.
- **Recovery Procedure**:
  ```bash
  docker compose restart nats
  # Verify JetStream status
  curl -s http://localhost:8222/varz | grep jetstream
  ```
- **Post-Recovery**: Worker automatically resumes consumption from JetStream consumer group. Deduplication on `request_id` via PostgreSQL primary key guarantees zero duplicate usage records.

### 6.4 PostgreSQL Database Outage & Recovery
- **Symptoms**: Control API and Worker report database connection refused on `localhost:5432`.
- **System Impact**: Control API enters read/write freeze (returns 503). Gateway serves existing in-cache credential and policy sessions. Worker holds unacknowledged JetStream messages in backlog.
- **Recovery Procedure**:
  ```bash
  docker compose restart postgres
  # Wait for PostgreSQL readiness
  docker compose exec postgres pg_isready -U gateway -d ai_gateway
  ```
- **Post-Recovery**: Worker flushes queued JetStream backlog with automatic deduplication.

### 6.5 Service Container Restarts
To restart individual services without downtime to dependent infrastructure:
```bash
docker compose restart gateway-v2
docker compose restart worker
docker compose restart control-api
```
All services support clean SIGTERM graceful shutdown, completing in-flight requests and flushing remaining telemetry batches within 5 seconds.
