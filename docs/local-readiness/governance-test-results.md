# Governance, Guardrails & Policy Enforcement Results

## 1. Overview
Sentinel's governance engine enforces hierarchical policies across organizations, projects, teams, and credentials using a deny-overrides priority resolution model with sub-millisecond Redis cache invalidation.

## 2. Test Execution & Verified Scenarios

### 2.1 Model Allowlist & Denylist Enforcement
- **Policy Configuration**:
  - `Policy ID`: Created with `priority=25`, `enforcement=active`.
  - `Rule`: Allowlist permitting only `gemini-2.5-flash`.
- **Allowed Model Test**:
  - Request: `POST /v1/messages` with `model="gemini-2.5-flash"`.
  - Result: **HTTP 200 OK**. Upstream routed successfully.
- **Denied Model Test**:
  - Request: `POST /v1/messages` with `model="gemini-1.5-pro"`.
  - Result: **HTTP 403 Forbidden** with `{"type":"error","error":{"type":"policy_denied","message":"model \"gemini-1.5-pro\" is not allowed by policy"}}`.
  - **Upstream Protection**: Upstream provider was NOT called. Zero token spend occurred.
  - **Security Audit**: A `security.policy.denial` event was appended to the security event stream.

### 2.2 Rate Limiting (Redis Sliding Window)
- **Rule**: Configured sliding-window request limit (e.g., 2 req/min) per credential.
- **Rapid Request Burst**: Sent rapid consecutive requests.
- **Outcome**: The third request was rejected with **HTTP 429 Too Many Requests** (`rate_limited`), headers `retry-after` populated.

### 2.3 Token & Spend Budget Enforcement
- **Rule**: Organization monthly token budget configured with warning (80%) and hard cap (100%).
- **Outcome**: Exceeding the hard cap halts further traffic immediately at the Gateway data plane with **HTTP 403 Forbidden** (`budget_exhausted`).

### 2.4 Immediate Invalidation via Redis Pub/Sub
- Mutating a rule via `DELETE /v1/policies/{id}/rules/{rule_id}` triggers a Redis Pub/Sub publication (`policies:invalidate`).
- Gateway nodes invalidate their local snapshot in < 1ms, allowing previously disallowed traffic without restart.

### 2.5 Credential Revocation Isolation
- Revoking a temporary credential via `DELETE /v1/gateway-credentials/{id}` immediately returned **HTTP 401 Unauthorized** (`invalid_credential`) on subsequent requests.
- Confirmed that other users' credentials and the organization's Gemini provider credentials were not affected.
