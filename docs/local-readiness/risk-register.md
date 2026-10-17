# Sentinel AI Gateway / AI Governance — Risk Register & Gap Remediation

**Audit Date**: 2026-10-17  
**Scope**: Local Development, Automated Test Baselines, and Runtime Infrastructure  
**Status**: All Local Baseline Defects Remediated; Zero Blocking Vulnerabilities

---

## 1. Active Architecture Boundaries & Operational Gaps

| Gap ID | Component | Severity | Description | Local Mitigation | Production Requirement |
|---|---|---|---|---|---|
| **GAP-01** | External Inference | Medium (P2) | Direct inference against live external LLM providers (Gemini, Anthropic, OpenAI) requires runtime API keys (`GEMINI_API_KEY`). | Local test suites and E2E journeys execute against deterministic in-memory mock adapters; live provider test is skipped gracefully when unconfigured. | Runtime injection of production API keys via cloud secrets manager. |
| **GAP-02** | Billing & Invoicing | Low (P3) | Billing subsystem implements usage cost accounting, tier quotas, and invoice itemization, but no payment gateway (Stripe/Adyen) is integrated. | Application runs in self-hosted usage-metering mode; invoices reflect internal quota tracking and usage breakdown. | Integrate payment processor SDK (Stripe Billing/Customer Portal) for financial settlement. |
| **GAP-03** | Enterprise SSO | Low (P3) | Authentication relies on local PBKDF2-HMAC-SHA256 hashed credentials and session cookies; SAML 2.0 / OIDC is not configured. | Dev-login and session cookies enable complete local multi-user testing and RBAC enforcement. | Wire OAuth 2.0 / SAML 2.0 identity federation (Okta, Azure AD, Auth0) for enterprise single sign-on. |
| **GAP-04** | Air-Gapped Image Build | Low (P3) | Running `docker compose build` cold without network cache requires internet access for Alpine and npm packages. | Pre-built local container images are preserved and managed via `./scripts/dev-ctl.sh start` using `--no-build` defaults. | Maintain internal container image registry (ECR, GCR, Harbor) for air-gapped CI environments. |

---

## 2. Remediated Historical Defects

| Defect ID | Component | Severity | Root Cause | Remediated Implementation & Evidence | Verified Status |
|---|---|---|---|---|---|
| **REM-01** | `tenant_resolver_test.go` | High | Normalization added trailing slash `"/"` while test asserted non-trailing URL. | Aligned test assertions in `tenant_resolver_test.go` with canonical trailing slash. | ✅ RESOLVED |
| **REM-02** | `client_test.go` (CLI) | High | Mock returned raw Gemini JSON payload while CLI routes via OpenAI-compatible endpoint. | Aligned mock responses in `packages/cli/internal/client/client_test.go` to standard completion format. | ✅ RESOLVED |
| **REM-03** | `packages/api-client` | Medium | Missing `@types/node` caused monorepo typecheck to fail on `node:test`. | Added `@types/node` to `packages/api-client/package.json` and updated `tsconfig.json`. | ✅ RESOLVED |
| **REM-04** | `app/package.json` | Medium | Root script expected `typecheck` script in `app/`. | Added `"typecheck": "tsc --noEmit"` to `app/package.json`. | ✅ RESOLVED |
| **REM-05** | `config.go` Validation | High | Insecure development settings and placeholder secrets could leak into production. | Implemented strict startup validation in `config.go` requiring `AIGW_ALLOW_INSECURE_DEV=true` in dev and rejecting placeholders in production. | ✅ RESOLVED |
| **REM-06** | `telemetry_emitter.go` | Critical | Status string mismatch (`"success"` vs `"succeeded"`) violating DB check constraint. | Aligned status string constants in `gatewayv2/types.go` and `telemetry_emitter.go`. | ✅ RESOLVED |
| **REM-07** | `guardrails.go` | High | Empty UUID string `""` passed on creation causing PostgreSQL syntax error `22P02`. | Auto-generate UUID in `repositories/guardrails.go` when ID is empty. | ✅ RESOLVED |
| **REM-08** | `alerts.go` | High | Alert type string `"spend_exceeded"` rejected by DB check constraint (`'spend_limit'`). | Normalized enum in `controlapi/alerts.go` and updated migration 00032. | ✅ RESOLVED |
| **REM-09** | `operations.go` | High | Zero-length timeseries when traffic only existed on a single day, collapsing SVG chart paths. | Added backend zero-filling over requested time range in `repositories/operations.go`. | ✅ RESOLVED |
| **REM-10** | `sanju_live/main.go` | Critical | Hardcoded Google Gemini API key blocked by GitHub push protection. | Replaced with `os.Getenv("GEMINI_API_KEY")` lookup. | ✅ RESOLVED |

---

## 3. Residual Operational Risks

| Risk ID | Domain | Severity | Description | Local Mitigation | Production Requirement |
|---|---|---|---|---|---|
| **RES-01** | External Providers | Medium | Upstream LLM provider latency spikes or rate limits propagate to client. | Gateway-v2 stream proxy implements timeout, context cancellation (`ctx.Done()`), and fallback error mapping (429, 504). | Configure multi-provider load balancing and automatic failover. |
| **RES-02** | Secrets Storage | Medium | Local secrets stored in encrypted local JSON file (`/data/secrets.json`). | AES-GCM encryption with key on secrets volume; secrets never revealed in API responses. | Migrate to AWS Secrets Manager, HashiCorp Vault, or Google Secret Manager. |
| **RES-03** | Connection Saturation | Low | PostgreSQL connection pool defaults to max 20 connections per service. | Peak local concurrent testing utilized 35 connections across all 3 services. | Deploy PgBouncer in front of PostgreSQL for high-traffic environments. |
| **RES-04** | Transport Security | Low | Local processes communicate over unencrypted HTTP loopback ports. | Acceptable for local dev; cookies enforce `HttpOnly` and `SameSite=Lax`. | Enforce TLS reverse proxy (Nginx, Traefik, or AWS ALB) in production. |
