# Sentinel AI Gateway — Remaining Gaps and Known Limitations

## Scope and Context

This document provides a transparent, objective assessment of architectural boundaries, unintegrated external capabilities, and known limitations in the local development build of Sentinel AI Gateway. In compliance with repository verification standards, no simulated or mocked behavior is represented as production-ready cloud infrastructure.

---

## 1. Summary of System Capabilities

| Capability Area | Local Implementation Status | Production Gap / Architecture Boundary |
|---|---|---|
| **Multi-Tenant Isolation** | **Complete** (PostgreSQL FORCE RLS, Actor Context) | Cloud KMS secret envelope encryption not integrated locally |
| **Authentication & RBAC** | **Complete** (PBKDF2-HMAC-SHA256, Session Cookies) | Enterprise SAML 2.0 / OIDC identity provider integration not configured |
| **Gateway Request Proxy** | **Complete** (Streaming SSE, Anthropic/OpenAI protocol) | Multi-region edge deployment and Anycast DNS routing not configured |
| **Governance & Policies** | **Complete** (Precedence hierarchy, <10ms cache bound) | Dynamic policy authoring DSL (e.g. OPA/Rego) not embedded |
| **Content Guardrails** | **Complete** (PII, Secrets, Injection, Token Limits) | Cloud ML-based toxicity and embedding classifiers require external API |
| **Usage & Cost Metering** | **Complete** (Exact micros arithmetic, worker rollups) | External billing settlement & payment processor (Stripe) not integrated |
| **Event Bus & Worker** | **Complete** (NATS JetStream, Consumer Group, Deduplication) | Multi-cluster distributed event replication not tested locally |
| **Telemetry & Observability** | **Complete** (Redis ring buffer, timeseries, heatmaps) | OpenTelemetry OTLP remote collector export not active in default profile |

---

## 2. Detailed Technical Gaps

### 2.1 Live External Cloud Provider Calls
- **Current State**: Sentinel includes production-grade provider adapters for Anthropic, OpenAI, and Google Gemini (`backend/internal/infrastructure/provider/*`). Deterministic mock providers are used in automated test suites and local CI.
- **Limitation**: Direct live inference against external cloud LLM APIs requires a valid external API key (e.g., `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`). Without these secrets configured in `.env`, upstream network calls are blocked at the provider boundary.
- **Impact**: Feature matrix marks Live External Inference as `BLOCKED` in offline/airgapped environments.

### 2.2 Payment Gateway and Financial Settlement
- **Current State**: The billing subsystem implements internal usage cost calculation, plan tiers (`developer`, `scale`), and invoice generation with itemized token consumption.
- **Limitation**: No payment gateway (such as Stripe or Adyen) is integrated. The application does not process credit cards, manage bank settlements, or initiate automated collection webhooks.
- **Impact**: Billing operates in deterministic usage-reporting mode. Invoices represent internal cost accounting and quota tracking.

### 2.3 Enterprise SSO & Federated Identity
- **Current State**: Standard email/password authentication with PBKDF2-HMAC-SHA256 (600,000 iterations), constant-time verification, session rotation, and CSRF protection is fully implemented.
- **Limitation**: Enterprise Single Sign-On (SAML 2.0, Okta, Azure AD OIDC) is not implemented in the local development profile.

### 2.4 CLI Output Parsing Edge Case
- **Current State**: Sentinel CLI (`packages/cli`) fully supports authentication, project listing, model inspection, and interactive gateway streaming.
- **Limitation**: Gemini-formatted completion responses with nested candidates containing null text segments require explicit fallback handling in `packages/cli/client_test.go:278`.

### 2.5 Air-Gapped Container Image Rebuilding
- **Current State**: Docker Compose pre-built container images run reliably offline.
- **Limitation**: Attempting a complete cold Docker build (`docker compose build`) without local npm/Go package caching requires internet access to fetch Alpine packages and npm modules; network interruptions trigger `ECONNRESET`.
- **Mitigation**: Pre-built container images are preserved and managed via `./scripts/dev-ctl.sh start` using `--no-build`.

---

## 3. Operational Risk Summary

| Risk | Likelihood | Impact | Current Defense / Mitigation |
|---|---|---|---|
| Inadvertent master key exposure | Low | High | Strict key masking; raw keys returned once at creation; headers sanitized in logs |
| Database outage during streaming | Low | Medium | Gateway spools telemetry to disk/NATS; worker retries with exponential backoff |
| Double counting on worker replay | Medium | High | `request_id` unique constraint on `llm_requests`; atomic `ON CONFLICT DO NOTHING` |
| Upstream provider rate limits | Medium | Medium | Fallback model routing, upstream status code mapping, client retry headers |
