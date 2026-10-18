# Secure Provider Credential Management Architecture

This document specifies Sentinel's organization-level AI provider connection and credential architecture, credential ownership boundaries, lifecycle operations, and security controls.

---

## 1. Credential Ownership & Boundary Model

Sentinel strictly separates three distinct layers of identity, credentials, and access control:

| Layer | Type | Scope | Storage & Representation | Public Visibility |
| :--- | :--- | :--- | :--- | :--- |
| **Org Provider Master Key** | Upstream provider secret (`sk-ant-...`, `sk-proj-...`, `AIzaSy...`, `hf_...`) | Organization-owned master credential | Encrypted in `SecretStore` via unique reference (`provider/creds/{uuid}`); database stores only metadata, SHA-256 fingerprint, and last 4 characters | Never returned to frontend, client, or logs. Only accessible server-side by gateway router. |
| **Sentinel Gateway Credential** | Client bearer token (`sgw_live_...`) | Individual employee user or service account | SHA-256 token hash stored in PostgreSQL with salt; raw token shown exactly once at creation time | One-time display to user; hash verified on incoming requests |
| **Model Access & Policy Rules** | Governance policy (`model_access_rules`) | User, team, project, or organization scope | PostgreSQL policy tables with real-time Redis invalidation cache (<10ms) | Evaluated before any upstream forwarding; connecting a provider does not grant automatic access |

### Critical Security Invariant
Employees and client SDKs are **never** given organization provider master keys. The gateway receives requests bearing `Authorization: Bearer sgw_...`, authenticates the identity and evaluates policy, and securely attaches the organization's upstream master key in internal Go memory during proxying.

---

## 2. Supported Providers & Validation Operations

Sentinel validates credentials against upstream providers using safe, non-mutating, zero-cost operations:

| Provider | Supported Kind | Safe Validation Operation | Model Discovery Endpoint |
| :--- | :--- | :--- | :--- |
| **Anthropic** | `anthropic` | `GET /v1/models` (or lightweight header probe) | `GET /v1/models` |
| **OpenAI** | `openai` | `GET /v1/models` | `GET /v1/models` |
| **Google Gemini** | `gemini` | `GET /v1beta/models` | `GET /v1beta/models` |
| **Hugging Face** | `huggingface` | `GET https://huggingface.co/api/whoami-v2` | `GET https://huggingface.co/api/models` |
| **Azure OpenAI** | `azure_openai` | Deployment endpoint probe | Configured deployment catalog |
| **Custom** | `custom` | OpenAI-compatible `/v1/models` probe | `/v1/models` |

Mock test keys (e.g. `sk-ant-dummy`, `sk-proj-test`, `test-gemini-key`, `hf_test_dummy`) are explicitly recognized by `isDummyKey()` for local development and deterministic CI fixtures without making live network requests.

---

## 3. Account & Credential Lifecycle

### A. Connection Registration & Credential Addition
1. **Account Creation**: Org Admin registers a provider account with `POST /v1/provider-accounts`. Status defaults to `active`.
2. **Credential Storage**: Admin submits key via `POST /v1/provider-accounts/{id}/credentials`.
3. **Upstream Validation**: Backend performs live validation against upstream provider using `ValidateCredentials`.
4. **Secret Storage**: Upon successful validation, secret is written to `SecretStore` with a random UUID reference (`provider/creds/{uuid}`).
5. **Metadata Persistence**: Only `fingerprint` and `tail` are stored in PostgreSQL.
6. **Cache Invalidation**: Redis Pub/Sub broadcasts invalidation to all gateway replicas to evict stale upstream connections.

### B. Account-Level Validation & Model Discovery
Admins can test connection and sync models without re-entering the secret:
- `POST /v1/provider-accounts/{id}/validate`: Retrieves the active stored credential internally from `SecretStore` and executes upstream validation.
- `POST /v1/provider-accounts/{id}/health`: Probes provider health using stored credentials and updates `provider_health` records.
- `POST /v1/provider-accounts/{id}/sync`: Queries provider model catalog, discovers accessible models, and upserts them into the model registry.

### C. Zero-Downtime Key Rotation
When rotating an active credential (`POST /v1/provider-credentials/{id}/rotate`):
1. **Pre-Validation**: New secret is validated against the upstream provider.
2. **Dual-Key Staging**: New active credential row is inserted with `Name: "<existing_name> (rotated)"`. Both credentials are briefly active so in-flight requests never drop.
3. **Old Key Revocation**: Old credential row is updated to `status: revoked`.
4. **Old Secret Purge**: Old secret is deleted from `SecretStore`.
5. **Rollback Safety**: If revoking the old row fails, the transaction rolls back: the new credential is immediately revoked and its secret purged from `SecretStore`.

### D. Disabling Connections
- `PATCH /v1/provider-accounts/{id}` with `{ "status": "disabled" }`.
- Evicts cached upstream entries via Redis Pub/Sub.
- Gateway `TenantResolver` filters out disabled accounts (`status == "active"` required), instantly blocking inference requests for this provider with `upstream_unavailable`.

### E. Deletion & Cascade Secret Cleanup
When a provider account is removed (`DELETE /v1/provider-accounts/{id}`):
1. All associated credentials for the account are listed.
2. Each credential is marked revoked.
3. Every secret material reference (`SecretRef`) is deleted from `SecretStore`.
4. Account row is deleted from the database.
5. Invalidation event evicts upstream cache entries across all gateway instances.

---

## 4. RBAC & Multi-Tenant Isolation

### Authorization Policies
- `PermissionProvidersManage` & `PermissionCredentialsManage`: Restricted to `owner` and `admin` roles.
- `PermissionProvidersRead` & `PermissionCredentialsRead`: Restricted to `owner`, `admin`, and `auditor` roles.
- Standard employees (`member`) and service accounts cannot read provider accounts, create/rotate credentials, or initiate health probes (`403 Forbidden`).

### Tenant Isolation
- PostgreSQL Row-Level Security (FORCE RLS) combined with repository application context (`actor.OrganizationID`) enforces strict tenant boundary separation.
- Any attempt to access, mutate, or delete an account or credential from another organization results in immediate `404 Not Found`.

---

## 5. Secret Protection & Leak Prevention

1. **No Plaintext Storage**: Secrets are never saved in database tables.
2. **Never Returned in Full**: Control API responses for accounts and credentials omit secrets, returning only:
   ```json
   {
     "id": "71d48424-971b-45a3-a69e-440c65c611f1",
     "provider_account_id": "00000000-0000-0000-0000-00000000000a",
     "name": "Production Key",
     "kind": "api_key",
     "fingerprint": "a3f5c1...",
     "tail": "...1234",
     "status": "active"
   }
   ```
3. **No Logging**: Request loggers and audit log sinks sanitize `Authorization` headers, request bodies containing secrets, and error payloads.
4. **Sanitized Error Messages**: Upstream provider error responses during validation or health probes are sanitized to prevent secret echo in error descriptions.
