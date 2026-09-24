# Contributing

## Engineering rules

- **No "TODO" placeholders for core functionality.** If a behavior is not
  implemented, it is either implemented or deliberately out of scope — never a
  stub.
- **Never silently rewrite unrelated existing functionality.** Change only what
  the task requires.
- **No dependency without a reason.** Prefer the standard library; add a
  dependency only when it clearly pays for itself (choose libraries in
  `docs/architecture`, not ad-hoc).
- **No hardcoded provider pricing or finite model enums.** Provider models and
  pricing are data, imported and versioned — never code.
- **Provider-native on the wire.** Requests and responses pass through in the
  provider's own format (including Claude Code). The gateway never maps a
  native request into an impoverished universal schema.
- **Honest telemetry.** Only observable events are presented as observable.
  The gateway observes requests it serves — never claims to see ordinary
  browser sessions on an external provider's website.

## Backend conventions (Go)

- Module: `ai-governance-saas/backend`. Go 1.25.
- Structure: `internal/{domain,application,ports,interfaces,infrastructure,platform,bootstrap}`.
  Dependency direction is strict: HTTP/CLI → application → domain → ports ←
  infrastructure. Domain code never imports infrastructure (pgx, redis, cloud
  SDKs).
- Formatting: `gofumpt`; imports grouped stdlib/external/local (`goimports
  -local ai-governance-saas/backend`).
- Linting/verification gate (strict):
  ```bash
  gofumpt -l . && golangci-lint run ./... && nilaway ./... && \
    go test -race -shuffle=on -count=1 ./...
  ```
- Errors: wrapped with `%w`; use `errors.Is`/`errors.As`; no panics in
  library code.
- DB access via `sqlc` + `jackc/pgx/v5` only — never raw hand-rolled SQL
  marshalling in application code.
- Logging: `log/slog` (static attributes where possible).
- `context.Context` is the first parameter; no `context.Background()` inside
  leaf functions; no goroutine without context-driven shutdown.

## Tests

- Every change ships tests at the layers it touches:
  unit (fast, no external deps), integration (testcontainers: Postgres/Redis),
  contract (provider payload passthrough), and security/authorization
  (cross-tenant isolation) where applicable.
- Run with `go test -race -shuffle=on -count=1 ./...`.

## Web app conventions (Next.js)

- TypeScript strict. Do not suppress type errors with `as any` /
  `@ts-ignore`.
- Server components by default; client components only where interactivity
  requires them.
- Access the control API only through `packages/api-client` — never hand-rolled
  fetch calls in pages.

## Definition of done

A change is done when: implementation + unit tests pass; integration/contract
tests pass where touched; `go vet`/`golangci-lint`/`nilaway` and
`tsc --noEmit` are clean; documentation that describes the changed behavior is
updated; performance-sensitive hot-path changes include a benchmark.