# Tenant Isolation and Security Verification Guide

This document specifies the security architecture, tenant isolation boundaries, and regression verification for Sentinel's multi-provider AI gateway and dual payment-gateway billing system.

---

## 1. Multi-Tenant Provider Account & Credential Isolation

### Boundaries
1. **Organization Data Scoping**:
   - Provider accounts and credentials exist strictly within an organization boundary (`organization_id`).
   - Every read (`GET /v1/provider-accounts/{id}`, `GET /v1/provider-credentials/{id}`, list endpoints) queries by `(actor.OrganizationID, id)`. If an authenticated actor attempts an IDOR (Insecure Direct Object Reference) using another tenant's account or credential UUID, the repository returns `repositories.ErrNotFound` (HTTP 404).
   - Mutations (`PATCH`, `DELETE`, `rotate`, `sync`, `validate`, `health`) strictly verify that the target account or credential belongs to `actor.OrganizationID`.
   - Attaching a credential to an account (`POST /v1/provider-accounts/{id}/credentials`) enforces that `{id}` belongs to `actor.OrganizationID`.

### Secret Non-Leakage
- **Storage**: Raw provider keys (Anthropic, OpenAI, Gemini, Hugging Face) are validated live against upstream, stored in AES-256 encrypted `SecretStore`, and referenced only by opaque `SecretRef`.
- **Wire Representation**: Responses returned to browser clients or API consumers (`providerCredentialDTO`) only expose `fingerprint` (e.g. SHA-256 fingerprint) and `tail` (last four characters), strictly omitting `Secret` and `SecretRef`.
- **Audit Trails**: Security audits log credential mutations without `Before`/`After` secret payloads.

---

## 2. Gateway Credential Scope Bounding & Parameter Tampering Defense

### Gateway Credential Scoping
1. **`AllowedProviders`**:
   - Gateway credentials can be pinned to specific provider kinds (e.g. `["anthropic"]`).
   - If an incoming request attempts to invoke an unauthorized provider (e.g. OpenAI `gpt-4o`, Gemini `gemini-1.5-pro`, Hugging Face `meta-llama/*`, or sets `x-provider: openai`), Gateway v2 immediately rejects the request with HTTP 403 `permission_error`.
   - If a credential has `AllowedProviders` configured and no provider can be deduced or specified, the request is rejected with HTTP 403 `permission_error`.
2. **`AllowedModels`**:
   - Gateway credentials can be restricted to an explicit model whitelist (e.g. `["claude-3-5-sonnet"]`).
   - Any attempt to invoke another model (even under the same provider) is rejected with HTTP 403 `permission_error`.
   - **Blank Model Defense**: If a credential has `AllowedModels` configured, omitting the `model` parameter is strictly blocked with HTTP 403 `permission_error` (`"credential requires an authorized model; none specified in request"`).
3. **`ProviderAccountID` Binding**:
   - A gateway credential can be pinned to an exact organization provider connection. Gateway v2 resolves upstream credentials strictly through `ResolveUpstreamForAccount(ctx, cred.OrganizationID, *cred.ProviderAccountID)`. Cross-organization connection hijacking is impossible.

### API Parameter Tampering Defense
1. **Query Model vs Body Model Consistency**:
   - If a caller supplies a query parameter `?model=...` and a payload `{"model":"..."}`, Gateway v2 verifies that `queryModel == payloadModel`. Any mismatch is rejected with HTTP 400 Bad Request `parameter_tampering`.
2. **Path Model vs Body Model Consistency**:
   - In provider-native endpoints (e.g. `/v1beta/models/{model}:generateContent` or `/v1/models/{model}`), the model in the URL path is verified against the payload model. Mismatches are rejected with HTTP 400 Bad Request `parameter_tampering`.
3. **Query Provider vs Header/Body Provider Consistency**:
   - Conflicting query parameter `?provider=...` vs header `x-provider` or payload provider is rejected with HTTP 400 Bad Request `parameter_tampering`.
4. **Header Spoofing Rejection**:
   - `X-Organization-ID` mismatching the authenticated credential organization is rejected with HTTP 403 `tenant_isolation_violation`.
   - `X-User-ID` mismatching the authenticated credential subject is rejected with HTTP 403 `permission_error`.

---

## 3. Billing & Payment History Tenant Isolation

1. **Subscriptions & Plans**:
   - `GET /v1/billing/subscription` queries strictly by `actor.OrganizationID`. Organization A cannot view Organization B's active plan, payment provider, or renewal status.
2. **Invoices**:
   - `GET /v1/billing/invoices` queries `s.invoices.List(ctx, actor.OrganizationID)`. Invoice numbers, itemizations, micro-dollar amounts, and payment IDs of other tenants are completely inaccessible.
3. **Usage Rollups**:
   - `GET /v1/billing/usage` queries `s.rollups.GetSummary(ctx, actor.OrganizationID, filter)`. Cross-tenant usage leakage is blocked at the database query level.

---

## 4. Webhook Forgery Rejection

1. **Stripe Webhook Defense (`POST /v1/billing/webhooks/stripe`)**:
   - Requires valid `Stripe-Signature` header (`t=<timestamp>,v1=<hmac>`).
   - Missing or empty `Stripe-Signature` header returns HTTP 400 Bad Request (`payment.ErrInvalidSignature`).
   - Forged or mismatched HMAC signatures are rejected with HTTP 400 Bad Request.
   - Timestamp tolerance (5 minutes) prevents replay attacks.
2. **Razorpay Webhook Defense (`POST /v1/billing/webhooks/razorpay`)**:
   - Requires valid `X-Razorpay-Signature` header containing HMAC-SHA256 of the raw request payload signed with `webhook_secret`.
   - Missing or empty `X-Razorpay-Signature` returns HTTP 400 Bad Request (`payment.ErrInvalidSignature`).
   - Forged signatures are rejected with HTTP 400 Bad Request.
3. **Browser Payment Callback Verification (`POST /v1/billing/verify-payment`)**:
   - Requires cryptographic verification of `order_id|payment_id` signature against `key_secret`. Forged signatures return HTTP 400 Bad Request.

---

## 5. Role-Based Access Control (RBAC)

| Resource / Action | Owner | Admin | Billing Admin | Auditor | Member |
|---|---|---|---|---|---|
| Read Provider Accounts & Credentials | Allowed | Allowed | Denied (403) | Allowed | Denied (403) |
| Manage Provider Accounts (`POST`/`PATCH`/`DELETE`) | Allowed | Allowed | Denied (403) | Denied (403) | Denied (403) |
| Manage Provider Credentials (`POST`/`ROTATE`/`REVOKE`) | Allowed | Allowed | Denied (403) | Denied (403) | Denied (403) |
| Read Billing & Invoices | Allowed | Allowed | Allowed | Allowed | Denied (403) |
| Manage Subscriptions & Checkout | Allowed | Allowed | Allowed | Denied (403) | Denied (403) |
| View Audit Logs | Allowed | Allowed | Denied (403) | Allowed | Denied (403) |

---

## 6. Real-Time Revocation & Disablement Enforcement

1. **Gateway Credential Revocation**:
   - Revoking a gateway credential (`DELETE /v1/gateway-credentials/{id}`) sets its status to `revoked` and evicts cache entries in memory.
   - Gateway v2 `authenticator.validateCredential` immediately returns `ErrCredentialRevoked` (HTTP 401 `authentication_error`).
2. **Provider Account Disablement**:
   - Disabling an account (`PATCH /v1/provider-accounts/{id}` with `status: disabled`) invalidates the organization's upstream cache.
   - Gateway v2 `executePipeline` and `tenantResolver` reject calls with HTTP 503 `upstream_unavailable` (`"provider account for <kind> is not active"`).
3. **Provider Credential Revocation**:
   - Revoking an upstream key (`DELETE /v1/provider-credentials/{id}`) sets status to `revoked`, deletes the stored AES secret, and invalidates the upstream cache.
   - Subsequent calls reject with HTTP 503 `upstream_unavailable`.

---

## 7. Automated Test Verification Evidence

All security constraints are guarded by regression tests:
- `backend/internal/interfaces/controlapi/security_tenant_isolation_test.go`:
  - `TestSecurity_TenantIsolation_ProviderCredentials_CrossTenantAccess`: 8 test cases verifying IDOR, deletion, rotation, credential attachment, and listing isolation.
  - `TestSecurity_TenantIsolation_BillingAndInvoices`: Verifies subscription and invoice isolation across organizations.
  - `TestSecurity_WebhookForgeryRejection`: 5 test cases verifying missing/forged Stripe and Razorpay signatures rejection and authentic signature acceptance.
  - `TestSecurity_RBAC_ProviderAndBillingManagement`: 5 test cases verifying Member and Auditor mutation denials.
  - `TestSecurity_SecretNonLeakageInAPIResponses`: Verifies zero raw secret or internal reference leakage in DTOs.
- `backend/internal/interfaces/gatewayv2/security_tenant_isolation_test.go`:
  - `TestGatewaySecurity_ScopeEscapeAndRestrictions`: Verifies provider and model bounding, model escape blocking, and blank parameter restriction enforcement.
  - `TestGatewaySecurity_APIParameterTampering`: Verifies query vs body, path vs body, and query vs provider tampering rejections.
  - `TestGatewaySecurity_CrossTenantHeaderSpoofing`: Verifies rejection of spoofed `X-Organization-ID` and `X-User-ID` headers.
  - `TestGatewaySecurity_RevocationAndDisablement`: Verifies instant enforcement of revoked credentials and disabled accounts.
  - `TestGatewaySecurity_SecretMaterialNonLeakage`: Verifies errors never leak upstream credentials.
