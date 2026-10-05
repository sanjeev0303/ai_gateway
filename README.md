# AI Gateway — Governance for Enterprise AI Usage

A multi-tenant AI gateway and governance control plane. Teams route their AI
traffic through one gateway, while a control plane governs who can call which
provider/models and enforces budgets and rate limits in the request path.

## Why it exists

- **Route and govern**: organizations, teams, projects, and users get scoped
  access to AI providers (Anthropic, OpenAI, Gemini, …) with API credentials.
- **Policy in the hot path**: authorization, rate limits, and budget checks run
  at request time, backed by compiled, immutable policy snapshots.
- **Native provider compatibility**: provider payloads pass through
  provider-native (including Claude Code passthrough) — the gateway does not
  force a universal schema on the wire.
- **Usage accounting, schema first**: `usage_events` carries a unit, a quantity,
  and a unique key on `(organization_id, request_id, event_type)` that makes
  ingest idempotent, plus hourly/daily/monthly rollup tables. `gateway-v2`
  parses provider-reported usage (Anthropic, OpenAI, Gemini) and persists
  `llm_requests` + `usage_events` off the request path; re-ingesting a request
  is idempotent. The async worker prices `llm_requests` from the price sheet
  effective at request time, then rolls it up into hourly/daily/monthly buckets.
  Not wired up yet: `gateway` (v1) still writes no usage events, there is no
  control API for price sheets and no invoice settlement, and retention windows
  and privacy modes are not implemented.
- **Self-serve control plane**: a web app and REST API to manage tenants,
  policies, credentials, audit, and billing.

## Architecture at a glance

Six Go processes plus a web app, in one monorepo:

| Process | Role |
|---|---|
| `backend/cmd/gateway-v2` | **Canonical data plane**: the hot-path proxy (auth → policy snapshot → rate/budget → forward → stream) + usage accounting |
| `backend/cmd/control-api` | Control plane API: tenants, providers, credentials, audit, billing |
| `backend/cmd/worker` | Async worker: usage rollups, plus health probes |
| `backend/cmd/gateway` | Data plane v1 — **frozen and not deployed** ([ADR-5](docs/adr/0005-gateway-v2-canonical.md)); kept one release as a rollback path |
| `backend/cmd/migrate` | One-shot: applies DB migrations |
| `backend/cmd/seed` | One-shot: applies migrations + seeds demo data |
| `app/` | Next.js dashboard over the control API |

Design decisions live in [`docs/adr`](docs/adr); the roadmap is in
[`docs/plan/milestones.md`](docs/plan/milestones.md).

## Repository layout

```
backend/    Go module: cmd/{gateway,gateway-v2,control-api,worker,migrate,seed} + internal/ (domain, application, ports, interfaces, infrastructure, platform, bootstrap)
app/        Next.js dashboard (app router)
packages/   Shared TS: api-client, contracts, cli
deploy/     Docker, docker-compose, Kubernetes, Terraform (AWS)
docs/       Architecture, ADRs, API, security, operations, runbooks, plan
scripts/    Dev and ops helpers
```

## Status

Platform foundations, persistence, identity/tenants, the provider layer, and the
gateway-v2 hot path are implemented and covered by a green `-race` suite.

The gateway-v2 hot path enforces policy, rate limits, and per-organization budget
caps on every request from compiled, immutable policy snapshots. What is missing is
the control plane that would author those rules: no routes manage policy rules,
price sheets, or budget caps, and nothing settles an invoice, so derived cost
cannot be configured or billed through the product yet.

The OpenAPI spec now documents all 83 implemented `/v1` operations across 69
paths, plus the 3 health endpoints, in parity with the registered control API.
The remaining gap is schema depth, not coverage: `packages/contracts` exports 5
domain types for those 83 operations.

[`docs/architecture/completeness-matrix.md`](docs/architecture/completeness-matrix.md)
is the item-by-item inventory with evidence. See
[`docs/plan/milestones.md`](docs/plan/milestones.md) for the phased plan.

## Development

Requires Go 1.25+, Node 20+ (for the web app), Docker (for local Postgres/Redis).

### Run the full stack (docker compose)

The compose file in [`deploy/compose`](deploy/compose) runs everything: the DB,
Redis, NATS, migrations, all six backend processes, and the web dashboard.

```bash
cp deploy/compose/.env.example deploy/compose/.env   # adjust secrets if desired
make dev-up                                          # build + start the whole stack
make dev-seed                                        # optional: load demo data
make dev-down                                        # stop everything
make dev-logs                                       # tail all service logs
```

`make dev-up` also runs the one-shot `migrate` service for you (it exits after
applying migrations; the backend services wait for it via
`depends_on: service_completed_successfully`). `seed` is behind the `seed`
profile, so it only runs when you explicitly ask (`make dev-seed`).

| Service | Host port | Notes |
|---|---|---|
| `postgres` | 5432 | control-plane DB (`ai_gateway`) |
| `redis` | 6379 | reserved; consume once the Redis milestone lands |
| `nats` | 4222 / 8222 | event bus; reserve until the eventbus milestone lands |
| `gateway` | 8080 | data plane v1 — **not started by default** (profile `legacy-v1`; [ADR-5](docs/adr/0005-gateway-v2-canonical.md)) |
| `gateway-v2` | 8084 | **canonical data plane** — the provider gateway |
| `control-api` | 8081 | control plane API |
| `worker` | 8082 | async worker |
| `web` | 3000 | Next.js dashboard |

Context: `deploy/compose/.env` holds the compose-level credentials
(POSTGRES_*, REDIS_PASSWORD, AUTH_HMAC_SIGNING_KEY). The backend processes
receive them as `AIGW_*` env vars via the compose `x-backend-env` anchor — you
normally never edit per-service env by hand.

Health probes: every backend process exposes `/livez` and `/readyz` (plus
`/healthz`); compose uses them for dependency ordering and restart checks.

Usage guide:

- Dashboard: http://localhost:3000 (web talks to `control-api` on :8081).
- Control plane API: http://localhost:8081 (OpenAPI under `/docs`; dev login
  enabled for local use).
- Provider gateway: http://localhost:8084 (`gateway-v2`; route provider traffic through it).
- Remote envs / local runs can read [`backend/.env.example`](backend/.env.example)
  and export the same `AIGW_*` variables.

### Without Docker (backend only)

```bash
make build       # build all Go binaries
make test        # run Go tests with race detector
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for engineering rules.