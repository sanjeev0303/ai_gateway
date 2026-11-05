# Performance Budgets & Regression Prevention

This directory holds the performance-engineering spine for AI Gateway: the
budgets, the committed benchmark baseline, and the gate that enforces them.
It is built **around the tools the repo already has** rather than adding new
infrastructure.

## What enforces what

| Layer | Tool | When |
|---|---|---|
| Micro-benchmark regressions (CPU/alloc on hot-path functions) | **`cmd/perfgate`** (new) vs `perf/baseline.json` | Every PR (`.github/workflows/ci.yml` → `perf-gate`) |
| End-to-end API / resource / DB budgets (throughput, latency, memory growth, goroutine leaks, pool, dedup, cancellation) | **`cmd/perf`** (existing load + resilience harness) → `docs/local-readiness/local_performance_report.md` | Locally / scheduled tier, against `cmd/mockupstream` |
| CPU / heap / goroutine profiling | **pprof** via `AIGW_SERVER_PPROF_ADDR` (opt-in, off by default) | On demand under load |

Nothing new was introduced for load generation (reuse `cmd/perf`), profiling
(reuse the pprof endpoint), or metrics. Prometheus/Grafana/OpenTelemetry/k6/
Lighthouse were **deliberately not added** — none exist in the repo today, and
adding them would be the duplication and dependency bloat the brief warns
against. Add them only if a running-service monitoring need appears.

## Budgets — `budgets.json`

Central, environment-specific thresholds. The API/resource/DB sections document
the budgets `cmd/perf` already checks; the `regression` section is consumed by
`cmd/perfgate`. Values are calibrated from the measured local baseline
(`docs/local-readiness/`), not guessed.

### Regression policy and the noise problem
ns/op on a shared developer or CI machine varies 20–40% run-to-run (observed
directly: one gate run measured a benchmark 22% *faster* than baseline). So:

- **allocs/op and B/op are the hard gate** — they are deterministic (identical
  every run for the same code and input) and portable across machines, so an
  allocation regression is caught precisely with a strict threshold.
- **ns/op is advisory** — reported and flagged `warn` when it exceeds the
  latency budget, but it never fails the build. Benchmarks below `noise_floor_ns`
  are not latency-flagged at all.

This is the statistical safeguard against false positives: an algorithmic
regression almost always shows up as more allocations or bytes; a pure
constant-factor CPU regression is noise-masked across runs anyway and needs a
dedicated quiet runner to measure reliably.

## Baseline — `baseline.json`

Committed median benchmark results plus environment metadata (git commit/branch
when available, Go version, GOOS/GOARCH, CPU count, host, timestamp). It is
**environment-specific** — regenerate it on a representative runner when the
benchmark set or the environment materially changes:

```
cd backend && go run ./cmd/perfgate -update
```

Keep materially different environments on separate baselines; do not compare
across incompatible hardware/runtime as if equivalent.

## Running

```
cd backend

# Micro-benchmark gate (what CI runs):
go run ./cmd/perfgate                 # PASS/FAIL vs perf/baseline.json, writes perf/last_report.json
go run ./cmd/perfgate -update         # (re)record the baseline
go run ./cmd/perfgate -bench <regex> -pkg <path> -count <n>   # widen/adjust the suite

# End-to-end load + resilience (needs the stack + mock upstream):
go run ./cmd/mockupstream &           # stand-in provider on :9099
DATABASE_URL=... AIGW_PERF_UPSTREAM_URL=http://localhost:9099 go run ./cmd/perf

# Profiling under load (gateway-v2 started with AIGW_SERVER_PPROF_ADDR=127.0.0.1:6060):
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=20   # CPU
go tool pprof http://127.0.0.1:6060/debug/pprof/heap                 # heap
```

## SLO vs budget vs error budget (kept distinct)

- **Performance budget** — the maximum acceptable latency / resource use for a
  workload (what this directory enforces).
- **SLO** — a reliability/performance objective over a window (e.g. 99.9%
  availability). Not yet tracked; needs the running-service monitoring layer.
- **Error budget** — the permitted unreliability within the SLO window. Follows
  from an SLO; out of scope until SLOs are tracked.

## Adding benchmarks

Add a `Benchmark*` with `b.ReportAllocs()` to the relevant package, then
`go run ./cmd/perfgate -bench <NewName> -update` to fold it into the baseline.
Prefer deterministic, DB-free, non-accumulating benchmarks so the gate stays
low-noise.

## Current limitations

- The suite starts with the request-body parse hot path (`promptTurn`); extend
  it as other hot paths are identified from profiles.
- Reliable ns/op gating needs a dedicated quiet runner; today ns/op is advisory.
- SLO / error-budget tracking and dashboards require a running-service
  monitoring stack that the repo does not have yet.
