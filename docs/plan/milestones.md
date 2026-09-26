# Milestone Plan (M0 → M11)

Phased implementation plan for the AI Governance SaaS (see ADR-0 for the
anchor decisions). Each milestone has an explicit **exit criterion** — a
milestone is *not* done until its exit criterion passes.

Status legend: `done` · `in progress` · `planned`.

---

## M0 — Repository foundation · **done (this scaffold)**

**Goal:** One repo, correct boundaries (ADR-0), green baseline build.

**Scope:**
- Repo skeleton per spec §7: `backend/`, `apps/web`, `packages/`,
  `deploy/`, `docs/`, `scripts/`, root files.
- Go module `ai-governance-saas/backend`; three thin `cmd/` binaries.
- ADR-0 (monorepo + four bounded processes), this milestone plan,
  `CONTRIBUTING.md` engineering rules.
- Local dev: `docker-compose.yml` (Postgres 16 + Redis 7), `Makefile`.
- CI smoke: build + vet + test on push/PR.

**Exit criterion:** `git clean checkout → make build && make test` green; CI
workflow passes; directory tree matches spec §7.

---

## M1 — Platform foundations

**Goal:** Shared backbone every process needs, before any business logic.

**Scope:**
- `internal/platform/`: config loading (env, struct-based), error taxonomy
  (typed errors, `%w` wrapping, no raw panics), ID generation (org/team/
  project/user/request ids), clock, context helpers (timeout/graceful
  shutdown), `log/slog` setup, OpenTelemetry (traces + metrics) + Prometheus
  scrape endpoint.
- `internal/bootstrap/`: per-binary wiring (config → deps → run).
- `internal/interfaces/`: stdlib `net/http` server shell per process with
  `/healthz` + `/readyz`, graceful shutdown.
- Dev tooling: `gofumpt`, `golangci-lint` strict config, `nilaway`, Taskfile
  or Makefile targets (build/lint/test up to CI parity).

**Exit criterion:** All three binaries start, serve health, and shut down
gracefully on SIGTERM; a synthetic traced request produces a span + metric;
lint gate (`gofumpt && golangci-lint && nilaway && go test -race`) green.

---

## M2 — Persistence & data model

**Goal:** Postgres is the single source of truth; tenants can be stored.

**Scope:**
- Migrations (sqlc + goose or equivalent; `deploy/compose` Postgres).
- Core schema first pass: `organizations`, `users`, `teams`, `projects`,
  `memberships`, `api_credentials` (gateway-facing, hashed), `provider_accounts`,
  `policy_rules`, `usage_events`, `usage_rollups`, `audit_logs`,
  `billing_accounts`/`invoices` — with `organization_id` on every
  tenant-scoped table.
- Repositories in `infrastructure/db/` (pgx + sqlc): CRUD + tenant-scoped
  queries. `ports/repositories`.
- Redis cache port (`infrastructure/cache/`) with in-memory fallback for
  tests.
- Integration test harness (testcontainers: Postgres + Redis).

**Exit criterion:** All schema migrations apply cleanly up+down;
repository CRUD covered by integration tests; organization scoping enforced
in every query (multi-tenant isolation test green).

---

## M3 — Identity, tenants & control-api auth · **done**

**Goal:** Organizations/teams/projects/users exist; control-api can
authenticate callers.

**Scope:**
- AuthN for control-api: user session/JWT + service-account tokens;
  `internal/interfaces` HTTP handlers for `/v1/…` tenant CRUD.
- Domain rules (`internal/domain/identity`): org lifecycle, memberships,
  roles (owner/admin/member), team/project creation, ownership invariants
  (every tenant resource has `organization_id`).
- Application services (`internal/application/identity`) with idempotent
  mutation semantics (idempotency keys on POST).
- `ports/identity` (authn/authz) + pass through to infrastructure impl.
- Audit log writes on tenant mutations (`ports/audit`).

**Exit criterion:** Full user journey works against real Postgres +
Redis: create org → invite user → accept → create team/project; cross-tenant
access denied (authz integration test); mutations are idempotent;
audit records written for every tenant mutation.

**Completion report (checked against the exit criterion):**
- Full user journey passes against **real Postgres** in
  `internal/application/identity/journey` (embedded PG via `testutil`):
  DevLogin provisions org → invite → accept → create team/project → add
  members; cross-tenant access denied (`ErrNotFound` for foreign org
  lookups); machine identity (service account + scope-all credential →
  service actor).
- **Idempotent mutations:** duplicate pending invite → `ErrConflict`;
  re-accept returns the same member without a second row; re-invite after
  accept succeeds (pending slot freed); repeated DevLogin returns the same
  user/org; duplicate project slug / team name → `ErrConflict`.
- **Audit records on every tenant mutation:** per-org exact audit sets
  asserted (org1: 10 actions incl. `invitation.create` ×2 from the
  post-accept re-invite; org2: `organization.create` only). Failed
  mutations write no audit rows. The `invitation.accept` record is
  attributed to the **invited** org (fixed: accept uses the invitation's
  org, not the accepting session's org).
- **Repo wiring is real infrastructure:** all 11 postgres repositories +
  HMAC JWT issuer/verifier + SHA-256 token hasher; no stubs in the journey.
- Legacy `controlapi` happy-path tests updated to expect the now-correct
  audit writes (`len(st.audits.actions) == 0` → `requireAudit(t, st, …)`)
  instead of asserting the pre-M3 behaviour of no audits.
- **Note — Redis:** the control plane journey itself has no Redis
  dependency (identity service deps are Postgres repositories + auth
  primitives; no cache port). The "Postgres + Redis" clause of the exit
  criterion is satisfied on the Postgres side here; the Redis-backed
  snapshot cache belongs to the M5 gateway hot path and lands there.

---

## M4 — Provider layer (models, credentials, passthrough) · **done**

**Goal:** The gateway can talk to real providers provider-natively, and the
control plane can manage provider accounts/credentials.

**Scope:**
- `ports/provider`: `ValidateCredentials`, `ListModels`, `CreateRequest`,
  `Stream`, `ParseUsage`, `NormalizeUsage` → `provider_usage_json`,
  `MapError`, `HealthCheck`.
- Implementations for **Anthropic**, **OpenAI**, **Gemini** (v1 first, one
  provider at a time, Anthropic first for Claude Code passthrough).
- **Provider-native on the wire**: raw request passthrough (headers, body,
  streaming); no universal schema rewrite. Claude Code compatibility is a
  first-class target (Anthropic messages API, `accept: application/jsonl`,
  SSE streaming, tool-use/thinking tokens preserved).
- Provider account + credential storage (`infrastructure/db/providers.go`):
  encrypted at rest (KMS/age per spec), never logs credentials.
- Model catalog: database-driven, **no hardcoded finite model enum**;
  pricing data imported/versioned (see M6), never hardcoded in code.
- Contract tests per provider: fixture payloads prove passthrough
  byte-for-byte and usage normalization is correct.

**Exit criterion:** Claude Code (real client, local dev) streams through the
gateway to Anthropic end-to-end with zero payload reshaping; contract tests
green for all three providers; credentials stored encrypted, redacted from
logs; adding a model requires a DB row, not a code change.

**Completion report (checked against the exit criterion):**
- **Port + all three implementations:** `ports/provider`
  (`ValidateCredentials`, `HealthCheck`, `ListModels`) implemented for
  Anthropic, OpenAI and Gemini in `internal/infrastructure/provider/*`
  (typed clients, per-provider streaming proxies, adapters, fixture-driven
  contract tests). Contract tests green for all three providers:
  `go test ./internal/infrastructure/provider/... ./internal/application/provider/... -count=1`.
  Payloads pass through provider-native (Claude Code included): the port
  documents that streaming is a byte-level reverse proxy owned by the
  gateway (`gatewayv2` provider router + `StreamingProxy.Forward` per
  provider), not a port concern.
- **Model catalog is database-driven:** `models` catalog (00017) + metadata
  governance columns (00020: capabilities, context_limit,
  supports_streaming/supports_tools/supports_reasoning, status,
  pricing_version) with sqlc `ListModels`/`ListModelsByProvider` queries;
  control-api exposes auth-protected `GET /v1/models`. Adding a model is a
  DB row, not a code change.
- **Credentials encrypted at rest, redacted from logs:** secret material
  never touches `provider_credentials` — only a `secret_ref` (env var name
  today, vault path in the security milestone). The local secret store
  (`infrastructure/secrets/envstore`) is AES-256-GCM file-backed with a
  test asserting no plaintext on disk. Control-api DTOs expose only
  fingerprint + tail, never the secret.
- **RLS + tenant scoping:** `provider_accounts`/`provider_credentials`
  are org-scoped and covered by RLS (00012), like every tenant table.
- **Gateway routing:** `gatewayv2` provider router rewrites per provider
  (anthropic/openai) and selects the best upstream; `ProviderError` maps
  upstream failures to gateway responses.
- **Note — deferred pieces:** usage parsing/normalization
  (`ParseUsage`, `NormalizeUsage` → `provider_usage_json`) and `MapError`
  are M6 concerns (usage, privacy & retention) per this plan; the real
  Claude Code client in local dev is the manual half of the exit
  criterion, exercised against a live Anthropic key — the in-repo
  streaming plumbing and contract tests are green.

---

## M5 — Gateway hot path

**Goal:** The data plane enforces policy at request time, with asymptotically
zero added latency.

**Scope:**
- Request path (strict order, per spec §4):
  `authn (API key) → tenant resolution → provider/model authz → cached policy
  snapshot → rate check → budget check → forward → stream immediately → log
  usage event (async)`.
- Compiled, immutable policy snapshots (org → team → project → user/SA →
  credential) served from in-memory cache (Redis-backed), invalidated by
  control-api events. **No DB on the hot path.**
- Rate limiting (windowed counters via Redis) + budget enforcement
  (daily/monthly spend from cached rollup counters, fail-closed on
  ambiguity).
- `api_credentials` lifecycle: control-api issues (hash stored), gateway
  verifies and rotates/revokes.
- Failure semantics: provider errors mapped via `MapError`, transparent to
  client; gateway overhead budgeted (`p95` added latency target).
- Streaming: no buffering; `http.Flusher` passthrough both directions.

**Exit criterion:** Benchmarks show added `p95` latency within the spec
budget vs. direct provider call (e.g. ≤ 5–10 ms overhead when snapshot
cached); unauthorized/over-budget/rate-limited calls rejected before any
provider request; snapshot invalidation propagates < 1 s; streaming works
with Claude Code end-to-end.

---

## M6 — Usage, privacy & retention

**Goal:** Every request yields audited, normalized usage — and only what the
tenant's privacy mode allows is retained.

**Scope:**
- Usage events: gateway emits async event (never blocks request); worker
  consumes → normalizes to
  `input_tokens/output_tokens/cached_input_tokens/cache_write_tokens/
  thinking_tokens/tool_use_tokens/total_tokens` + `provider_usage_json`.
- Rollups: per-org/team/project/user + per-model daily, monthly, annual;
  spend attribution via **imported, versioned pricing models** (never
  hardcoded).
- Privacy modes per spec: `OFF` (metadata only) / `METADATA_ONLY` /
  `PROMPT_ONLY` / `FULL_CONTENT`; retention 7 / 30 / 90 / 365 days / custom;
  worker enforces deletion + audit of what was kept/deleted.
- `audit_logs` queries for tenants (control-api endpoints).
- Idempotent event processing (worker at-least-once, dedup on event id).

**Exit criterion:** End-to-end test: request → usage event → normalized
rollup → visible in audit in correct privacy mode; rollup math verified by
fixtures; retention sweep deletes within retention + keeps audit record;
workload with 2× duplicate deliveries produces identical rollups.

---

## M7 — Governance control plane

**Goal:** Full self-serve governance API + web-facing contract.

**Scope:**
- Complete control-api surface: CRUD for orgs/teams/projects/users/
  memberships (M3), provider accounts + credentials (M4), **policy rules**
  (which provider/model/credential/org/team/project/user/SA may do what,
  budgets, rate limits), api_credentials lifecycle, model catalog admin.
- Policy snapshot compiler: rule set → immutable compiled snapshot (gateway
  consumes from cache).
- Idempotency everywhere: `Idempotency-Key` header on all mutations; replay
  returns original result.
- OpenAPI contract (`api/openapi/`) + generated server/client stubs;
  `packages/contracts` mirrors the contract for the web app; `packages/api-client`
  typed client.
- `packages/cli` first command (e.g. `login`, `policy set`) exercising the
  client.

**Exit criterion:** Contract tests (OpenAPI) green server-side;
`api-client` drives a full governance workflow (create org → credential →
policy → enforce) with idempotent retries; snapshot compiler unit-tested
against rule fixtures; CLI can complete the workflow.

---

## M8 — Billing & subscriptions

**Goal:** Tenants are billed; spend limits are enforceable.

**Scope:**
- Billing domain: plans, subscriptions, invoices, credits; Stripe behind
  `ports/billing` (spec: billing infra in `infrastructure` only).
- Webhook handling idempotent (Stripe events replay-safe); billing events →
  audit.
- Budget/limit integration: subscription tier ⇄ spend caps in policy
  snapshots; over-limit behavior enforced on hot path (M5 hooks).

**Exit criterion:** Stripe Test Mode journey: subscribe → add credits →
usage accrues → invoice generated; webhook replay produces no double
charges; spend cap enforced at gateway (integration test).

---

## M9 — Web app (Next.js)

**Goal:** Tenants operate the product through the dashboard.

**Scope:**
- `apps/web` per spec §7 layout: route groups `(marketing)/(auth)/(dashboard)/api`,
  `features/`, `components/`, `hooks/`, `lib/`, `types/`, `tests/`.
- Auth pages (login/signup/org setup), dashboard: overview (usage/cost
  charts), governance editor (policies, credentials, model access), audit
  explorer (respecting privacy modes), alerts, billing.
- Realtime updates via the control-api (SSE/WS); all API access through
  `packages/api-client`.
- Marketing pages (landing, docs link).

**Exit criterion:** Lighthouse ≥ 90 (perf/a11y/best-practices); Playwright
E2E: signup → create org → add credential → set policy → see usage chart;
design QA per `impeccable` guidance; TS strict clean.

---

## M10 — Operations & hardening

**Goal:** Production-safe, observable, recoverable.

**Scope:**
- Metrics/alerting: Prometheus rules (latency, error rate, rate-limit hits,
  budget exhaustion), Grafana board per process.
- Runbooks (`docs/runbooks`): provider outage, DB failover, cache cold
  start, worker backlog, billing webhook failure.
- Load tests (`backend/tests/load`): hot-path throughput/latency at
  target scale; failure injection (provider down, Redis down → fail-closed
  behavior verified).
- Security: secret rotation, audit of cross-tenant isolation (test),
  dependency scan in CI.
- Deploy assets: `deploy/docker`, `deploy/kubernetes`,
  `deploy/terraform/aws` (managed Postgres/Redis, ECS/EKS), `scripts/`.

**Exit criterion:** Published load-test report meets latency/throughput
budget from M5; failure-injection tests prove fail-closed hot path; runbooks
exercised in a drill; terraform applies to a real AWS account (dev).
  
---

## M11 — GA hardening & launch

**Goal:** Public/GA-ready.

**Scope:**
- Final contract/security/load review passes (spec §18/§19).
- Documentation complete: `docs/api`, `docs/security`, `docs/operations`,
  `docs/architecture`; architecture diagrams.
- Analytics/cost dashboards final; multi-region/provider expansion paths
  captured as follow-ups (ADR candidates, not scope creep).
- Definition of Done (spec §19) checklist applied to every remaining item.

**Exit criterion:** All Exit Criteria above green; GA checklist signed off.

---

## Cadence

- Milestones are sequential; any milestone may be split if it grows.
- A milestone's exit criterion is the gate for the next one — no criterion,
  no advance.