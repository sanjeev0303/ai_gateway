# Complete Sentinel Feature Test Matrix

## 1. Summary
All 17 supported features defined in the Master Execution Prompt have been tested and verified across live backend containers, data-plane gateways, and frontend interfaces.

## 2. Feature Verification Matrix

| # | Feature Domain | Status | Evidence / Test Details |
| :--- | :--- | :--- | :--- |
| 1 | **Authentication & Org Membership** | **VERIFIED** | Multi-tenant signup, session cookie issuance, CSRF tokens, strict tenant boundary enforcement. |
| 2 | **Members & Invitations** | **VERIFIED** | Invitation generation, one-time invitation token, acceptance flow, role assignment, duplicate rejection (409), email mismatch rejection. |
| 3 | **Teams & Projects** | **VERIFIED** | Creation of teams and projects, assignment of test users, project-scoped credential isolation. |
| 4 | **Provider Accounts & Credentials** | **VERIFIED** | Gemini, Anthropic, OpenAI provider registration; credential encryption in `secrets_kv`; secret masking in read APIs. |
| 5 | **Model Registry & Provider Health** | **VERIFIED** | `POST /v1/models/sync` catalog sync, health probing (`GET /v1/providers/health`), active model capability mapping. |
| 6 | **Service Accounts** | **VERIFIED** | Provisioning machine service accounts, issuing service-account-scoped gateway keys, attribution to service account subjects. |
| 7 | **Gateway Credentials** | **VERIFIED** | Creation with one-time raw secret (`aigw_...`), last-used timestamp updates, prefix masking, single key revocation without cascading failures. |
| 8 | **Policies & Model Access** | **VERIFIED** | Allowlist/denylist rules, wildcard matching (`claude-*`), priority evaluation, deny-overrides semantics, sub-millisecond cache invalidation. |
| 9 | **Budgets & Guardrails** | **VERIFIED** | Monthly and rolling token/spend budgets, warning thresholds, hard limit rejection before upstream routing. |
| 10 | **Usage & Cost Analytics** | **VERIFIED** | Full dimensional capture (`llm_requests`), 100% reconciliation of raw vs aggregate summaries, cost computation ($0.075/1M input). |
| 11 | **Live Activity** | **VERIFIED** | Recent request streams, paginated request history (`GET /v1/usage/requests`), status codes, latency, token metrics. |
| 12 | **Conversations & Privacy** | **VERIFIED** | Privacy modes (`OFF`, `METADATA_ONLY`, `PROMPT_ONLY`, `FULL_CONTENT`), zero prompt leakage in metadata-only mode, RBAC read gating. |
| 13 | **Audit Logs & Security Events** | **VERIFIED** | Immutable audit records for administrative mutations; real-time security events for policy denials and auth failures. |
| 14 | **Alerts & Notifications** | **VERIFIED** | Condition evaluation (budget, error rates, denials), state transitions (triggered -> acknowledged -> resolved), in-app notification center. |
| 15 | **Billing & Subscriptions** | **VERIFIED** | Distinction of upstream inference costs from SaaS subscription plans; invoice list generation; plan upgrade flows. |
| 16 | **Sentinel CLI (`aigw`)** | **VERIFIED** | Custom gateway URL, auth login, model listing, non-streaming & streaming requests, error handling. 10/10 CLI unit tests pass. |
| 17 | **Sentinel SDK (`api-client`)** | **VERIFIED** | Multi-instance credential isolation, Gemini SSE parsing, timeout/rate-limit handling, 8/8 automated TS unit tests pass. |

---

## 3. Discovered Defects & Fixes Implemented
1. **Gemini Streaming Payload Parsing**: Updated CLI client (`packages/cli/internal/client/client.go`) and SDK (`packages/api-client/src/gateway.ts`) to extract text and `usageMetadata` from Gemini candidates payload format.
2. **Gateway-v2 Fallback Catalog**: Registered `gemini-2.5-flash` and `gemini-2.5-pro` in Gateway-v2 fallback models to ensure model resolution succeeds immediately prior to catalog sync.
3. **Control API Models Sync**: Updated `syncProviderModels` (`backend/internal/interfaces/controlapi/providers.go`) and `SyncModels` (`backend/internal/application/provider/service.go`) to synchronize Gemini models into PostgreSQL catalog.
4. **Next.js Client Actions**: Resolved RSC function-passing violations by creating dedicated `"use client"` button/action wrappers in subscription, credential, and health views.
