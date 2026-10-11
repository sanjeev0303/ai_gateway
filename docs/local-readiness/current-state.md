# Sentinel AI Gateway / AI Governance — Current State Architecture & Runtime Audit

## 1. System Overview

Sentinel AI Gateway is a production-grade AI governance and proxy gateway platform built as a hybrid architecture:
- **Frontend**: Next.js 16 (App Router, React 19, TypeScript, Vanilla CSS design tokens with modern UI shell) listening on `http://localhost:3000`.
- **Control Plane API**: Go HTTP service (`ai-governance-saas/backend/cmd/control-api`) listening on `http://localhost:8081`. Manages identity, organizations, teams, projects, credentials, provider accounts, policies, guardrails, alerts, security events, audit logs, and billing.
- **Data Plane (Gateway-v2)**: Low-latency streaming proxy (`ai-governance-saas/backend/cmd/gateway-v2`) listening on `http://localhost:8084`. Handles key authentication, tenant isolation, upstream provider routing (Anthropic, OpenAI, Gemini), real-time policy evaluation, rate limiting, token budget checks, and asynchronous telemetry emission.
- **Background Worker**: Go daemon (`ai-governance-saas/backend/cmd/worker`) listening on `http://localhost:8082`. Subscribes to telemetry events from NATS JetStream / in-process event bus, records raw requests to `llm_requests` and `usage_events`, computes hourly/daily usage rollups (`usage_hourly`, `usage_daily`), enforces retention, and evaluates watchdog alerts.
- **Persistence & Storage**: PostgreSQL 16 (with Goose migrations 00001 through 00032) and Redis 7 (rate limiting, real-time activity tracker, credential caching, pub/sub invalidation).

---

## 2. Infrastructure & Topology

| Service | Container / Process | Port | Health Probes | Storage / Dependencies |
|---|---|---|---|---|
| PostgreSQL | `ai-gateway-postgres-1` | 5432 | `pg_isready -U gateway` | Docker volume `pgdata`, 32 migrations applied |
| Redis | `ai-gateway-redis-1` | 6379 | `redis-cli ping` | In-memory with optional persistence, pub/sub channels |
| NATS JetStream | `ai-gateway-nats-1` | 4222 / 8222 | `wget /healthz` | File-backed durable JetStream stream `AIGW_USAGE` |
| Control API | `ai-gateway-control-api-1` | 8081 | `GET /livez`, `GET /readyz` | PostgreSQL pool, Redis client, Secrets store |
| Gateway-v2 | `ai-gateway-gateway-v2-1` | 8084 | `GET /livez`, `GET /readyz` | PostgreSQL pool, Redis client, Event Bus, Upstream Providers |
| Worker | `ai-gateway-worker-1` | 8082 | `GET /livez`, `GET /readyz` | PostgreSQL pool, Event Bus, Scheduled Cron |
| Frontend | `ai-gateway-app-1` | 3000 | `GET /api/health` | Node.js Next.js standalone server, proxies to :8081 |

---

## 3. Request Lifecycle & Pipelines

### 3.1 Control Plane Request Flow
1. **Client / Browser**: Next.js Server Components call `fetchApi()` or Client Components call direct `:8081`.
2. **Auth Layer**: Cookie `session=<jwt>` parsed with HMAC token manager. CSRF double-submit token validated for state-mutating requests (`POST`, `PATCH`, `DELETE`).
3. **Actor Context**: Organization ID, User ID, and Role extracted and injected into `context.Context`.
4. **Tenant Isolation**: Handlers never trust client-provided `organization_id`; queries enforce `WHERE organization_id = $1`.
5. **Audit Logging**: Successful mutations emit audit log entries to `audit_logs` table via transaction hooks.

### 3.2 Gateway-v2 Data Plane Flow
1. **Client Request**: `POST /v1/messages` with `x-api-key: aigw_...` or `Authorization: Bearer aigw_...`.
2. **Authentication**: `AuthenticatorImpl` hashes the raw key with SHA-256, queries Redis cache (L1), falls back to `gateway_credentials` table (L2).
3. **Tenant & Upstream Resolution**: Resolves organization's active provider accounts and decrypted secrets.
4. **Policy Snapshot**: Cached in-memory policy rules evaluate model access, prompt guardrails, and tenant boundaries without database round-trips.
5. **Guardrail Check**: In-memory inspection evaluates PII and prompt injection rules against the input payload.
6. **Rate Limiting & Budgets**: Redis token bucket checks credential rate limits and budget caps.
7. **Stream Proxy**: HTTP request forwarded to upstream provider; response SSE chunks streamed directly back to client with `x-request-id`. For deterministic testing, built-in offline mock handles `sk-dummy-key`.
8. **Telemetry Emission**: Async worker channel publishes `UsageRecord` to NATS JetStream event bus.

### 3.3 Worker Telemetry & Rollup Pipeline
1. `UsageConsumer` receives `UsageEventMessage` from JetStream subject `aigw.usage.events`.
2. Writes to `llm_requests` (detailed request ledger) and `usage_events` (token counts).
3. Scheduled worker executes `RecomputeAll` every 5 minutes to aggregate usage into `usage_hourly` and `usage_daily`.
4. Alerts engine queries `usage_hourly` against active alert rules and fires notifications.
5. Retention job purges raw requests and message bodies based on organization-level retention policies.

---

## 4. Database Schema & Migration Status

All 32 Goose migrations have been verified and applied against PostgreSQL:
- `00001_identity.sql`: Organizations, users, memberships, invitations.
- `00002_projects_teams.sql`: Projects, teams, and team memberships.
- `00003_service_accounts.sql`: Machine service accounts.
- `00004_api_credentials.sql`: API credentials and key hashing.
- `00005_audit_logs.sql`: Structured immutable audit log entries.
- `00006_providers.sql`: Provider accounts, credentials, and model registry.
- `00007_usage.sql`: Raw `llm_requests`, `usage_events`, `usage_hourly`, `usage_daily`.
- `00008_policies.sql`: Policies and declarative rules engine.
- `00009_billing.sql`: Plans, subscriptions, and invoices.
- `00010_conversations.sql`: Multi-turn chat sessions and message logs.
- `00011_alerts.sql`: Organization watchdog and alert rules.
- `00012_rls.sql`: PostgreSQL Row-Level Security policies.
- `00013_gateway_credentials.sql`: High-performance gateway credential records.
- `00014_guardrails.sql`: Guardrails table with detector configurations.
- `00015` through `00030`: Telemetry enhancements, latency tracking (TTFT, total latency), service account attribution, and privacy controls.
- `00031_usage_reconciliation_indexes.sql`: Composite performance indexes for usage rollups and reconciliation.
- `00032_alerts_enhancement_and_history.sql`: Alert execution logs and notification history tables.

---

## 5. Audit Conclusions & Resolved Defect Status

1. **Backend Core Services**: Identity, Providers, Usage, Activity, Conversations, AuditLogs, Alerts, SecurityEvents, Billing, Policy, Budget, and Guardrails are all implemented and verified.
2. **Gateway-v2 Data Plane**: Authenticates gateway tokens, enforces tenant isolation, checks policies and guardrails, proxies streams, and emits async usage records.
3. **Previously Identified Gaps — All Resolved**:
   - `guardrails.go`: UUID empty string syntax error when creating guardrails without pre-assigned ID — **FIXED** (UUID generated automatically).
   - `alerts.go`: Check constraint mismatch on `alert_type` (`spend_exceeded` vs `spend_limit`) — **FIXED** (Normalized enum and updated migrations).
   - `telemetry_emitter.go`: Status string mismatch (`'success'`/`'error'` vs `'succeeded'`/`'failed'`) breaking `llm_requests` insertion — **FIXED** (Aligned enum values).
   - Offline Provider Mocking: Upstream requests with `sk-dummy-key` previously returned 401 from external Anthropic API — **FIXED** (Added deterministic in-memory mock provider handling for streaming and non-streaming in `stream_proxy.go`).
   - Loopback SSE chunk coalescing in stream proxy unit tests — **FIXED** (Added flush pause to guarantee distinct socket reads).
