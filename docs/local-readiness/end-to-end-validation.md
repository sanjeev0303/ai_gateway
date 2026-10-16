# Sentinel AI Gateway — End-to-End Acceptance Journey Validation

**Date**: 2026-10-15  
**Test Suite**: `backend/cmd/e2e/acceptance_journey.go` via `backend/cmd/e2e/main.go`  
**Execution Environment**: Local Docker Stack (PostgreSQL 16, Redis 7, NATS, Control API, Gateway v2, Worker, Next.js Web)  
**Status**: 21/21 ACCEPTANCE CRITERIA VERIFIED [PASS]  

---

## 1. Overview & Verification Strategy

The 21-Step Acceptance Journey runs a fully automated end-to-end operational validation that simulates the entire lifecycle of an enterprise using Sentinel. The test suite exercises:
- Organization provisioning, invitation, and role assignments
- Multi-provider configuration with credential validation
- Model synchronization and registry discovery
- Gateway credential issuance with single-view secret semantics
- Multi-scope governance rules: rate limits, token/spend budgets, model allowlists/denylists, and PII guardrails
- Real-time Redis cache invalidation on policy mutations
- Gateway request execution across streaming and non-streaming modes with token attribution and latency capture
- Asynchronous worker ingestion from NATS to PostgreSQL usage ledgers
- Analytics rollup verification (reconciling raw events to summary totals)
- Privacy policy enforcement (metadata-only vs full-content capture)
- Automated alert evaluation, threshold alerting, and notification delivery
- Data retention cleanup rules

---

## 2. Step-by-Step Acceptance Journey Results

| Step # | Acceptance Criteria Description | Key Assertions & Actions | Execution Result |
|---|---|---|---|
| **Step 1** | Organization & Administrator Creation | Provision tenant `Acme Global Corp`, configure settings, assert default teams/projects | `[PASS]` |
| **Step 2** | Multiple Employee Invitations & Roles | Invite `Alice` (Admin/Engineer) & `Bob` (Member), verify invite tokens and role acceptance | `[PASS]` |
| **Step 3** | Team & Project Structure Provisioning | Create `Core Systems` team and `Production App` project; assign employee memberships | `[PASS]` |
| **Step 4** | Primary & Secondary Provider Setup | Configure Anthropic and OpenAI accounts; assert credential storage safety (fingerprint/mask) | `[PASS]` |
| **Step 5** | Provider Health Checks & Offline Probes | Probe provider health via `GET /v1/provider-health`; verify offline dummy key semantics | `[PASS]` |
| **Step 6** | Model Registry Discovery & Sync | Synchronize model catalog via `POST /v1/models/sync`; assert models available in registry | `[PASS]` |
| **Step 7** | Gateway Credential Issuance & Single-Display | Issue gateway keys `aigw_...`; verify secret returned only once, stored as SHA-256 hash | `[PASS]` |
| **Step 8** | Organization Model Allowlist Rule | Enact policy restricting access to `claude-3-5-sonnet` and `gpt-4o`; reject unapproved models | `[PASS]` |
| **Step 9** | Rate Limit Policy Enforcement | Configure sliding-window limit (5 req/min); verify 6th request triggers `429 Too Many Requests` | `[PASS]` |
| **Step 10** | Monthly Spend Budget Allocation | Allocate $100 budget to `Production App`; verify concurrency-safe balance tracking | `[PASS]` |
| **Step 11** | Guardrail Activation (PII Scanner) | Enable PII detection rule; verify requests containing synthetic SSNs are blocked with `403 Forbidden` | `[PASS]` |
| **Step 12** | Valid Gateway Request Routing | Dispatch valid `POST /v1/messages` request; verify provider routing, status `200 OK`, response body | `[PASS]` |
| **Step 13** | Streaming Response & Client Cancellation | Initiate SSE stream; verify chunks received with TTFT measurement; test client disconnect handling | `[PASS]` |
| **Step 14** | Asynchronous Telemetry Ingestion | Emit usage payload via NATS; verify worker consumes event and writes to `llm_requests` ledger | `[PASS]` |
| **Step 15** | Token & Latency Attribution Verification | Verify prompt tokens, completion tokens, TTFT, and total duration match request metadata | `[PASS]` |
| **Step 16** | Usage Ledger & Rollup Reconciliation | Verify `usage_daily` and `usage_hourly` match the exact sum of individual `llm_requests` | `[PASS]` |
| **Step 17** | Executive Dashboard Overview Verification | Query `GET /v1/usage/summary` & `GET /v1/usage/timeseries`; assert non-zero real metrics | `[PASS]` |
| **Step 18** | Privacy Mode Enforcement (Metadata-Only) | Set privacy to `METADATA_ONLY`; verify prompt and completion text are omitted from storage | `[PASS]` |
| **Step 19** | Alert Threshold & In-App Notification | Incur spend exceeding budget warning threshold (80%); assert alert fires and in-app event generated | `[PASS]` |
| **Step 20** | Immutable Audit Trail Logging | Inspect `GET /v1/audit-logs`; verify all admin actions, policy edits, and denials are logged | `[PASS]` |
| **Step 21** | Data Retention & Policy Pruning | Trigger retention cleanup job; verify expired temporary records purged while audit records preserved | `[PASS]` |

---

## 3. Synthetic Seed Data vs Live Provider Isolation

To ensure absolute fidelity without incurring external vendor costs or leaking proprietary keys during local testing:
- **Synthetic Test Keys**: Keys formatted with `sk-dummy-*` or `dummy-*` are deterministically routed to the local high-fidelity provider simulators.
- **Explicit Labeling**: All synthetic accounts, models, and usage records created by `seed.sql` and the E2E suite are explicitly prefixed with `[Synthetic Seed]` or `perf-test`.
- **Zero Mock Bypasses**: The gateway request pipeline executes full authentication, authorization, budget checking, rate limiting, and telemetry logging regardless of whether the backend adapter reaches a simulated or live endpoint.

---

## 4. Execution Command & Logs

The acceptance suite can be re-run at any time against the running local stack:

```bash
cd backend
go run ./cmd/e2e
```

Output:
```text
>>> [Acceptance Step 1/21] Creating organization & setting up administrator...
  [PASS] Organization 'Acme Global Corp' created with owner 'admin@acme.example'
...
>>> [Acceptance Step 21/21] Verifying category-specific data retention engine...
  [PASS] Retention rules verified: Expired transient data pruned, audit logs intact
=========================================================================
   21-STEP ACCEPTANCE JOURNEY COMPLETED SUCCESSFULLY (ALL STEPS PASSED)  
=========================================================================
```
