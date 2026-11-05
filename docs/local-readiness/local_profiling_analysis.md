# Sentinel — pprof Profiling Analysis (gateway-v2)

Companion to `local_performance_report.md`. The auto-generated report records
throughput/latency; this file records the CPU / heap / goroutine profiling the
load run enabled, and the conclusion drawn from it.

**Captured:** 2026-11-03, against `gateway-v2` running natively with an isolated
Postgres (Redis degraded-local, eventbus off), driven by `cmd/perf` through
`cmd/mockupstream`. Profiles taken from the opt-in pprof listener
(`AIGW_SERVER_PPROF_ADDR=127.0.0.1:6060`).

## Heap (in-use)
Total ~13 MB, entirely **boot-time fixed buffers** — no per-request garbage:

| Site | Share |
|---|---|
| `eventbus.NewInMemory` (ring buffer) | 38.6% |
| `gatewayv2.NewTelemetryEmitter` | 25.5% |
| `runtime.mallocgc` (misc startup) | 19.9% |
| pgx stmt cache / pgtype map | ~8% |

## Goroutines
**15 total, all parked** — no leak (matches the harness's `+0` goroutine delta
over sustained traffic).

## CPU (under load, 20 s window)
Dominated by **`runtime.cgocall` (46%)** and scheduler sema ops — i.e. network
and DB **syscalls**. Application code barely registers: per-request
`sqlc.GetModelBySlug` ~10 ms and JSON parse ~10 ms out of ~540 ms total samples.

## Conclusion
The gateway request path is **I/O-bound, not CPU-bound, and already lean** —
there is no CPU or allocation hotspot to optimize. Latency (p50 ~12 ms, p95
~36–70 ms across runs) is spent in I/O, not in the gateway's own code.
Optimizing the code further is not warranted on this evidence.

If latency ever becomes a target, the profile points at I/O, not CPU: the only
per-request app DB call visible is `GetModelBySlug` (a model-metadata lookup —
a natural caching candidate), plus upstream connection keep-alive. Both are
negligible today; revisit only if a real workload shows them mattering.

## Reproduce
```
# 1. Postgres on a free port, migrate + seed (AIGW_AUTH_HMAC_SIGNING_KEY required when DB enabled)
# 2. control-api (:8081, AIGW_AUTH_CSRF_ENABLED=false for the harness) + gateway-v2 (:8084, AIGW_SERVER_PPROF_ADDR=127.0.0.1:6060)
# 3. go run ./cmd/mockupstream           # :9099 by default
# 4. DATABASE_URL=... AIGW_PERF_UPSTREAM_URL=http://localhost:9099 go run ./cmd/perf
# 5. while perf runs:
go tool pprof -top http://127.0.0.1:6060/debug/pprof/profile?seconds=20   # CPU
go tool pprof -top http://127.0.0.1:6060/debug/pprof/heap                 # heap
```
A snapshot CPU profile from this run is saved alongside as `cpu.prof`
(`go tool pprof docs/local-readiness/cpu.prof`).
