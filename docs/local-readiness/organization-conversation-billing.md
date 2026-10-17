# Organization Workflows, Conversation History, and Local Billing Verification

## Executive Summary

This report documents the architectural audit, automated test coverage, and verification results for Sentinel's organization lifecycle, invitation workflows, team/project hierarchies, gateway credentials, conversation history with server-side privacy redaction, and local billing accounting.

---

## 1. Organization Lifecycle & Hierarchy

### 1.1 Organization Management & Multi-Tenancy
- Organizations serve as the root multi-tenant isolation boundary. Every operational entity (`projects`, `teams`, `members`, `policies`, `budgets`, `credentials`, `conversations`) is strictly scoped by `organization_id`.
- Tenant boundaries are enforced via PostgreSQL `FORCE ROW LEVEL SECURITY` and application actor context resolution (`backend/internal/application/identity`).

### 1.2 Invitation Lifecycle
- **Creation & Token Generation**:
  - Owners and Admins create invitations (`POST /v1/invitations`).
  - High-entropy 32-byte hex tokens are generated; only the SHA-256 hash is persisted in `invitations.token_hash`. The raw token is returned exactly once in the response.
- **Duplicate Prevention**:
  - The database enforces a partial unique constraint:
    ```sql
    CREATE UNIQUE INDEX invitations_org_email_pending_idx
        ON invitations (organization_id, lower(email)) WHERE status = 'pending';
    ```
  - Attempting to issue a duplicate pending invite for the same email in the same organization immediately returns `409 Conflict` (`repositories.ErrConflict`).
- **Acceptance & Expiration**:
  - Authenticated users accept invitations via `POST /v1/invitations/accept`. Acceptance is atomic and idempotent.
  - Expired invitations (`now() > expires_at`) fail validation and cannot be accepted.
- **Revocation**:
  - Admins can revoke pending invitations (`DELETE /v1/invitations/{id}`), transitioning the state to `revoked`.

### 1.3 Role Hierarchy & Privilege Escalation Defenses
- Supported organization roles: `owner`, `admin`, `member`, `auditor`, `billing_admin`.
- **Privilege Safeguards**:
  - Non-owners cannot promote themselves or any other user to `owner` or `admin` (`403 Forbidden`).
  - Members cannot modify membership roles or change their own role (`403 Forbidden`).
  - The last remaining owner of an organization cannot be demoted or removed.

### 1.4 Team and Project Associations
- Teams and projects are organization-scoped resources.
- Adding members (`POST /v1/teams/{id}/members`, `POST /v1/projects/{id}/members`) and removing members strictly verify tenant ownership. Cross-tenant modification attempts are rejected with `404 Not Found` or `403 Forbidden`.

### 1.5 Individual Gateway Credentials
- Gateway credentials can be attached to a specific `user` or `service_account`.
- **Key Generation & Storage**:
  - Raw secret generated as `aigw_<hex>` and hashed via SHA-256 before persistence. Only `key_prefix` (e.g. `aigw_ab12cd...`) is readable after creation.
- **Rotation**:
  - Calling `POST /v1/gateway-credentials/{id}/rotate` revokes the old credential, generates a new active credential with the same scopes, and invalidates in-memory and Redis caches.
- **Revocation**:
  - Calling `DELETE /v1/gateway-credentials/{id}` marks the credential `revoked`. Revoked keys are rejected immediately at the gateway edge (`401 Unauthorized`).

---

## 2. Conversation History and Privacy

### 2.1 Attribution and Persistence
- Conversations (`conversations`) and individual turns (`conversation_messages`) record `organization_id`, `user_id`, optional `project_id`, `tokens`, and message timestamps.
- Assistant turns count toward `request_count` and provider inference usage.

### 2.2 Ownership & Subject-Level Access Control
- **Regular Members**: Restricted to listing and reading only conversations where `user_id == actor.UserID`. Accessing another user's conversation yields `403 Forbidden` (`TestConversationHistory_SubjectAccessAndPrivacyModes`).
- **Privileged Roles**: `owner`, `admin`, and `auditor` can inspect conversations across all members within their own organization.
- **Multi-Tenant Isolation**: Cross-organization conversation access is blocked by RLS and tenant validation.

### 2.3 Server-Side Privacy Redaction Modes
Sentinel supports 4 deterministic privacy modes configured at the organization level (`privacy_default`) with optional per-conversation overrides:

| Privacy Mode | System / User Prompts | Assistant Turns | Tokens & Metadata | Intended Use Case |
|---|---|---|---|---|
| `METADATA_ONLY` | Redacted (omitted) | Redacted (omitted) | Preserved | Strict enterprise privacy & zero data retention |
| `PROMPT_ONLY` | Verbatim Content | Redacted (omitted) | Preserved | Audit prompt compliance without storing model outputs |
| `FULL_CONTENT` | Verbatim Content | Verbatim Content | Preserved | Full debuggability and conversation history inspection |
| `OFF` | Verbatim Content | Verbatim Content | Preserved | Redaction disabled |

- **Security Guarantee**: Privacy redaction is applied server-side prior to serialization (`conversationMessageDTO.Content` has `omitempty`). Callers cannot bypass redaction via query parameters.

### 2.4 Data Retention and Pruning
- The background worker executes scheduled retention pruning (`backend/internal/interfaces/worker/retention.go`).
- When `retention_conversations_days > 0`, conversations and cascading message turns older than the cutoff timestamp are permanently purged via `PurgeExpiredConversations`.

---

## 3. Usage Accounting and Local Billing

### 3.1 Local Usage Reporting Scope
> [!IMPORTANT]
> **No Payment Processing or Financial Settlement**:
> Sentinel's local billing module is designed strictly for deterministic usage accounting, cost attribution, and governance tier limits. It does **not** process credit cards, initiate payment gateway webhooks, or perform financial settlement transactions.

### 3.2 Granular Cost and Token Attribution
- Usage is metered at three distinct levels:
  1. **Request Level**: Stored in `llm_requests` with exact prompt tokens, completion tokens, and estimated cost calculated from the inference-time price sheet.
  2. **User & Team Level**: Aggregated in `usage_hourly` and `usage_daily` tables with project, team, and user dimensions.
  3. **Organization Totals**: Trailing 30-day and monthly spend aggregates.
- Historical usage remains consistent and idempotent; replayed events cannot double-count spend or tokens.

### 3.3 Subscription Tiers
- Plan catalog (`GET /v1/billing/plans`) provides available tiers (e.g. `developer`, `scale`).
- Active subscription (`GET /v1/billing/subscription`) indicates the current governance tier, active cycle start/end dates, and cancellation status.
- Plan modifications (`PUT /v1/billing/subscription`) update internal quota and governance limits.

### 3.4 Invoices & Line Items
- Invoices (`GET /v1/billing/invoices`) record closed-period usage summaries.
- Each invoice includes itemized lines (`invoice_lines`) indicating model usage, token quantities, unit rates in micros, and line totals.
- **Integer Micros Precision**:
  - Currency calculations strictly use integer micro-dollars (`1 USD = 1,000,000 micros`), eliminating floating-point rounding drift.
  - Verified in `TestBilling_UsageAccountingAndInvoiceTotals`:
    $$\text{TotalMicros} = \sum (\text{Line AmountMicros})$$

---

## 4. Frontend Resilience and Route Integrity

### 4.1 Section Root Redirects
Section roots in `app/next.config.ts` are configured with automatic redirects to prevent 404 navigation errors:
- `/app/billing` $\rightarrow$ `/app/billing/usage`
- `/app/organization` $\rightarrow$ `/app/organization/members`
- `/app/observe` $\rightarrow$ `/app/observe/live-activity`

### 4.2 UI State Resilience
- **Loading & Empty States**: Subscription, invoice, and conversation views render structured `EmptyState` cards when no data exists rather than blank screens.
- **Error Boundaries**: Secondary metric failures (e.g., sparklines or activity heatmaps) do not crash or blank primary data tables or navigation headers.

---

## 5. Verification Test Matrix

| Component | Test Suite / Function | Verification Target | Result |
|---|---|---|---|
| Identity / Org | `TestOrgLifecycle_InvitesAndDuplicatesAndRBAC` | Pending invite creation, duplicate 409 conflict, owner role safeguards, self-role modification block, invitation revocation | **PASS** |
| Identity / Hierarchy | `TestOrgLifecycle_TeamsProjectsMembership` | Team member add/remove, project member add/remove, tenant association | **PASS** |
| Identity / Keys | `TestOrgLifecycle_GatewayCredentials` | User gateway key creation, raw key return, key rotation, revocation | **PASS** |
| Conversations | `TestConversationHistory_SubjectAccessAndPrivacyModes` | Member subject access isolation (403 on other user), METADATA_ONLY, PROMPT_ONLY, FULL_CONTENT redactions | **PASS** |
| Conversations | `TestConversationService_PrivacyModes` | Token availability preservation under all privacy modes | **PASS** |
| Billing | `TestBilling_UsageAccountingAndInvoiceTotals` | Subscription retrieval/upgrade, invoice line arithmetic, micros reconciliation | **PASS** |
| Worker | `retention_test.go` | Pruning expired conversations and usage events older than retention cutoff | **PASS** |
| Frontend | `npm run typecheck` & `npm test` | Next.js compilation, routing contracts, 24/24 client tests | **PASS** |
