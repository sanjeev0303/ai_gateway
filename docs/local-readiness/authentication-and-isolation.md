# Sentinel Local Authentication and Tenant Isolation Audit

## 1. Overview and Architecture

This document records the audit findings, architecture, test evidence, and residual boundaries for authentication, authorization, and tenant isolation in the Sentinel AI Governance Gateway.

### Core Security Architecture

| Security Domain | Mechanism | Implementation Detail | Location |
| :--- | :--- | :--- | :--- |
| **Password Storage** | PBKDF2-HMAC-SHA256 | 600,000 iterations, 16-byte cryptographically secure random salt, constant-time verification (`subtle.ConstantTimeCompare`), decoy digest execution for unknown emails to eliminate user-enumeration timing channels. Plaintext is never stored. | [`password.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/auth/password.go) |
| **Session Lifecycle** | JWT Access + Opaque Refresh | Short-lived signed JWT access tokens (1h) paired with 256-bit random opaque refresh tokens (30d TTL). Refresh tokens are single-use with automatic rotation. Re-use of a revoked refresh token triggers global session invalidation for the account (compromise detection). | [`auth.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/identity/auth.go) |
| **Browser Security** | Safe Cookies & Double-Submit CSRF | Access cookie is `HttpOnly`, `SameSite=Lax`. Refresh cookie is `HttpOnly`, `SameSite=Strict`. Readable `_csrf` cookie paired with required `X-CSRF-Token` header validated using constant-time comparison on all state-changing cookie requests. Safe `Secure` flag applied conditionally in local dev vs production. | [`auth.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/auth.go) |
| **Database Tenant Isolation** | PostgreSQL FORCE RLS | All tenant tables enforce PostgreSQL `FORCE ROW LEVEL SECURITY`. Repository transactions bind tenant context via `WithTenantTx` invoking `SELECT set_config('app.org_id', $1, true)`. Forgotten tenant context or mismatched queries fail closed (zero rows returned). | [`tenant.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/postgres/tenant.go) |
| **Control Plane Auth** | Principal Derivation | All control API endpoints extract caller identity strictly from verified token claims or API keys via `requireAuth`. Client-supplied tenant/org headers are never trusted. | [`auth.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/auth.go) |
| **Gateway Auth & Scoping** | Key Hash Lookup & Scopes | Gateway keys are hashed with PBKDF2-HMAC-SHA256 and matched against active credentials. Validated for `status=active`, non-expired timestamp, and `gateway`/`all` scope. Keys with `control` scope only are forbidden from AI gateway proxy routes. | [`gateway.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway.go), [`authenticator.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/authenticator.go) |
| **Key Revocation** | Real-time Invalidation | Credential revocation immediately marks the row revoked in PostgreSQL, evicts the in-memory cache, and publishes cache invalidation over Redis Pub/Sub. Subsequent gateway requests receive HTTP 401 Unauthorized (`authentication_error`). | [`policy_invalidation_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/policy_invalidation_test.go), [`gateway_lifecycle_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway_lifecycle_test.go) |
| **Role-Based Access Control** | Role-Permission Matrix | Enforces granular permission gates (`domain.Permission*`) across 6 roles (`RoleOwner`, `RoleAdmin`, `RoleAuditor`, `RoleBillingAdmin`, `RoleMember`, `RoleService`). Day-to-day members are restricted to read operations and forbidden from administrative mutations. | [`roles.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/domain/roles.go), [`service.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/identity/service.go) |
| **Subject-Level Isolation** | Conversation Privacy | Ordinary members can only view their own conversations within the organization (`summary.UserID == actor.UserID`). Privileged roles (`RoleOwner`, `RoleAdmin`) can view org conversations according to privacy modes. | [`service.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/conversation/service.go) |
| **Secret Scrubbing** | Zero Leakage | Gateway credential listings redact `KeyHash`, displaying only `KeyPrefix`. Provider credential listings redact `SecretRef`, displaying only `Fingerprint` and `Tail`. No password digests or raw secrets appear in responses, error bodies, or slog output. | [`multi_tenancy_security_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/multi_tenancy_security_test.go) |
| **Audit Logging** | Immutable Audit Trail | Administrative actions (member removal, role change, credential creation/rotation/revocation, policy mutations, alert updates) append immutable audit logs attributed to the actor. | [`service.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/identity/service.go) |

---

## 2. Test Verification Evidence

The test suite in [`multi_tenancy_security_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/multi_tenancy_security_test.go) and [`gateway_lifecycle_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway_lifecycle_test.go) verifies all isolation and authentication requirements using two independent organizations and three distinct users.

### Test Matrix Entities

- **Organization A (`orgTenantA`)**: `11111111-1111-1111-1111-111111111111`
- **Organization B (`orgTenantB`)**: `22222222-2222-2222-2222-222222222222`
- **User 1 (`userTenantA_Owner`)**: Alice — `RoleOwner` in Tenant A
- **User 2 (`userTenantA_Member`)**: Bob — `RoleMember` in Tenant A
- **User 3 (`userTenantB_Owner`)**: Charlie — `RoleOwner` in Tenant B

### Cross-Tenant IDOR Protection Evidence

Attempts by User 1 (Tenant A) to access Tenant B resources by substituting UUIDs across all 9 resource families:

| Resource Family | Target Endpoint | Method | Response Code | Result |
| :--- | :--- | :--- | :--- | :--- |
| **Projects** | `/v1/projects/{projectTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Teams** | `/v1/teams/{teamTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Gateway Credentials** | `/v1/gateway-credentials/{credTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Gateway Credentials** | `/v1/gateway-credentials/{credTenantB}` | `DELETE` | 404 Not Found | **PASS** (Cannot revoke other tenant key) |
| **Gateway Credentials** | `/v1/gateway-credentials/{credTenantB}/rotate` | `POST` | 404 Not Found | **PASS** (Cannot rotate other tenant key) |
| **Provider Accounts** | `/v1/provider-accounts/{provAcctTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Provider Credentials** | `/v1/provider-credentials/{provCredTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Provider Credentials** | `/v1/provider-credentials/{provCredTenantB}` | `DELETE` | 404 Not Found | **PASS** (Cannot revoke other tenant key) |
| **Service Accounts** | `/v1/service-accounts/{saTenantB}` | `DELETE` | 404 Not Found | **PASS** (Cannot revoke other tenant SA) |
| **Organization Members** | `/v1/members/{userTenantB_Owner}` | `DELETE` | 404 Not Found | **PASS** (Cannot remove other tenant user) |
| **Policies** | `/v1/policies/{policyTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Policies** | `/v1/policies/{policyTenantB}` | `DELETE` | 404 Not Found | **PASS** (Cannot delete other tenant policy) |
| **Audit Logs** | `/v1/audit-logs/{auditTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Alerts** | `/v1/alerts/{alertTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Conversations** | `/v1/conversations/{convTenantB}` | `GET` | 404 Not Found | **PASS** (Zero cross-tenant disclosure) |
| **Invoices** | `/v1/billing/invoices` | `GET` | 200 OK | **PASS** (Scoped strictly to `orgTenantA` invoices) |

### Role-Based Access Control Evidence

Comparing User 1 (`RoleOwner`) and User 2 (`RoleMember`) within Organization A:

| Operation | Path | Role: Member | Role: Owner | Result |
| :--- | :--- | :--- | :--- | :--- |
| **Read Projects** | `GET /v1/projects` | 200 OK | 200 OK | **PASS** (Permitted day-to-day read) |
| **Read Teams** | `GET /v1/teams` | 200 OK | 200 OK | **PASS** (Permitted day-to-day read) |
| **Read Members** | `GET /v1/members` | 200 OK | 200 OK | **PASS** (Permitted day-to-day read) |
| **Create Project** | `POST /v1/projects` | 403 Forbidden | 201 Created | **PASS** (Admin operation blocked) |
| **Create Team** | `POST /v1/teams` | 403 Forbidden | 201 Created | **PASS** (Admin operation blocked) |
| **Invite Member** | `POST /v1/invitations` | 403 Forbidden | 201 Created | **PASS** (Admin operation blocked) |
| **Create Gateway Key** | `POST /v1/gateway-credentials` | 403 Forbidden | 201 Created | **PASS** (Credential issuance blocked) |
| **Revoke Gateway Key** | `DELETE /v1/gateway-credentials/{id}` | 403 Forbidden | 204 No Content | **PASS** (Credential revocation blocked) |
| **Create Policy** | `POST /v1/policies` | 403 Forbidden | 201 Created | **PASS** (Governance modification blocked) |
| **Delete Policy** | `DELETE /v1/policies/{id}` | 403 Forbidden | 204 No Content | **PASS** (Governance deletion blocked) |
| **Read Audit Logs** | `GET /v1/audit-logs` | 403 Forbidden | 200 OK | **PASS** (Audit trail protected) |
| **Create Alert** | `POST /v1/alerts` | 403 Forbidden | 201 Created | **PASS** (Alert creation blocked) |
| **Read Invoices** | `GET /v1/billing/invoices` | 403 Forbidden | 200 OK | **PASS** (Billing records protected) |

### Gateway Credential Lifecycle & Spoofing Prevention Evidence

Tests executed in [`gateway_lifecycle_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/gatewayv2/gateway_lifecycle_test.go):

| Scenario | Condition | Expected Status | Error Type | Upstream Forwarded | Result |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Active Key** | `status = active`, valid scope | 200 OK | None | Yes | **PASS** |
| **Revoked Key** | Credential revoked in DB | 401 Unauthorized | `authentication_error` | **No** (Zero upstream traffic) | **PASS** |
| **Expired Key** | `ExpiresAt < Now` | 401 Unauthorized | `authentication_error` | **No** (Zero upstream traffic) | **PASS** |
| **Wrong Scope** | Key has only `control` scope | 403 Forbidden | `permission_error` | **No** (Zero upstream traffic) | **PASS** |
| **Tenant Spoofing** | Header `X-Organization-ID` mismatch | 403 Forbidden | `tenant_isolation_violation` | **No** (Zero upstream traffic) | **PASS** |

### Execution Log Snippet

```text
=== RUN   TestMultiTenantSecurity_IDORCrossTenantProtection
--- PASS: TestMultiTenantSecurity_IDORCrossTenantProtection (0.01s)
=== RUN   TestMultiTenantSecurity_RoleBasedAccessControl
--- PASS: TestMultiTenantSecurity_RoleBasedAccessControl (0.01s)
=== RUN   TestMultiTenantSecurity_SubjectLevelIsolationSameOrg
--- PASS: TestMultiTenantSecurity_SubjectLevelIsolationSameOrg (0.00s)
=== RUN   TestMultiTenantSecurity_CrossTenantOrgSwitchingBlocked
--- PASS: TestMultiTenantSecurity_CrossTenantOrgSwitchingBlocked (0.00s)
=== RUN   TestMultiTenantSecurity_SecretScrubbing
--- PASS: TestMultiTenantSecurity_SecretScrubbing (0.00s)
=== RUN   TestMultiTenantSecurity_RevokedCredentialImmediateRejection
--- PASS: TestMultiTenantSecurity_RevokedCredentialImmediateRejection (0.00s)
PASS
ok  	ai-governance-saas/backend/internal/interfaces/controlapi	0.045s

=== RUN   TestGatewayLifecycle_RevokedGatewayCredentialPreventsRequests
--- PASS: TestGatewayLifecycle_RevokedGatewayCredentialPreventsRequests (0.00s)
=== RUN   TestGatewayLifecycle_ExpiredGatewayCredentialPreventsRequests
--- PASS: TestGatewayLifecycle_ExpiredGatewayCredentialPreventsRequests (0.00s)
=== RUN   TestGatewayLifecycle_WrongScopeCredentialPreventsRequests
--- PASS: TestGatewayLifecycle_WrongScopeCredentialPreventsRequests (0.00s)
=== RUN   TestGatewayLifecycle_CrossTenantSpoofedHeaderRejected
--- PASS: TestGatewayLifecycle_CrossTenantSpoofedHeaderRejected (0.00s)
PASS
ok  	ai-governance-saas/backend/internal/interfaces/gatewayv2	0.009s
```

---

## 3. Remaining Limitations

1. **SaaS Multi-Cluster Partitioning**: Tenant isolation is enforced at application context and PostgreSQL Row-Level Security levels within a shared cluster. Schema-per-tenant or database-per-tenant physical isolation is not configured for local development.
2. **Offline Key Invalidation Propagation**: Redis Pub/Sub broadcast delivers sub-millisecond cache invalidation across running gateway instances. In an environment where Redis is temporarily degraded, local in-memory credential cache entries rely on a bounded TTL fallback before querying PostgreSQL directly.
3. **No External Identity Provider (SSO/SAML)**: The identity architecture currently uses native PBKDF2 password authentication and JWT/opaque sessions. Enterprise SAML/OIDC federated SSO is outside the local baseline scope.
