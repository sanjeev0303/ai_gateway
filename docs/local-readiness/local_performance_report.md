# Sentinel Local Performance & Resilience Validation Report (Phase 16)

**Execution Date:** 2026-10-18 01:52:24 UTC  
**Test Environment:** Linux / 12 CPU Cores / 7.5 GiB RAM (Local Docker Compose stack)

---

## 1. Local Acceptance Thresholds & Measured Baseline

| Metric | Acceptance Threshold | Measured Value | Result |
|---|---|---|---|
| **Gateway Throughput** | >= 10.0 req/s (local mock proxy) | **419.18 req/s** | ✅ PASS |
| **Gateway Latency (p50)** | <= 250 ms | **11ms** | ✅ PASS |
| **Gateway Latency (p95)** | <= 1500 ms | **54ms** | ✅ PASS |
| **Gateway Latency (p99)** | <= 2500 ms | **68ms** | ✅ PASS |
| **Streaming TTFT (p50)** | <= 500 ms | **10ms** | ✅ PASS |
| **Streaming TTFT (p95)** | <= 1000 ms | **50ms** | ✅ PASS |
| **Memory Growth (Sustained)** | < 30.0 MB heap delta | **-0.14 MB** | ✅ PASS |
| **Goroutine Leak Delta** | <= 5 goroutines | **+0** | ✅ PASS |
| **Database Connections** | <= 50 concurrent | **36** | ✅ PASS |
| **Duplicate Usage Events** | Strictly 0 | **0** | ✅ PASS |
| **SSR Page Latency** | <= 600 ms | **< 250 ms** | ✅ PASS |

---

## 2. Workload Breakdown (8 Concurrent Workers, 80 Mixed Requests)
- **Total Requests Dispatched:** 80
- **Successful Proxied Requests:** 64
- **Governance / Error Handled Requests:** 16
- **Failed Requests:** 0
- **Total Duration:** 191ms
- **Latency Distribution:**
  - Min: 5ms
  - p50: 11ms
  - p90: 29ms
  - p95: 54ms
  - p99: 68ms
  - Max: 68ms
  - Mean: 17ms

---

## 3. Resilience, Fault Tolerance & Gate Audit
- **Unbounded Queues:** None. Event bus and worker channels operate with bounded ring buffers.
- **Goroutine Leaks:** Verified zero leaks after sustained traffic (post-GC goroutine count stable).
- **Database Connection Pool:** Connection pool active/idle states remain within pgxpool limits.
- **N+1 Queries:** Telemetry aggregation queries utilize pre-computed rollups (usage_hourly, usage_daily).
- **Blocking Writes on Data Plane:** Asynchronous telemetry publishing offloaded to background event bus.
- **Duplicate Usage Events:** Zero duplicate request IDs recorded in llm_requests ledger.
- **Request Cancellation:** Client context deadlines abort immediately without hanging data plane goroutines.
