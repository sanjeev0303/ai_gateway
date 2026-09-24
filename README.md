# AI Gateway — Governance for Enterprise AI Usage

A multi-tenant AI gateway and governance control plane. Teams route their AI
traffic through one gateway, while a control plane governs who can call which
provider/models, enforces budgets and rate limits, and records audited usage.

## Why it exists

- **Route and govern**: organizations, teams, projects, and users get scoped
  access to AI providers (Anthropic, OpenAI, Gemini, …) with API credentials.
- **Policy in the hot path**: authorization, rate limits, and budget checks run
  at request time, backed by compiled, immutable policy snapshots.
- **Native provider compatibility**: provider payloads pass through
  provider-native (including Claude Code passthrough) — the gateway does not
  force a universal schema on the wire.
- **Audit-grade usage**: every request yields normalized usage
  (input/output/cached/thinking/tool tokens + `provider_usage_json`) with
  configurable retention and privacy modes.
- **Self-serve control plane**: a web app and REST API to manage tenants,
  policies, credentials, audit, and billing.

## Architecture at a glance

Three Go processes plus a web app, in one monorepo:

| Process | Role |
|---|---|
| `backend/cmd/gateway` | Data plane: the hot-path proxy (auth → policy snapshot → rate/budget → forward → stream → async usage) |
| `backend/cmd/control-api` | Control plane API: tenants, policies, credentials, audit, billing |
| `backend/cmd/worker` | Async: usage rollups, retention, alerts |
| `apps/web` | Next.js dashboard over the control API |

Design decisions live in [`docs/adr`](docs/adr); the roadmap is in
[`docs/plan/milestones.md`](docs/plan/milestones.md).

## Repository layout

```
backend/    Go module: cmd/{gateway,control-api,worker} + internal/ (domain, application, ports, interfaces, infrastructure, platform, bootstrap)
apps/web    Next.js web app
packages/   Shared TS: api-client, contracts, cli
deploy/     Docker, docker-compose, Kubernetes, Terraform (AWS)
docs/       Architecture, ADRs, API, security, operations, runbooks, plan
scripts/    Dev and ops helpers
```

## Status

Scaffolded (M0). See [`docs/plan/milestones.md`](docs/plan/milestones.md) for
the phased implementation plan and current milestone.

## Development

Requires Go 1.25+, Node 20+ (for the web app), Docker (for local Postgres/Redis).

```bash
make dev-up      # start local Postgres + Redis
make build       # build all Go binaries
make test        # run Go tests with race detector
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for engineering rules.