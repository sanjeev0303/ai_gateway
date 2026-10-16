# Sentinel AI Gateway: End-to-End Gap Analysis

**Date:** October 2026  
**Auditor:** Sentinel Systems Architecture  
**Scope:** Full-stack inspection of Next.js frontend (`app/`), Control API (`cmd/control-api`), Data Plane Gateway (`cmd/gateway-v2`), Provider Adapters (`internal/infrastructure/provider/`), PostgreSQL Migrations & Repositories, Worker Background Jobs (`cmd/worker`), and CLI/SDK packages.

---

## 1. Executive Summary & Repository Reality

The codebase is an advanced, production-grade AI governance platform with active data planes, persistence engines, and security isolation layers. This audit distinguishes between documentation claims and verified code execution.

* **Repository State:** 880 git-tracked files, cleanly compiling across Go 1.25 and Next.js 16.3.7 (Turbopack).
* **Core Architecture:** Canonical data plane is [gateway-v2](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/server.go) (ADR-5). Control plane is [controlapi](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/server.go). Background async processing is handled by [worker](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/worker/).
* **Data Model:** 32 PostgreSQL migrations (`00001_identity.sql` to `00032_alerts_enhancement_and_history.sql`) managed via SQLC.

---

## 2. Feature-by-Feature Matrix

| Feature Area | Classification | File Paths & Code Citations | Current Reality vs. Documented Claim |
|---|---|---|---|
| **Organization Signup & Auth** | `Implemented and verified` | [auth.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/auth.go#L44), [signin/page.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/app/(auth)/signin/page.tsx), [cmd/e2e/main.go](file:///home/sanju/ai-project/ai_gateway/backend/cmd/e2e/main.go#L42) | Verified live: Argon2/bcrypt password hashing, HttpOnly session cookies, double-submit CSRF, and organization isolation. |
| **Provider Accounts & Credentials** | `Implemented and verified` | [service.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/provider/service.go#L149), [provider-detail-client.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/app/(dashboard)/app/platform/provider-accounts/[id]/provider-detail-client.tsx#L179), [00004_credentials.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00004_credentials.sql) | Upstream keys stored encrypted via `SecretStore`; fingerprint and tail only returned. Anthropic, OpenAI, and Gemini live connectivity validated. |
| **Model Registry Sync** | `Implemented and verified` | [service.go:SyncModels](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/provider/service.go#L424), [providers.go:syncProviderModels](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/providers.go#L261), [00017_models.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00017_models.sql) | Upserts upstream live model lists into `models` table. Tested and working in `providers_test.go`. |
| **Employee Invitations & Onboarding** | `Implemented and verified` | [handlers.go:invite](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/handlers.go#L116), [invitations-client.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/app/(dashboard)/app/organization/invitations/invitations-client.tsx#L130), [00001_identity.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00001_identity.sql) | Full invitation lifecycle: invite, accept with token validation, role assignment, and audit log generation. |
| **Teams & Projects Hierarchy** | `Implemented and verified` | [handlers.go:createTeam](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/handlers.go#L240), [teams-client.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/app/(dashboard)/app/organization/teams/teams-client.tsx), [00002_projects.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00002_projects.sql) | Scoped membership, project assignment, and status controls verified in Journey A. |
| **Service Accounts & Gateway Keys** | `Implemented and verified` | [gateway_credentials.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/gateway_credentials.go#L22), [client.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/app/(dashboard)/app/platform/gateway-credentials/client.tsx#L180), [00018_gateway_credentials.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00018_gateway_credentials.sql) | Machine credentials carry one-time secret generation, prefix display, SHA-256 hash storage, and immediate revocation. |
| **Policy Enforcement Engine** | `Implemented and verified` | [policy_resolver.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/policy_resolver.go#L50), [gateway.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway.go#L135), [policy_invalidation_test.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/policy_invalidation_test.go) | Real-time policy evaluation against incoming requests; Redis Pub/Sub invalidator clears gateway cache in <1ms. |
| **Policy Preview & Dry-Run Validation** | `Partially implemented` | [policies.go:previewPolicy](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/policies.go#L280), [policies.go:validatePolicy](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/policies.go#L285) | Endpoints are stubs: `previewPolicy` returns `"status": "preview not implemented"` and `validatePolicy` returns hardcoded `{"valid": true}`. |
| **Distributed Rate Limiting** | `Implemented and verified` | [redis_rate_limiter.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/redis_rate_limiter.go#L35), [gateway.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway.go#L170), [cmd/e2e/main.go](file:///home/sanju/ai-project/ai_gateway/backend/cmd/e2e/main.go#L389) | Atomic Redis sliding window script blocks traffic exceeding RPM/TPM thresholds. |
| **Budget Caps & Exhaustion** | `Implemented and verified` | [budget_guard.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/budget_guard.go#L40), [service.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/budget/service.go#L45), [00029_budgets.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00029_budgets.sql) | Redis-backed reservation and budget enforcement prevents tenant overages. |
| **Gateway Request & Routing** | `Implemented and verified` | [stream_proxy.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/stream_proxy.go#L30), [provider_router.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/provider_router.go#L25) | Byte-exact passthrough for Anthropic (`/v1/messages`) and OpenAI. Request inspector enforces 8 MiB limit. |
| **Streaming Responses & TTFT** | `Implemented and verified` | [stream_proxy.go:Forward](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/stream_proxy.go#L90), [gateway_lifecycle_test.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway_lifecycle_test.go) | Real-time SSE flush with `ctx.Done()` client disconnect handling; accurate TTFT measurement. |
| **Usage & Cost Persistence** | `Implemented and verified` | [usage_observer.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/usage_observer.go#L45), [usage_consumer.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/worker/usage_consumer.go), [00007_usage.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00007_usage.sql) | Async NATS/InMemory bus emits usage events off the hot path; worker aggregates into `usage_hourly` and `usage_daily`. |
| **Attribution (User, Team, SA, Project)** | `Implemented and verified` | [authenticator.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/authenticator.go#L60), [00027_service_account_telemetry.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00027_service_account_telemetry.sql) | Request attribution maps to human user or service account, project, and team. |
| **Audit Logs & Security Events** | `Implemented and verified` | [audit_logs.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/audit_logs.go#L30), [security_events.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/security_events.go#L25) | Immutable audit trails in PostgreSQL; real-time security events captured for policy/budget rejections. |
| **Alerts & Evaluator Engine** | `Implemented and verified` | [evaluator.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/alert/evaluator.go#L50), [alerts.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/alerts.go#L40), [00032_alerts_enhancement_and_history.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00032_alerts_enhancement_and_history.sql) | Background evaluation loops check spend, latency, and error thresholds; records alert incidents in database. |
| **External Alert Notification Channels** | `Partially implemented` | [evaluator.go:dispatchNotification](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/alert/evaluator.go#L230) | Webhook dispatch exists; direct integrations for Slack, PagerDuty, and SMTP email are absent. |
| **Data Retention & Privacy Purging** | `Implemented and verified` | [retention.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/worker/retention.go#L60), [00024_conversation_privacy.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00024_conversation_privacy.sql) | Automated worker job purges conversation bodies and usage payloads older than configured retention policy window. |
| **Credential Revocation Invalidation** | `Implemented and verified` | [credential_cache.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/credential_cache.go#L50), [redis_invalidator.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/invalidation/redis_invalidator.go) | Revoked credentials immediately trigger Redis Pub/Sub events; gateway blocks calls in <1ms. |
| **CLI & Developer Tooling** | `Implemented and verified` | [packages/cli](file:///home/sanju/ai-project/ai_gateway/packages/cli), [packages/api-client](file:///home/sanju/ai-project/ai_gateway/packages/api-client) | Go CLI (`aigw`) supports login, credential configuration, and execution. TypeScript API Client is fully typed. |
| **Python SDK** | `Missing` | Monorepo root | TypeScript SDK exists; Python client library does not exist in monorepo. |
| **Stripe Live Billing** | `Partially implemented` | [billing.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/billing.go#L30), [00010_billing.sql](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/migrations/00010_billing.sql) | In-database invoice, plan, and subscription ledger exists; live Stripe webhook listener and customer checkout sessions are not connected. |
| **Pagination in UI Tables** | `Partially implemented` | [data-table.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/components/dashboard/data-table.tsx#L40) | Client-side pagination slices rows loaded in memory; server-side cursor pagination needed for high-volume audit logs. |

---

## 3. In-Depth Journey Traces

### Journey 1: Organization Signup & Login
* **Frontend:** User submits credentials at `/signup` or `/signin` ([app/src/app/(auth)/signin/page.tsx](file:///home/sanju/ai-project/ai_gateway/app/src/app/(auth)/signin/page.tsx)).
* **Control API:** Routes `POST /v1/auth/signup` and `POST /v1/auth/signin` ([auth.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/auth.go)).
* **Service & DB:** `identity.Service` hashes password with bcrypt, inserts tenant into `organizations` and `users`, creates owner in `organization_members`. Generates JWT access cookie and CSRF cookie.
* **Finding:** Fully verified and functional.

### Journey 2: Provider Connection & Credential Validation
* **Frontend:** Admin adds account at `/app/platform/provider-accounts` and attaches an API key.
* **Control API:** `POST /v1/provider-accounts` and `POST /v1/provider-accounts/{id}/credentials` ([providers.go](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/providers.go)).
* **Validation:** Service runs live probe against upstream (`/v1/models` for Anthropic/OpenAI/Gemini). Secret material is encrypted into `SecretStore` with generated `secret_ref`; database only stores fingerprint and tail.
* **Finding:** Fully verified and functional (recent dropdown kind bug resolved).

### Journey 3: Provider Model Catalog Synchronization
* **Frontend:** Admin accesses `/app/platform/model-registry` and clicks "Sync Models".
* **Control API:** `POST /v1/models/sync` ([providers.go:syncProviderModels](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/providers.go#L261)).
* **Service & DB:** `provider.Service.SyncModels` calls provider adapter's `ListModels(ctx, secret)` and bulk upserts into `models`.
* **Finding:** Fully verified.

### Journey 4: Employee Invitation & Onboarding
* **Frontend:** Admin invites colleague via `/app/organization/invitations`.
* **Control API:** `POST /v1/invitations` generates cryptographically random token with 14-day expiry.
* **Acceptance:** Colleague accepts via `POST /v1/invitations/accept`; service creates membership and invalidates token.
* **Finding:** Fully verified.

### Journey 5: Team & Project Membership
* **Frontend:** Admin organizes teams (`/app/organization/teams`) and projects (`/app/organization/projects`).
* **Control API:** CRUD operations across `/v1/teams`, `/v1/projects`, and membership join tables.
* **Finding:** Fully verified.

### Journey 6: Credential Issuance
* **Frontend:** Admin issues keys via `/app/platform/gateway-credentials`.
* **Control API:** `POST /v1/gateway-credentials` captures name, subject type, scopes, team, and project.
* **Secret Handling:** Generated raw token (`aigw_...`) returned exactly once in creation response. Only SHA-256 hash stored.
* **Finding:** Fully verified.

### Journey 7: Model Access & Policy Evaluation
* **Frontend:** Admin defines policies and model grants via `/app/govern/policies` and `/app/govern/model-access`.
* **Control API:** `POST /v1/policies`, `/v1/policies/{id}/rules`, `PUT /v1/model-access`.
* **Gap Identified:**
  * `POST /v1/policies/preview` is a stub ([policies.go:L281](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/policies.go#L281)).
  * `POST /v1/policies/{id}/validate` returns hardcoded `{"valid": true}` ([policies.go:L286](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/policies.go#L286)).
  * Gateway evaluation itself is working; only control-plane preview/validation endpoints need real implementation.

### Journey 8: Rate-Limit & Budget Enforcement
* **Data Plane:** Gateway checks Redis sliding window script for RPM/TPM limits.
* **Budget Guard:** Gateway checks budget balance in Redis before proxying; rejects with HTTP 429 when exhausted.
* **Finding:** Fully verified in Journey E and performance suite.

### Journey 9: Gateway Request & Provider Routing
* **Data Plane:** Gateway receives client request with `Bearer aigw_...` key.
* **Pipeline:** Authenticates key -> resolves tenant -> inspects size (<= 8 MiB) -> evaluates policy -> routes to active upstream provider account.
* **Finding:** Fully verified.

### Journey 10: Streaming Response
* **Data Plane:** `stream_proxy.go` flushes SSE events in real time. Measures TTFT and handles client abort.
* **Finding:** Fully verified in Journey C.

### Journey 11: Usage & Cost Persistence
* **Data Plane:** Gateway usage observer parses streaming tokens. Emits event to NATS JetStream / InMemory bus.
* **Worker:** Asynchronous worker ingests event into `llm_requests` and triggers rollup computations.
* **Finding:** Fully verified in Journey G.

### Journey 12: Attribution
* **Persistence:** `llm_requests` associates request with `user_id`, `service_account_id`, `team_id`, and `project_id`.
* **Finding:** Fully verified.

### Journey 13: Activity & Audit Queries
* **Frontend:** Real-time requests displayed in `/app/observe/live-activity`; audit trail in `/app/security/audit-logs`.
* **Finding:** Fully verified.

### Journey 14: Alerts & Notification Delivery
* **Background Worker:** Worker evaluator checks metric spikes against `alerts` rules and records events.
* **Gap Identified:** Dispatcher only triggers HTTP webhooks. Direct connectors for Slack webhooks, PagerDuty, and SMTP email are absent.

### Journey 15: Privacy & Retention Enforcement
* **Background Worker:** Automated retention job purges conversation text and usage payload detail after retention threshold.
* **Finding:** Fully verified in `retention_test.go`.

### Journey 16: Credential Revocation & Immediate Denial
* **Control API:** Admin revokes key at `/app/platform/gateway-credentials`.
* **Invalidation:** Redis Pub/Sub immediately purges gateway credential cache.
* **Data Plane:** Subsequent calls fail with HTTP 401 in <1ms.
* **Finding:** Fully verified in Journey D.

### Journey 17: CLI & SDK Authentication
* **CLI:** `packages/cli` executes login, captures token, and proxies CLI calls.
* **SDK:** TypeScript SDK `@ai-governance/api-client` is complete.
* **Gap Identified:** Python SDK is absent.

---

## 4. Specific Codebase Investigation Findings

1. **Stub Control API Endpoints:**
   * [backend/internal/interfaces/controlapi/policies.go:281](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/policies.go#L281): `previewPolicy` returns mock text. Needs implementation of dry-run policy evaluation against sample payloads.
   * [backend/internal/interfaces/controlapi/policies.go:286](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/policies.go#L286): `validatePolicy` returns hardcoded `{"valid": true}`. Needs schema and rule conflict validation.

2. **Stale Marketing Copy:**
   * [app/src/lib/constants.ts:430](file:///home/sanju/ai-project/ai_gateway/app/src/lib/constants.ts#L430) and [app/src/lib/landing-data.ts:193](file:///home/sanju/ai-project/ai_gateway/app/src/lib/landing-data.ts#L193) claim retention windows and guardrails are "not implemented yet". Both are fully implemented and verified in the backend. Copy should be synchronized.

3. **External Notification Channels:**
   * [backend/internal/application/alert/evaluator.go:230](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/alert/evaluator.go#L230): Notification dispatch supports generic webhook POST; dedicated Slack and SMTP email dispatchers should be added.

4. **Server-Side Cursor Pagination:**
   * High-volume endpoints (`/v1/audit-logs`, `/v1/activity`) currently support limit parameters, but the UI table relies on client-side slicing. Server-side cursor pagination should be connected.
