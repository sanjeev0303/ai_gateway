# ADR-0: Monorepo with four bounded execution processes

- **Status:** Accepted (M0)
- **Date:** 2026-09-25
- **Deciders:** Project owner, Sisyphus (implementation)

## Context

The product is a multi-tenant AI gateway plus a governance control plane. It
must:

- Route provider-native AI traffic (Anthropic, OpenAI, Gemini, …) on a hot
  path: authenticate → resolve a cached, compiled policy snapshot →
  authorize model/credential → enforce rate/budget → forward → stream →
  record usage asynchronously.
- Expose a control plane API + web app for tenants: organizations, teams,
  projects, users, credentials, policies, audit, billing.
- Run async work: usage rollups, retention, alerts.
- Pass through Claude Code (and other provider-native clients) without
  reshaping payloads into a universal schema.

The spec mandates: one repository, a Go backend, a Next.js web app, strict
clean-architecture dependency direction, `net/http` (stdlib), OpenTelemetry +
Prometheus, SQLite-free Postgres persistence, Docker Compose for local dev
(managed infra in prod), and "no Kafka unless justified".

## Decision

A **single monorepo** (`ai-governance-saas/`) containing **four bounded
execution processes**, plus a web app:

| Process | Binary / app | Responsibility |
|---|---|---|
| Gateway (data plane) | `backend/cmd/gateway` | Hot path only: authn, cached policy snapshot, model authz, rate/budget, forwarding, streaming, async usage events |
| Control API | `backend/cmd/control-api` | Control plane REST API: tenants, policies, credentials, audit, billing. The only writer of governance state |
| Worker | `backend/cmd/worker` | Async: consume usage events, rollups, retention, alerts |
| Web app | `apps/web` (Next.js) | Dashboard over the control API, via `packages/api-client` |

### Backend structure (strict dependency direction)

```
interfaces/  (HTTP handlers, CLI)  → application/  →  domain/  →  ports/ ← infrastructure/
```

- HTTP/CLI → application → domain → ports ← infrastructure. Domain must never
  import infrastructure (pgx, Redis, cloud SDKs).
- `internal/bootstrap/` wires configuration + dependencies for each `cmd/`
  binary. Each command is thin: read config, assemble, run.
- `domain/` holds entities and rule logic shared by all three processes;
  `ports/` are the interfaces infrastructure satisfies (repositories,
  providers, cache, eventbus, secrets, identity, billing, notifications).

### Communication boundaries

- **Control API is the single writer** of governance state (Postgres).
- **Gateway never queries Postgres on the hot path.** It consumes compiled,
  immutable policy snapshots (org → team → project → user/service-account →
  credential) served from an in-memory cache (Redis-backed at startup),
  invalidated by the control API via event.
- **Usage recording is asynchronous.** The gateway emits a normalized usage
  event (input/output/cached/thinking/tool/total + `provider_usage_json`)
  onto an internal queue; the worker persists and rolls it up. The hot path
  never waits on usage persistence.
- **Provider-native payloads on the wire** (including Claude Code passthrough).
  Normalization to `provider_usage_json` happens after the request, in the
  worker/application layer — never by rewriting the request.

### Infrastructure choices (M0 anchors)

- **Postgres** (primary store) + **Redis** (policy snapshot cache, rate-limit
  counters) via Docker Compose locally; managed equivalents in prod.
- **Eventbus via port** — in-process/Postgres-backed queue first; a
  NATS/queue-backed implementation is behind the same `ports/eventbus`
  interface and introduced only when volume justifies it.
- `net/http` stdlib, `log/slog`, OpenTelemetry SDK (traces + metrics),
  Prometheus scrape endpoint per process.
- sqlc + pgx for DB access, generated in `infrastructure/db/`.

## Consequences

- One language for all backend processes (Go) minimizes cross-process
  convention drift; strict dependency direction keeps them *bounded* despite
  sharing one module.
- The gateway hot path stays lean by construction: no DB, no billing SDK, no
  web framework dependencies in its request path.
- Added processes/directories later are cheap, but every addition must respect
  the dependency direction above (enforced by review + `go list` checks in CI
  eventually).
- Trade-off: single-module coupling means a domain change can touch all three
  binaries — acceptable at this scale, and kept safe because domain changes
  are the intended shared surface.

## Alternatives considered

- **Microservices per process** — rejected: premature operational overhead
  (N+1 deploys, version skew, distributed debugging) with no scaling need.
- **Shared library repo + app repos** — rejected: cross-cutting domain changes
  would force coordinated multi-repo releases.
- **Gateway-owned DB access** — rejected: violates the hot-path rule and the
  single-writer control plane requirement.