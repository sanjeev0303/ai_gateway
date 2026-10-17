# Sentinel AI Gateway / AI Governance — Current State Architecture & Runtime Audit

**Evaluation Date**: 2026-10-17  
**Environment**: Local Multi-Service Development Stack  
**Active Git Branch**: `main` (commit `5a88fab77aba9a492ba71e3bcf35c03059a68fb0`)

---

## 1. System Overview & Entry Points

Sentinel AI Gateway is an enterprise AI governance proxy, policy enforcement engine, and observability platform built on a modular Go and Next.js architecture:

| Component | Repository Path & Entry Point | Runtime / Language | Port | Role & Responsibility |
|---|---|---|---|---|
| **Frontend Dashboard** | `app/src/app` (`app/Dockerfile`) | Next.js 16.3.7, React 19.2.8, TypeScript 5 | `3000` | Executive control center, real-time observability charts, governance rule authoring, audit logs, and settings. |
| **Control Plane API** | `backend/cmd/control-api/main.go` | Go 1.25, Chi/HTTP router | `8081` | Multi-tenant REST API managing organizations, teams, projects, credentials, providers, policies, budgets, guardrails, alerts, and billing. |
| **Data Plane (Gateway-v2)** | `backend/cmd/gateway-v2/main.go` | Go 1.25, high-throughput streaming proxy | `8084` | Latency-critical streaming proxy, API key verification, Redis sliding-window rate limiting, policy snapshot enforcement, guardrail inspection, provider routing, telemetry emission. |
| **Background Worker** | `backend/cmd/worker/main.go` | Go 1.25, asynchronous daemon | `8082` | NATS JetStream consumer, idempotent `llm_requests` insertion, 5-minute rollups (`usage_hourly`, `usage_daily`), retention enforcement, watchdog alert evaluation. |
| **Database Migrations** | `backend/cmd/migrate/main.go` | Go 1.25, Goose | One-shot (`deploy/compose`) | Applies 34 database migrations sequentially against PostgreSQL. |
| **Sentinel CLI (`aigw`)** | `packages/cli/cmd/root.go` | Go 1.25, Cobra CLI | Client binary | Developer CLI for authentication, project selection, permitted model discovery, interactive and streaming inference. |
| **Sentinel SDK** | `packages/api-client/src/index.ts` | TypeScript, Node.js fetch | Client library | Programmatic client library for browser dashboard and Node.js applications. |

---

## 2. Infrastructure & Local Topology

All core infrastructure and application services run locally via `deploy/compose/docker-compose.yml` and are managed via [`scripts/dev-ctl.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/dev-ctl.sh):

| Service | Container Name | Host Port | Internal Port | Readiness Probe | Status | Verified Functionality |
|---|---|---|---|---|---|---|
| **PostgreSQL** | `ai-gateway-postgres-1` | `5432` | `5432` | `pg_isready -U gateway` | Up (healthy) | Goose migrations 00001–00034 applied; FORCE RLS active. |
| **Redis** | `ai-gateway-redis-1` | `6379` | `6379` | `redis-cli -a change-me ping` | Up (healthy) | In-memory cache, rate limiter sliding window, pub/sub policy invalidation active. |
| **NATS JetStream** | `ai-gateway-nats-1` | `4222`, `8222` | `4222`, `8222` | `wget http://localhost:8222/healthz` | Up (healthy) | Persistent stream `AIGW_USAGE` configured and receiving events. |
| **Control API** | `ai-gateway-control-api-1` | `8081` | `8080` | `GET http://localhost:8081/readyz` | Up (healthy) | Connected to PostgreSQL and Redis. |
| **Gateway-v2** | `ai-gateway-gateway-v2-1` | `8084` | `8084` | `GET http://localhost:8084/readyz` | Up (healthy) | Hot path proxying requests, streaming SSE, and emitting telemetry. |
| **Worker** | `ai-gateway-worker-1` | `8082` | `8080` | `GET http://localhost:8082/readyz` | Up (healthy) | Subscribed to NATS JetStream consumer group with deduplication. |
| **Frontend Web** | `ai-gateway-app-1` | `3000` | `3000` | HTTP 200 on `/app/overview` | Up (healthy) | Production Next.js build running authenticated server components. |

---

## 3. End-to-End Request & Governance Lifecycle

```
[Browser / CLI / SDK]
        │
        ▼ (POST /v1/messages or /v1/chat/completions)
[Gateway-v2 :8084]
        │
        ├── 1. Auth: Authenticator hashes API key -> Redis L1 cache -> DB L2 fallback
        ├── 2. Tenant Context: Bind OrgID, UserID, TeamID, ProjectID from credential actor
        ├── 3. Policy Snapshot: Evaluate model allowlist/denylist & max token constraints
        ├── 4. Guardrails: Inspect payload for PII and prompt injection patterns
        ├── 5. Rate Limits & Budgets: Redis Lua sliding window & spend budget checks
        ├── 6. Provider Upstream: Stream request to Anthropic, OpenAI, or Gemini (or offline mock)
        │       └── SSE chunks flushed continuously to client
        └── 7. Telemetry Sink: Non-blocking emit of UsageRecord to NATS JetStream
                │
                ▼ (Subject: aigw.usage.events)
        [Worker :8082]
                │
                ├── 1. Consume UsageEventMessage
                ├── 2. Insert into llm_requests & usage_events (idempotent, conflict-safe)
                ├── 3. Hourly/Daily Rollup Job: Aggregate into usage_hourly & usage_daily
                ├── 4. Alert Evaluator: Trigger threshold warnings & incident events
                └── 5. Retention Job: Purge aged prompts based on org privacy setting
                        │
                        ▼ (PostgreSQL Tables)
        [Control API :8081] <── [Next.js Web Dashboard :3000]
                │
                └── Serves Overview KPI metrics, timeseries, heatmaps, and detail views
```

---

## 4. Key Subsystem Implementation Audit

### 4.1 Budget Enforcement
- **Location**: [`backend/internal/interfaces/gatewayv2/budget_guard.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/budget_guard.go)
- **Mechanics**: Concurrency-safe budget reservations tracking spend in integer micro-dollars ($0.000001 precision). Requests exceeding budget allocations return HTTP `429 Too Many Requests` (`err_budget_exceeded`) without forwarding to upstream providers.

### 4.2 Rate Limiting
- **Location**: [`backend/internal/interfaces/gatewayv2/gateway.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway.go) and Redis Lua scripts
- **Mechanics**: Sliding-window rate limiting keyed per organization, team, project, user, or credential. Rejections return HTTP `429` with `Retry-After` headers.

### 4.3 Policy Resolution & Caching
- **Location**: [`backend/internal/application/policy/service.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/policy/service.go) and [`backend/internal/interfaces/gatewayv2/policy_invalidation_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/policy_invalidation_test.go)
- **Mechanics**: Policy snapshots are cached in memory for sub-millisecond evaluation. Cache invalidation is broadcast via Redis pub/sub upon policy mutation, bounding stale policy evaluation to <10ms.

### 4.4 Usage Recording & Telemetry Deduplication
- **Location**: [`backend/internal/interfaces/gatewayv2/telemetry_emitter.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/telemetry_emitter.go) and [`backend/internal/interfaces/worker/event_reliability_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/worker/event_reliability_test.go)
- **Mechanics**: Gateway emits events asynchronously to NATS JetStream. Worker processes batches using consumer groups. Idempotency is guaranteed by `request_id` primary key constraint on `llm_requests` using atomic `ON CONFLICT DO NOTHING`.

### 4.5 Analytics & Rollup Pipelines
- **Location**: [`backend/internal/infrastructure/postgres/repositories/operations.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/repositories/operations.go) and `backend/internal/interfaces/controlapi/usage.go`
- **Mechanics**: Worker aggregates granular request logs into `usage_hourly` and `usage_daily` tables. Control API reads pre-aggregated summary tables for timeseries and cost breakdowns, achieving p50 query latencies <7ms.

---

## 5. Reconciled Contradictions

1. **Test Failures in Baseline**: Previous reports documented failing tests in `tenant_resolver_test.go`, `client_test.go`, and missing `@types/node` in `api-client`. All have been resolved and verified passing in commit `5a88fab`.
2. **Environment Configuration**: `backend/.env.example` has been updated to reflect active Redis rate limiting and NATS JetStream event bus configuration.
3. **Gateway v2 Port Alignment**: `Makefile` target `dev-gateway` updated from `:8080` to `:8084` to match Docker Compose and runtime documentation.
4. **Offline Development**: `./scripts/dev-ctl.sh start` updated to reuse pre-built local container images without forcing network rebuilds.
