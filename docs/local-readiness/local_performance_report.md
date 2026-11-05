# Sentinel Local Performance & Resilience Validation Report (Phase 16)

**Execution Date:** 2026-11-03 13:01:32 UTC  
**Test Environment:** Linux / 12 CPU Cores / 7.5 GiB RAM (Local Docker Compose stack)

---

## 1. Local Acceptance Thresholds & Measured Baseline

| Metric | Acceptance Threshold | Measured Value | Result |
|---|---|---|---|
| **Gateway Throughput** | >= 10.0 req/s (local mock proxy) | **409.83 req/s** | ✅ PASS |
| **Gateway Latency (p50)** | <= 250 ms | **14ms** | ✅ PASS |
| **Gateway Latency (p95)** | <= 1500 ms | **40ms** | ✅ PASS |
| **Gateway Latency (p99)** | <= 2500 ms | **59ms** | ✅ PASS |
| **Streaming TTFT (p50)** | <= 500 ms | **13ms** | ✅ PASS |
| **Streaming TTFT (p95)** | <= 1000 ms | **43ms** | ✅ PASS |
| **Memory Growth (Sustained)** | < 30.0 MB heap delta | **-0.11 MB** | ✅ PASS |
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
- **Total Duration:** 195ms
- **Latency Distribution:**
  - Min: 8ms
  - p50: 14ms
  - p90: 31ms
  - p95: 40ms
  - p99: 59ms
  - Max: 59ms
  - Mean: 18ms

---

## 3. Resilience, Fault Tolerance & Gate Audit
- **Unbounded Queues:** None. Event bus and worker channels operate with bounded ring buffers.
- **Goroutine Leaks:** Verified zero leaks after sustained traffic (post-GC goroutine count stable).
- **Database Connection Pool:** Connection pool active/idle states remain within pgxpool limits.
- **N+1 Queries:** Telemetry aggregation queries utilize pre-computed rollups (usage_hourly, usage_daily).
- **Blocking Writes on Data Plane:** Asynchronous telemetry publishing offloaded to background event bus.
- **Duplicate Usage Events:** Zero duplicate request IDs recorded in llm_requests ledger.
- **Request Cancellation:** Client context deadlines abort immediately without hanging data plane goroutines.
