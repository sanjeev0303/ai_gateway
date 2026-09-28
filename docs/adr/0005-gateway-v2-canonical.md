# ADR-5: gateway-v2 is the canonical data plane

- **Status:** Accepted
- **Date:** 2026-09-26
- **Deciders:** Project owner, Sisyphus (implementation)

## Context

Two gateway data planes now exist in the same binary family:

- `cmd/gateway` → `internal/interfaces/gateway` (v1)
- `cmd/gateway-v2` → `internal/interfaces/gatewayv2` (v2)

Both are wired from `internal/bootstrap/run.go` (`RunGateway` and
`RunGatewayV2`), both are real HTTP servers, and both are reachable in local
development. Nothing in `config` selects between them — they are two separate
binaries with no feature flag, so "which gateway is running" is decided by
which image a deployment happens to start.

The two implementations are not peers:

| | v1 | v2 |
|---|---|---|
| Production LOC | ~394 (`server.go`, `proxy.go`) | ~3,636 across 15 files |
| Tenant resolution | absent | `tenant_resolver.go` |
| Authentication | absent | `authenticator.go` |
| Credential resolution | absent | `credential_cache.go` |
| Policy evaluation | absent | `policy_resolver.go`, `policy_authorizer.go` |
| Rate limiting | absent | `rate_limiter.go` |
| Budget enforcement | absent | `budget_guard.go` |
| Request inspection | absent | `request_inspector.go` |
| Provider routing | absent | `provider_router.go` |
| Usage capture | absent | `usage_observer.go` |
| Telemetry | none | `telemetry_emitter.go` |
| Streaming | single path | `stream_proxy.go` |

Every governance control the product is named for — tenant isolation,
authentication, policy, budgets, rate limits, cost attribution — exists only in
v2. v1 is a transparent proxy with no controls, so running it in production
means running the product with enforcement silently absent.

v2 also now owns the two correctness properties the data plane must not
regress. Body handling is bounded and lossless: `readAndRestoreBody` and
`RequestInspectorImpl.Inspect` both cap input at 8 MiB
(`maxRequestBodyBytes`) and reject rather than truncate, and the exact bytes
plus unknown `ContentLength` are restored before forwarding. Streaming is
carried by `streamProxy.Forward` rather than a local buffer.

Current deployment reality is inconsistent, which is the actual problem this
ADR resolves:

- `deploy/compose/docker-compose.yml` ships `gateway-v2` (host `:8084`).
- The root `Makefile` dev target still runs `go run ./cmd/gateway` (v1).
- `deploy/kubernetes/` contains only `.gitkeep` — no data plane is defined yet.

Open questions: which data plane is canonical, is v1 deleted or frozen, and
what is the concrete cutover sequence for the deployments that do not yet
agree?

## Decision

1. **gateway-v2 is the only canonical data plane.** All new data-plane work
   lands in `internal/interfaces/gatewayv2`. `cmd/gateway-v2` is the canonical
   binary; `cmd/gateway` is deprecated and must not be deployed to any
   environment that serves tenant traffic.
2. **v1 is frozen, not deleted.** `internal/interfaces/gateway` and
   `cmd/gateway` stay in the tree, unchanged, for one release as a migration
   path for operators who still pin the old image. Freezing (not deleting)
   keeps a rollback available without keeping two data planes in active
   development. No new feature, bug fix, or test lands in v1.
3. **v1 is deleted once no deployment references it.** Deletion is gated on
   grepping the deploy manifests and infrastructure for `cmd/gateway` /
   `gateway` and reaching zero references — not on a calendar date. At that
   point `internal/interfaces/gateway`, `cmd/gateway`, and
   `internal/bootstrap.RunGateway` are removed in one change, together with the
   v1 test suite.
4. **The `Makefile` dev target switches to v2.** `make run-gateway` must start
   `cmd/gateway-v2` on the same port it used for v1, so local development
   exercises the canonical path. The v1 target is retained but renamed to make
   its deprecated status explicit (`run-gateway-v1`).
5. **Kubernetes ships v2 only.** When `deploy/kubernetes/` is written, the data
   plane Deployment/Service references `cmd/gateway-v2`. v1 is not offered as a
   selectable manifest variant, because a selectable runtime reintroduces the
   silent-enforcement-absent failure this ADR exists to prevent.
6. **No runtime feature flag.** A flag letting an operator silently switch
   data planes would let the unsafe path become the default through
   configuration drift. The cutover is a deployment change, which is visible in
   the image tag and reviewable.
7. **v2's bounded-body and streaming contracts are regression-locked.** The
   8 MiB cap, reject-not-truncate behavior, exact body restoration, and
   `streamProxy.Forward` streaming path are covered by
   `internal/interfaces/gatewayv2/gateway_proxy_test.go` and
   `request_inspector_test.go`. They are the properties most likely to be
   "simplified" away by a later optimization, so they are explicitly called out
   as contractual rather than incidental.

## Consequences

- One data plane is canonical, so a deployment can no longer ship without
  tenant isolation, policy, budgets, rate limits, or usage capture simply by
  picking the wrong binary. Enforcement presence becomes a property of the
  image, not a deployment decision.
- All data-plane capacity planning, latency budgets, and incident response
  target one implementation. The `TestProxyStreamingFlushesChunks` flake in the
  v1 suite (`internal/interfaces/gateway/proxy_test.go`) stops being a source of
  noise for the canonical path, which has its own stream coverage.
- The v1 code stays as ~394 lines of dead weight for one release. That is the
  deliberate cost of keeping rollback available; deletion is the follow-up
  gated on reference count, so it cannot rot into permanent duplication.
- A single gateway means the remaining v2 gaps (Redis-backed atomic rate
  limiting, event-driven policy invalidation, the async worker and ledger) are
  now on the critical path for the whole product, with no second implementation
  to fall back on. This raises the stakes on completing them rather than
  lowering it.
- Operators upgrading must repoint the data plane deployment to the v2 image.
  Because v2 enforces controls v1 did not, some traffic that previously
  succeeded anonymously or un-metered will now be rejected or billed. That is
  the intended behavior change and must be called out in release notes and
  customer-facing migration docs, not treated as a regression.

## Alternatives considered

- **Keep both data planes in active development** — rejected: v1 has no
  governance controls, so every control would have to be written twice and the
  two would inevitably diverge. That divergence is the bug class this ADR
  removes.
- **Delete v1 immediately** — rejected: removes the rollback path for operators
  mid-migration with no deprecation window. The v1 suite also still contains a
  flaky streaming test, so an immediate cut would be rushed.
- **Feature-flag the data plane at runtime** — rejected: makes "are controls
  enforced?" a configuration question that can silently drift to the unsafe
  value, and it reintroduces the need to keep v1 correct. The cutover belongs in
  a reviewed deployment change.
- **Make v1 canonical and backport controls into it** — rejected: v2 already
  implements the full control set, so this means rewriting ~3,200 lines that
  work, discarding the bounded-body and streaming fixes, and leaving the
  legacy proxy as the hot path.
