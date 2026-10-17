# Implementation Task Tracker

Updated after each completed prompt.

## ACTIVE TASK: Git Commit and Push to GitHub on Explicit User Command
- [ ] Step 1: Stage verified modifications and untracked readiness files (`git add -A`)
- [ ] Step 2: Commit changes with descriptive message
- [ ] Step 3: Push to remote repository (`git push origin main`)
- [ ] Step 4: Verify clean git status and remote synchronization

## PREVIOUS TASK: Prompt 8 — Final Sentinel Cleanup, Consistency, and Local Release Gate
- [x] Step 1: Inventory & Audit: analyze untracked files, scratch files, dead code candidates, duplicate test helpers, doc contradictions, and secret exposure risks.
- [x] Step 2: Safe Dead Code & Fixture Pruning: remove verified obsolete/unused files with proof of unreachability, preserving migrations, SDK/CLI interfaces, scripts, adapters, and tooling.
- [x] Step 3: Consistency & Secret Redaction Hardening: resolve contradictions across docs, env examples, API contracts, and ensure startup/config errors never leak secrets.
- [x] Step 4: Test Independence & Provider Transparency: ensure tests do not rely on hidden env vars or accidental state, and explicitly identify mock vs fixture vs live providers.
- [x] Step 5: Final Release Gate & Verification: execute full release-readiness verification (`./scripts/verify-local.sh`, `go run ./cmd/e2e`, `go run ./cmd/perf`).
- [x] Step 6: Deliverables: produce `docs/local-readiness/final-cleanup-report.md`, update `docs/local-readiness/feature-matrix.md`, and update `task.md`.

## PREVIOUS TASK: Prompt 7 — Sentinel Full Local End-to-End and Recovery Verification
- [x] Scenario 1: Fresh Local Setup & Topology Verification: validate clean local environment setup, services (Postgres, Redis, NATS, Control API, Gateway v2, Worker, Frontend), migrations, and readiness probes.
- [x] Scenario 2: Three-User Request Flow & Governance Matrix: test 3 users with individual gateway credentials in one organization, SDK & CLI, streaming vs non-streaming, invalid models, revoked credentials, policy denials, guardrails, rate limits, and attribution.
- [x] Scenario 3: Dashboard & API Cross-Verification: verify UI and API data consistency with persisted records across traffic charts, activity, usage, models, tokens, cost, conversations, guardrails, alerts, and audit logs.
- [x] Scenario 4: Failure & Recovery Testing: test provider failures, service restarts (gateway, worker, event bus), database and Redis unavailability, timeouts, stream interruption, fail-closed/fail-open behavior, and zero usage duplication/loss.
- [x] Scenario 5: Full Regression & Quality Checks: execute all test suites (Go unit/integration/race/vet, TypeScript lint/tsc/test/build, SDK/CLI tests, migrations, API contracts, E2E journeys).
- [x] Step 6: Final Deliverables: create `docs/local-readiness/final-test-report.md`, `docs/local-readiness/remaining-gaps.md`, update `docs/local-readiness/local-runbook.md`, update `docs/local-readiness/feature-matrix.md`, and update `task.md`.

## PREVIOUS TASK: Prompt 6 — Complete Organization Workflows, Conversation History, and Local Billing
- [x] Step 1: Organization Lifecycle & Hierarchy: audit & test org creation, invitation lifecycle (creation, validation, acceptance, expiration, revocation, duplicate checks), membership management, role changes, team & project membership persistence, and privilege escalation prevention.
- [x] Step 2: Gateway Credentials Management: audit individual user gateway credentials creation, display, rotation, revocation, scoping, and authorization checks.
- [x] Step 3: Conversation History & Privacy Redaction: trace user/org request attribution, persistence, retrieval (`GET /v1/conversations`, `/{id}/messages`), pagination, filters, ownership enforcement, 4 privacy redaction modes, retention policy, and leak prevention.
- [x] Step 4: Usage & Local Billing Mechanics: audit request/user/org cost aggregation, verify historical stability, test subscription and invoice endpoints using real DB data vs labeled demo fixtures, document usage-only reporting scope, verify rounding, currencies, date boundaries, and reconciliation.
- [x] Step 5: Frontend Interface Resilience: verify loading, empty, error, and populated states across organization, conversation, and billing pages; fix broken routes/links/contracts; ensure isolated widget failure handling.
- [x] Step 6: Test Suite, Documentation & Matrix: write regression tests for org workflows, conversations, and billing; update `docs/local-readiness/feature-matrix.md`; create `docs/local-readiness/organization-conversation-billing.md`; run `./scripts/verify-local.sh`; update `task.md`.

## PREVIOUS TASK: Prompt 5 — Complete Sentinel Governance and Security Enforcement
- [x] Step 1: Policies & Model Access: audit and test policy CRUD (create, read, update, enable, disable, delete), allow/deny rule precedence, conflict resolution, unsupported models, pre-request server enforcement, cache invalidation timebounds, and gateway restart recovery.
- [x] Step 2: Budgets & Rate Limits: audit organization & user limits, threshold edge cases (below, at, above), concurrency race defense, zero provider usage/cost accrual on rejection, reset windows, and remaining budget calculation.
- [x] Step 3: Guardrails Pipeline & Content Redaction: audit regex, keyword, PII, and length guardrails; enforce pre-forwarding execution; record blocked requests without upstream forwarding; verify sensitive data redaction in logs and audit trails.
- [x] Step 4: Alerts Engine & Trigger Lifecycle: audit all alert condition evaluators, test threshold crossing, no-trigger conditions, repeat suppressions, auto-recovery/resolution, trigger history persistence, and API/dashboard visibility.
- [x] Step 5: Audit Logs & Security Events: audit security-relevant event emission (auth failures, credential mutations, policy/guardrail updates, role changes), tamper resistance, org isolation, RBAC read gating, and secret/prompt redaction.
- [x] Step 6: Test Suite, Feature Matrix & Documentation: expand automated regression tests across all governance domains, update `docs/local-readiness/feature-matrix.md`, create `docs/local-readiness/governance-and-security-verification.md`, run `./scripts/verify-local.sh`, and update `task.md`.

## PREVIOUS TASK: Prompt 4 — Make Usage Metering, Event Processing, and Analytics Reliable
- [x] Step 1: Event Bus & Worker Pipeline Audit: inspect NATS JetStream / Memory broker, stream definitions, subject bindings, consumer group config, ack/nack behavior, and redelivery semantics.
- [x] Step 2: Idempotency & Deduplication in Worker Persistence: ensure replayed or duplicate events on `request_id` are idempotent and cannot double-count usage metrics or billing aggregates.
- [x] Step 3: Usage Metering, Cost Calculation & Historical Explainability: verify model pricing lookup, token estimation vs exact provider-reported tags, streaming/interrupted request handling, and cost snapshot storage.
- [x] Step 4: Analytics Aggregation, Timezones & Redis Cache Safety: audit `/v1/usage/*` and `/v1/analytics/*`, date boundaries, UTC/local timezone handling, pagination, empty states, and tenant-scoped Redis caching with proper TTL and invalidation.
- [x] Step 5: Comprehensive Event Reliability Test Suite: test duplicate delivery, worker crashes/restarts, event bus backlog recovery, database outage handling, missing token metadata, streaming interruption, failed provider calls, 3-user reconciliation, and cache invalidation.
- [x] Step 6: Documentation & Verification: create `docs/local-readiness/usage-and-event-reliability.md`, run `./scripts/verify-local.sh`, and update `task.md`.

## PREVIOUS TASK: Prompt 3 — Complete Sentinel Gateway Request Lifecycle and Client Integrations
- [x] Phase A: Request Lifecycle Audit & Gap Closure: trace full lifecycle (client -> gateway credential -> auth -> org/user resolution -> model access -> guardrails -> budget/rate limit -> provider adapter -> response -> telemetry recording -> DB persistence -> dashboard analytics).
- [x] Phase B: Multi-Provider Integration Hardening: verify Anthropic, OpenAI, and Gemini formats, error mapping (timeouts, 401, 429, 500), streaming/SSE flush, client disconnects, and token capture/estimation.
- [x] Phase C: Sentinel SDK and CLI Audit & Hardening: audit public APIs, user gateway key auth, streaming, retries/idempotency, timeouts, and error handling.
- [x] Phase D: Multi-User Local Fixture & End-to-End Attribution: test 3 independent users with individual gateway credentials in one organization, attributing requests accurately; execute deterministic mock tests and live smoke test.
- [x] Phase E: Deliverables & Local Readiness Documentation: create `docs/local-readiness/gateway-sdk-cli-verification.md` with client/provider/scenario test matrix and usage instructions.
- [x] Phase F: Verification: run test suites and `scripts/verify-local.sh`, then update `task.md`.

## PREVIOUS TASK: Prompt 2 — Complete Local Authentication and Tenant Isolation
- [x] Step 1: Audit Authentication & Session Lifecycle: audit password hashing (PBKDF2-HMAC-SHA256, 600k iterations, decoy timing defense), login/logout, session expiry, revoked sessions, cookie security flags (`HttpOnly`, `SameSite=Lax/Strict`), and CSRF double-submit protection.
- [x] Step 2: Audit Tenant Isolation & IDOR Protection: inspect all control-plane and data-plane routes ensuring cross-tenant queries and manipulated IDs are strictly blocked via PostgreSQL FORCE RLS and application actor contexts.
- [x] Step 3: Audit RBAC & Gateway Credential Scoping: verify role-based access for administrative vs member operations (`RoleOwner`, `RoleAdmin`, `RoleMember`), key scoping (`gateway`, `control`, `all`), rotation, and real-time revocation behavior.
- [x] Step 4: Implement Identified Gaps: patched policy and guardrails service mutations to enforce administrative role authorization checks (`isPolicyManager`, `isGuardrailManager`).
- [x] Step 5: Regression & Multi-Tenant Test Suite: updated `backend/internal/interfaces/controlapi/multi_tenancy_security_test.go` and `backend/internal/interfaces/gatewayv2/gateway_lifecycle_test.go` with 2 independent organizations, 3 users, multiple roles, manipulated IDs across all 9 resource families, subject-level privacy, and gateway key revocation.
- [x] Step 6: Documentation: produced `docs/local-readiness/authentication-and-isolation.md` documenting test evidence, security architecture, and limitations.
- [x] Step 7: Verify & Update Task Tracker: ran `scripts/verify-local.sh` (15/15 checks passed) and updated `task.md`.

## PREVIOUS TASK: Prompt 1 — Secure and Reproducible Local Development Environment
- [x] Step 1: Config & Environment Validation: enhance `backend/internal/platform/config/config.go` with strict startup validation, placeholder secret detection, dev-convenience controls, and safe redaction.
- [x] Step 2: Automated Tests for Config Validation: update `backend/internal/platform/config/config_test.go` to test missing variables, placeholder rejection, dev-mode flags, and secret redaction.
- [x] Step 3: Environment Examples: audit and sanitize `deploy/compose/.env.example` and `backend/.env.example` with clear placeholder conventions and security guidance.
- [x] Step 4: Docker Compose & Startup Ordering: update `deploy/compose/docker-compose.yml` to ensure healthchecks use functional readiness (`/readyz`) and dependencies include postgres, redis, and nats JetStream.
- [x] Step 5: Safe Local Service Lifecycle Script: create `scripts/dev-ctl.sh` and update `Makefile` with safe start, stop, status, and confirmed reset commands.
- [x] Step 6: Fix Verification Script & Baseline Defects: update `scripts/verify-local.sh` to never overwrite existing `.env`, display checkout commit and endpoints, and resolve baseline test issues in `tenant_resolver_test.go`, `client_test.go`, and node types.
- [x] Step 7: Local Readiness Documentation: create `docs/local-readiness/environment-verification.md` and update `docs/local-readiness/local-setup.md`.
- [x] Step 8: Execution & Verification: run test suites, verify local services and lifecycle commands, record results, and update `task.md`.

## PREVIOUS TASK: Prompt 0 — Sentinel Local-Stage Audit and Baseline
- [x] Step 1: Inspect repository structure, git status, compose configuration, migrations, entry points
- [x] Step 2: Verify running processes/containers, ports, and alignment with checkout
- [x] Step 3: Execute test suites across Go, TypeScript/Next.js, SDK, CLI, and integration tests; record results
- [x] Step 4: Trace full request and telemetry lifecycle across auth, gateway, worker, DB, and dashboard
- [x] Step 5: Audit and classify features (PASS, FAIL, PARTIAL, BLOCKED, NOT TESTED)
- [x] Step 6: Produce deliverables (`current-state.md`, `feature-matrix.md`, `risk-register.md`, `baseline-test-results.md`)
- [x] Step 7: Update `task.md`

## PREVIOUS TASK: Git Push & Enforce Git Add/Push Policy by Command
- [x] Step 1: Stage verified codebase modifications (`git add -A`)
- [x] Step 2: Commit changes with descriptive message
- [x] Step 3: Push changes to remote GitHub repository (`git push origin main`)
- [x] Step 4: Record policy constraint: only perform git add and git push upon explicit user command

## PREVIOUS TASK: Fix Conversations Visibility, Dashboard Graphs, Heatmaps, and Data Presentation Across All Dashboard Pages
- [x] Step 1: Diagnose & Fix Conversations endpoint and persistence for live user traffic (`/app/observe/conversations`, `GET /v1/conversations`)
- [x] Step 2: Diagnose & Fix Overview Page graphs, timeseries, heatmap, and presentation components (`/app/overview`, `/v1/usage/timeseries`, etc.)
- [x] Step 3: Inspect & Fix Usage page charts and breakdowns (`/app/observe/usage`)
- [x] Step 4: Inspect & Fix Cost & Billing page charts and metrics (`/app/billing/usage`, `/app/billing/invoices`)
- [x] Step 5: Inspect & Fix Model Access, Budgets, Guardrails, Credentials, Events, Alerts, Provider Health pages for real data display
- [x] Step 6: Verify all components and charts render rich live data for `sanju.04044@gmail.com`

## PREVIOUS TASK: Live Developer Account Gemini Integration & Multi-User Verification (`sanju.04044@gmail.com`)
- [x] Step 1: Developer Account Authentication (`sanju.04044@gmail.com`, Organization `cb4f2b7b-78cf-4c7e-b376-6be86f484d59`)
- [x] Step 2: Live Gemini Provider Account & Credential Registration (`499aff99-fd9a-4ba8-8f57-1edd45b261eb`)
- [x] Step 3: Model Catalog Live Synchronization (30+ Gemini models synced)
- [x] Step 4: Three Independent Test Users Created & Invited to Organization
- [x] Step 5: Individual Sentinel Gateway Credentials Issued for Each Test User
- [x] Step 6: Sentinel CLI (`aigw`) Validation with Live Gemini Prompts (Streaming & Non-Streaming)
- [x] Step 7: Sentinel SDK Validation with Live Gemini Endpoint (Streaming & Non-Streaming)
- [x] Step 8: Guardrails & Governance Enforcement Testing (Disallowed model rejection)
- [x] Step 9: Multi-User Concurrent Traffic Generation & Telemetry Ingestion
- [x] Step 10: Live Dashboard Verification (Overview, Usage, Billing, Models, Guardrails, Audit Logs HTTP 200)

## PREVIOUS TASK: Master Execution — End-to-End Gemini Integration, Test Users, CLI/SDK Validation, and Full Sentinel Feature Testing

## PREVIOUS TASK: Dashboard Pages UI & Actionable Functionality Overhaul
- [x] Section 1: Audit Logs & Security Events UI (`/app/security/audit-logs`, `/app/security/events`): add live filter bar, export CSV, severity tags, real-time trigger inspection
- [x] Section 2: Alerts & Notifications UI (`/app/security/alerts`): add active incident tabs, acknowledge/resolve action buttons, in-app notification center
- [x] Section 3: Credentials & Gateway Credentials (`/app/govern/credentials`, `/app/platform/gateway-credentials`, `/app/platform/provider-credentials`): issue key modals, one-time secret copy, revoke & rotate buttons
- [x] Section 4: Providers & Models (`/app/platform/provider-health`, `/app/platform/provider-accounts`, `/app/platform/model-registry`): probe live button, add provider dialog, sync models button, status indicators
- [x] Section 5: Organization Management (`/app/organization/members`, `/app/organization/invitations`, `/app/organization/teams`, `/app/organization/projects`, `/app/platform/service-accounts`, `/app/organization/settings`): member invite/role modal, create team/project dialogs, service account issuance, privacy/retention forms
- [x] Section 6: Usage, Subscription & Invoices (`/app/observe/usage`, `/app/billing/usage`, `/app/billing/subscription`, `/app/billing/invoices`): real usage breakdown, plan status/upgrade dialogs, simulated invoice downloads, payment method actions
- [x] Section 7: Quality validation, docker container build and browser client verification


## PREVIOUS TASK: Prompt 7 — Local Production-Readiness and End-to-End Acceptance
- [x] Section 1: Local Environment & Deterministic Seed Data (seed one org, admin, multiple employees, teams, projects, provider accounts, models, gateway credentials, model-access rules, budgets, guardrails, requests/usage records, audit events, alerts; clearly distinguish synthetic seed data from live provider data)
- [x] Section 2: Complete 21-Step Acceptance Journey Execution & Verification
- [x] Section 3: Quality Gates (Go tests, Go race tests, Frontend type checks, Frontend lint, Frontend build, API contract checks, Migration checks, Integration tests, End-to-end tests, Local load tests for gateway and usage endpoints)
- [x] Section 4: Final Documentation Deliverables (feature-completeness.md, end-to-end-validation.md, known-limitations.md, runbook.md in docs/local-readiness/)
- [x] Section 5: Update `task.md` on completion
- [x] Section 6: Client UI Refresh & Guardrails Implementation (built guardrails client module, full CRUD & test UI in `/app/govern/guardrails`, re-evaluated alert descriptions, seeded user passwords, and rebuilt `ai-gateway-web:dev` container)

## PREVIOUS TASK: Prompt 6 — Complete Alerting, Privacy, and Retention
- [x] Section 1: Real Alert Evaluation Engine & Conditions (budget warning/exhaustion, usage/spend spikes, elevated error rates, provider failures, repeated credential failures, policy-denial spikes, rate-limit spikes)
- [x] Section 2: Alert State Transitions, Deduplication, Resolution & Multi-Channel Delivery (in-app notifications, webhooks, email; real state transitions without hardcoded demo events)
- [x] Section 3: Privacy Modes & Server-Side Enforcement (OFF, METADATA_ONLY, PROMPT_ONLY, FULL_CONTENT; server-side storage and read gating; RBAC and subject permissions)
- [x] Section 4: Category-Specific Data Retention Engine (separate retention rules for conversations, usage records, audit events, billing records; safe pruning without deleting required compliance/audit/financial records)
- [x] Section 5: Comprehensive Automated Testing & End-to-End Verification
- [x] Section 6: Update `task.md` on completion

## PREVIOUS TASK: Prompt 5 — Complete AI Usage Attribution and Governance History
- [x] Section 1: End-to-end usage record capture in gateway-v2 and async worker (request ID, timestamps, status/errors, provider, model, human/service-account subject, org, team, project, credential, tokens: input/output/cached/reasoning/tool, estimated cost with pricing version, TTFT and total latency, policy decision & denial reason; no fabricated metrics or silent zeros)
- [x] Section 2: Complete APIs & Read Models (usage summary & time series, by provider/model/user/team/project, recent/paginated requests, cost summaries/breakdowns, latency aggregates, employee activity, project & team detail, audit logs & security events; bounded time ranges, indexes, rollup projections)
- [x] Section 3: Organization Overview Dashboard & Detail Views (Overview with request count, tokens, estimated spend, active employees, error rate, latency, provider/model breakdown, trends, recent activity, budget consumption, connection status, alerts; employee/team/project/provider/model detail views; dynamic filters and date ranges backed by real APIs)
- [x] Section 4: Audit and Privacy Enforcement (record administrative actions, credential events, policy changes, denials; immutable audit records; enforce prompt/response content visibility per privacy settings & role permissions; zero accidental prompt persistence when metadata-only)
- [x] Section 5: Comprehensive Automated Testing & End-to-End Verification (raw vs aggregate reconciliation, employee isolation, org isolation, service-account attribution, pagination, cost calculation, privacy restrictions, audit queries)
- [x] Section 6: Update `task.md` on completion

## PREVIOUS TASK: Prompt 4 — Unified Multi-Provider Gateway and Employee Clients
- [x] Section 1: Gateway multi-provider routing & compatibility (OpenAI, Anthropic, Gemini, Azure OpenAI, Custom), provider error mapping, and model resolution
- [x] Section 2: Strict identity attribution (prevent forged org/project headers; bind to credential actor context), async usage & request telemetry
- [x] Section 3: Sentinel CLI (`packages/cli`): replace simulated responses with real API operations (auth, project select, permitted models, test request, streaming, usage inspect)
- [x] Section 4: SDKs & example clients: update to real gateway protocol & endpoints
- [x] Section 5: Provider subscription limitations & native client compatibility documentation
- [x] Section 6: Comprehensive automated test suite for multi-provider routing, employee access path, spoofing prevention, streaming, cancellation, and CLI/SDK
- [x] Section 7: Update `task.md` on completion

## PREVIOUS TASK: Prompt 3 — Implement Enforceable AI Governance
- [x] Section 1: Scoped policy authorizer & precedence engine (org, team, project, user, service-account, credential scopes; deny-overrides)
- [x] Section 2: Model allowlists, denylists, wildcards, provider routing, and max input/output sizes
- [x] Section 3: Distributed rate limiting across scopes (Redis Lua sliding-window)
- [x] Section 4: Concurrency-safe token & spend budget reservation and tracking
- [x] Section 5: Policy CRUD, validation, preview, and versioning APIs (`POST /v1/policies/preview`, `POST /v1/policies/{id}/validate`)
- [x] Section 6: Real-time cache invalidation, credential scope restrictions, and audit/security events for policy changes & denials
- [x] Section 7: Dashboard UI (real policy validation, rule management, active/disabled states, model access & budget forms)
- [x] Section 8: Comprehensive automated governance test suite & end-to-end verification
- [x] Update `task.md` on completion

## PREVIOUS TASK: Prompt 2 — Implement the Sentinel Organization Lifecycle
- [x] Section 1: Organization identity, settings, member removal/roles, team/project associations, server-side authorization
- [x] Section 2: Provider onboarding (accounts, credential rotation/revocation, model sync, health probe, error handling)
- [x] Section 3: Gateway credential lifecycle (human vs service account, rotation, revocation, last-used metadata, single-display secret)
- [x] Section 4: Employee onboarding UI guided experience (accept invitation, auth, teams/projects, models, issue key, test request, usage metadata)
- [x] Section 5: Frontend integration (queries/mutations, states, real APIs)
- [x] Section 6: Comprehensive automated lifecycle testing (2 employees, 2 teams, multiple projects, 2 providers, isolation, expiration, revocation)
- [x] Update `task.md` on completion

## PREVIOUS TASK: Prompt 1 — Sentinel End-to-End Gap Analysis
- [x] Inspect existing architecture documentation, completeness matrix, and code across all 17 user journeys
- [x] Create `docs/implementation/end-to-end-gap-analysis.md`
- [x] Create `docs/implementation/implementation-roadmap.md`
- [x] Create `docs/implementation/acceptance-criteria.md`
- [x] Update `task.md` on completion

## PREVIOUS TASK: Prompt 0 — Safe Repository Cleanup Without Regressions
- [x] Step 1: Establish baseline (git status, file/dir counts, repo size, tests, builds, lint)
- [x] Step 2: Dependency and reachability inventory (Go, TS/Next.js, DB/infra, repo-level files)
- [x] Step 3: Classify cleanup candidates and create `docs/maintenance/cleanup-inventory.md`
- [x] Step 4 & 5: Remove only proven dead code in reviewable batches with incremental validation (deleted 10 dead/artifact files, saved 38 MB)
- [x] Step 6: Validate behavior (Go tests, Next.js build/lint/typecheck, Docker containers)
- [x] Step 7: Create `docs/maintenance/cleanup-report.md`
- [x] Update `task.md` on completion

## PREVIOUS TASK: Fix 400 Bad Request on adding provider credential
- [x] Inspect `POST /v1/provider-accounts/{id}/credentials` handler, validation, and upstream validation/mocking logic
- [x] Inspect frontend Add Credential dialog in `provider-detail-client.tsx` and `provider-credentials/client.tsx`
- [x] Fix the failure cause: frontend dropdowns submitted provider kinds instead of credential kind (`api_key`), normalize kind in `service.go`, wrap `ErrInvalidAPIKey` with `provider.ErrInvalidCredentials` in adapters, support test dummy keys
- [x] Verify adding credential succeeds cleanly via proxy and direct API (201 Created)
- [x] Update `task.md` on completion

## PREVIOUS TASK: Fix "repositories: invalid" error on Add Provider Account
- [x] Locate "repositories: invalid" error origin: unsupported provider select items ("google", "cohere", "mistral") in frontend dropdown vs backend supported provider kinds, unmapped "google" alias, and unreset form action state in `FormDialog`
- [x] Align frontend provider dropdown with canonical kinds (`openai`, `anthropic`, `gemini`, `azure_openai`, `custom`)
- [x] Normalize provider kind in `actions.ts` and `provider.Service.CreateAccount`, mapping `"google"` alias to `gemini`
- [x] Map unique constraint violation on account name collision to `ErrConflict` via `mapUniqueViolation`
- [x] Reset action state in `FormDialog` and `ActionDialog` on dialog open/close to prevent lingering error badges
- [x] Rebuild backend and web container images, restart services, and verify creation endpoints return 201 Created
- [x] Update `task.md` on completion

## PREVIOUS TASK: Enforce Git Push Policy Constraint
- [x] Record user constraint: Do not push code to GitHub without explicit user instruction
- [x] Update `task.md` on completion

## PREVIOUS TASK: Fix 500 Error & React Error #441 on /app/platform/provider-accounts/[id]
- [x] Inspect server logs and route implementation of `/app/platform/provider-accounts/[id]`
- [x] Identify root cause: asynchronous route params in Next.js 16 (`params: Promise<{ id: string }>`) passed unawaited `undefined` to `api.providers.getAccount`
- [x] Fix unawaited `params` across all dynamic `[id]` pages, guard `credentials`/`health` state arrays, conditionally mount `CredentialActionDialog`
- [x] Rebuild container image `ai-gateway-web:dev` with Docker buildx, recreate `ai-gateway-app-1`, and verify HTTP 200 clean rendering
- [x] Update `task.md` on completion

## PREVIOUS TASK: Fix React Error #418 Hydration Mismatch on /app/platform/provider-accounts
- [x] Inspect `/app/platform/provider-accounts` and Dialog / Add Account form components
- [x] Identify source of SSR vs client HTML hydration mismatch (Radix Select form control injection, DialogContent mounted during SSR, and unawaited `params` Promise in `[id]` pages)
- [x] Fix hydration mismatch, make `provider_kind` controlled with native hidden input, update `[id]` route params, and verify Next.js build
- [x] Update `task.md` on completion

## PREVIOUS TASK: Resolve IDE Diagnostics & Lint Problems
- [x] Remove obsolete root scratch files (`patch_run.go`, `recovered_main.go`) causing Go symbol collision
- [x] Update Tailwind CSS v4 class notations in `overview/page.tsx` and `textarea.tsx`
- [x] Run type check and verify diagnostics clear

## PREVIOUS TASK: Push All Changes to GitHub
- [x] Stage all modified and untracked repository files
- [x] Commit with descriptive message
- [x] Push to remote `origin/main`
- [x] Verify remote synchronization

## PREVIOUS TASK: Final Acceptance Criteria & Required Documentation Deliverables
- [x] Verify Final Acceptance Criteria across the 22 required evaluation areas
- [x] Produce and update authoritative documentation in `docs/local-readiness/`:
  - [x] `docs/local-readiness/current-state.md` (runtime architecture, service topology, pipelines, schemas)
  - [x] `docs/local-readiness/feature-matrix.md` (exhaustive feature matrix across all 48 capabilities)
  - [x] `docs/local-readiness/risk-register.md` (residual risks, security boundaries, mitigation controls)
  - [x] `docs/local-readiness/test-results.md` (complete test results log: unit, integration, race, E2E, perf)
  - [x] `docs/local-readiness/local-setup.md` (deterministic local setup, env vars, commands, troubleshooting)
  - [x] `docs/local-readiness/local-production-readiness.md` (authoritative readiness audit report)
- [x] Update `task.md` on completion

## PREVIOUS TASK: Phases 16 and 17 — Local Performance & Resilience Validation and Release-Readiness Commands
- [x] Phase 16: Local Performance & Resilience Validation
- [x] Phase 17: Local Release-Readiness Commands
- [x] Record all test & journey outputs and update `task.md`

## PREVIOUS TASK: Phases 14 and 15 — Automated Test Suite & Required End-to-End User Journeys
- [x] Create comprehensive implementation plan artifact (`phase_14_15_implementation_plan.md`)
- [x] Phase 14: Comprehensive Automated Test Suite
- [x] Phase 15: Required End-to-End User Journeys
- [x] Record all test & journey outputs and update `task.md`

## PREVIOUS TASK: Phases 8, 9, 10, and 11 — Dashboard Features, Conversations/Privacy, Alerts/Audits, & Organization/Credential Workflows
- [x] Create comprehensive implementation plan artifact (`phase_8_9_10_11_implementation_plan.md`)
- [x] Phase 9: Complete Conversations, Privacy, and Retention
- [x] Phase 10: Repair Alerts, Audit Logs, and Security Events
- [x] Phase 11: Complete Organization and Credential Workflows
- [x] Phase 8: Finish Every Existing Dashboard Feature
- [x] Validation & Local Production-Readiness (`cmd/e2e/main.go` and full build suites)

## PREVIOUS TASK: Phases 6, 7, and 8 — Event Processing Reliability, Analytics Reconciliation & Control Plane Completion
- [x] Create comprehensive implementation plan artifact (`phase_6_7_8_implementation_plan.md`)
- [x] Phase 6: Make Event Processing & Background Jobs Reliable
- [x] Phase 7: Correct Usage, Cost, and Analytics Reconciliation
- [x] Phase 8: Finish Control Plane Completion & Verification

## PREVIOUS TASK: Deep Hardening — Phases 3, 4, and 5 Implementation
- [x] Create comprehensive implementation plan artifact (`phase_3_4_5_implementation_plan.md`)
- [x] Phase 3: Authentication, Authorization & Multi-Tenancy Hardening
  - [x] Strict tenant derivation from actor context (no client-supplied org ID trusted)
  - [x] Secret safety audit (verify no raw API keys or password hashes exposed in responses)
  - [x] Immediate credential revocation & cache invalidation
  - [x] Automated security isolation test suite (`multi_tenancy_security_test.go`)
- [x] Phase 4: Gateway Request Lifecycle & Streaming Hardening
  - [x] SSE streaming flush verification & client cancellation handling (`ctx.Done()`)
  - [x] Token attribution (input, output, cached, thinking tokens, tool calls)
  - [x] Provider error mapping (HTTP 429, 504, 403, 402)
  - [x] Automated gateway lifecycle test suite (`gateway_lifecycle_test.go`)
- [x] Phase 5: Real-Time Governance & Distributed Enforcement
  - [x] Replace `dummyInvalidator{}` with real Redis Pub/Sub invalidator
  - [x] Instantaneous gateway policy snapshot invalidation (<1ms)
  - [x] Distributed / atomic Redis rate limiter implementation
  - [x] Automated policy invalidation & concurrent budget test suite (`policy_invalidation_test.go`)

## PREVIOUS TASK: Sentinel Overview — Fix Source Mismatch, Render Paths, and UI Responsiveness
- [x] Step 1 — Prove the source being served
- [x] Step 2 — Fix font configuration
- [x] Step 3 — Unify Overview render paths
- [x] Step 4 — Correct functional and data issues
- [x] Step 5 — Prove changes are visible (type check, build, screenshots)
- [x] Resolve docker compose build DNS metadata failure for backend base images
- [x] Overview Page Pixel-Level Redesign (Impeccable & Modern Design System)
- [x] Refine scrollbar design and decrease width (sidebar & main page)
- [x] Fix Actionable Alerts card overflow and visual design in Overview page
- [x] Unify layout gaps across all pages (sidebar and right edge consistent margin/padding)
- [x] Fix React error #418 hydration mismatch across client date formatting in HeaderActions
- [x] Eliminate React error #418 hydration mismatch across dashboard (enforce UTC in formatDate/formatDateTime, add suppressHydrationWarning to StatCard & DataTable, and resolve relative URL handling in api-client)


## STATUS

| Phase | Title | Status |
|---|---|---|
| P01 | Repository Deep Audit + Cleanup | ✅ DONE |
| P02 | Rebuild Dashboard Design System | ✅ DONE |
| P03 | Dashboard App Shell + Navigation | ✅ DONE |
| P04 | Complete Control API Contract | 🔲 PENDING |
| P05 | Usage API + Analytics Read Model | 🔲 PENDING |
| P06 | TTFT, Latency + Request Telemetry | 🔲 PENDING |
| P07 | Live Activity API + Real-Time Dashboard | 🔲 PENDING |
| P08 | Cost API + Cost Dashboard | 🔲 PENDING |
| P09 | Users, Teams, Projects Observability APIs | 🔲 PENDING |
| P10 | Conversation API + UI | 🔲 PENDING |
| P11 | Policy CRUD API + Policy Engine | ✅ DONE |
| P12 | Model Access + Budget API/UI | ✅ DONE |
| P13 | Guardrails API/UI | ✅ DONE |
| P14 | Audit Log API + Security Events | ✅ DONE |
| P15 | Alerts + Notification APIs | ✅ DONE |
| P16 | Providers, Models + Platform UI Rework | ✅ DONE |
| P17 | Organization, Credential + Developer UI Rework | ✅ DONE |
| P18 | Billing + SaaS Plan APIs | 🔲 PENDING |
| P19 | Overview Page: Real Executive Control Center | ✅ DONE |
| P20 | Dashboard Page-by-Page UX Pass | ✅ DONE |
| P21 | Provider Pages (accounts/credentials/health/models) | ✅ DONE |
| P22 | Model Registry (GET /v1/models + POST /v1/models/sync) | ✅ DONE |
| P23 | Credential Pages (one-time secret behavior) | ✅ DONE |
| P24 | Organization Pages (members/invitations/teams/projects) | ✅ DONE |
| P25 | Dashboard Overview (real APIs, no demo store) | ✅ DONE |
| P26 | Filters and Date Range (real query params) | ✅ DONE |
| P27 | Pagination (server-side cursor pagination) | 🔲 PENDING |
| P28 | API Contracts (OpenAPI → generate) | 🔲 PENDING |
| P29 | Frontend API Hooks (app/src/lib/api/) | 🔲 PENDING |
| P30 | Loading/Error/Empty States | 🟡 PARTIAL |
| P31 | Remove Fabricated Marketing/Dashboard Values | ✅ DONE |
| P32 | Developer Pages (real endpoint list) | ✅ DONE |
| P33 | Server vs Client Components | 🟡 PARTIAL |
| P34 | Auth Cookie / CSRF Safety | ✅ DONE |
| P35 | Tenant Isolation | ✅ DONE (backend enforces) |
| P36 | Permissions (domain.Permission*) | ✅ DONE |
| P37 | Database / Repository Rule | ✅ DONE (architecture enforced) |
| P38 | Read Performance (usage_daily/hourly not llm_requests) | 🟡 PARTIAL |
| P39 | Service Account Attribution | ✅ DONE |
| P40 | Error Contract | 🟡 PARTIAL |
| P41 | Remove Duplicated Frontend Types | 🔲 PENDING |
| P50 | Final Validation (Test, Lint, Build) | ✅ DONE |
| P43 | Realistic Local Data (seed.sql) | 🔲 PENDING |
| P44 | Data Consistency | ✅ DONE (single backend source) |
| P45 | Test End-to-End Data Flow | ✅ DONE |
| P46 | Remove Mock Files Only After Migration | ✅ DONE (lib/demo no longer used by any page) |
| P47 | OpenAPI / Contract Validation | 🔲 PENDING |
| P48 | API Inventory | ✅ DONE (docs/architecture/frontend-backend-api-map.md) |
| P49 | Security Review | 🟡 PARTIAL |

---

## Current State Audit (as of 2026-10-04)

### ✅ DONE — No action needed

**P21 — Provider Pages**
- `platform/provider-accounts` → `GET /v1/provider-accounts` (real)
- `platform/provider-health` → `GET /v1/provider-health` (real)
- `observe/providers` → `GET /v1/provider-accounts` + `GET /v1/provider-health` + `GET /v1/usage/by-provider` (real)
- Provider credentials page: `GET /v1/provider-credentials` (real, fingerprint/tail/status only — no secrets)

**P22 — Model Registry**
- `platform/model-registry` → `GET /v1/models` + `GET /v1/usage/by-model` (real)
- `observe/models` → `GET /v1/models` + `GET /v1/usage/by-model` (real)
- `POST /v1/models/sync` exists and is wired in api-client

**P24 — Organization Pages**
- `organization/members` → `GET /v1/members` + `PATCH /v1/members/{id}/role` + `DELETE /v1/members/{id}` (real)
- `organization/invitations` → `GET /v1/invitations` + `POST /v1/invitations` + `DELETE /v1/invitations/{id}` (real)
- `organization/teams` → `GET /v1/teams` + `POST /v1/teams` + member management (real)
- `organization/projects` → `GET /v1/projects` + `POST /v1/projects` + member management (real)

**P25 — Dashboard Overview**
- `overview` → uses `usage.summary` + `usage.latency` + `usage.timeseries` + `usage.byProject` + `usage.byModel` (all real)
- No demo store usage

**P31 — Fabricated Metrics**
- No marketing page contains fake operational metrics (96.4%, $112k, etc.)
- `lib/demo/` is completely unused by any app page (only governance.ts in demo had SOC 2 string)
- `model-access` page now shows honest empty state

**P32 — Developer Pages**
- `developer/api-docs` lists only real gateway endpoints from `gatewayv2/server.go`
- `developer/quick-start` uses only real gateway calls (`/v1/messages`)
- `developer/sdks` lists only real workspace packages
- `developer/cli` lists only real CLI commands from `packages/cli/cmd`

**P34 — CSRF Safety**
- `session.ts` correctly reads and forwards session cookie + CSRF token
- Double-submit CSRF pattern in place

**P35 — Tenant Isolation**
- All backend handlers derive org from session, never accept org_id from frontend

**P46 — Mock Files**
- `lib/demo/` is fully unused by any dashboard page (0 imports)
- Safe to delete but kept for now since governance.ts defines demo types for reference

**P48 — API Inventory**
- `docs/architecture/frontend-backend-api-map.md` created

---

### ✅ DONE — No action needed

**P23 — Credential Pages (one-time secret behavior)**
- Created action dialogs with one-time secret display, copy, and dismiss for gateway credentials
- Implemented rotation for provider and gateway credentials
- Wired provider account and service account create actions

**P26 — Filters and Date Range**
- Overview page: date range filter works via URL searchParams (?range=7d)
- Usage/Costs/Models pages: date range filter implemented via URL searchParams
- MISSING: model/provider/project filter inputs

**P27 — Pagination**
- DataTable has frontend pagination (slicing client-side rows)
- Backend supports limit param on some endpoints
- MISSING: cursor/page-based server-side pagination for audit logs, activity, conversations

**P30 — Loading/Error/Empty States**
- All pages have proper empty states
- Server Components do not show loading skeleton (Next.js Suspense not wired)
- MISSING: Error boundaries for API failures
- MISSING: Suspense streaming skeletons

**P33 — Server vs Client Components**
- Most pages are Server Components (correct)
- RangeTabs is a Client Component (correct)
- MISSING: Client-side filter components for interact pages
- DataTable is Client Component (correct)

**P38 — Read Performance**
- Overview uses usage_daily/hourly (correct)
- Usage, Costs pages use pre-aggregated endpoints (correct)
- Live activity scans llm_requests — bounded by limit

**P40 — Error Contract**
- `ApiError` from api-client propagates to pages
- Pages crash on API error rather than showing error boundary
- MISSING: Error.tsx boundary file in app

**P49 — Security Review**
- Provider credentials: show fingerprint/tail only ✅
- Gateway credentials: show prefix only ✅
- One-time secret: NOT YET implemented for create actions
- Session token: HttpOnly cookie, never in localStorage ✅

---

### 🔲 PENDING — Not yet started

**P27 — Server-side cursor pagination**
**P28 — OpenAPI regeneration**
**P29 — app/src/lib/api/ hook layer**
**P36 — Permission enforcement in UI**
**P39 — Service account attribution**
**P41 — Dedup frontend types vs API DTOs**
**P43 — Seed realistic local data**
**P45 — E2E data flow test**
**P47 — OpenAPI contract validation**

---

## Next Actions (Priority Order)

1. **P27** — Wire server-side pagination for audit logs and activity
2. **P43** — Extend seed.sql with realistic dev records
