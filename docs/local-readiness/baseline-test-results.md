# Sentinel AI Gateway / AI Governance — Baseline Test Execution Results

**Date**: 2026-10-16  
**Environment**: Local Docker Compose Stack (`postgres:16`, `redis:7`, `nats:2`, `control-api`, `gateway-v2`, `worker`, `app:3000`)  
**Commit**: `c8e84d8` (clean working tree)

---

## 1. Test Execution Summary

| Suite / Subsystem | Command | Passed | Failed | Skipped / Blocked | Status |
|---|---|---|---|---|---|
| **Go Backend Unit & Repositories** | `go test ./...` in `backend/` | 38 pkgs | 1 pkg | 0 | ⚠️ PARTIAL |
| **Go Gateway v2 (Tenant Resolver)** | `go test ./internal/interfaces/gatewayv2/...` | 13 tests | 2 tests | 0 | ❌ FAIL |
| **Sentinel CLI (`aigw`)** | `go test ./...` in `packages/cli/` | 10 tests | 1 test | 0 | ⚠️ PARTIAL |
| **TypeScript SDK (`@ai-governance/api-client`)** | `npm test` in `packages/api-client/` | 8 tests | 0 | 0 | ✅ PASS |
| **TypeScript SDK Typecheck** | `tsc --noEmit` in `packages/api-client/` | 0 | 2 errors | 0 | ❌ FAIL |
| **Frontend Unit Tests** | `npm test` in `app/` | 24 tests | 0 | 0 | ✅ PASS |
| **Frontend TypeScript Typecheck** | `npx tsc --noEmit` in `app/` | Clean | 0 errors | 0 | ✅ PASS |
| **Frontend ESLint Audit** | `npm run lint` in `app/` | 0 errors | 0 errors | 474 warnings | ✅ PASS |
| **Frontend Production Build** | `npm run build` in `app/` | 49 routes | 0 errors | 0 | ✅ PASS |
| **End-to-End User Journeys** | `go run ./cmd/e2e` in `backend/` | 14 journeys | 0 | 0 | ✅ PASS |
| **Live External Upstream LLM Calls** | Live call without `GEMINI_API_KEY` | 0 | 0 | Blocked | ⏸️ BLOCKED |

---

## 2. Failed Tests & Exact Root Causes

### 2.1 Backend: `gatewayv2/tenant_resolver_test.go`
- **Failing Tests**: `TestTenantResolver_HappyPath`, `TestTenantResolver_CustomBaseURL`
- **Command**: `go test -v ./internal/interfaces/gatewayv2`
- **Output**:
  ```text
  --- FAIL: TestTenantResolver_HappyPath (0.00s)
      tenant_resolver_test.go:64: expected default anthropic base URL, got "https://api.anthropic.com/"
  --- FAIL: TestTenantResolver_CustomBaseURL (0.00s)
      tenant_resolver_test.go:78: expected account base URL override, got "https://eu.api.anthropic.com/"
  ```
- **Evidence**: `backend/internal/interfaces/gatewayv2/tenant_resolver.go:157-159`
  ```go
  if !strings.HasSuffix(baseURL, "/") {
      baseURL += "/"
  }
  ```
- **Root Cause**: `tenant_resolver.go` normalizes all provider base URLs to have a trailing slash `"/"` so downstream path concatenation cannot result in malformed URLs. The unit test asserts strict equality without a trailing slash (`"https://api.anthropic.com"` and `"https://eu.api.anthropic.com"`).
- **Proposed Fix**: Update unit test assertions in `tenant_resolver_test.go:63` and `77` to expect the normalized trailing slash, or normalize comparison with `strings.TrimRight(upstream.BaseURL, "/")`.
- **Priority**: High (P1)

---

### 2.2 CLI: `packages/cli/internal/client/client_test.go`
- **Failing Test**: `TestGatewayClient_SendRequest_Gemini`
- **Command**: `go test -v ./internal/client/...` in `packages/cli/`
- **Output**:
  ```text
  === RUN   TestGatewayClient_SendRequest_Gemini
      client_test.go:278: unexpected output: ""
      client_test.go:281: unexpected tokens: &{RequestID:req-nonstream-gemini StatusCode:200 Duration:722.563µs InputTokens:0 OutputTokens:0 TotalTokens:0 Model:gemini-2.5-flash}
  --- FAIL: TestGatewayClient_SendRequest_Gemini (0.00s)
  ```
- **Evidence**: `packages/cli/internal/client/client.go:245-250` and `client_test.go:265`
- **Root Cause**: `SendRequest` routes `gemini-` models through `/v1/chat/completions` (OpenAI format), while the mock HTTP server in `client_test.go:265` returns Google Generative Language raw candidates payload (`{"candidates":[{"content":{"parts":[{"text":"Gemini non-streaming response"}]}}]}`).
- **Proposed Fix**: Align the unit test mock payload in `client_test.go` to return standard OpenAI-compatible completions response as expected by `/v1/chat/completions`.
- **Priority**: High (P1)

---

### 2.3 SDK Typecheck: `packages/api-client/src/gateway.test.ts`
- **Command**: `npm run typecheck` in workspace root / `tsc --noEmit` in `packages/api-client`
- **Output**:
  ```text
  src/gateway.test.ts:1:30 - error TS2307: Cannot find module 'node:test' or its corresponding type declarations.
  1 import { describe, it } from "node:test";
  src/gateway.test.ts:2:20 - error TS2307: Cannot find module 'node:assert/strict' or its corresponding type declarations.
  2 import assert from "node:assert/strict";
  ```
- **Root Cause**: `packages/api-client` devDependencies lacked `@types/node` and its `tsconfig.json` was not configured with `types: ["node"]`.
- **Proposed Fix**: Add `@types/node` to `packages/api-client/package.json` devDependencies.
- **Priority**: Medium (P2)

---

## 3. Blocked Tests

### 3.1 External Upstream Live LLM Provider Inference
- **Blocked Path**: `backend/cmd/sanju_live` and live Gemini calls via `gateway-v2:8084`
- **Condition**: Requires `GEMINI_API_KEY` set in host environment.
- **Current Behavior**: E2E suite falls back to deterministic registered provider credential and mock response path (`sk-dummy-key`). Live external upstream call is blocked until runtime key is exported.
- **Impact**: Zero impact on local governance, rate limits, budgets, token attribution, or dashboard testing.

---

## 4. Passing Test Breakdown

### 4.1 Go Backend Unit Tests (Passed)
- `internal/application/alert` (evaluator, notifications, service)
- `internal/application/billing` (subscriptions, invoices, services)
- `internal/application/conversation` (retention, privacy enforcement)
- `internal/application/identity` (auth, members, session tokens)
- `internal/application/identity/journey` (multi-tenant journeys)
- `internal/application/provider` (adapters, account credentials)
- `internal/application/securityevent` (event dispatching)
- `internal/application/usage` (analytics rollups, summaries)
- `internal/infrastructure/auth` (token manager, password hashes)
- `internal/infrastructure/eventbus` (JetStream, memory bus)
- `internal/infrastructure/postgres/repositories` (RLS, operations, resources)
- `internal/infrastructure/provider/{anthropic,gemini,openai}` (adapters)
- `internal/infrastructure/secrets/envstore` (AES-GCM encryption)
- `internal/interfaces/controlapi` (all CRUD routes, middleware, DTOs)
- `internal/interfaces/gateway` (legacy v1 handlers)
- `internal/interfaces/worker` (event consumers, retention jobs)

### 4.2 Frontend Unit Tests (Passed: 24/24)
- API client construction, query formatting, error status mapping (401, 403, 404).
- Credential lifecycle (one-time raw secret display, secret masking on list, instant revocation).
- Dashboard widget formatting (currency, compact tokens, percentages, UTC timestamps).
- Error resilience (secondary widget degradation handling).
- Cache invalidation and form validation.
- RBAC permissions (owner, admin, billing_admin, auditor, member).

### 4.3 End-to-End Master Execution Journeys (Passed: 14/14)
- Journey A: First-Time Organization Setup
- Journey B: Policy Configuration & Enforcement
- Journey C: Concurrent Traffic & Live Activity
- Journey D: Credential Lifecycle & Audit Trail
- Journey E: Budget Enforcement & Spend Limits
- Journey F: Guardrail Interception & Blocking
- Journey G: Usage Analytics & Executive Overview
- Organization Lifecycle (teams, projects, members, service accounts)
- Governance Lifecycle (model access allow/block, rate limiting)
- Provider Gateway Lifecycle (multi-provider routing, anti-spoofing)
- Usage Attribution Lifecycle (5-dimension attribution)
- Alerting, Privacy & Retention Lifecycle (modes OFF, METADATA_ONLY, PROMPT_ONLY, FULL_CONTENT)
- Complete 21-Step Acceptance Journey Validation
- Gemini Master Execution & Multi-User Telemetry
