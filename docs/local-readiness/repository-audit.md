# Sentinel AI Gateway — Comprehensive Repository Audit (Phase 1)

**Audit Date**: 2026-10-19  
**Repository Revision**: `HEAD` (`main`)  
**Assessment Toolchain**: Go `1.25.10 linux/amd64`, Node.js `v24.15.0`, npm `11.12.1`, Docker `29.5.3`  
**Actual Readiness Assessment**: **85–90% Functional Local Readiness** (Production-grade core, fully verified hermetic local test suites, documented external cloud/payment boundaries)

---

## 1. Executive Summary & Verification Methodology

A thorough, ground-up inspection was conducted across the entire codebase—encompassing Go control plane binaries, the Gateway-v2 streaming orchestrator, 4 upstream provider adapters, the Next.js 16 web application, Sentinel CLI, TypeScript SDK, 36 PostgreSQL database migrations, Redis Lua rate limiters/budget guards, NATS JetStream event bus pipelines, and dual-processor billing integrations.

### Audit Invariants & Rules Enforced:
1. **No Documentation Claims Taken as Proof**: Every subsystem was verified by inspecting source code, tracing execution paths, and executing unit, integration, race-detection, and linting suites.
2. **Deterministic Hermetic Boundaries**: Verified that in-memory mocks cleanly isolate offline local execution from external third-party cloud LLMs and payment gateways.
3. **Strict Layer Boundary Isolation**: Confirmed that `platform` low-level plumbing does not import `domain` or `application` layers (`scripts/check-boundaries.sh` passed with 0 violations).

---

## 2. Deep Subsystem Inspection

### 2.1 Control API & Gateway-v2
- **Source Paths**:
  - Control API: [`backend/internal/interfaces/controlapi`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi)
  - Gateway-v2: [`backend/internal/interfaces/gatewayv2`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2)
- **Architecture & Implementation**:
  - **Control API (`:8081`)**: RESTful management API handling authentication, organizations, members, invitations, projects, teams, service accounts, gateway credentials, policies, guardrails, provider accounts, usage analytics, conversations, and billing. Protected by session cookies, rotating JWTs, and CSRF double-submit cookies.
  - **Gateway-v2 (`:8084`)**: Canonical high-performance data plane (ADR-5). Implements a 10-stage synchronous pipeline:
    1. Authentication & API Key extraction (SHA-256 hash lookup with in-memory/Redis caching).
    2. Request inspection & routing hints extraction (8 MiB buffer limit, payload parameter tampering defense).
    3. Provider & model authorization (strict check of `AllowedProviders` and `AllowedModels`).
    4. Policy evaluation (precedence-ordered rule checks).
    5. Upstream connection resolution (AES-GCM decrypted secrets).
    6. Distributed rate limiting (Redis Sorted Sets sliding window).
    7. Atomic budget reservation (Redis Sorted Sets + Hash, sweeping expired reservations).
    8. Provider routing & streaming proxy (SSE chunk forwarding, error translation).
    9. Token usage extraction & telemetry publishing (asynchronous NATS JetStream events).
    10. Real-time activity recording (Redis ring buffer).

### 2.2 Provider Adapters & Dynamic Model Discovery
- **Source Paths**:
  - Adapters: [`backend/internal/infrastructure/provider/{anthropic,gemini,openai,huggingface}`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/provider)
  - Discovery: [`backend/internal/application/provider/service.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/provider/service.go)
- **Architecture & Implementation**:
  - **Anthropic**: Maps Messages API (`/v1/messages`), preserves tool definitions, tool use blocks, reasoning/thinking parameters, and extracts token usage (`input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens`).
  - **Gemini**: Supports native GenerateContent and OpenAI-compatible `/v1beta/openai/chat/completions`, maps Gemini safety settings (`HARM_CATEGORY_*`), and formats SSE streams.
  - **OpenAI**: Maps Chat Completions (`/v1/chat/completions`), preserves reasoning effort (`low`, `medium`, `high`) and temperature gating.
  - **Hugging Face**: Supports Dedicated Endpoints (`x-hf-endpoint` with strict HTTPS/SSRF validation) and Serverless Inference.
  - **Dynamic Discovery**: Directly queries upstream discovery endpoints (`/v1/models`, `/v1beta/models`, `/api/models`). Does **not** return fake fallback models on connection failure; marks stale records inactive.

### 2.3 Next.js Frontend & API Integration
- **Source Paths**:
  - Web Application: [`app/src`](file:///home/sanju/ai-project/ai_gateway/app/src)
  - SDK Package: [`packages/api-client`](file:///home/sanju/ai-project/ai_gateway/packages/api-client)
- **Architecture & Implementation**:
  - Next.js 16 (App Router) with 49 static and server-rendered routes.
  - Server-side cookie forwarding via `getApiClient()` to maintain authenticated caller sessions.
  - Rich UI views: Overview dashboard (activity timeline, spend charts, heatmap), Observe (usage, costs, models, providers, conversations), Govern (policies, model-access, budgets, guardrails), Platform (gateway credentials, provider accounts, service accounts), Organization (members, invitations, settings), and Billing (plans, checkout, subscriptions, invoices).
  - Verified with 34/34 passing unit tests, zero TypeScript errors (`tsc --noEmit`), and zero ESLint errors.

### 2.4 Sentinel CLI & TypeScript SDK
- **Source Paths**:
  - CLI: [`packages/cli`](file:///home/sanju/ai-project/ai_gateway/packages/cli)
  - SDK: [`packages/api-client`](file:///home/sanju/ai-project/ai_gateway/packages/api-client)
  - Contracts: [`packages/contracts`](file:///home/sanju/ai-project/ai_gateway/packages/contracts)
- **Architecture & Implementation**:
  - **CLI**: Cobra-based interactive REPL and single-shot command runner. Dynamically filters slash commands based on provider session (`/safety` for Gemini, `/endpoint` for Hugging Face, `/max-tokens` for Anthropic). Includes conversation compaction (`/compact`) and history clearing (`/clear`).
  - **SDK**: Fully typed TypeScript client with streaming SSE chunk parser, exponential backoff retry on 503, idempotency key support, request cancellation via `AbortSignal`, and token estimation indicators.

### 2.5 PostgreSQL Schemas, Migrations & Transactions
- **Source Paths**:
  - Migrations: [`backend/internal/infrastructure/postgres/migrations`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations) (36 SQL migrations)
  - Repositories: [`backend/internal/infrastructure/postgres/repositories`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/repositories)
- **Architecture & Implementation**:
  - Managed via Goose. All forward migrations apply cleanly on boot.
  - Multi-tenancy enforced at database engine level via `FORCE ROW LEVEL SECURITY` (`00012_rls.sql`, `00021_gateway_credentials_rls.sql`).
  - Append-only immutable tables with PostgreSQL triggers preventing `UPDATE` or `DELETE` on `audit_logs` (`00009_audit.sql`) and `security_events` (`00025_security_events.sql`).
  - Strong transactional integrity: atomic balance reservations, idempotent duplicate handling with `ON CONFLICT DO NOTHING`.

### 2.6 Redis Rate Limiting, Budgets & Caching
- **Source Paths**:
  - Rate Limiter: [`backend/internal/interfaces/gatewayv2/redis_rate_limiter.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/redis_rate_limiter.go)
  - Budget Guard: [`backend/internal/interfaces/gatewayv2/budget_guard.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/budget_guard.go)
  - Invalidation: [`backend/internal/infrastructure/invalidation`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/invalidation)
- **Architecture & Implementation**:
  - Sliding-window rate limiting executed via atomic Redis Lua scripts (`redisSlidingWindowLua`).
  - Budget reservation guard tracks in-flight requests using dual Redis structures: Sorted Set (`res_zset`) scored by expiration time and Hash (`res_hash`) mapping request ID to reserved cost, atomically sweeping abandoned or timed-out reservations before checking spend limits.
  - Explicit fail-closed (`AIGW_RATE_LIMIT_FAIL_CLOSED=true`, `AIGW_BUDGET_FAIL_CLOSED=true`) vs degraded-local outage policy.
  - Real-time cache invalidation pub/sub over Redis channels (`aigw:invalidation:policy`, `aigw:invalidation:credential`).

### 2.7 NATS Event Bus & Worker Processing
- **Source Paths**:
  - Event Bus: [`backend/internal/infrastructure/eventbus`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/eventbus)
  - Worker: [`backend/internal/interfaces/worker`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/worker)
- **Architecture & Implementation**:
  - Decouples high-throughput gateway proxying from database write load.
  - NATS JetStream stream `aigw.usage.events` with persistent consumer group `aigw-usage-worker`.
  - Deduplication via NATS message ID and database unique constraints on `llm_requests.request_id`.
  - In-process channel fallback when NATS is disabled in lightweight local development.
  - Background worker executes periodic usage rollup aggregations (`usage_hourly`, `usage_daily`), alert evaluation watchdog, and data retention purges.

### 2.8 Authentication, Sessions, RBAC & Tenant Isolation
- **Source Paths**:
  - Identity: [`backend/internal/application/identity`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/identity)
  - Auth Repositories: [`backend/internal/infrastructure/auth`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/auth)
  - Roles: [`backend/internal/domain/roles.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/domain/roles.go)
- **Architecture & Implementation**:
  - Passwords hashed using PBKDF2-HMAC-SHA256 (600,000 iterations).
  - Multi-tenant role hierarchy: `owner` > `admin` > `billing_admin` > `auditor` > `member`.
  - Subject access control strictly enforced: members cannot query foreign credentials, billing records, invoices, or other users' conversation messages.

### 2.9 Policies, Guardrails, Audit Logs & Security Events
- **Source Paths**:
  - Policies: [`backend/internal/application/policy`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/policy)
  - Guardrails: [`backend/internal/application/guardrails`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/guardrails)
  - Audit Logs: [`backend/internal/application/auditlog`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/auditlog)
  - Security Events: [`backend/internal/application/securityevent`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/securityevent)
- **Architecture & Implementation**:
  - Policy engine enforces model access matrices, prompt template controls, rate limits, and spend quotas.
  - Guardrail evaluator detects and blocks PII (regex pattern matching), secret keys, and prompt injection attacks with configurable thresholds.
  - Immutable audit logs capture administrative actions with actor, IP, timestamp, and diff attributes.
  - Security event logging records anomaly events and supports CSV export for compliance auditing.

### 2.10 Usage Tracking, Pricing & Analytics
- **Source Paths**:
  - Usage Service: [`backend/internal/application/usage`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/usage)
  - Pricing Domain: [`backend/internal/domain/pricing.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/domain/pricing.go)
- **Architecture & Implementation**:
  - Cost calculations use exact integer micro-dollar arithmetic (`costMicros = int64(math.Round(cost * 1_000_000.0))`).
  - Multi-dimensional attribution: usage tracked across organization, project, team, user, service account, provider, and model dimensions.
  - Rollup tables (`usage_hourly`, `usage_daily`) provide instantaneous query response times for overview charts and heatmaps.

### 2.11 Dual-Gateway Billing Integrations (Stripe & Razorpay)
- **Source Paths**:
  - Adapters: [`backend/internal/infrastructure/payment/{stripe,razorpay}`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/payment)
  - Billing Service: [`backend/internal/application/billing`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/billing)
  - Handlers: [`backend/internal/interfaces/controlapi/billing.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/billing.go)
- **Architecture & Implementation**:
  - Three explicitly segregated modes: `disabled`, `simulation`, and `provider_test`.
  - Stripe and Razorpay webhook handlers mandate cryptographic HMAC signature verification.
  - Event replay is verified idempotent (duplicate webhook posts return HTTP 200 without grant duplication).
  - Invoices feature itemized line items with exact token counts and micro-dollar amounts.

### 2.12 Docker Compose & Development Automation Scripts
- **Source Paths**:
  - Compose: [`deploy/compose/docker-compose.yml`](file:///home/sanju/ai-project/ai_gateway/deploy/compose/docker-compose.yml)
  - Control Script: [`scripts/dev-ctl.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/dev-ctl.sh)
  - Verification Script: [`scripts/verify-local.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/verify-local.sh)
- **Architecture & Implementation**:
  - Compose orchestrates PostgreSQL, Redis, NATS, Control API, Gateway-v2, Worker, and Web UI.
  - Readiness probes test functional readiness (`pg_isready` + `SELECT 1`, `redis-cli ping`, NATS `/healthz`, `/readyz` HTTP endpoints).
  - `verify-local.sh` runs a complete 15-stage release-readiness battery non-destructively.

---

## 3. End-to-End User Workflow Tracing

```mermaid
flowchart TD
    User([Client / CLI / Web Browser]) -->|1. Auth: Session Cookie or Bearer Key| ControlAPI[Control API :8081]
    User -->|2. Inference: x-api-key / Bearer| GatewayV2[Gateway v2 :8084]
    
    ControlAPI -->|Manage Tenants, Keys, Policies| Postgres[(PostgreSQL 16\nFORCE RLS)]
    ControlAPI -->|Invalidate Caches| RedisPubSub[Redis PubSub Channel]
    
    GatewayV2 -->|Stage 1: Authenticate Key| CredCache[(In-Memory / Redis Key Cache)]
    GatewayV2 -->|Stage 4: Evaluate Policies| PolicyCache[(Policy Cache)]
    GatewayV2 -->|Stage 6: Rate Limiting| RedisRate[(Redis Sliding Window ZSET)]
    GatewayV2 -->|Stage 7: Budget Guard| RedisBudget[(Redis Spend ZSET + Hash)]
    
    GatewayV2 -->|Stage 8: Route & Stream SSE| ProviderAdapter[Provider Adapter\nAnthropic / Gemini / OpenAI / HF]
    ProviderAdapter -->|Upstream Request| UpstreamLLM[Upstream LLM Provider]
    
    GatewayV2 -->|Stage 9: Publish Usage Event| NATS[NATS JetStream\naigw.usage.events]
    GatewayV2 -->|Stage 10: Real-time Metric| RedisRing[(Redis Activity Ring Buffer)]
    
    NATS -->|Consume Telemetry| Worker[Background Worker :8082]
    Worker -->|Persist LLM Requests & Rollups| Postgres
    
    WebUI[Next.js Web UI :3000] -->|Read Analytics & Heatmaps| ControlAPI
```

### Detailed Workflow Traces:
1. **User Signin & Onboarding**:
   - `POST /v1/auth/signup` creates user, organization, and owner membership in a single transaction. Sets `session` and `session_csrf` cookies.
2. **Gateway Credential Issuance**:
   - Owner/Admin creates credential with restricted `allowed_providers` and `allowed_models`, setting budget and rate limits. Raw secret returned exactly once. SHA-256 hash stored in DB and cached in Redis.
3. **Inference Request Execution**:
   - Client sends inference request to `:8084/v1/chat/completions` or `:8084/v1/messages`.
   - Gateway verifies key, inspects body hints, validates provider/model scoping, checks rate limits in Redis, reserves estimated cost in Redis budget guard, and routes to provider adapter.
   - Streamed chunks are proxied via SSE. Actual tokens extracted from completion. Budget reservation released with actual cost adjustment.
4. **Telemetry Ingestion & Dashboard Rollup**:
   - Gateway emits event to NATS JetStream `aigw.usage.events`.
   - Worker consumes event, deduplicates by `request_id`, records in `llm_requests`, and updates `usage_hourly`/`usage_daily` aggregate rollups.
   - Web UI queries `GET /v1/usage/summary` and `/v1/usage/timeseries` to render interactive charts and activity heatmaps.
5. **Billing Lifecycle**:
   - Admin initiates plan checkout via `POST /v1/billing/checkout`.
   - Depending on mode, system generates simulation URL or provider test session.
   - Provider webhook (`POST /v1/billing/webhooks/{stripe,razorpay}`) arrives with HMAC signature. Handler verifies cryptographic signature, processes subscription entitlement update, and records immutable billing event idempotently.

---

## 4. Verification of Stubs, Mocks & Claims

| Component | Documentation Claim | Actual Implementation Reality | Status |
| :--- | :--- | :--- | :--- |
| **Model Discovery** | "Queries provider dynamic endpoints" | Validates credentials via Hub/v1beta/models APIs; rejects fake fallback models on failure. | **VERIFIED** |
| **CLI Dynamic Menu** | "Filters slash commands by provider" | `GetApplicableCommands(provider)` dynamically bounds menu; SSRF protection on `/endpoint`. | **VERIFIED** |
| **Billing Safety** | "Strict separation of modes" | `disabled` rejects with 403; `simulation` prevents real entitlement grant; `provider_test` requires valid test keys. | **VERIFIED** |
| **Budget Concurrency** | "Zero reservation leakage" | Dual Redis Sorted Set + Hash with atomic Lua sweep cleans up expired or abandoned reservations. | **VERIFIED** |
| **Tenant Isolation** | "Complete isolation between tenants and roles" | PostgreSQL FORCE RLS; RBAC forbids members from reading foreign credentials, invoices, or conversations. | **VERIFIED** |
| **Live External Requests** | "Requires external cloud keys" | Tests execute against deterministic in-memory mock adapters; live provider calls skip when keys are absent. | **VERIFIED BOUNDARY** |

---

## 5. Identified Gaps & Final Readiness Rating

### Remaining Documented Boundaries:
1. **External LLM Cloud API Keys**: Live cloud inference requires user-provided provider keys in `.env`.
2. **Production Merchant Accounts**: Production credit card processing requires live Stripe/Razorpay accounts.
3. **Enterprise SSO**: SAML 2.0 / OIDC federated login is not wired in local dev profile.

### Final Readiness Rating:
**85–90% Functional Local Readiness — READY WITH DOCUMENTED LIMITATIONS**
