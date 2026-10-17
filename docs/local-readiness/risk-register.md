# Sentinel AI Gateway / AI Governance — Risk Register & Gap Remediation

**Audit Date**: 2026-10-16  
**Scope**: Local Development, Automated Test Baselines, and Runtime Infrastructure

---

## 1. Active Gaps & Audit Findings (Prioritized)

| Gap ID | Component & Exact File Path | Severity & Priority | Evidence & Root Cause | Impact | Dependencies | Proposed Fix | Verification Method |
|---|---|---|---|---|---|---|---|
| **GAP-01** | `backend/internal/interfaces/gatewayv2/tenant_resolver_test.go:64,78` | **High** (P1) | `tenant_resolver.go:157-159` normalizes base URLs to have a trailing slash `"/"`, while `tenant_resolver_test.go` asserts strict equality against URLs without trailing slash (`"https://api.anthropic.com"`). | Backend unit test suite `gatewayv2` fails on local and CI execution. | Go standard library | Update test assertions in `tenant_resolver_test.go:64,78` to expect trailing slash or use `strings.TrimRight(upstream.BaseURL, "/")`. | `go test -v ./internal/interfaces/gatewayv2/...` |
| **GAP-02** | `packages/cli/internal/client/client_test.go:265,278` | **High** (P1) | `SendRequest` in `packages/cli/internal/client/client.go:245-250` routes `gemini-` models through `/v1/chat/completions` (OpenAI wire format), but `client_test.go:265` returns Google Gemini candidate payload (`{"candidates":[{"content":...}]}`). | CLI unit test suite `packages/cli/internal/client` fails during `go test ./...`. | CLI HTTP client | Update mock HTTP server in `client_test.go` to return standard OpenAI chat completion JSON matching `/v1/chat/completions` response structure. | `go test -v ./...` in `packages/cli/` |
| **GAP-03** | `packages/api-client/src/gateway.test.ts:1,2` | **Medium** (P2) | `gateway.test.ts` imports from `"node:test"` and `"node:assert/strict"`, but `packages/api-client/package.json` does not include `@types/node` in `devDependencies`. | `npm run typecheck` across the monorepo fails with `TS2307: Cannot find module 'node:test'`. | Node.js typings | Add `"@types/node": "^20"` to `packages/api-client/package.json` devDependencies and update `tsconfig.json`. | `npm run typecheck --workspace=@ai-governance/api-client` |
| **GAP-04** | `app/package.json` | **Medium** (P2) | Root `package.json` executes `npm run typecheck --workspaces`, but `app/package.json` has no `"typecheck"` script entry. | Monorepo root typecheck exits with code 1 (`Missing script: "typecheck"` in workspace `ai_governence`). | Next.js TypeScript compiler | Add `"typecheck": "tsc --noEmit"` to `app/package.json` scripts. | `npm run typecheck` in repository root |
| **GAP-05** | `deploy/compose/docker-compose.yml:43,59,61,62` | **Medium** (P2) | Permissive defaults: `AIGW_AUTH_SESSION_COOKIE_SECURE="false"`, `AIGW_AUTH_CSRF_ENABLED="false"`, and default passwords `change-me`. | Development defaults are unsuitable for public staging or production deployments. | Deployment configuration | Document strict production configuration overrides in runbook and add validation check preventing startup with default secrets when `AIGW_ENV=production`. | Automated configuration audit in `config_test.go` |
| **GAP-06** | `backend/cmd/sanju_live/main.go:24` | **Low** (P3) | External upstream inference calls require `GEMINI_API_KEY` set in the local shell environment. | Local developers without API keys are blocked from testing live external Google Gemini API. | External GCP account | Offline mock provider path exists in `gatewayv2/stream_proxy.go` (`sk-dummy-key`); document environment variable prerequisite in `docs/local-readiness/local-setup.md`. | `go run ./cmd/sanju_live` with valid key |

---

## 2. Remediated Historical Defects

| Defect ID | Component | Severity | Root Cause | Remediated Implementation & Evidence | Verified Status |
|---|---|---|---|---|---|
| **REM-01** | `telemetry_emitter.go` | Critical | Status string mismatch (`"success"` vs `"succeeded"`) violating DB check constraint. | Aligned status string constants in `gatewayv2/types.go` and `telemetry_emitter.go`. | ✅ RESOLVED |
| **REM-02** | `guardrails.go` | High | Empty UUID string `""` passed on creation causing PostgreSQL syntax error `22P02`. | Auto-generate UUID in `repositories/guardrails.go` when ID is empty. | ✅ RESOLVED |
| **REM-03** | `alerts.go` | High | Alert type string `"spend_exceeded"` rejected by DB check constraint (`'spend_limit'`). | Normalized enum in `controlapi/alerts.go` and updated migrations (00032). | ✅ RESOLVED |
| **REM-04** | `operations.go` | High | Zero-length timeseries when traffic only existed on a single day, collapsing SVG chart paths. | Added backend zero-filling over requested time range in `repositories/operations.go`. | ✅ RESOLVED |
| **REM-05** | `charts/utils.ts` | Medium | SVG single-point paths collapsed to 0 dimensions. | Added single-point circle rendering and area band fallbacks in `line-chart.tsx` and `area-chart.tsx`. | ✅ RESOLVED |
| **REM-06** | `govern/budgets/page.tsx` | Medium | Missing `useEffect` hook prevented client from loading budgets on component mount. | Added `useEffect` and `useCallback` fetching; verified active 2 budgets render in browser. | ✅ RESOLVED |
| **REM-07** | `sanju_live/main.go` | Critical | Hardcoded Google Gemini API key blocked by GitHub push protection. | Replaced with `os.Getenv("GEMINI_API_KEY")` lookup. | ✅ RESOLVED |

---

## 3. Residual Operational Risks

| Risk ID | Domain | Severity | Description | Local Mitigation | Production Requirement |
|---|---|---|---|---|---|
| **RES-01** | External Providers | Medium | Upstream LLM provider latency spikes or rate limits propagate to client. | Gateway-v2 stream proxy implements timeout, context cancellation (`ctx.Done()`), and fallback error mapping (429, 504). | Configure multi-provider load balancing and automatic failover in production. |
| **RES-02** | Secrets Storage | Medium | Local secrets stored in encrypted local JSON file (`/data/secrets.json`). | AES-GCM encryption with key on secrets volume; secrets never revealed in API responses. | Migrate to AWS Secrets Manager, HashiCorp Vault, or Google Secret Manager. |
| **RES-03** | Connection Saturation | Low | PostgreSQL connection pool defaults to max 20 connections per service. | Peak local concurrent testing utilized 34 connections across all 3 services. | Deploy PgBouncer in front of PostgreSQL for high-traffic environments. |
| **RES-04** | Transport Security | Low | Local processes communicate over unencrypted HTTP loopback ports. | Acceptable for local dev; cookies enforce `HttpOnly` and `SameSite=Lax`. | Enforce TLS reverse proxy (Nginx, Traefik, or AWS ALB) in production. |
