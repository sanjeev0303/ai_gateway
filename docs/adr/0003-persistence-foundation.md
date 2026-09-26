# ADR-3: Persistence foundation — Postgres, migrations, RLS, repositories

- **Status:** Accepted (M2)
- **Date:** 2026-09-25
- **Deciders:** Project owner, Sisyphus (implementation)

## Context

M2 ships the control-plane database every process builds on: the tenancy
model (organizations, users, memberships, projects), API credentials,
policies and rules, per-request usage records, subscriptions, and pricing
models. Three concerns drove the design:

1. **Multi-tenancy is the core promise** (ADR-0). Every table carries
   `organization_id`, and a bug that leaks rows across tenants is a
   correctness failure, not a test failure.
2. **Audit-grade usage** (README) requires usage and audit records that are
   append-only — no UPDATEs, no DELETEs, ever.
3. Three Go processes (`gateway`, `control-api`, `worker`) share the schema;
   the schema must be versioned, forward-only, and reproducible in test,
   CI, and production from a single source of truth.

## Decision

### Stack

- **Postgres 16** as the control-plane database (`deploy/compose` runs
  `postgres:16-alpine`; the same major version is assumed in production).
- **`pgx/v5` + `pgxpool`** for the connection pool — no ORM, no query
  builder. SQL is explicit and reviewable in the hot path.
- **goose v3 as an embedded library** for migrations — migrations are Go
  code paths (`internal/infrastructure/postgres/migrations`, embedded via
  `//go:embed`), applied by `cmd/migrate`, `cmd/seed`, and the test harness.
  No `goose` CLI dependency anywhere.
- **sqlc** for type-safe query codegen from the `-- +goose Up` sections
  (`backend/sqlc.yaml`): generated code lands in
  `internal/infrastructure/postgres/sqlc` and is committed.

### Migrations are forward-only

- Migrations are numbered `000NN_*.sql`, applied in order, recorded in
  `goose_db_version`. **Applied migrations are never edited** — schema
  corrections ship as new forward migrations (e.g. 00014 redefines the
  credential resolver first declared in 00012 rather than patching 00012).
- Dollar-quoted function bodies are wrapped in
  `-- +goose StatementBegin/StatementEnd` because goose splits statements
  naively on `;`.
- `cmd/migrate` applies all pending migrations and is idempotent — safe to
  run during deploys. `cmd/seed` runs migrations plus the idempotent demo
  seed (`internal/infrastructure/postgres/seed.sql`) and is safe to re-run.

### Tenant isolation is enforced by RLS, not by application discipline

- An `app` schema holds the tenancy plumbing: `app.current_org_id()` is a
  `STABLE SECURITY DEFINER` function reading the `app.org_id` GUC (set per
  transaction, never globally).
- **Every tenant-owned table** (projects, api_credentials, policies,
  policy_rules, usage, audit_log, subscriptions, pricing_models, …) has a
  `FORCE RLS` policy: rows are visible/insertable only when
  `organization_id = app.current_org_id()`.
- `FORCE RLS` makes even the table owner (the app's pool role) subject to
  the policy — isolation holds for repository code, raw SQL in tests, and
  future ad-hoc queries alike. The RLS test (`TestRLSEnforcesTenantScope`)
  additionally proves the policies with a non-superuser role: no GUC ⇒ zero
  rows; GUC = org ⇒ only that org's rows.
- Repositories bind tenancy explicitly:
  - `WithTenantTx(ctx, orgID, fn)` — opens a transaction, sets
    `app.org_id = orgID`, and runs `fn` inside it.
  - `WithTx(ctx, fn)` — a plain transaction used by tenant-creation flows
    that must write across several tenants' tables atomically.
- API-credential resolution (`app.resolve_api_credential`) is a
  `SECURITY DEFINER` function matching `key_hash` and returning
  **`SETOF api_credentials`** — a non-matching hash yields zero rows
  (`pgx.ErrNoRows` → `ErrNotFound`) instead of a NULL composite row.
  Key material (hash) is checked via this function; the plain `SELECT` from
  any other code path is pointless because the table is RLS-isolated.

### Audit and usage are append-only

- `audit_log` and usage records are immutable: an `audit_no_update_delete`
  trigger rejects UPDATE/DELETE. Retention is a later worker concern
  (M6) — enforcement mechanics are in place now.
- Generated UUIDs (never client-supplied) key every table; no natural-key
  surprises in multi-tenant uniqueness (`organization_id` + slug unique
  constraints, not global slugs).

### Repositories

- One repository per aggregate under
  `internal/infrastructure/postgres/repositories`, each implementing a
  `ports/repositories` interface; domain code depends only on the port
  (ADR-0/ADR-2).
- pgx errors map at the boundary: `pgx.ErrNoRows` → `ErrNotFound`,
  `23505` → `ErrConflict`, `23503`/`23514` surface as raw constraint errors
  (the caller decides; tests use `requireRawConstraint`). No sentinel —
  including `ErrNotFound` — is faked from application logic; only the DB
  declares truth.

### Testing without Docker

- Integration tests boot a real Postgres via
  `fergusstrange/embedded-postgres` (one instance per package run, fresh
  from migrations every time), so `go test -race -shuffle=on ./...` needs
  no external services and is reproducible in CI. `TestMain` in
  `repositories` applies migrations + seed, exactly like `cmd/seed`.
- Every repository test creates tenants through the same
  `newTenant(t) → (orgID, userID, projectID)` helper the production flow
  uses, so cross-tenant assertions exercise real RLS outcomes, not
  lookalike fixtures. Byte-exact assertions on `jsonb` columns are compare
  semantically (jsonb normalizes formatting, e.g. `{"scope": "gateway"}`).

## Consequences

- One source of truth (migrations) drives production, `cmd/migrate`,
  `cmd/seed`, sqlc codegen, and the test harness — no drift between
  environments.
- RLS plus FORCE means a mistake in repository WHERE clauses is caught by
  the policy layer, and the embedded-Postgres suite catches it in CI with
  no external infrastructure.
- Forward-only migrations keep deploys safe and idempotent; the cost is
  that schema corrections require a new migration (rare, cheap).
- Cost: every new tenant-owned table must declare RLS policies, append-only
  triggers where required, and registration in the RLS test — intentional
  friction owned by the milestone checklist.

## Alternatives considered

- **ORM (GORM/ent)** — rejected: implicit queries fight RLS and audit
  requirements; sqlc keeps SQL explicit and reviewable.
- **Schema-per-tenant** — rejected: connection/config explosion and
  migration fan-out; row-level RLS in one schema is simpler at this scale
  and keeps cross-tenant reporting possible for the owner.
- **Application-only tenant scoping** — rejected: defense in depth requires
  the database to enforce isolation even for direct SQL (support, metrics,
  future tools).
- **Atomic/other SQL migration tooling** — rejected: goose-as-library gives
  the same ordering/versioning with no external CLI and one embed path
  already used by tests.
- **Editing applied migrations in place** — rejected during M2 (00012 →
  resolver fix went through a new 00014): forward-only is the contract that
  keeps deploy-time migrations deterministic.