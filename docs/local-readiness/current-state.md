# Sentinel AI Gateway / AI Governance — Current State Architecture & Runtime Audit

**Date**: 2026-10-16  
**Environment**: Local Multi-Service Development Stack  
**Active Git Branch**: `main` (commit `c8e84d8`)

---

## 1. System Overview & Entry Points

Sentinel AI Gateway is an enterprise AI governance proxy and observability platform built on a hybrid architecture:

| Component | Repository Path & Entry Point | Runtime / Language | Port | Role & Responsibility |
|---|---|---|---|---|
| **Frontend Dashboard** | `app/src/app` (`app/Dockerfile`) | Next.js 16.3.7, React 19.2.8, TypeScript 5 | `3000` | Administrative control center, observability charts, governance authoring, audit logs, and settings. |
| **Control Plane API** | `backend/cmd/control-api/main.go` | Go 1.25, Chi/HTTP router | `8081` (maps to internal `:8080`) | Multi-tenant REST API managing identity, teams, projects, credentials, providers, policies, budgets, guardrails, alerts, and billing. |
| **Data Plane (Gateway-v2)** | `backend/cmd/gateway-v2/main.go` | Go 1.25, low-latency streaming proxy | `8084` (maps to internal `:8080`) | High-throughput streaming proxy, key hashing, Redis Lua rate limiting, policy snapshot enforcement, guardrail inspection, provider routing, telemetry emission. |
| **Background Worker** | `backend/cmd/worker/main.go` | Go 1.25, async daemon | `8082` (maps to internal `:8080`) | NATS JetStream consumer, `llm_requests` insertion, 5-minute rollups (`usage_hourly`, `usage_daily`), retention enforcement, watchdog alert evaluation. |
| **Database Migrations** | `backend/cmd/migrate/main.go` | Go 1.25, Goose | One-shot (`deploy/compose`) | Applies 34 database migrations sequentially against PostgreSQL. |
| **Sentinel CLI (`aigw`)** | `packages/cli/cmd/root.go` | Go 1.25, Cobra CLI | Client binary | Developer command-line interface for auth, project selection, permitted model discovery, streaming/non-streaming inference. |
| **Sentinel SDK** | `packages/api-client/src/index.ts` | TypeScript, Node.js fetch | Client library | Programmatic client library for browser dashboard and Node.js applications. |

---

## 2. Infrastructure & Topology

All services are containerized via `deploy/compose/docker-compose.yml` and verified running locally:

| Service | Container Name | Host Port | Internal Port | Health Probe | Status | Verified Alignment |
|---|---|---|---|---|---|---|
| **PostgreSQL** | `ai-gateway-postgres-1` | `5432` | `5432` | `pg_isready -U gateway` | Up (healthy) | Goose migrations 00001-00034 applied. |
| **Redis** | `ai-gateway-redis-1` | `6379` | `6379` | `redis-cli -a change-me ping` | Up (healthy) | Cache, rate limiter, pub/sub invalidation active. |
| **NATS JetStream** | `ai-gateway-nats-1` | `4222`, `8222` | `4222`, `8222` | `wget http://localhost:8222/healthz` | Up (healthy) | Persistent stream `AIGW_USAGE` configured. |
| **Control API** | `ai-gateway-control-api-1` | `8081` | `8080` | `GET http://localhost:8080/livez` | Up (healthy) | Built from current `backend/` commit. |
| **Gateway-v2** | `ai-gateway-gateway-v2-1` | `8084` | `8080` | `GET http://localhost:8080/livez` | Up (healthy) | Built from current `backend/` commit. |
| **Worker** | `ai-gateway-worker-1` | `8082` | `8080` | `GET http://localhost:8080/livez` | Up (healthy) | Built from current `backend/` commit. |
| **Frontend Web** | `ai-gateway-app-1` | `3000` | `3000` | HTTP 200 on `/app/overview` | Up (healthy) | Built from current `app/` commit. |

---

## 3. End-to-End Request & Telemetry Lifecycle

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

## 4. Database Schema & Migration Status

A total of **34 sequential Goose migrations** are present in `backend/internal/infrastructure/postgres/migrations/` and verified applied in PostgreSQL:

- `00001` - `00005`: Identity, projects, teams, API credentials, audit logs.
- `00006` - `00010`: Provider accounts, usage tables, policies, billing, conversations.
- `00011` - `00015`: Alerts, RLS policies, membership lookup, credential resolver, identity m3.
- `00016` - `00020`: Provider credential fingerprints, models, gateway credentials, provider health, model metadata.
- `00021` - `00025`: Gateway credentials RLS, auth credentials, request telemetry, conversation privacy, security events.
- `00026` - `00030`: Invoices, service account attribution, guardrails, budgets, telemetry & privacy.
- `00031` - `00032`: Usage reconciliation composite indexes, alerts enhancement & history.
- `00033_usage_dimensions_and_audit.sql`: Team and project attribution columns on `llm_requests`.
- `00034_alerting_and_retention.sql`: Alert events status tracking and retention rule definitions.

---

## 5. Contradictions Between Documentation & Implementation

1. **Migration Count Mismatch**: Previous readiness documents stated 32 migrations; actual repository contains 34 applied migrations.
2. **Mock vs Real Backend Status**: Previous reports claimed conversations and budgets were "pending" or "mock-only". In reality, the Control API and backend repositories are fully implemented, and the frontend connects to live endpoints.
3. **Workspace Script Mismatch**: Root `package.json` ran `npm run typecheck --workspaces`, but `app/package.json` lacked a `typecheck` script, causing workspace-level type checking to fail even though `npx tsc --noEmit` in `app/` is clean.
4. **URL Trailing Slash Mismatch**: `tenant_resolver.go` normalizes all provider base URLs to have a trailing slash `"/"`, but `tenant_resolver_test.go` asserted equality without a trailing slash.
5. **CLI Test Payload Mismatch**: `SendRequest` in `packages/cli/internal/client/client.go` routes `gemini-` models through `/v1/chat/completions` (OpenAI format), but `client_test.go` mocked raw Gemini Google API JSON.

---

## 6. Unsafe Development Defaults Identified

1. **Hardcoded Secrets in Compose**:
   - `AIGW_REDIS_PASSWORD: change-me`
   - `AIGW_AUTH_HMAC_SIGNING_KEY: change-me-change-me-change-me-change-me`
2. **Permissive Cookie Settings**:
   - `AIGW_AUTH_SESSION_COOKIE_SECURE: "false"` (HTTP allowed in dev)
   - `AIGW_AUTH_CSRF_ENABLED: "false"`
3. **Dev Login Enabled**:
   - `AIGW_AUTH_DEV_LOGIN_ENABLED: "true"` (bypasses production OAuth/SAML)
4. **Missing Rate Limit on Public Auth**:
   - `POST /v1/auth/signin` and `POST /v1/auth/signup` do not enforce Redis sliding-window brute-force limits.

---

## 7. Prioritized Implementation Plan

| Priority | Component | Item | Proposed Action | Verification Method |
|---|---|---|---|---|
| **P1** | Backend Gateway | Trailing slash in `tenant_resolver_test.go` | Normalize test assertions to match canonical trailing-slash URLs. | `go test ./internal/interfaces/gatewayv2/...` |
| **P1** | Sentinel CLI | Mock response format in `client_test.go` | Update Gemini mock response in CLI client test to match `/v1/chat/completions` contract. | `go test ./...` in `packages/cli/` |
| **P2** | SDK (`packages/api-client`) | Missing `@types/node` in `api-client` | Add `@types/node` to `packages/api-client/package.json` and update `tsconfig.json`. | `npm run typecheck` across monorepo |
| **P2** | Monorepo Scripts | Missing `typecheck` in `app/package.json` | Add `"typecheck": "tsc --noEmit"` to `app/package.json`. | `npm run typecheck` in workspace root |
| **P3** | Security Hardening | Development defaults documentation | Document production environment overrides for cookie security, CSRF, and secret rotation. | `docs/local-readiness/runbook.md` |
