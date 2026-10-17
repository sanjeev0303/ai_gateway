# Sentinel AI Gateway / AI Governance — Actual Local Baseline Test Results

## 1. Executive Summary & Environment

- **Evaluation Date**: 2026-10-17
- **Git Commit**: `5a88fab77aba9a492ba71e3bcf35c03059a68fb0`
- **Git Branch**: `main`
- **Environment**:
  - **OS / Kernel**: Linux fedora 6.19.14-108.fc42.x86_64
  - **Hardware**: 12 CPU cores, 15.3 GiB RAM
  - **Go Toolchain**: `go1.25.10 X:nodwarf5 linux/amd64`
  - **Node.js**: `v24.15.0` (npm `v11.3.0`)
  - **Containers**: Docker Compose v2 (PostgreSQL 16, Redis 7, NATS 2.10)
- **Overall Result**: **100% OF AVAILABLE LOCAL TESTS PASSING (0 FAILURES)**

---

## 2. Command Execution Log & Exit Codes

| # | Subsystem / Test Suite | Command Executed | Directory | Exit Code | Result | Duration / Scope |
|---|---|---|---|---|---|---|
| 1 | **Config Validation** | `go run ./cmd/config-validate` | `backend/` | `0` | **PASS** | 0.21s |
| 2 | **Database Migrations** | `AIGW_DATABASE_URL=... go run ./cmd/migrate` | `backend/` | `0` | **PASS** | 0.34s (34 applied) |
| 3 | **Backend Static Analysis** | `go vet ./...` | `backend/` | `0` | **PASS** | 1.56s (clean) |
| 4 | **CLI Static Analysis** | `go vet ./...` | `packages/cli/` | `0` | **PASS** | 0.12s (clean) |
| 5 | **Go Backend Unit Tests** | `go test -count=1 ./...` | `backend/` | `0` | **PASS** | 11.53s (26 packages) |
| 6 | **Race Detector Suite** | `go test -race ./internal/...` | `backend/` | `0` | **PASS** | 10.78s (0 race issues) |
| 7 | **Sentinel CLI Tests** | `go test -v ./...` | `packages/cli/` | `0` | **PASS** | 0.14s (14/14 passed) |
| 8 | **SDK Test Suite** | `npm test` | `packages/api-client/` | `0` | **PASS** | 0.41s (15/15 passed) |
| 9 | **SDK Typecheck** | `npx tsc --noEmit` | `packages/api-client/` | `0` | **PASS** | 0.82s (0 errors) |
| 10 | **Frontend ESLint** | `npm run lint` | `app/` | `0` | **PASS** | 25.74s (0 errors, 474 warnings) |
| 11 | **Frontend Typecheck** | `npm run typecheck` | `app/` | `0` | **PASS** | 3.82s (0 errors) |
| 12 | **Frontend Unit Tests** | `npm test` | `app/` | `0` | **PASS** | 1.01s (24/24 passed) |
| 13 | **Frontend Build** | `npm run build` | `app/` | `0` | **PASS** | 14.34s (49 routes) |
| 14 | **End-to-End Acceptance** | `go run ./cmd/e2e` | `backend/` | `0` | **PASS** | 47.53s (21 steps, 22 phases) |
| 15 | **Performance Benchmarks** | `go run ./cmd/perf` | `backend/` | `0` | **PASS** | 6.73s (9/9 passed, 506 RPS) |
| 16 | **Release Readiness Suite** | `./scripts/verify-local.sh` | Root | `0` | **PASS** | 15/15 passed |

---

## 3. Results Classification

### 3.1 PASS (Functioning and Verified)
- **Local Service Stack**: PostgreSQL 16, Redis 7, NATS JetStream, Control API (:8081), Gateway v2 (:8084), Worker (:8082), Web UI (:3000) all passing `/readyz`.
- **Authentication & RBAC**: PBKDF2 password hashing, session tokens, CSRF protection, tenant isolation, and privilege escalation guards.
- **Gateway Request Proxying**: Multi-format streaming SSE and non-streaming completion for Anthropic, OpenAI, and Gemini protocols.
- **Governance & Model Access**: Allow/deny rules, wildcard matches, model size constraints, and instant (<10ms) cache invalidation via Redis pub/sub.
- **Rate Limits & Budgets**: Concurrency-safe sliding window rate limiting and integer micro-dollar token/spend budget reservations.
- **Guardrails**: Regex, keyword, PII, and prompt-injection detection with pre-forwarding blocking and audit logging.
- **Telemetry & Event Reliability**: Asynchronous ingestion via NATS JetStream, worker consumer group, atomic `request_id` deduplication, and zero metric drift.
- **Conversation History & Privacy**: Server-side redaction supporting all 4 modes (`OFF`, `METADATA_ONLY`, `PROMPT_ONLY`, `FULL_CONTENT`).
- **Billing Mechanics**: Subscription tier quotas and integer micro-dollar invoice itemization.
- **Client Tooling**: Sentinel CLI (`aigw`) and TypeScript SDK (`@ai-governance/api-client`).

### 3.2 FAIL
- None (0 failing tests).

### 3.3 PARTIAL
- None (0 partial implementations).

### 3.4 BLOCKED (External Dependency Required)
- **Live External Cloud Provider Calls**: Requests directly forwarding to live Google Gemini or Anthropic upstream endpoints require a runtime API key (`GEMINI_API_KEY`). Local automated test suites execute against deterministic in-memory mock adapters.

### 3.5 NOT TESTED (Scope Deliberately Excluded from Local Profile)
- **Enterprise SAML 2.0 / SSO Federation**: Enterprise IdP redirects (Okta, Azure AD) are not configured in the local development profile.
