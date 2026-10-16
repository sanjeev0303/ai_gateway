# Sentinel Local Performance & Resilience Validation Report (Phase 16)

**Execution Date:** 2026-10-15 21:25:53 UTC  
**Test Environment:** Linux / 12 CPU Cores / 7.5 GiB RAM (Local Docker Compose stack)

---

## 1. Local Acceptance Thresholds & Measured Baseline

| Metric | Acceptance Threshold | Measured Value | Result |
|---|---|---|---|
| **Gateway Throughput** | >= 10.0 req/s (local mock proxy) | **367.20 req/s** | ✅ PASS |
| **Gateway Latency (p50)** | <= 250 ms | **9ms** | ✅ PASS |
| **Gateway Latency (p95)** | <= 1500 ms | **82ms** | ✅ PASS |
| **Gateway Latency (p99)** | <= 2500 ms | **114ms** | ✅ PASS |
| **Streaming TTFT (p50)** | <= 500 ms | **10ms** | ✅ PASS |
| **Streaming TTFT (p95)** | <= 1000 ms | **97ms** | ✅ PASS |
| **Memory Growth (Sustained)** | < 30.0 MB heap delta | **-0.13 MB** | ✅ PASS |
| **Goroutine Leak Delta** | <= 5 goroutines | **+0** | ✅ PASS |
| **Database Connections** | <= 50 concurrent | **29** | ✅ PASS |
| **Duplicate Usage Events** | Strictly 0 | **0** | ✅ PASS |
| **SSR Page Latency** | <= 600 ms | **< 250 ms** | ✅ PASS |

---

## 2. Workload Breakdown (8 Concurrent Workers, 80 Mixed Requests)
- **Total Requests Dispatched:** 80
- **Successful Proxied Requests:** 64
- **Governance / Error Handled Requests:** 16
- **Failed Requests:** 0
- **Total Duration:** 218ms
- **Latency Distribution:**
  - Min: 5ms
  - p50: 9ms
  - p90: 35ms
  - p95: 82ms
  - p99: 114ms
  - Max: 114ms
  - Mean: 20ms

---

## 3. Resilience, Fault Tolerance & Gate Audit
- **Unbounded Queues:** None. Event bus and worker channels operate with bounded ring buffers.
- **Goroutine Leaks:** Verified zero leaks after sustained traffic (post-GC goroutine count stable).
- **Database Connection Pool:** Connection pool active/idle states remain within pgxpool limits.
- **N+1 Queries:** Telemetry aggregation queries utilize pre-computed rollups (usage_hourly, usage_daily).
- **Blocking Writes on Data Plane:** Asynchronous telemetry publishing offloaded to background event bus.
- **Duplicate Usage Events:** Zero duplicate request IDs recorded in llm_requests ledger.
- **Request Cancellation:** Client context deadlines abort immediately without hanging data plane goroutines.
