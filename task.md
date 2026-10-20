 -# Implementation Task Tracker

Updated after each completed prompt.

## COMPLETED TASK: Push Pricing Management Implementation to GitHub
- [x] Stage all modified and untracked pricing and billing changes
- [x] Commit staged changes with descriptive message
- [x] Push commits to remote origin main
- [x] Verify repository synchronization

## COMPLETED TASK: Phase 3 — Implement Pricing Management
- [x] 3.1 Inspect pricing domain models, schema, repositories, cost calculator, and control API routes
- [x] 3.2 Implement complete pricing-management routes (List, Create, Update/Version, GetVersion, CalculateCost) in Control API
- [x] 3.3 Validate provider/model support, currency, token categories (cached tokens only where supported), and price validation
- [x] 3.4 Wire authorization, audit logging, and dashboard client methods
- [x] 3.5 Ensure cost calculation precision, unknown pricing handling, and historical version reproducibility
- [x] 3.6 Implement comprehensive pricing regression tests (create/update/get, invalid prices, unsupported tokens, version history, cost calc, auth)

## COMPLETED TASK: Phase 2 — Fix Billing Correctness
- [x] 2.1 Remove hardcoded payment amounts ($49/$499) and validate server-side plan/checkout records
- [x] 2.2 Stop ignoring persistence failures in subscription, invoice, billing events, and audit writes
- [x] 2.3 Reconcile invoice status and migrations, database constraints, repository queries, and domain states
- [x] 2.4 Make billing state transitions reliable (idempotency, out-of-order handling, mode enforcement)
- [x] 2.5 Add comprehensive billing regression tests covering all error, edge case, and mode scenarios

## COMPLETED TASK: Phase 1 — Audit, Baseline & Matrix Update
- [x] Inspect billing service, application layer, interfaces, infrastructure, domain, bootstrap
- [x] Inspect migrations (billing, gateway-credentials), Next.js dashboard, CLI, SDK, Redis, NATS, workers
- [x] Update `docs/architecture/completeness-matrix.md`
- [x] Update `docs/local-readiness/current-baseline.md`
- [x] Update `docs/local-readiness/final-completion-plan.md`
- [x] Update `docs/local-readiness/feature-verification-matrix.md`
- [x] Verify test execution evidence across all packages and layers

## COMPLETED TASK: Full Verification, Defect Fixing, and Final Completion Reports (Phases 11-13)
- [x] Run full repository verification suite and underlying checks (formatting, boundaries, vet, unit, integration, race, contracts, frontend, build)
- [x] Verify Workflows A through H (Org/Provider setup, inference, governance, tenant isolation, usage reconciliation, recovery, billing, regression)
- [x] Fix any discovered defects with minimal safe changes and regression tests
- [x] Update `docs/local-readiness/current-baseline.md`
- [x] Update `docs/local-readiness/final-completion-plan.md`
- [x] Update `docs/local-readiness/final-test-report.md`
- [x] Update `docs/local-readiness/remaining-gaps.md`
- [x] Commit and push final reports and changes

## COMPLETED TASK: Phases 5-10 Full Verification (CLI/SDK, Auth/RBAC, Governance, Telemetry, Billing, Frontend)
- [x] Section 5: CLI and TypeScript SDK entire request lifecycle, SSRF defense, streaming & options
- [x] Section 6: Authentication, RBAC, tenant isolation, cross-org denial, server-side enforcement
- [x] Section 7: Governance effectiveness, budgets, rate limits, guardrails, Redis outage policy
- [x] Section 8: Usage accounting, pricing resolution, idempotent event processing, worker rollups
- [x] Section 9: Dual billing verification (Stripe & Razorpay), HMAC signature checks, simulation vs test mode
- [x] Section 10: Frontend dashboard integration, route compilation, live API binding, permission-denied states
- [x] Update documentation and task tracking

## COMPLETED TASK: Complete Local Startup & Provider Integration
- [x] Verify local startup reproducibility, dependency & lockfile consistency, env validation, migration ordering
- [x] Verify Redis, NATS, PostgreSQL connectivity, lifecycle commands, and startup failure diagnostics
- [x] Inspect and test Gemini, OpenAI, Anthropic, Hugging Face provider adapters
- [x] Verify model discovery, streaming/non-streaming inference, token extraction, error mapping, and correlation IDs
- [x] Ensure live smoke tests are safely env-gated and zero secrets are hardcoded
- [x] Update readiness documentation and task tracker

## COMPLETED TASK: Establish Reproducible Baseline & Update Readiness Documents
- [x] Inspect README.md, Makefile, scripts, go.mod, package manifests, Docker Compose, migrations, tests
- [x] Record exact toolchain versions, command execution logs, and environment limitations
- [x] Update `docs/local-readiness/current-baseline.md` with verified execution evidence
- [x] Update `docs/local-readiness/final-completion-plan.md` with explicit validation matrix
- [x] Verify clean git working tree and production readiness

## COMPLETED TASK: Architectural Integrity and Defect Audit
- [x] Verify architectural boundary preservation (`scripts/check-boundaries.sh`)
- [x] Verify Go backend, CLI, SDK, and contracts stability
- [x] Confirm no fabricated states or mocks replace real implementations in production paths
- [x] Confirm clean git working tree and production readiness

## COMPLETED TASK: Full Application Local Readiness, Acceptance Evidence, and Completion Reports
- [x] Step 1: Establish baseline and generate `docs/local-readiness/current-baseline.md`
- [x] Step 2: Formulate roadmap and create `docs/local-readiness/final-completion-plan.md`
- [x] Step 3: Run full verification suite across all layers (Backend, Race, CLI, SDK, Contracts, Frontend, E2E Workflows A-H)
- [x] Step 4: Update `docs/local-readiness/final-test-report.md` with honest pass/fail counts, tool versions, and blocker documentation
- [x] Step 5: Update `docs/local-readiness/remaining-gaps.md` with exact external prerequisites and final readiness rating
- [x] Step 6: Verify clean git status, commit, push to remote, and mark tasks complete

## COMPLETED TASK: Push to Git Repository
- [x] Step 1: Verify clean working directory and all local changes committed
- [x] Step 2: Push commits to remote origin main (`b1a5141..252e2c4`)
- [x] Step 3: Verify remote is synchronized and up-to-date

## COMPLETED TASK: Phases 10 and 11 — Complete Local Acceptance Testing & Final Documentation
- [x] Phase 10: Run complete local acceptance tests
  - [x] A. Static & unit checks: `go vet`, formatting, TypeScript typecheck, ESLint, Go unit tests, NPM test suite
  - [x] B. Integration checks: Postgres repositories, migrations, Redis operations, NATS/eventbus, worker, adapters, auth, billing
  - [x] C. End-to-end scenarios (1 through 8):
    - [x] Scenario 1 — New organization: Admin signup, org setup, provider connection, credential validation, model discovery
    - [x] Scenario 2 — Employee inference: Member invite, role assignment, gateway credential, CLI/SDK prompt, usage record
    - [x] Scenario 3 — Governance enforcement: Model restrictions, budget exhaustion, policy rejection, server-side block
    - [x] Scenario 4 — Tenant isolation: Multi-org isolation across API, CLI, SDK, and DB
    - [x] Scenario 5 — Usage reconciliation: Known token counts, event processing, cost calculation, dashboard reconciliation
    - [x] Scenario 6 — Recovery: Redis outage (fail-closed vs degraded), worker restarts, recovery without duplicate usage
    - [x] Scenario 7 — Billing: Disabled/simulation modes, HMAC signature checks, idempotent webhooks, subscription changes
    - [x] Scenario 8 — Regression: Complete test suite execution across backend, CLI, SDK, contracts, frontend
- [x] Phase 11: Documentation and final report
  - [x] Update `docs/local-readiness/local-setup.md` with verified commands and prerequisites
  - [x] Update `docs/local-readiness/provider-verification.md` with model discovery & capability matrix
  - [x] Update `docs/local-readiness/security-verification.md` with tenant isolation & SSRF defense
  - [x] Update `docs/local-readiness/billing-verification.md` with dual payment modes & signature checks
  - [x] Update `docs/local-readiness/final-test-report.md` with pass/fail counts, blockers, and final readiness status
  - [x] Update `docs/local-readiness/remaining-gaps.md` with honest remaining non-critical limitations

## COMPLETED TASK: Phases 5, 6, 7, 8, and 9 — Governance, Telemetry/Pricing, Billing, Frontend/API, Security/Privacy
- [x] Phase 5: Make governance effective
  - [x] Org, team, project, and user access control + model access restrictions
  - [x] Policy creation, validation, evaluation, guardrails, and policy rejection responses
  - [x] User, team, project, and organization budget limits + rate limiting and concurrency control
  - [x] Audit events for privileged actions + policy cache invalidation after updates
  - [x] Server-side enforcement decisions + fail-closed Redis outage verification
- [x] Phase 6: Make usage, pricing and event processing reliable
  - [x] Consistent Request IDs and correlation IDs across gateway, bus, worker, and database
  - [x] Durable usage events with idempotent processing (no duplicate counting or dropped records)
  - [x] Reliable worker restart recovery, retries, and dead-letter handling
  - [x] Exact provider-reported token counts and accurate multi-tier pricing resolution
  - [x] Explicit unknown pricing handling + concurrent budget reservations and releases
  - [x] Accurate service-account attribution and reconciliation against dashboard aggregate records
- [x] Phase 7: Complete billing safely
  - [x] Explicit billing modes: disabled, simulation, and real provider test mode
  - [x] Configuration validation, startup errors, and safe local development defaults
  - [x] Real checkout session creation with valid test credentials + webhook signature verification
  - [x] Idempotent webhook handling, out-of-order event resilience, and subscription lifecycle
  - [x] Entitlement synchronization, invoice reconciliation, and simulation non-entitlement safety
- [x] Phase 8: Complete frontend and API integration
  - [x] Connect all dashboard routes to live backend APIs (auth, orgs, credentials, models, policies, guardrails, budgets, billing)
  - [x] Verify loading, empty, partial-failure, permission-denied, and backend-unavailable states
  - [x] Eliminate fake/mock production UI data and fix broken links/labels
- [x] Phase 9: Security and privacy verification
  - [x] Tenant-scoped authorization at every backend entry point + cross-tenant leak prevention
  - [x] Secure session handling, CSRF defense, credential encryption, and secret redaction
  - [x] Server-side SSRF validation for outbound provider endpoints + safe CORS/cookies
  - [x] Administrative audit logging + retention, deletion, and privacy controls

## COMPLETED TASK: Phases 2, 3, and 4 — Local Reproducibility, Provider Integration, CLI/SDK E2E
- [x] Phase 2: Make local development reproducible
  - [x] Verify toolchain versions (Go, Node.js, package manager) against repository configs
  - [x] Verify Compose services, ports, dependency ordering, health checks, volumes, environment variables
  - [x] Verify database migrations run in correct order and fail visibly on error
  - [x] Verify PostgreSQL, Redis, NATS and app services connectivity and usage
  - [x] Ensure application startup fails clearly when required dependencies are unavailable
  - [x] Provide safe, explicit local-development defaults without insecure defaults in other modes
  - [x] Ensure logs contain actionable diagnostics without credentials, tokens, API keys, or prompt content
  - [x] Add/repair repository local setup, startup, shutdown, migration, and health-check commands
- [x] Phase 3: Complete provider integration
  - [x] Verify Gemini, OpenAI, Anthropic, Hugging Face adapters for credential validation and secure storage
  - [x] Verify provider connection status and actionable error reporting
  - [x] Verify credential-specific model discovery (no unverified fallbacks) and capability metadata
  - [x] Verify model selection, provider routing, streaming (SSE) and non-streaming inference
  - [x] Verify provider-specific parameter preservation, usage/token extraction, timeouts/retries
  - [x] Verify upstream auth errors, unavailable models, malformed responses, usage event persistence
  - [x] Verify mock-based deterministic tests and real-provider env-gated smoke tests
- [x] Phase 4: Finish CLI and SDK end-to-end behavior
  - [x] Inspect and trace every advertised CLI command and SDK feature
  - [x] Verify login, logout, credential handling, and revocation
  - [x] Verify model selection, listing, provider-specific settings, streaming/non-streaming
  - [x] Verify usage and cost commands, conversation clearing and compaction
  - [x] Enforce SSRF protection and server-side allowlists on custom endpoints
  - [x] Ensure consistency between interactive REPL, single-request mode, and SDK
  - [x] Run and add regression tests for all repaired commands and options

## COMPLETED TASK: Phase 1 — Establish Actual Repository State & Deliverables
- [x] Step 1: Deep inspection of all repository subsystems (Control API, Gateway-v2, Adapters, Frontend, CLI, SDK, Postgres schemas/migrations, Redis limiters/budgets, NATS worker, Auth/RBAC, Policies/Guardrails, Usage/Billing, Docker Compose/scripts, test suites).
- [x] Step 2: Trace end-to-end user workflows across frontend -> API -> database/services -> gateway -> provider -> event processing -> dashboard.
- [x] Step 3: Create/update `docs/local-readiness/repository-audit.md` with honest findings, source files, actual behavior, and defects.
- [x] Step 4: Create/update `docs/local-readiness/implementation-plan.md` detailing the execution roadmap to achieve complete local readiness.
- [x] Step 5: Create/update `docs/local-readiness/feature-verification-matrix.md` with source files, persistence, authorization, tests, known defects, and completion criteria for every feature.
- [x] Step 6: Update `task.md`.

## COMPLETED TASK: Push Verified Multi-Provider, Billing, and Test Changes to Remote
- [x] Step 1: Stage all verified changes and untracked files (`git add -A`).
- [x] Step 2: Commit staged changes with descriptive message.
- [x] Step 3: Push commits to remote origin main (`git push origin main`).
- [x] Step 4: Verify clean working directory and remote synchronization.
- [x] Step 5: Mark task complete in `task.md`.

## COMPLETED TASK: Phase H — Run Tests and Produce Evidence
- [x] Step 1: Discover repository commands, environment prerequisites, Go version, Node dependencies, and Docker availability.
- [x] Step 2: Run Go formatting, vetting, race detection, unit and integration test suites (`backend`, `packages/cli`, `packages/sdk`).
- [x] Step 3: Run Frontend type checking, linting, and build validation (`app`).
- [x] Step 4: Run database migration validation, provider-adapter contract tests, model discovery tests, slash-command execution tests, payment adapter tests, and tenant isolation tests.
- [x] Step 5: Create/update required deliverables in `docs/local-readiness/`: `critical-defects.md`, `provider-discovery-verification.md`, `cli-command-verification.md`, `billing-verification.md`, `local-configuration-verification.md`, `final-test-report.md`, and `remaining-gaps.md`.
- [x] Step 6: Assign honest final status and update `task.md`.

## COMPLETED TASK: Phase G — End-to-End Test Scenario
- [x] Step 1: Create reproducible local test fixtures with 1 organization, 3 independent users, separate Sentinel credentials, different provider/model access restrictions, explicit test budgets, rate limits, and deterministic mock providers.
- [x] Step 2: Verify user authentication, Sentinel CLI session startup, dynamic `/` command menu inspection, and execution of common vs provider-specific slash commands for each user.
- [x] Step 3: Verify CLI inference requests, provider/model routing, unauthorized model rejection, token usage, cost calculation, status, and latency metrics.
- [x] Step 4: Verify corresponding dashboard records and assert strict tenant/user isolation (prevent cross-user reading of conversations, credentials, usage, or billing records).
- [x] Step 5: Verify immediate credential revocation enforcement and separation of hermetic mock-provider fixtures from live provider tests.
- [x] Step 6: Execute end-to-end verification suites, verify zero regressions, and update `task.md`.

## COMPLETED TASK: Phase F — Verify the Full Multi-Provider Gateway
- [x] Step 1: Audit multi-provider adapters (Gemini, OpenAI, Anthropic, Hugging Face), adapter contracts (`ports/provider`), gateway routing (`provider_router.go`), and control plane discovery.
- [x] Step 2: Verify credential validation and dynamic model discovery / availability verification across all 4 providers.
- [x] Step 3: Verify provider-bound Sentinel credential enforcement, authorized model selection, and scope escape prevention.
- [x] Step 4: Verify streaming (SSE) and non-streaming request/response mapping, provider-specific options preservation, token usage extraction, error translation, and timeout/cancellation handling.
- [x] Step 5: Verify usage and cost persistence, multi-dimensional request attribution (user/org/project), and dashboard analytics aggregation.
- [x] Step 6: Execute comprehensive multi-provider test suite, document capabilities and known limitations per provider, and update `task.md`.

## COMPLETED TASK: Phase E — Correct Local Configuration and Service Wiring
- [x] Step 1: Inspect `deploy/compose/docker-compose.yml`, environment templates, startup scripts, and application bootstrap (`run.go`).
- [x] Step 2: Validate unified Redis credentials and verify gateway Redis-backed budget and rate-limit wiring with explicit fail-open/fail-closed outage policy.
- [x] Step 3: Ensure strict placeholder secret rejection and make development login, disabled CSRF, and insecure session defaults explicit opt-ins.
- [x] Step 4: Verify NATS configuration alignment with publisher/consumer, service readiness probes, log sanitization, and remove obsolete comments.
- [x] Step 5: Ensure verification scripts (`scripts/verify-local.sh`, `scripts/dev-ctl.sh`) preserve `.env` files and target correct checkout/endpoints.
- [x] Step 6: Implement automated regression tests for configuration, service wiring, and Redis outage policies, and verify full test suites.

## COMPLETED TASK: Phase D — Make Billing Modes Explicit and Safe
- [x] Step 1: Inspect existing billing configuration, payment gateways, controlapi handlers, and adapters (Stripe & Razorpay).
- [x] Step 2: Implement explicit billing modes (Disabled, Simulation, Provider Test Mode) across config, billing service, and API handlers.
- [x] Step 3: Enforce strict safety rules: reject fake checkout URLs when not in simulation mode; prevent simulated checkouts from granting real entitlements; require valid webhook signatures; verify idempotency.
- [x] Step 4: Verify live/test-mode API interactions using user-provided Stripe and Razorpay test credentials.
- [x] Step 5: Implement comprehensive regression tests covering all modes, missing/invalid credentials, webhooks, duplicate processing, failures, and cancellations.
- [x] Step 6: Verify full test suites and update task.md.

## COMPLETED TASK: Configure User-Provided Stripe and Razorpay Test Credentials
- [x] Step 1: Update `backend/.env` and `backend/.env.example` with Razorpay key ID/secret and Stripe secret/publishable keys.
- [x] Step 2: Update root `.env` and `.env.example` with the active test credentials.
- [x] Step 3: Update `deploy/compose/.env` and `deploy/compose/.env.example` with the active test credentials.
- [x] Step 4: Create `app/.env.example` and `app/.env.local` with public frontend keys (`NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`, `NEXT_PUBLIC_RAZORPAY_KEY_ID`).
- [x] Step 5: Verify backend configuration loading, payment adapters, and update task.md.

## COMPLETED TASK: Phase C — Fix Provider-Aware CLI Command Execution
- [x] Step 1: Trace command registry, slash-command handlers, gateway request DTOs, and provider adapters.
- [x] Step 2: Implement capability-gated commands (/safety for Gemini only, /endpoint with SSRF validation, /model permission filtering, /usage and /cost from gateway, /compact and /clear).
- [x] Step 3: Ensure request-path consistency between CLI interactive session, single-shot requests, gateway-v2 DTOs, and provider adapters.
- [x] Step 4: Add comprehensive regression tests for slash commands, state transitions, payload serialization, and SSRF prevention.
- [x] Step 5: Verify all CLI and backend test suites and update task.md.

## COMPLETED TASK: Provide Stripe and Razorpay Credentials in .env and .env.example
- [x] Step 1: Configure Stripe and Razorpay test credentials in backend/.env.example and backend/.env.
- [x] Step 2: Configure consolidated .env.example and .env at repository root.
- [x] Step 3: Configure Stripe and Razorpay environment variables in deploy/compose/.env.example and deploy/compose/.env.
- [x] Step 4: Verify backend configuration loading and test suites.
- [x] Step 5: Update task.md.

## COMPLETED TASK: Phase B — Fix Model Discovery and Availability
- [x] Step 1: Audit existing model discovery logic across Hugging Face, Anthropic, OpenAI, and Gemini adapters.
- [x] Step 2: Ensure model discovery never returns hardcoded fallback models as verified; query provider listing endpoints with actual credential.
- [x] Step 3: Track discovery status, last sync timestamp, and connection association in provider account & model registry.
- [x] Step 4: Ensure UI & Gateway distinguish discovered, authorized, ready, unavailable, and revoked connection states.
- [x] Step 5: Write comprehensive regression tests (successful discovery, empty lists, invalid credentials, timeouts, unsupported endpoints, revoked credentials, org isolation, stale models).
- [x] Step 6: Verify all test suites and update task.md.

## COMPLETED TASK: Phase A — Establish Current Baseline and Audit Integration Gaps
- [x] Step 1: Inspect Git status, commit SHA, and toolchain versions.
- [x] Step 2: Inspect implementation files across Hugging Face, Anthropic, Gateway V2, Bootstrap, CLI, and Payment adapters.
- [x] Step 3: Trace end-to-end flows (provider discovery, credential auth, CLI slash commands, billing checkout, webhooks, entitlements).
- [x] Step 4: Run baseline test suites (Go backend, CLI, SDK, Frontend) and record exact exit codes.
- [x] Step 5: Create or update docs/local-readiness/critical-defects.md with confirmed findings, root causes, and verification status.
- [x] Step 6: Update task.md with completed steps.

## COMPLETED TASK: Push Multi-Provider and Billing Implementation to GitHub
- [x] Step 1: Stage all verified files and untracked artifacts (`git add -A`).
- [x] Step 2: Commit staged changes with comprehensive commit message.
- [x] Step 3: Push commits to remote origin main (`git push origin main`).
- [x] Step 4: Verify clean working directory and remote synchronization.

## COMPLETED TASK: Phase J — Complete end-to-end local verification and final deliverables
- [x] Step 1: Create reproducible test fixture with 1 organization, 3 independent users with distinct Sentinel credentials, and mock provider connections for Gemini, OpenAI, Anthropic, and Hugging Face.
- [x] Step 2: Verify independent model discovery for all 4 providers, confirming unavailable models are marked inactive/unselectable.
- [x] Step 3: Verify Sentinel gateway credentials bound to providers, enforcing strict provider and model scoping.
- [x] Step 4: Verify Sentinel CLI slash-command registry resolution and command execution across common and provider-specific commands.
- [x] Step 5: Verify streaming and non-streaming inference requests, routing, token usage attribution, cost tracking, policies, budgets, rate limits, and revocation.
- [x] Step 6: Verify failure simulation, recovery, and dual-payment checkout (Stripe and Razorpay), webhook authenticity, idempotency, and entitlements.
- [x] Step 7: Run full Go, frontend, SDK, CLI, and integration test suites.
- [x] Step 8: Produce final evidence-based report `docs/local-readiness/multi-provider-and-billing-completion.md` and update `task.md`.

## COMPLETED TASK: Phase I — Security and tenant isolation
- [x] Step 1: Audit multi-provider and billing architecture for tenant isolation and security constraints (cross-tenant credentials, unauthorized connection usage, gateway credential scope escape, model parameter tampering, billing isolation, webhook forgery, RBAC, secret redaction, error sanitization, credential revocation & disabled connections).
- [x] Step 2: Implement hardening and fixes for any identified isolation or authorization vulnerabilities across Control API, Gateway pipeline, and Billing services.
- [x] Step 3: Implement comprehensive automated security regression tests covering cross-tenant credentials, unauthorized connection use, scope escape/tampering, billing isolation, forged webhook rejection, RBAC enforcement, secret leakage prevention, and revocation/disablement.
- [x] Step 4: Run full backend and frontend regression suites (`go test -C backend ./...` and `npm test`).
- [x] Step 5: Document security guarantees and verification in `docs/security/tenant-isolation-verification.md` and update `task.md`.

## COMPLETED TASK: Phase H — Frontend provider and billing management
- [x] Step 1: Audit existing frontend provider management (`provider-accounts`, `[id]/provider-detail`, `provider-health`, `model-registry`) and billing views (`subscription`, `invoices`, `usage`).
- [x] Step 2: Complete Provider Management UI with catalog, connect flow, secure credential submission, status badges (connected, credential invalid, unreachable, discovery unavailable, unauthorized vs authorized/ready), model capabilities, model-access assignment, and rotation/revocation.
- [x] Step 3: Complete Billing Management UI with current & available plans, dual-gateway checkout (Stripe & Razorpay), subscription state indicators, payment failure alerts, paid invoice history, and quota/usage views.
- [x] Step 4: Ensure all data binds directly to real backend API endpoints with zero fake mock states in production mode.
- [x] Step 5: Write automated frontend component/integration tests and verify full typecheck (`npm run typecheck` and `npm test`).
- [x] Step 6: Create architectural and operational documentation in `docs/architecture/frontend-provider-billing-management.md` and update `task.md`.
\\

## COMPLETED TASK: Phase G — Stripe and Razorpay Payment Integration
- [x] Step 1: Audit existing billing domain, schemas, repositories, and interfaces (`ports/payment`, `infrastructure/payment`, `application/billing`).
- [x] Step 2: Implement Stripe adapter with customer creation/retrieval, subscription checkout, lifecycle webhook parsing, and signature verification.
- [x] Step 3: Implement Razorpay adapter with customer association, subscription/order creation, signature verification (`order_id|payment_id` and webhook HMAC), and event parsing.
- [x] Step 4: Implement shared billing application service with idempotent webhook processing, event deduplication, subscription state machine, invoice persistence, and entitlement enforcement.
- [x] Step 5: Wire Control API endpoints (`/v1/billing/checkout`, `/v1/billing/webhooks/stripe`, `/v1/billing/webhooks/razorpay`, `/v1/billing/verify-payment`) with auth and signature enforcement.
- [x] Step 6: Connect frontend billing workflows (provider selection, checkout initiation, payment status, payment history, and entitlement display).
- [x] Step 7: Write comprehensive automated unit, integration, and mock webhook tests (successful/failed checkout, signature verification, idempotency/deduplication, out-of-order events, renewal, cancellation, entitlement changes).
- [x] Step 8: Create authoritative documentation in `docs/billing/stripe-integration.md`, `docs/billing/razorpay-integration.md`, and `docs/billing/billing-state-machine.md` and verify full test suites.

## COMPLETED TASK: Phase F — Preserve provider-specific features without breaking compatibility
- [x] Step 1: Design capability-aware provider parameters validation and mapping in `ports/provider/provider.go`.
- [x] Step 2: Implement provider-specific parameter preservation and strict unsupported parameter rejection across Anthropic, OpenAI, Gemini, and Hugging Face adapters.
- [x] Step 3: Preserve tool definitions, tool responses (`tool_result`, `function`, `tool`), multimodal data, and system instruction semantics across adapters.
- [x] Step 4: Implement streaming event and usage metadata preservation (request IDs, tool call deltas, cached/reasoning tokens) across all 4 adapters.
- [x] Step 5: Add comprehensive adapter contract tests covering parameter preservation, rejection of unsupported parameters, tool semantics, and streaming conversion.
- [x] Step 6: Create comprehensive documentation in `docs/providers/provider-specific-features.md` and verify full test suites.

## COMPLETED TASK: Phase E — Provider-aware slash commands in the Sentinel CLI
- [x] Step 1: Design and implement the centralized slash-command registry (`Command`, `CommandRegistry`, provider applicability, capability requirements, arguments, execution handler).
- [x] Step 2: Implement common Sentinel commands (`/help`, `/model`, `/provider`, `/status`, `/usage`, `/cost`, `/permissions`, `/clear`, `/compact`, `/stream`, `/exit`) with real conversation compaction and history management.
- [x] Step 3: Implement provider-specific command sets (Anthropic, Gemini, OpenAI, Hugging Face) matching authorized connection and model capabilities.
- [x] Step 4: Implement interactive slash-command menu on `/` and dynamic command set updating on `/provider` or `/model` switch.
- [x] Step 5: Implement comprehensive unit and integration tests covering registry resolution, provider filtering, permission checks, `/clear`, and `/compact`.
- [x] Step 6: Create comprehensive documentation in `docs/cli/slash-commands.md` and verify full test suites.

## COMPLETED TASK: Phase D — Provider-bound Sentinel gateway credentials
- [x] Step 1: Extend Gateway Credential domain models, types, and database queries (`AllowedProviders`, `AllowedModels`, `ProviderAccountID`, `RateLimit`, `BudgetLimit`).
- [x] Step 2: Update identity service and Control API (`/v1/gateway-credentials`) to persist and return provider connection bindings and model restrictions.
- [x] Step 3: Implement server-side scope enforcement in Gateway pipeline (`executePipeline`) and model listing (`handleListModels`) preventing model and provider escapes.
- [x] Step 4: Adapt CLI session, slash commands (`/provider`, `/model`), and client to authorized providers and permitted models.
- [x] Step 5: Implement automated tests verifying provider bounding, model restriction enforcement, revocation, rotation safety, and spoofing resistance.
- [x] Step 6: Create comprehensive architecture documentation in `docs/security/provider-bound-credentials.md` and verify full regression suite.

## COMPLETED TASK: Phase C — Dynamic Model Discovery and Registry
- [x] Step 1: Extend model metadata storage (`domain.Model`, `ports/repositories`, sqlc queries) with capabilities, context limits, pricing version, and status.
- [x] Step 2: Implement dynamic provider model discovery with capabilities discovery across adapters (OpenAI, Gemini, Hugging Face, Anthropic supported models).
- [x] Step 3: Implement provider-aware model filtering in Gateway (`/v1/models`) and Control API (`/v1/models`), strictly eliminating un-connected provider models and static fallbacks.
- [x] Step 4: Enforce connection existence, active status, valid credentials, and model availability in Gateway request routing.
- [x] Step 5: Update Frontend Model Registry and selection UI to display rich metadata and provider connection status.
- [x] Step 6: Write automated regression tests covering 2 organizations with different provider connections, unavailable/revoked models, and unauthorized policy denials.
- [x] Step 7: Document dynamic model discovery architecture in `docs/models/dynamic-model-discovery.md` and verify full test suite.

## COMPLETED TASK: Phase B — Secure Provider Credential Management
- [x] Step 1: Audit existing provider accounts and credentials subsystem against security and workflow requirements.
- [x] Step 2: Implement Provider Account disable/enable (`UpdateStatus`) and deletion (`Delete`) with cascade secret cleanup in repository, application service, and control API.
- [x] Step 3: Implement Account-level validation (`ValidateAccount`), health check (`CheckAccountHealth`), and model discovery (`SyncAccountModels`) using stored credentials.
- [x] Step 4: Verify credential ownership boundaries (Org Provider Keys vs Gateway Credentials vs User Model-Access Scope).
- [x] Step 5: Update Frontend Provider Accounts UI to support testing connection, model discovery, disabling/enabling, rotation, and deletion.
- [x] Step 6: Write comprehensive unit and integration tests for tenant isolation, RBAC, rotation rollback, secret non-leakage, and account lifecycle.
- [x] Step 7: Document security model and credential architecture in `docs/security/provider-credential-management.md` and run full regression suite.

## COMPLETED TASK: Phase A — Multi-Provider Adapter Architecture & Compatibility Matrix
- [x] Step 1: Design and extend unified provider contract and normalized inference representation in `ports/provider/provider.go`.
- [x] Step 2: Implement `InferenceProvider` interface on Anthropic adapter with capability discovery and message conversion.
- [x] Step 3: Implement `InferenceProvider` interface on OpenAI adapter with capability discovery and chat conversion.
- [x] Step 4: Implement `InferenceProvider` interface on Google Gemini adapter with capability discovery and content generation conversion.
- [x] Step 5: Implement `InferenceProvider` interface on Hugging Face adapter with capability discovery and inference endpoints conversion.
- [x] Step 6: Update application provider `Registry` to support `InferenceProvider` resolution without breaking existing stubs.
- [x] Step 7: Write comprehensive provider-specific unit tests and backward-compatibility tests across all 4 adapters.
- [x] Step 8: Create authoritative architectural documentation in `docs/providers/provider-adapter-architecture.md` and verify full suite.

## COMPLETED TASK: Extend Sentinel with Hugging Face, Dual Payments (Stripe & Razorpay), and CLI Slash Commands
- [x] Step 1: Inspect repository, Git status, provider interfaces, billing modules, model registry, and CLI structure.
- [x] Step 2: Produce dependency-aware implementation plan artifact (`multi_provider_dual_payments_plan.md`).
- [x] Step 3: Implement Hugging Face Inference Providers adapter with dynamic model discovery and gateway routing.
- [x] Step 4: Implement dual payment-provider subsystem (Stripe + Razorpay) for checkout, subscriptions, and webhooks.
- [x] Step 5: Implement provider-aware slash commands in the Sentinel CLI (`/provider`, `/model`, `/stream`, `/tokens`, `/cost`, etc.).
- [x] Step 6: Update frontend routes for provider selection (Hugging Face) and dual payment gateways (Stripe & Razorpay).
- [x] Step 7: Write comprehensive automated regression tests for Hugging Face, Stripe, Razorpay, and CLI slash commands.
- [x] Step 8: Verify full test suite (`go test ./...`, `./scripts/verify-local.sh`), update documentation, and update `task.md`.

## COMPLETED TASK: Push Changes to GitHub on Explicit User Command
- [x] Step 1: Stage verified modifications and untracked readiness files (`git add -A`).
- [x] Step 2: Commit changes with descriptive message.
- [x] Step 3: Push changes to remote repository (`git push origin main`).
- [x] Step 4: Verify clean git status and remote synchronization.

## COMPLETED TASK: Phase F, G, H, & Final Deliverables
- [x] Phase F: Verify event reliability, duplicate delivery deduplication, worker & broker restart/backlog recovery, DB/Redis failure handling, and usage reconciliation.
- [x] Phase G: Test live Gemini integration separately (check for valid user-supplied `GEMINI_API_KEY`; execute live inference if present, or mark BLOCKED with precise reason; never expose key).
- [x] Phase H: Run full final regression suite (Go vet/tests/race, Frontend lint/tsc/test/build, SDK/CLI tests, migrations, isolation, E2E journeys, verify-local).
- [x] Final Deliverables: Produce/update `docs/local-readiness/critical-defects.md`, `final-test-report.md`, `remaining-gaps.md`, and `local-runbook.md`, assign evidence-based status (`READY WITH DOCUMENTED LIMITATIONS`), and update `task.md`.

## COMPLETED TASK: Phase C, D, & E — Rate-Limiting, Configuration Safety, and Request-to-Dashboard Lifecycle
- [x] Phase C: Inspect `redis_rate_limiter.go` & bootstrap wiring; verify atomic concurrency, auth identity scoping, Redis outage policy, revoked credential enforcement, and live Redis regression tests.
- [x] Phase D: Inspect compose, env templates, auth config, CSRF, sessions, migrations, readiness, and scripts (`dev-ctl.sh`, `verify-local.sh`); ensure zero secret leakage and safe opt-in local dev.
- [x] Phase E: Verify end-to-end request lifecycle across providers, 3-user scenario (SDK & CLI, streaming/non-streaming, governance/denial, revocation), usage persistence, analytics, and dashboard metrics.
- [x] Verification: Execute regression suites (`go test ./...`, `./scripts/verify-local.sh`, `cmd/e2e`), update documentation deliverables, and update `task.md`.

## COMPLETED TASK: Phase B — Correct Redis-Backed Budget Accounting
- [x] Step 1: Design & audit Redis budget keys, Lua scripts, and window boundary calculation.
- [x] Step 2: Implement deterministic window boundary calculation (UTC, daily, monthly, weekly, custom) preventing expired window blocking.
- [x] Step 3: Implement atomic ZSet + Hash reservation tracking in Redis Lua scripts to eliminate aggregate inflation on reservation expiry.
- [x] Step 4: Implement explicit Redis failure policy (`fail_closed` vs `degraded_local`) and non-silent error handling.
- [x] Step 5: Ensure pricing consistency between gateway estimates, provider reporting, and persisted ledger.
- [x] Step 6: Write comprehensive regression tests covering all required scenarios against live Redis.
- [x] Step 7: Run verification suite (`go test ./...`, `./scripts/verify-local.sh`) and update deliverables.

## COMPLETED TASK: Phase A — Repository Verification, Architecture Tracing & Baseline Defect Audit
- [x] Step 1: Inspect Git status and identify exact repository revision (`30d9a1d12101ca325d516a763098f87287fe4689`).
- [x] Step 2: Inspect target files (`budget_guard.go`, `gateway.go`, `redis_rate_limiter.go`, `bootstrap/run.go`, `deploy/compose/docker-compose.yml`).
- [x] Step 3: Trace budget lifecycle, rate-limiting & fallback, pricing calculation, usage persistence, event processing, and analytics.
- [x] Step 4: Run all feasible baseline tests and record exact results (passing, failing, skipped, blocked).
- [x] Step 5: Audit existing defects against latest code and create/update `docs/local-readiness/critical-defects.md`.


## COMPLETED TASK: Phase J — Full Regression and Recovery Tests
- [x] Step 1: Go Test Suite, Formatting, Vetting & Race: run `gofmt`, `go vet`, `go test ./...`, and `go test -race` across backend packages.
- [x] Step 2: Frontend Validation: run `npm run typecheck`, `npm run lint`, `npm test`, and `npm run build` in `app/`.
- [x] Step 3: SDK & CLI Test Suites: execute test suites in `packages/api-client` and `packages/cli`.
- [x] Step 4: Database Migrations, API Contracts & Multi-Tenant Isolation: validate migrations, auth, tenant isolation, and contract tests.
- [x] Step 5: Gateway, Usage, Cost & Worker Recovery Tests: run gateway integration, rate limiting/budget recovery, and durable worker tests.
- [x] Step 6: End-to-End Suite & Final Verification: run `go run ./cmd/e2e`, `./scripts/verify-local.sh`, and `go run ./cmd/perf`.
- [x] Step 7: Final Deliverables: update `docs/local-readiness/final-test-report.md`, `docs/local-readiness/remaining-gaps.md`, `docs/local-readiness/local-runbook.md`, and `docs/local-readiness/known-limitations.md`, assign final status (`READY WITH DOCUMENTED LIMITATIONS`), and update `task.md`.

## COMPLETED TASK: Phase I — Test Three Users Through SDK and CLI
- [x] Step 1: Fixture Setup & Secret Protection: create 1 organization, 3 independent users with separate Sentinel credentials, explicit model access rules, and budget settings; verify employee credentials never contain provider master API keys.
- [x] Step 2: SDK & CLI Inference Protocol Verification: verify requests through CLI (`packages/cli`) and SDK (`packages/api-client`) across all 3 users (non-streaming, streaming SSE, valid/invalid models) authenticating with individual Sentinel keys.
- [x] Step 3: Multi-Dimension Attribution & Analytics: verify usage, token counts, latency, and cost attribution per user/org/project across `/v1/usage/*` and Next.js analytics views.
- [x] Step 4: Authorization, Isolation & Credential Revocation: verify subject-level isolation (users cannot access another user's requests/history without permission) and immediate 401 revocation enforcement without affecting other users.
- [x] Step 5: Deterministic Fixtures & Live Gemini Smoke Test: support deterministic mock responses for CI/repeatable testing, and execute a live Gemini smoke test when `GEMINI_API_KEY` is provided.
- [x] Step 6: Automated Testing & Deliverables: update `backend/cmd/e2e/gemini_master_execution.go`, add unit/regression test `backend/internal/interfaces/controlapi/three_user_fixture_test.go`, execute verification suite, update `task.md`, and document in `docs/local-readiness/three-user-sdk-cli-verification.md` and `docs/local-readiness/final-test-report.md`.

## PREVIOUS TASK: Phase H — Verify the Frontend and Dashboard
- [x] Step 1: Frontend Route & Layout Audit: audit Next.js routes under `app/src/app/(dashboard)/app/` (Overview, Observe, Govern, Security, Platform, Organization, Billing, Developer), navigation links, and redirects.
- [x] Step 2: Widget Resilience & Isolation: verify loading, empty, error, and populated states; ensure error boundaries or try/catch blocks prevent one failed widget from blanking unrelated widgets.
- [x] Step 3: API Contract & Data Consistency: verify data formatting and agreement between backend JSON envelopes (`/v1/usage/*`, `/v1/audit-logs`, `/v1/alerts`, `/v1/guardrails`, `/v1/conversations`) and UI components (charts, token counts, cost micro-dollars, request history, guardrails, alerts).
- [x] Step 4: Filtering, Pagination & Authorization: test date filtering, pagination controls, search parameters, and role-based view enforcement.
- [x] Step 5: Automated Testing & Deliverables: execute frontend test suite (`npm run typecheck`, `npm test`, `npm run build`), verify e2e UI tests (`cmd/e2e`), update `task.md`, and document verification results in `docs/local-readiness/frontend-dashboard-verification.md`.

## PREVIOUS TASK: Phase G — Verify Policies, Guardrails, and Alerts
- [x] Step 1: Policy Management & Model Access: audit and test policy CRUD (creation, editing, enabling, disabling, deletion), allow/deny rule precedence, model access restrictions, and gateway cache invalidation timebounds (<10ms).
- [x] Step 2: Guardrails Pipeline: verify pre-forwarding blocking for PII, secrets, unsafe keywords, and prompt length; confirm zero upstream calls on violation and security event recording.
- [x] Step 3: Budgets & Rate Limits: verify concurrent sliding window rate limits (50 concurrent workers) and budget guard zero-accrual on rejection.
- [x] Step 4: Alerts Engine & Incident Persistence: verify alert rules trigger strictly on real conditions (spend thresholds, error rate, provider failures, policy violations); verify repeat suppression, auto-recovery resolution, and timestamp persistence.
- [x] Step 5: Audit Events & Tenant Isolation: verify immutable audit trail triggers, sensitive token redaction, and multi-tenant isolation across policies, alerts, and audit endpoints.
- [x] Step 6: Automated Testing & Deliverables: add `backend/internal/application/policy/service_test.go`, execute verification tests, and update `task.md`.

## PREVIOUS TASK: Phase F — Verify Event Processing and Durable Usage
- [x] Step 1: Event Bus & Worker Pipeline Audit: audit publisher in `gatewayv2/telemetry_emitter.go`, `telemetry_sink.go`, `nats_publisher.go`, and consumer in `backend/internal/interfaces/worker/` and repositories for ack semantics, deduplication, backoff retries, dead-letter/unrecoverable event observability, and queue groups.
- [x] Step 2: Durable Idempotency & Aggregation Verification: verify that duplicate events (same `request_id`) are deduplicated at ingestion and in rollup tables (`ON CONFLICT DO NOTHING` / unique constraint), and verify reconciliation between request records and org/user rollup aggregates.
- [x] Step 3: Outage & Recovery Lifecycle Verification: verify worker restarts, NATS restarts (spooling / buffer drain), temporary DB outages, and retry limits with exponential backoff.
- [x] Step 4: Multi-Tenant Attribution & Analytics Cache Isolation: verify Redis analytics caching keys (`aigw:analytics:<orgID>:...`), TTLs, and cross-tenant isolation.
- [x] Step 5: Verification & Deliverables: write automated regression/recovery tests in `backend/internal/interfaces/worker/`, execute full test suite, update `task.md`, and document findings in `docs/local-readiness/durable-event-processing.md`.

## PREVIOUS TASK: Phase E — Complete the End-to-End Request Lifecycle
- [x] Step 1: Provider Adapters & Gateway Pipeline Audit: audit OpenAI, Anthropic, and Gemini implementations in `backend/internal/infrastructure/provider/*`, `gatewayv2/stream_proxy.go`, `gateway.go`, and `inspector.go` for auth retrieval, model validation, request/response translation, streaming/non-streaming, token accounting, error handling, and cancellation.
- [x] Step 2: Protocol Translation & Token Accounting Hardening: verify and fix parsing of provider-specific usage payloads (e.g. Gemini `usageMetadata`, OpenAI `usage`, Anthropic `message_delta`/`message_start`), missing usage metadata fallbacks, and cost calculations.
- [x] Step 3: Stream Interruption, Timeout & Error Lifecycle: ensure client disconnects (`context.Canceled`), timeouts, and malformed provider chunks cancel upstream calls, finalize reserved budgets, and record attribution with appropriate status/error codes.
- [x] Step 4: Comprehensive Multi-Provider Test Suite: write deterministic mock-server tests covering all 3 providers (OpenAI, Anthropic, Gemini) across streaming/non-streaming, valid/invalid models, 429/500 errors, client disconnects, missing usage metadata, correlation IDs, latency tracking, policy/guardrail enforcement, and persistence.
- [x] Step 5: Verification & Deliverables: execute tests (`go test ./...`, `./scripts/verify-local.sh`), update `task.md`, and document findings in `docs/local-readiness/provider-lifecycle-matrix.md`.

## PREVIOUS TASK: Phase D — Harden the Local Environment
- [x] Step 1: Configuration & Secret Audit: audit `deploy/compose/docker-compose.yml`, `.env.example`, `config.go`, and diagnostics for placeholder secret detection, missing configs, never printing secret values, and dev-bypass opt-in flags.
- [x] Step 2: Auth, CSRF & Lifecycle Verification: audit and verify CSRF protection, session expiry, logout revocation, and credential revocation behavior.
- [x] Step 3: Script & Database Guarding: audit startup and migration scripts (`dev-ctl.sh`, `verify-local.sh`, `migrate`, `seed`) to ensure explicit confirmation on destructive resets, correct checkout/endpoints testing, and clean readiness probes.
- [x] Step 4: Automated Testing: add tests for configuration validation, authentication lifecycle, startup failures, and safe setup scripts.
- [x] Step 5: Verification & Deliverables: run full verification suite (`./scripts/verify-local.sh`, `go test ./...`), update `task.md`, and document security posture in `docs/local-readiness/local-security-hardening.md`.

## PREVIOUS TASK: Phase C — Fix Rate-Limit Behavior
- [x] Step 1: Inspect `redis_rate_limiter.go`, `rate_limiter.go`, `gateway.go`, and `bootstrap/run.go`: audit key derivation, Redis usage, fallback behavior, failure modes, and header spoofing protections.
- [x] Step 2: Implement robust rate limiting: ensure authentic credential scope binding, explicit documented degraded/fail-closed policy when Redis is unavailable, correct Redis sliding window Lua script, counter TTLs, and reject header spoofing.
- [x] Step 3: Attribution & usage integrity: ensure rejected 429 requests are attributed and recorded in security/audit logs without creating false provider token/spend usage.
- [x] Step 4: Write comprehensive tests: concurrency, window expiration, Redis outage & recovery, degraded mode guarantees, key revocation, and cross-user/cross-org isolation.
- [x] Step 5: Verify regression tests and full test suite (`go test ./...`, `./scripts/verify-local.sh`).
- [x] Step 6: Update documentation & task tracker: update `task.md`, `docs/local-readiness/feature-matrix.md`.

## PREVIOUS TASK: Phase B — Fix Budget Enforcement
- [x] Step 1: Design & Architecture: define source of truth, reservation lifecycle (reserve, finalize, release, timeout expiry), Redis store with in-memory fallback, and idempotency guarantees.
- [x] Step 2: Update `BudgetGuard` interface and implementation: add atomic reservation with request ID tracking, Redis integration with in-memory fallback, abandoned reservation expiry, and duplicate event deduplication.
- [x] Step 3: Wire `BudgetGuard` in `gateway.go` and `bootstrap/run.go`: pass `redisClient` and SpendLoader, execute `Reserve` in Stage 7 before upstream forwarding, and finalize/release in stream handling.
- [x] Step 4: Write comprehensive tests covering all required scenarios: below/at/above limit, concurrent competition, gateway restart, duplicate events, failed/interrupted provider calls, reservation expiration, cross-org isolation, and persistence outages.
- [x] Step 5: Verify regression tests and full test suite (`go test ./...`, `./scripts/verify-local.sh`).
- [x] Step 6: Deliverables & documentation update: update `task.md`, `docs/local-readiness/feature-matrix.md`.

## PREVIOUS TASK: Phase A — Establish the Actual Baseline
- [x] Step 1: Structural & Entry Point Inventory: inspect repo layout, git status, compose files, env examples, migrations, and exact entry points (Web UI, Control API, Gateway v2, Worker, SDK, CLI).
- [x] Step 2: Implementation & Engine Audit: audit budget enforcement, rate limiting, policy resolution, usage recording, event handling, and analytics pipelines.
- [x] Step 3: Comprehensive Test Execution: run all available test suites, record exact commands, exit codes, durations, and outputs (Go vet/tests/race, Frontend lint/typecheck/tests/build, SDK tests, CLI tests, E2E journeys, and performance benchmarks).
- [x] Step 4: Capability Classification & Gap Reconciliation: categorize features across PASS, FAIL, PARTIAL, BLOCKED, and NOT TESTED against current source code.
- [x] Step 5: Deliverables: update `docs/local-readiness/current-state.md`, `docs/local-readiness/feature-matrix.md`, `docs/local-readiness/baseline-test-results.md`, and `docs/local-readiness/risk-register.md`, then update `task.md`.

## PREVIOUS TASK: Git Commit and Push to GitHub on Explicit User Command
- [x] Step 1: Stage verified modifications and untracked readiness files (`git add -A`)
- [x] Step 2: Commit changes with descriptive message (`b1b39d9`)
- [x] Step 3: Push to remote repository (`git push origin main`)
- [x] Step 4: Verify clean git status and remote synchronization

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
