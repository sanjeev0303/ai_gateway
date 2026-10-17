# Usage Metering, Event Processing, and Analytics Reliability Report

This document records the verification and hardening of Sentinel's event bus, worker pipelines, persistence layer, usage metering, cost calculation, and analytics aggregation paths.

---

## 1. Verified Architecture & Pipeline Overview

```
                      Sentinel Gateway-v2 (:8084)
                                   │
              ┌────────────────────┴────────────────────┐
              ▼                                         ▼
   Active Requests Cache                       Telemetry Emitter
   (Redis: tenant:active_requests)             (Buffered Queue + Spool)
   - 10-minute automatic pruning                        │
   - Sub-millisecond latency                            ▼
                                                Event Bus Broker
                                      (NATS JetStream / In-Memory Channel)
                                       - Stream: LLM_EVENTS
                                       - Subject: llm.request.completed
                                       - Consumer Group: worker-usage-group
                                       - AckPolicy: Explicit, MaxDeliver: 5
                                                        │
                                                        ▼
                                                Usage Consumer Worker
                                             (ports.UsageEventConsumer)
                                                        │
                                                        ▼
                                              usage.Service.RecordRequest
                                       (Idempotent (org_id, request_id))
                                                        │
                                                        ▼
                                             PostgreSQL Database
                                   ┌────────────────────┴────────────────────┐
                                   ▼                                         ▼
                             llm_requests                              usage_events
                      (Canonical Status Check)                  (Unique Event Attribution)
                                   │
                                   ▼
                       Analytics Aggregation APIs (:8081)
                   /v1/usage/summary, timeseries, by-*
                   - Exact mathematical agreement
                   - Strict date boundary validation (from <= to)
                   - Guaranteed empty arrays [] (never null)
```

---

## 2. Event Bus & Delivery Guarantees

| Property | Implementation | Guarantee / Behavior |
| :--- | :--- | :--- |
| **Broker Modes** | NATS JetStream (Production/Compose) & Memory Event Bus (Test/Fallback) | Standardized over `ports.EventBus` interface. |
| **Stream Definition** | `LLM_EVENTS` stream bound to `llm.request.*` subjects | Durable retention across worker restarts. |
| **Consumer Group** | Durable consumer group `worker-usage-group` with explicit acknowledgement (`AckPolicyExplicit`) | Unacknowledged messages are redelivered upon worker crashes or timeout. |
| **Redelivery Semantics** | `MaxDeliver: 5` with backoff; `ports.ErrNonRetryable` immediately terminates delivery | Transient database failures trigger backoff; unrecoverable schema errors do not poison the queue. |
| **Publisher Spooling** | Gateway `TelemetryEmitterImpl` local spool channel | If the event bus is temporarily unavailable, messages are spooled locally in memory and flushed once connectivity is restored. |

---

## 3. Idempotency & Deduplication Guarantees

Every accepted LLM request has a stable `request_id` originating from client headers (`x-request-id` or `x-correlation-id`) or generated via UUIDv4 at the gateway boundary.

1. **Database Constraints**:
   - `llm_requests`: Unique constraint on `(organization_id, request_id)`
   - `usage_events`: Unique index on `(organization_id, request_id, event_type)` with `ON CONFLICT DO NOTHING`
2. **Repository Conflict Handling**:
   - PostgreSQL `ON CONFLICT DO NOTHING` returns 0 rows affected, mapped via `mapConflict` to `repositories.ErrConflict`.
3. **Application Layer Absorption**:
   - `usage.Service.RecordRequest` explicitly catches `repositories.ErrConflict` and returns `nil`, treating redeliveries as idempotent no-ops.
   - Replaying the same event 100 times results in exactly 1 record persisted and 0 double-counted tokens or cost.

---

## 4. Usage Metering, Cost Calculation & Historical Explainability

### Token Capture Provenance
- **Reported vs Estimated**: Gateway inspects provider metadata (`tokens_reported` or `estimated` flags). Requests with exact provider tokens are tagged `usage_source = "provider_reported"`; requests using the token heuristic are tagged `usage_source = "estimated"`.
- **Cost Status**:
  - Exact token calculations: `cost_status = "calculated"`
  - Estimated token calculations: `cost_status = "estimated"`
  - Failed upstream calls: `cost_status = "unavailable"` with `cost_micros = 0`

### Edge-Case Request Handling
| Scenario | Status Code | Canonical Status | Billable Cost | Provenance Metadata |
| :--- | :--- | :--- | :--- | :--- |
| **Completed Stream / Request** | `200 OK` | `succeeded` | Full calculated cost | `provider_reported` or `estimated` |
| **Client Disconnect** | `499 Client Closed Request` | `cancelled` | Calculated tokens to point of disconnect | `usage_source` stamped with streamed count |
| **Stream Truncation / Upstream Error**| `502 Bad Gateway` / `504 Timeout` | `failed` | `0` (`cost_status: "unavailable"`) | Non-billable failure |
| **Disallowed Policy Denial** | `403 Forbidden` | `failed` | `0` (`cost_status: "unavailable"`) | `policy_decision: "denied"` |

### Historical Cost Snapshotting & Backfill
- Cost is evaluated using effective pricing versions active at `created_at`.
- Historical reconciliation query (`BackfillLLMRequestCost`) joins `pricing_versions` matching `effective_from <= r.created_at`, updates `cost_micros`, locks `pricing_version`, and transitions `cost_status` from `estimated` or `unavailable` to `calculated`.

---

## 5. Analytics Aggregation & API Safety

1. **Mathematical Agreement**:
   - All analytics endpoints (`/v1/usage/summary`, `/v1/usage/by-user`, `/v1/usage/by-project`, `/v1/usage/by-model`, `/v1/usage/timeseries`) query from the authoritative `llm_requests` table.
   - Summing all `llm_requests` rows matches `GetUsageSummary` and the sum of `GetByUser` with zero discrepancy.
2. **Date Boundaries & Timezones**:
   - Timestamps are normalized to UTC `timestamptz`.
   - `parseUsageFilter` validates `from.After(to)` and rejects inverted time ranges with `HTTP 400 Bad Request` (`invalid_argument`).
3. **Empty State Serialization**:
   - Operations repositories (`GetByProject`, `GetByModel`, `GetByProvider`, `GetByUser`, `GetByTeam`, `GetByServiceAccount`, `GetLatencyByProviderModel`) initialize empty slices (`make(..., 0, len(rows))`) ensuring empty results serialize as `[]` instead of JSON `null`.
4. **Redis Cache Safety**:
   - Active requests in Redis (`tenant:active_requests`) are scoped by organization ID.
   - `ActiveRequests` automatically scans and prunes orphaned entries older than 10 minutes to protect against worker or gateway crashes.
   - Heavy aggregation queries remain strictly out of the gateway request hot path.

---

## 6. Automated Reliability & Regression Test Suite

| Test Suite | File | Scenarios Covered | Status |
| :--- | :--- | :--- | :--- |
| **Idempotency & Deduplication** | `event_reliability_test.go` | Duplicate event redelivery on same `request_id`, 100 replays produce 0 duplicate records. | `PASS` |
| **Worker Restarts & Crashes** | `event_reliability_test.go` | Worker termination midway through batch, recreation of consumer group, resumes backlog without loss. | `PASS` |
| **Event Bus Backlog Recovery** | `event_reliability_test.go` | Ingestion of 25 spooled events into consumer, batch processing completes with 100% accounting. | `PASS` |
| **Missing Token Metadata** | `event_reliability_test.go` | Upstream provider returns 0 tokens; honest fallback tagging as `estimated` without crashing worker. | `PASS` |
| **Stream Interruption & Abort** | `event_reliability_test.go` | Client abort (HTTP 499) recorded as `cancelled`, tokens captured up to disconnect, zero leakage. | `PASS` |
| **Failed Provider Calls** | `event_reliability_test.go` | Upstream timeout (HTTP 504) recorded as `failed`, billable cost stamped as 0 (`unavailable`). | `PASS` |
| **Multi-User Attribution** | `event_reliability_test.go` | 3 independent users issue concurrent traffic; requests strictly attributed to correct user/credential. | `PASS` |
| **Mathematical Reconciliation**| `event_reliability_test.go` | Sum of individual request tokens and costs reconciles exactly to org summary totals (Discrepancy: 0). | `PASS` |
| **Gateway Telemetry Spooling** | `event_reliability_test.go` | Transient NATS failure triggers in-memory spooling and recovery once broker is restored. | `PASS` |
| **Usage API Authentication** | `usage_test.go` | All 11 `/v1/usage/*` endpoints strictly require valid bearer tokens (HTTP 401 on unauthenticated). | `PASS` |
| **Date Boundary Validation** | `usage_test.go` | Inverted date filter (`from > to`) returns HTTP 400 `invalid_argument`. | `PASS` |
| **Empty State JSON Arrays** | `usage_test.go` | Endpoints with no records return `[]` instead of `null` for robust frontend rendering. | `PASS` |
| **Redis Orphan Pruning** | `service_test.go` | Active requests cache automatically prunes entries older than 10 minutes. | `PASS` |
| **Gateway Status Alignment** | `usage_capture_test.go` | Telemetry emitter emits canonical statuses (`succeeded`, `failed`, `cancelled`) matching DB constraints. | `PASS` |
