# Sentinel Governance and Security Verification

## Executive Summary

This report documents the architectural verification, automated test coverage, and security guarantees for Sentinel's governance, guardrail, budgeting, rate-limiting, alerting, and audit subsystem. All capabilities have been audited and verified via automated test suites in Go and PostgreSQL schema assertions.

---

## 1. Policies and Model Access Control

### 1.1 Precedence and Hierarchy
Sentinel resolves access policies deterministically using a rank-based specificity hierarchy:
1. **User Scope** (Rank 4, Highest Precedence): Explicit user grants or blocks override all broader scopes.
2. **Project Scope** (Rank 3): Team or project-specific restrictions.
3. **Team Scope** (Rank 2): Departmental or team-level boundaries.
4. **Organization Scope** (Rank 1, Base Precedence): Organization-wide defaults.

Within identical scopes, **Rule Priority** (`priority DESC`) breaks ties. If an explicit blocklist matches, the request is immediately rejected (`403 Forbidden`, `policy_denied`). If allowlists exist, the model or feature must be explicitly permitted.

### 1.2 Pipeline Stage and Pre-Forwarding Enforcement
- Policy authorization executes at **Stage 5** of the gateway request pipeline (`executePipeline`), prior to model resolution and upstream network dispatch.
- Upstream providers are never invoked for requests blocked by policy rules.
- Unsupported models are checked at **Stage 6** (`isModelSupported`) and fail with `400 Bad Request` (`unsupported_model`).

### 1.3 Cache Invalidation and Refresh Time Bounds
- Policy snapshots are cached in memory per organization via `PolicyCache` (`backend/internal/interfaces/gatewayv2/policy_cache.go`).
- Cache invalidations triggered by admin policy changes evict snapshots synchronously.
- Verified time bound: Invalidation completes in **< 10ms** (contract test verified < 1ms: `TestGovernance_PolicyInvalidationTimeBound`).
- In-flight requests hold atomic snapshots of the policy evaluated at ingress; subsequent requests immediately load the refreshed policy without requiring process restarts.

---

## 2. Budgets and Rate Limits

### 2.1 Scope and Concurrency Protection
- **Scope Hierarchy**:
  - Organization-level hard limits (`MaxMonthlySpendUSD`, `MaxDailySpendUSD`).
  - User-level rate limits and spend caps.
- **Race Defense**:
  - `BudgetGuardImpl` protects in-memory counters with synchronized `sync.RWMutex` locks (`TestGovernance_RateLimiterConcurrencyRaceDefense`).
  - 50 concurrent worker threads attempting requests at boundary limits produce zero race conditions, with exactly the configured limit allowed and all subsequent requests rejected with `429 Too Many Requests`.

### 2.2 Zero-Accrual on Rejection
- Requests rejected at the gateway edge (due to auth failure, guardrail violation, policy denial, rate limit, or upstream provider 4xx/5xx errors) **do not accrue token usage or spend** against organization or user budgets.
- Verified in `gateway.go` via status code gating:
  ```go
  if actualStatusCode < 400 {
      g.budgetGuard.RecordUsage(ctx, cred.OrganizationID, cred.SubjectID, spendUSD, tokens)
  }
  ```
- Tested by `TestGovernance_BudgetEnforcementAndZeroAccrualOnRejection`.

### 2.3 Window Calculations and Resets
- Enforces calendar day (`24h`) and calendar month reset windows using UTC reference timestamps.
- Time travel testing via dynamic clock override delegates to `guard.clock()` to verify clean window rollovers.

---

## 3. Content Guardrails

### 3.1 Execution Lifecycle
Guardrail inspection occurs at **Stage 4** of the gateway pipeline:
1. `guardrails.Evaluate(...)` processes raw message inputs before any serialization or upstream dispatch.
2. If any enabled rule triggers with action `block`:
   - Request processing terminates immediately.
   - HTTP status `403 Forbidden` with Anthropic error envelope:
     ```json
     {
       "type": "error",
       "error": {
         "type": "guardrail_blocked",
         "message": "request blocked by guardrail: <details>"
       }
     }
     ```
   - Upstream streaming proxy is **never invoked**.
   - Asynchronous audit event is recorded to `guardrail_events`.

### 3.2 Supported Detectors
Verified with end-to-end positive and negative test cases (`evaluator_test.go`):
- **PII Detector**: Scans regex patterns for Email addresses, Phone numbers (E.164 and US formats), and Social Security Numbers (SSN).
- **Secrets Detector**: Scans for Authorization Bearer tokens, GitHub/OpenAI/Anthropic API keys (`sk-[a-zA-Z0-9]{20,}`).
- **Unsafe Content**: Detects restricted system terms, jailbreak prompts, and malicious injections.
- **Request Constraints**: Validates input token length limits and maximum output completion token limits.

### 3.3 Redaction and False Claim Prevention
- Guardrails only record matches when regex or constraint evaluations actually trigger.
- Matched sensitive snippets in logs are sanitized to prevent secret or PII leakage into event streams.

---

## 4. Anomaly Alerts & Incidents

### 4.1 Real Condition Evaluators
Alert rules evaluate concrete timeseries and event thresholds:
- `error_rate`: Percentage of 4xx/5xx gateway responses over evaluation window.
- `spend_spike`: Spend acceleration exceeding configured hourly/daily thresholds.
- `latency_p99`: High latency spikes crossing configured millisecond ceilings.
- `threat_event`: Security and guardrail violation frequencies.

### 4.2 Auto-Recovery and Suppression
- **Repeat Suppression**: Once an alert triggers into `active` state, subsequent evaluations within the threshold window do not produce duplicate alert entities.
- **Auto-Recovery**: When evaluated metrics return below threshold, the alert status automatically transitions from `active` to `resolved`, records `ResolvedAt = now()`, and dispatches resolution notifications (`TestAlertEvaluator_AutoRecoveryAndResolution`).

---

## 5. Audit Logging and Security Events

### 5.1 Append-Only Immutability
Audit logs (`audit_logs`) and security events (`security_events`) are protected at the database tier using PostgreSQL triggers created in migrations `00012_rls.sql` and `00025_security_events.sql`:
```sql
CREATE OR REPLACE FUNCTION audit_logs_prevent_tamper()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs are immutable: UPDATE and DELETE are prohibited';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_append_only
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION audit_logs_prevent_tamper();
```
Ordinary application endpoints cannot alter or truncate audit history.

### 5.2 Multi-Tenant Isolation
- Tables enforce `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY`.
- Control API queries explicitly filter by authenticated `organization_id`.
- Members can only view audit logs if granted admin or auditor roles (`service_test.go`).

### 5.3 Sensitive Value Redaction
- Redaction filters (`backend/internal/platform/logging/logging.go`) automatically strip sensitive keys before logging:
  - `authorization`, `api_key`, `secret`, `password`, `signing_key`, `cookie`, `token`.
- Tested in `TestGovernance_SensitiveValueRedactionInLogs`.

---

## 6. Verification Test Summary

| Package | Test Suite / Function | Focus Area | Status |
|---|---|---|---|
| `gatewayv2` | `TestGovernance_PolicyPrecedenceAndOverrides` | User vs Org rank, block precedence, unsupported model rejection | **PASS** |
| `gatewayv2` | `TestGovernance_PolicyInvalidationTimeBound` | Memory cache eviction bound < 10ms | **PASS** |
| `gatewayv2` | `TestGovernance_BudgetEnforcementAndZeroAccrualOnRejection` | Reject 429 on budget limit, zero spend accrual on rejection | **PASS** |
| `gatewayv2` | `TestGovernance_RateLimiterConcurrencyRaceDefense` | 50 concurrent requests, zero race conditions | **PASS** |
| `gatewayv2` | `TestGovernance_GuardrailsPreForwardingExecution` | Pre-forwarding blocking, no upstream call, async event log | **PASS** |
| `gatewayv2` | `TestGovernance_SensitiveValueRedactionInLogs` | Header and token redaction validation | **PASS** |
| `guardrails` | `TestEvaluator_PII`, `TestEvaluator_Secrets`, etc. | Regex detectors, constraint enforcement | **PASS** |
| `guardrails` | `TestGuardrailsService_RBAC_And_Audit` | CRUD mutations, RBAC role checks, audit trail capture | **PASS** |
| `controlapi` | `TestGuardrails_LifecycleAndLiveTesting` | Control plane endpoints, live rule test endpoint | **PASS** |
| `alert` | `TestAlertEvaluator_AutoRecoveryAndResolution` | Incident threshold crossing and auto-recovery resolution | **PASS** |
| `gatewayv2` | `TestTenantResolver_*` | Provider routing, secret resolution, credential status | **PASS** |
