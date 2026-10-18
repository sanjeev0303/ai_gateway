# Provider-Bound Sentinel Gateway Credentials Architecture

This document specifies the architecture, data model, server-side enforcement, and CLI interactive experience for Sentinel gateway credentials bound to specific organizations, service accounts, provider connections, and model access scopes.

---

## 1. Overview & Scope Hierarchy

Sentinel gateway credentials (`sgw_live_...`) provide secure, auditable, and restricted client access to upstream AI providers. Rather than granting broad or ambient access, each gateway credential can be strictly bounded along five hierarchical dimensions:

```
+--------------------------------------------------------------+
| Organization Boundary (Mandatory Multi-Tenant Root)          |
+--------------------------------------------------------------+
                               |
                               v
+--------------------------------------------------------------+
| Subject Identity (User or Service Account ID)                |
+--------------------------------------------------------------+
                               |
                               v
+--------------------------------------------------------------+
| Provider Connection Scope (`provider_account_id` / providers)|
+--------------------------------------------------------------+
                               |
                               v
+--------------------------------------------------------------+
| Model Restrictions (`allowed_models`)                        |
+--------------------------------------------------------------+
                               |
                               v
+--------------------------------------------------------------+
| Governance Limits (Direct Rate Limits & Budgets + Org Rules) |
+--------------------------------------------------------------+
```

---

## 2. Gateway Credential Data Model

Gateway credentials are persisted with first-class provider connection and model restriction bindings:

| Field | Type | Description |
| :--- | :--- | :--- |
| `provider_account_id` | `UUID (Nullable)` | Pins the credential to a specific organization provider account (e.g. Anthropic Production). |
| `allowed_providers` | `TEXT[]` | Restricts access to specific providers (e.g. `["anthropic"]` or `["openai", "huggingface"]`). |
| `allowed_models` | `TEXT[]` | Restricts requests to explicit model identifiers (e.g. `["claude-3-5-sonnet-20241022"]`). |
| `rate_limit` | `INTEGER (Nullable)` | Direct credential-level requests-per-minute limit enforced in policy authorization. |
| `budget_limit` | `NUMERIC(12,4) (Nullable)` | Direct credential-level spending limit ($ USD) enforced alongside org budgets. |

All metadata is stored in PostgreSQL and cached in `CredentialCache` for sub-millisecond evaluation at the gateway edge.

---

## 3. Server-Side Scope Enforcement Pipeline

The Gateway v2 pipeline (`executePipeline`) enforces zero-trust server-side authorization:

### A. Untrusted Header & Parameter Rejection
- Callers cannot override authorization by sending `x-provider`, `X-Organization-ID`, or `X-Project-ID` headers.
- If a client supplies `x-provider`, the gateway verifies:
  1. The header value is authorized in `AllowedProviders`.
  2. The header matches the requested model's resolved provider kind.
- Any conflicting or unauthorized provider header immediately results in `403 Forbidden` (`permission_error`).

### B. Upstream Provider Resolution & Pinned Accounts
- If `ProviderAccountID` is set on the credential, the gateway resolves the upstream connection via `ResolveUpstreamForAccount`, ensuring the request is strictly fulfilled by that specific account.
- If `AllowedProviders` is configured, requests for models belonging to unauthorized providers are rejected before any upstream network call.

### C. Permitted Model Authorization
- If `AllowedModels` is defined on the credential, the requested model must be explicitly present in the allowlist.
- Model checks evaluate bare identifiers (`gpt-4o`), provider-qualified identifiers (`openai/gpt-4o`), and wildcards.
- Requests for unlisted models are blocked with `403 Forbidden` (`permission_error`).

### D. Direct Credential Limits Injection
- When a credential has direct `RateLimit` or `BudgetLimit` attributes, the policy authorizer dynamically injects a credential-scoped rate limit (`credential_direct:<id>`) and budget guard (`credential_direct:<id>`) into the evaluation set.
- Requests exceeding these limits receive `429 Rate Limit Exceeded` or `402 Budget Exceeded` without consuming upstream provider capacity.

### E. Model Catalog Scoping (`GET /v1/models`)
- `GET /v1/models` evaluates the caller's credential:
  - Excludes models from providers not in `AllowedProviders` or not matching `ProviderAccountID`.
  - Excludes models not present in `AllowedModels`.
  - Excludes models denied by organization governance policies.
- Clients only see models they are authorized to invoke.

---

## 4. Sentinel CLI Experience

The Sentinel interactive CLI (`sentinel`) dynamically inspects the gateway credential's scope on startup:

### A. Scope Discovery & Prompt Adaptation
- `SyncPermittedScope()` queries `/v1/models` using the authenticated gateway key.
- The interactive prompt adapts to show the active provider and model:
  ```text
  sentinel [anthropic/claude-3-5-sonnet-20241022]>
  ```
- If bound to a single provider (e.g. Anthropic), the prompt clearly reflects that context, and the CLI presents Anthropic-tailored behaviors.

### B. Interactive Slash Command Enforcement
- `/provider`: Lists only authorized providers. Attempts to switch to an unauthorized provider are blocked with:
  ```text
  Provider 'openai' is not permitted by your credential scope. Permitted: [anthropic]
  ```
- `/model`: Lists only permitted models. Attempts to switch to an unlisted model are rejected with:
  ```text
  Model 'gpt-4o' is not permitted by your credential scope.
  ```

### C. Multi-Provider Credential Support
- When an organization issues a multi-provider credential (e.g. `AllowedProviders: ["anthropic", "openai"]`), the CLI allows switching between authorized connections via `/provider anthropic` and `/provider openai`.
- `/model` automatically switches context to the permitted models for the newly selected provider.

---

## 5. Security Invariants & Secret Protection

1. **Zero Secret Leakage**:
   - Provider master secrets (`sk-ant-...`, `AIzaSy...`) are never transmitted to clients or displayed in the CLI.
   - Gateway credentials (`sgw_live_...`) are shown exactly once at creation and stored only as salted SHA-256 hashes.
2. **Immediate Revocation**:
   - Revoking a gateway credential takes effect immediately (<10ms).
   - Invalidation events clear in-memory caches and Redis keys, resulting in instant `401 Unauthorized`.
3. **Safe Provider Rotation**:
   - Rotating upstream provider master keys is completely transparent to gateway credentials.
   - The gateway dynamically re-fetches the active upstream key from the encrypted secret store without requiring client credentials to change.
