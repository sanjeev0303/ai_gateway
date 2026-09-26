# ADR-1: Hybrid architecture — two runtimes, one wire contract

- **Status:** Accepted (M1)
- **Date:** 2026-09-25
- **Deciders:** Project owner, Sisyphus (implementation)

## Context

ADR-0 settled the monorepo and the process split (gateway / control-api /
worker + web app), but not the *hybrid* boundary: how do two runtimes — Go
for every backend process, TypeScript/Next.js for the web app — share wire
types without drift? The spec mandates a Go backend using `net/http`
(stdlib), a Next.js web app, and a shared contracts package; it does not
mandate a framework, a code generator, or a runtime validation library.

Open questions:

- What is the single source of truth for request/response shapes?
- How do TS types get derived from it — generate, duplicate by hand, or
  import at runtime?
- Does the Go side also generate code from the spec, or implement by hand?
- May the web app consume generated types directly, or only through a client
  package?

## Decision

1. **One contract, two implementations.** `backend/api/openapi/openapi.yaml`
   (OpenAPI 3.0) is the single source of truth for every wire type. It grows
   milestone by milestone with the endpoints it actually ships — never
   speculatively.
2. **TS types are generated, never hand-written.** `packages/contracts`
   runs `openapi-typescript` over the spec (`make generate`) and emits raw
   TypeScript (`src/generated.ts`). The package ships source, no build step;
   consumers typecheck it through the npm-workspace symlink.
3. **The Go backend implements endpoints by hand with the stdlib.**
   No OpenAPI code generator on the Go side. Handlers stay explicit, the
   schema is small and stable, and the stdlib mandate (ADR-0) is preserved.
   JSON field validation is manual per handler, with the contract documenting
   the invariants.
4. **The web app never imports `@ai-governance/contracts` directly.**
   All network access goes through `packages/api-client`, which depends on
   the contracts package and owns parsing, the single permitted
   JSON-boundary cast, and error surfacing. This keeps client design changes
   out of the app and gives one place to harden transport behavior.
5. **Cross-runtime errors are contractual.** The `ErrorEnvelope` schema in
   the OpenAPI spec is the bridge between Go's `Kind*` error taxonomy and the
   TS client — a stable JSON shape beats any attempt to share a typed error
   library across languages.

## Consequences

- Drift between Go behavior and TS types is caught by contract review; the
  generated file is cheap to regenerate (`npm run generate` / `make generate`)
  and its diff is reviewable in CI.
- The raw-TS workspace pattern removes build orchestration for packages but
  requires every consumer to run strict `tsc` themselves (enforced by the
  root `typecheck` script and per-package tsconfigs).
- Hand-written Go handlers mean the spec and the handlers can disagree if
  review misses it; mitigated by keeping the spec minimal and endpoint
  additions deliberate.
- The api-client indirection costs one extra hop of indirection and buys one
  surface for transport/error policy across the whole web app.

## Alternatives considered

- **Hand-written TS mirror types** — rejected: silent drift is guaranteed as
  endpoints accumulate.
- **Runtime schema validation in TS** (zod/io-ts from spec) — deferred:
  adds a dependency and a validation layer before the client has real volume;
  revisit when the first business endpoint lands.
- **Generated Go server (oapi-codegen)** — rejected: diverges from the
  stdlib-first convention and couples handler shape to a generator.
- **Web app importing contracts directly** — rejected: spreads parsing and
  cast policy across the app and couples pages to generated type names.