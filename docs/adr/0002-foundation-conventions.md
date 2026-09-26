# ADR-2: Foundation conventions — errors, health, config, HTTP plumbing

- **Status:** Accepted (M1)
- **Date:** 2026-09-25
- **Deciders:** Project owner, Sisyphus (implementation)

## Context

M1 ships the shared foundation every process and the web app build on:
configuration loading, error surfaces, health probes, HTTP middleware, and
the enforcement scripts. Without explicit conventions, three Go processes and
one Next.js app would drift into per-process error formats, probe paths, and
logging styles — exactly what the shared `platform/` packages exist to
prevent (ADR-0).

## Decision

### Errors

- One taxonomy, implemented once in `platform/errors`: `KindInvalidArgument`,
  `KindUnauthenticated`, `KindPermissionDenied`, `KindNotFound`,
  `KindConflict`, `KindRateLimited`, `KindTimeout`, `KindUnavailable`,
  `KindInternal`. Each maps to a stable machine code
  (`invalid_argument`, `unauthenticated`, `permission_denied`, `not_found`,
  `conflict`, `rate_limited`, `timeout`, `unavailable`, `internal`) exposed
  as JSON via the `ErrorEnvelope` schema from ADR-1.
- Every API endpoint returns the envelope on failure — nothing else, no
  plain-text errors. A plain error (no `Kind`) maps to `internal`.
- Client-facing `message` is safe for end users; internal detail goes to logs
  only. **Never** put secrets, stack traces, or provider internals in a
  response.

### Health

- Every process serves `/livez` (liveness), `/readyz` (readiness, all
  registered dependency checks), and `/healthz` (aggregate health with
  per-check detail) — defined in the OpenAPI spec under `system`.
- Body is `{"status": "ok" | "degraded", "checks": {...}}`; 200 when ok,
  503 when degraded. Each process registers its own checks (none at M1;
  Postgres/Redis checks arrive with infrastructure).

### Configuration

- Environment-only configuration via `caarlos0/env` + `godotenv` in
  `platform/`, `AIGW_` prefix, validated at startup — a process refuses to
  boot on invalid config (fail fast).
- `.env` is for local dev and never committed; `backend/.env.example`
  documents every variable with placeholders and **never** contains real
  secrets. `caarlos0/env` prints redacted summaries only.

### HTTP plumbing (`platform/httpx`)

- Middleware chain shared by all three processes: request ID
  (`X-Request-ID`, generated when absent, echoed in response + logs),
  panic recovery (single 500 envelope, no stack leak, no double-write),
  request timeout (504 envelope), and per-request structured logging.
- One `statusRecorder` commits the status once; handlers never write after
  the response is committed.

### Engineering rules

- **Stdlib first:** `net/http`, `log/slog`, `encoding/json` in the hot path
  until profiling demonstrates a need; no dependency without a stated reason.
- **No domain → infrastructure imports** (ADR-0). `scripts/check-boundaries.sh`
  enforces the direction with `go list`, runs in CI, and guards against
  `internal/` imports of `cmd/*`.
- **Gates:** `gofmt`, `go vet`, `go test -race` in CI; `make format`,
  `make check-boundaries`, `make dev-*` targets for local workflow.
- **TS rules** (reflect the language gate): strict tsconfig with
  `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`,
  `verbatimModuleSyntax`; `import type` for type-only imports; no `any`, no
  non-null assertions, no `enum` (use `as const`), no `@ts-ignore`; the only
  permitted casts are `as const`/`satisfies` and the single JSON-boundary
  cast in `api-client`.

## Consequences

- A bug in one process behaves predictably in all three — one error format,
  one probe set, one log shape make cross-process debugging uniform.
- Fail fast on misconfiguration removes a whole class of half-broken
  deployments.
- The boundary script turns an architectural rule into a CI check, so drift
  is caught mechanically rather than in review.
- Cost: every new endpoint must conform (envelope, probe registration,
  strict TS), which is the intended friction.

## Alternatives considered

- **Per-process conventions** — rejected: defeats the purpose of `platform/`.
- **Shared cross-language error library** — rejected as impossible in
  practice; the JSON `ErrorEnvelope` *is* the shared format (ADR-1).
- **Frameworks/middleware libs for routing/recovery** — rejected at M1:
  stdlib covers the current surface; revisit only if handler volume or
  streaming complexity demands it.