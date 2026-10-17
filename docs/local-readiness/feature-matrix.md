# Sentinel AI Gateway / AI Governance — Comprehensive Feature Matrix

**Status Taxonomy**:
- `PASS`: Implementation complete, fully wired to real backend, active tests executing and passing.
- `FAIL`: Implementation present but automated unit or integration tests are currently failing.
- `PARTIAL`: Implementation functionally working but has minor test discrepancies or documentation gaps.
- `BLOCKED`: Capability requires external credentials or resources not available in local offline environment.
- `NOT TESTED`: Feature implemented in code but lacking dedicated automated regression coverage.

---

## Complete Feature Inventory

| # | Feature Area | Route / Entry Point | Backend API / Method | Persistent State | Worker / Event Bus | Evidence & Verification | Audit Status |
|---|---|---|---|---|---|---|---|
| 1 | **Authentication: Registration** | `/signup` | `POST /v1/auth/signup` | `users`, `organizations`, `org_members` | None | `auth_test.go`, `cmd/e2e` (Journey A) | **PASS** |
| 2 | **Authentication: Password Signin** | `/signin` | `POST /v1/auth/signin` | `users`, `refresh_tokens` | None | `auth_test.go`, `cmd/e2e` (Journey A) | **PASS** |
| 3 | **Authentication: Session Introspection** | Middleware | `GET /v1/auth/session` | `refresh_tokens` | None | `auth_test.go`, `cmd/e2e` | **PASS** |
| 4 | **Authentication: Logout** | User Menu | `POST /v1/auth/logout` | `refresh_tokens` | None | `auth_test.go` | **PASS** |
| 5 | **Overview: Operational Telemetry** | `/app/overview` | `GET /v1/usage/summary` | `usage_daily`, `llm_requests` | Worker Rollup | `usage_test.go`, Browser Screenshots | **PASS** |
| 6 | **Overview: Request Volume Timeseries** | `/app/overview` | `GET /v1/usage/timeseries` | `usage_daily`, `usage_hourly` | Worker Rollup | `operations.go`, Browser Screenshots | **PASS** |
| 7 | **Overview: Estimated AI Spend Chart** | `/app/overview` | `GET /v1/usage/timeseries` | `usage_daily`, `usage_hourly` | Worker Rollup | `operations.go`, Browser Screenshots | **PASS** |
| 8 | **Overview: Activity Heatmap** | `/app/overview` | `GET /v1/usage/timeseries` | `usage_hourly` | Worker Rollup | `HeatmapWidget`, Browser Screenshots | **PASS** |
| 9 | **Overview: Token Composition** | `/app/overview` | `GET /v1/usage/summary` | `usage_hourly` | Worker Rollup | `usage_test.go`, Browser Screenshots | **PASS** |
| 10 | **Observe: Real-Time Live Activity** | `/app/observe/live-activity` | `GET /v1/activity` | Redis Ring Buffer / `llm_requests` | Gateway Tracker | `activity_test.go`, `cmd/e2e` (Journey C) | **PASS** |
| 11 | **Observe: Usage Analytics & Sparklines** | `/app/observe/usage` | `GET /v1/usage/by-model`, `/by-project` | `usage_daily`, `usage_hourly` | Worker Rollup | `usage_test.go`, Browser Screenshots | **PASS** |
| 12 | **Observe: Cost Analytics & Attribution** | `/app/observe/costs` | `GET /v1/usage/summary`, `/by-project` | `usage_daily` | Worker Rollup | `usage_test.go`, Browser Screenshots | **PASS** |
| 13 | **Observe: Model Registry & Utilization** | `/app/observe/models` | `GET /v1/models`, `GET /v1/usage/by-model` | `provider_models`, `usage_daily` | None | `providers_test.go`, `cmd/e2e` | **PASS** |
| 14 | **Observe: Provider Health Monitoring** | `/app/observe/providers` | `GET /v1/provider-accounts`, `/health` | `provider_accounts`, `provider_health` | Health Probe | `providers_test.go`, HTTP 200 Probes | **PASS** |
| 15 | **Observe: Conversations & Transcripts** | `/app/observe/conversations` | `GET /v1/conversations`, `/{id}/messages` | `conversations`, `conversation_messages` | None | `conversations_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 16 | **Observe: Conversation Privacy Modes** | `/app/observe/conversations` | `GET /v1/conversations/{id}/messages` | Server-side redaction (4 modes) | None | `conversations_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 17 | **Govern: Policies & Rule Authoring** | `/app/govern/policies` | `GET /v1/policies`, `POST /v1/policies` | `policies`, `policy_rules` | Redis Invalidation | `policies_test.go`, `governance_security_test.go`, `cmd/e2e` | **PASS** |
| 18 | **Govern: Model Access Control Matrix** | `/app/govern/model-access` | `GET /v1/model-access`, `PUT /v1/model-access` | `policy_rules` | Redis Invalidation | `model_access_test.go`, `governance_security_test.go` | **PASS** |
| 19 | **Govern: Token & Spend Budgets** | `/app/govern/budgets` | `GET /v1/budgets`, `POST /v1/budgets` | `budgets`, `policies` | Gateway Budget Guard (Atomic Reserve) | `budget_concurrency_recovery_test.go` (8/8 pass), `budget_guard_test.go`, `cmd/perf` | **PASS** |
| 20 | **Govern: Content Guardrails (PII/Injection)** | `/app/govern/guardrails` | `GET /v1/guardrails`, `POST /v1/guardrails` | `guardrails`, `guardrail_events` | Guardrail Evaluator | `evaluator_test.go`, `service_test.go`, `guardrails_test.go` | **PASS** |
| 21 | **Platform: Gateway Credentials (API Keys)** | `/app/platform/gateway-credentials` | `GET /v1/gateway-credentials`, `POST /v1/...` | `gateway_credentials` (SHA-256) | Redis Key Cache | `gateway_credentials_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 22 | **Platform: Provider Accounts & Rotation** | `/app/platform/provider-accounts` | `GET /v1/provider-accounts`, `POST /...` | `provider_accounts`, `provider_credentials` | AES-GCM Secrets | `providers_test.go`, `cmd/e2e` (Journey A) | **PASS** |
| 23 | **Platform: Machine Service Accounts** | `/app/platform/service-accounts` | `GET /v1/service-accounts`, `POST /...` | `service_accounts` | None | `service_accounts_test.go`, `cmd/e2e` | **PASS** |
| 24 | **Security: Audit Trail Logging** | `/app/security/audit-logs` | `GET /v1/audit-logs`, CSV Export | `audit_logs` (Immutable) | Tx Audit Hook | `audit_logs_test.go`, `governance_security_test.go`, Append-Only Trigger | **PASS** |
| 25 | **Security: Anomaly Alerts & Incidents** | `/app/security/alerts` | `GET /v1/alerts`, `GET /v1/alert-events` | `alerts`, `alert_events` | Worker Watchdog | `evaluator_test.go` (Auto-Recovery), `alerts_test.go` | **PASS** |
| 26 | **Security: Threat & Security Events** | `/app/security/events` | `GET /v1/security-events`, CSV Export | `security_events` | Security Consumer | `security_events_test.go`, `cmd/e2e`, Append-Only Trigger | **PASS** |
| 27 | **Organization: Member Management** | `/app/organization/members` | `GET /v1/members`, `PATCH /v1/members/{id}/role` | `org_members` | None | `members_test.go`, `org_conversation_billing_regression_test.go`, `app/__tests__` | **PASS** |
| 28 | **Organization: Invitations & Onboarding** | `/app/organization/invitations` | `GET /v1/invitations`, `POST /...` | `invitations` | None | `invitations_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 29 | **Organization: Teams & Projects** | `/app/organization/teams`, `/projects` | `GET/POST /v1/teams`, `/v1/projects` | `teams`, `projects`, `team_members` | None | `teams_test.go`, `projects_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 30 | **Organization: Settings & Retention** | `/app/organization/settings` | `GET/PATCH /v1/organization` | `organizations` | Worker Retention | `organization_test.go`, `retention_test.go`, `cmd/e2e` | **PASS** |
| 31 | **Billing: Subscriptions & SaaS Plans** | `/app/billing/subscription` | `GET /v1/billing/subscription`, `/plans` | `subscriptions`, `plans` | None | `billing_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 32 | **Billing: Invoices & Line Items** | `/app/billing/invoices` | `GET /v1/billing/invoices` | `invoices`, `invoice_lines` | None | `billing_test.go`, `org_conversation_billing_regression_test.go` | **PASS** |
| 33 | **Gateway-v2: Streaming Proxy (SSE)** | `http://localhost:8084` | `POST /v1/messages` | `llm_requests` (via NATS) | NATS JetStream | `gateway_test.go`, `cmd/e2e` (Acceptance) | **PASS** |
| 34 | **Gateway-v2: Tenant Base URL Resolution** | `backend/internal/interfaces/gatewayv2` | `ResolveUpstream` | In-memory / Secrets Store | None | `tenant_resolver_test.go` (10/10 passing) | **PASS** |
| 35 | **Sentinel CLI: Interactive Commands** | `packages/cli` | `aigw auth`, `projects`, `models` | CLI Config (`~/.aigw`) | None | `cmd_test.go` | **PASS** |
| 36 | **Gateway-v2: Distributed Rate Limiting** | `http://localhost:8084` | Stage 6 Policy Enforcement | `policies`, Redis ZSET | Redis Sliding Window / Degraded Fallback | `rate_limit_concurrency_recovery_test.go` (8/8 pass), `rate_limiter_test.go`, `cmd/perf` | **PASS** |
| 37 | **Sentinel CLI: Gemini Completion Parsing** | `packages/cli` | `aigw request --model gemini-...` | CLI Client | None | `client_test.go` (10/10 passing) | **PASS** |
| 38 | **Sentinel SDK: Client Protocol & Streaming** | `packages/api-client` | `GatewayClient.sendRequest` | In-memory / Stream Parser | None | `gateway.test.ts` (15/15 pass) | **PASS** |
| 39 | **Live External Upstream Inference** | External Cloud APIs | Upstream Provider Endpoints | External Provider Servers | External Network | Blocked without runtime `GEMINI_API_KEY` | **BLOCKED** |
| 40 | **Enterprise SSO / SAML Authentication** | OAuth / SAML Redirect | OAuth Identity Provider | External IdP | None | Not implemented in dev profile | **NOT TESTED** |
| 41 | **Durable Telemetry & Event Ingestion** | Background Worker | `NATS JetStream` (`aigw.usage.events`) | `llm_requests`, `usage_events`, `usage_daily` | Worker Consumer `aigw-usage-worker` | `durable_event_lifecycle_test.go` (8/8 pass), `event_reliability_test.go` (11/11 pass) | **PASS** |

