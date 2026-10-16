# Sentinel AI Gateway: Acceptance Criteria

**Date:** October 2026  
**Architect:** Sentinel Core Systems Architecture  
**Scope:** Measurable, production-grade acceptance criteria for all 17 end-to-end user journeys and operational requirements.

---

## 1. Product Goal Definition

> A CEO or organization administrator signs in, connects approved AI providers, invites employees, organizes teams and projects, grants model access, sets budgets and guardrails, and monitors usage and security. Employees authenticate to Sentinel and use supported clients to access authorized models through a central gateway. Requests are attributable to the correct human or service account.

Every criterion below is binary (Pass/Fail), independently verifiable, and backed by automated tests.

---

## 2. End-to-End User Journey Criteria

### Journey 1: Organization Signup & Login
* **Criteria:**
  * Submitting valid credentials to `POST /v1/auth/signup` generates an organization, admin user, and sets an `HttpOnly`, `SameSite=Lax`, `Secure` session cookie.
  * Passwords must be hashed using bcrypt or Argon2 with cost factor >= 10; raw passwords must never be logged or persisted.
  * Subsequent `POST /v1/auth/signin` succeeds and returns a cryptographically secure CSRF token in the response or matching cookie.
  * Unauthenticated requests to protected `/v1/*` routes return HTTP 401 Unauthorized.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestAuth_SignupAndLogin ./internal/interfaces/controlapi/...
  ```

---

### Journey 2: Provider Connection & Credential Validation
* **Criteria:**
  * Creating a provider account (`POST /v1/provider-accounts`) and adding credentials (`POST /v1/provider-accounts/{id}/credentials`) triggers an immediate upstream probe.
  * Upstream secret keys are encrypted using AES-GCM via `SecretStore`; database stores only `secret_ref`, key prefix, and fingerprint.
  * Invalid keys immediately reject with HTTP 400 Bad Request or HTTP 422 Unprocessable Entity.
  * API responses never return the raw upstream secret key.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestProvider_Validation ./internal/application/provider/...
  ```

---

### Journey 3: Provider Model Catalog Synchronization
* **Criteria:**
  * Calling `POST /v1/models/sync` fetches all active models from connected upstream providers (Anthropic, OpenAI, Gemini).
  * New models are upserted into the `models` table with correct context window, input token cost, and output token cost.
  * Subsequent `GET /v1/models` returns the synchronized catalog scoped strictly to the organization.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestSyncProviderModels ./internal/interfaces/controlapi/...
  ```

---

### Journey 4: Employee Invitation & Onboarding
* **Criteria:**
  * Inviting an employee via `POST /v1/invitations` generates a cryptographically secure token with a 14-day expiry.
  * Accepting the invitation via `POST /v1/invitations/accept` creates an `organization_members` record with the invited role (`member`, `admin`, or `viewer`) and burns the token.
  * Reusing an expired or burned token returns HTTP 400 Bad Request.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestInvitations_Lifecycle ./internal/interfaces/controlapi/...
  ```

---

### Journey 5: Team & Project Membership
* **Criteria:**
  * Teams created via `POST /v1/teams` and projects via `POST /v1/projects` enforce unique names within the organization.
  * Adding an employee to a team associates them with all projects linked to that team.
  * Cross-tenant member assignment attempts return HTTP 403 Forbidden or HTTP 404 Not Found.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestTeamsAndProjects ./internal/interfaces/controlapi/...
  ```

---

### Journey 6: Credential Issuance (Employee & Service Account)
* **Criteria:**
  * Issuing a gateway credential (`POST /v1/gateway-credentials`) returns the raw API key (`aigw_...`) exactly once in the response payload.
  * Database persists only the SHA-256 hash of the key, along with the first 8 characters as prefix and the last 4 as tail.
  * Credentials must specify subject type (`user` or `service_account`), team ID, and project ID for attribution.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestGatewayCredentials_CreateAndHash ./internal/interfaces/controlapi/...
  ```

---

### Journey 7: Model Access & Policy Evaluation
* **Criteria:**
  * Gateway requests matching a Deny policy or unauthorized model return HTTP 403 Forbidden with standard error structure.
  * Policy changes published via Control API invalidate the gateway cache via Redis Pub/Sub in **< 1ms**.
  * Preview policy dry-run (`POST /v1/policies/preview`) evaluates simulated requests and returns accurate match traces without side effects.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestPolicyInvalidation ./internal/interfaces/gatewayv2/...
  ```

---

### Journey 8: Rate-Limit & Budget Enforcement
* **Criteria:**
  * Distributed rate limiter enforces RPM (requests per minute) and TPM (tokens per minute) using atomic Redis sliding window scripts.
  * Exceeding the rate limit returns HTTP 429 Too Many Requests with `Retry-After` header.
  * Budgets configured at organization, team, or project level decrement atomically; depleted budgets reject requests with HTTP 429.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestBudgetGuard ./internal/interfaces/gatewayv2/...
  ```

---

### Journey 9: Gateway Request & Provider Routing
* **Criteria:**
  * Requests to `/v1/messages` (Anthropic) or `/v1/chat/completions` (OpenAI) authenticate via `Bearer aigw_...` key.
  * Gateway inspects request size; payloads exceeding 8 MiB reject with HTTP 413 Payload Too Large.
  * Upstream requests carry the decrypted provider API key in the authorization header and preserve request parameters byte-for-byte.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestGateway_RoutingAndProxy ./internal/interfaces/gatewayv2/...
  ```

---

### Journey 10: Streaming Response & TTFT Measurement
* **Criteria:**
  * When `stream: true` is passed, response is sent via `text/event-stream` with immediate flushing of each upstream chunk.
  * Time to First Token (TTFT) must be **< 150ms** beyond upstream network latency.
  * Disconnecting the client terminates the upstream HTTP context via `ctx.Done()`, preventing leaked goroutines and unbilled tokens.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestStreamProxy_FlushAndCancel ./internal/interfaces/gatewayv2/...
  ```

---

### Journey 11: Usage & Cost Persistence
* **Criteria:**
  * Completed requests publish usage events containing token counts (input, output, cached, thinking) to NATS JetStream / InMemory bus off the hot path.
  * Worker aggregates usage asynchronously into `usage_hourly` and `usage_daily` tables.
  * Zero dropped usage events under bursts up to 5,000 req/sec.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestUsageObserver_AsyncPublish ./internal/interfaces/gatewayv2/...
  ```

---

### Journey 12: Attribution (User, Team, Project, Service Account)
* **Criteria:**
  * Each record in `llm_requests` accurately records `user_id`, `service_account_id`, `team_id`, and `project_id`.
  * Usage analytics queries (`GET /v1/usage/by-team`, `GET /v1/usage/by-project`) accurately group metrics without orphaned records.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestAttribution_PersistedFields ./internal/infrastructure/postgres/...
  ```

---

### Journey 13: Activity & Audit Queries
* **Criteria:**
  * Administrative mutations (policy update, member invite, credential revocation) append immutable records to `audit_logs`.
  * Queries to `/v1/audit-logs` support server-side pagination and return results ordered by `created_at DESC`.
  * Audit logs must be strictly isolated per tenant (`organization_id`).
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestAuditLogs_ImmutabilityAndIsolation ./internal/interfaces/controlapi/...
  ```

---

### Journey 14: Alerts & Notification Delivery
* **Criteria:**
  * Background worker evaluates spend and error thresholds every 60 seconds against active alert rules.
  * Threshold breaches generate incidents in `alert_history` and trigger configured channels (Webhook, Slack, SMTP).
  * Webhook retries apply exponential backoff (max 3 retries); channel failures do not crash the worker.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestAlertEvaluator_TriggerAndDispatch ./internal/application/alert/...
  ```

---

### Journey 15: Privacy & Retention Enforcement
* **Criteria:**
  * Automated worker retention job purges raw prompts and response bodies from `conversations` older than configured retention period (e.g., 30 days).
  * Aggregated token counts and spend metrics remain intact in `usage_daily`.
  * Retention sweep records an audit entry with count of purged records.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestRetention_PurgeOldRecords ./internal/interfaces/worker/...
  ```

---

### Journey 16: Credential Revocation & Immediate Request Denial
* **Criteria:**
  * Revoking a credential via `DELETE /v1/gateway-credentials/{id}` marks status as `revoked` in PostgreSQL and publishes revocation to Redis Pub/Sub.
  * Subsequent gateway requests using the revoked key must be denied with HTTP 401 Unauthorized within **< 1ms** of revocation.
* **Verification Command:**
  ```bash
  cd backend && go test -v -run TestCredentialRevocation_InstantDenial ./internal/interfaces/gatewayv2/...
  ```

---

### Journey 17: CLI & SDK Authentication and Execution
* **Criteria:**
  * Sentinel CLI (`aigw login`, `aigw config`, `aigw request`) successfully authenticates and routes requests through the gateway.
  * TypeScript client (`@ai-governance/api-client`) and Python client (`sentinel-ai`) provide complete type coverage and support streaming responses.
* **Verification Command:**
  ```bash
  pnpm --filter @ai-governance/api-client test
  cd packages/cli && go test -v ./...
  ```

---

## 3. Quantitative Non-Functional Benchmarks

| Metric | Target SLA | Verification Method |
|---|---|---|
| **Gateway TTFT Overhead** | < 15ms added proxy latency | Benchmark proxy vs direct mock upstream |
| **Policy Invalidation Latency** | < 1ms across all gateway nodes | Redis Pub/Sub subscriber latency test |
| **Credential Revocation Window** | < 1ms to HTTP 401 rejection | Concurrent revocation test suite |
| **Max Request Payload Size** | 8 MiB (HTTP 413 for oversized) | HTTP request inspector middleware test |
| **Tenant Isolation Leakage** | 0 cross-tenant data leaks | Automated multi-tenancy fuzzing test |
| **Secret Material in Logs/DB** | 0 raw secrets logged or in DB | Automated secret regex scanning test |

---

## 4. End-to-End Orchestrated Validation Command

The entire end-to-end user lifecycle across all 17 journeys is continuously verifiable using the master test harness:

```bash
cd backend && go run ./cmd/e2e/main.go
```
