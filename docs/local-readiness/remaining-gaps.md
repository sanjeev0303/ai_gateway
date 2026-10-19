# Sentinel AI Gateway — Remaining Gaps & Known Limitations

**Evaluation Date**: 2026-10-19  
**Status**: **READY WITH DOCUMENTED LIMITATIONS**  
**Repository Revision**: `HEAD` (`main`)

---

## 1. Scope & Context

This document provides a transparent assessment of architectural boundaries, unintegrated external capabilities, and known limitations in the local development build of Sentinel AI Gateway. In compliance with repository verification standards, no simulated or mocked behavior is represented as production-ready cloud infrastructure.

---

## 2. Summary of Capabilities & Known Boundaries

| Capability Area | Local Implementation Status | Production Gap / Architecture Boundary |
| :--- | :--- | :--- |
| **Multi-Provider Adapters** | **Complete** (Gemini, OpenAI, Anthropic, Hugging Face) | Direct live inference requires valid external cloud API keys in `.env` |
| **Model Discovery** | **Complete** (Authentic discovery endpoints, dynamic sync) | Live discovery skips when provider cloud keys are not configured |
| **Sentinel CLI** | **Complete** (Dynamic menus, provider-gated commands, SSRF protection) | Non-interactive CI environments render linear text without ANSI TTY cursor repositioning |
| **Dual-Gateway Billing** | **Complete** (Stripe & Razorpay adapters, modes, webhook HMAC, idempotency) | Merchant payout settlements and production credit card transactions require live provider accounts |
| **Tenant & User Isolation** | **Complete** (PostgreSQL RLS, RBAC, actor context) | Cloud KMS secret envelope encryption not integrated in local environment |
| **Redis Rate & Budget Limits** | **Complete** (Lua sliding window, ZSet+Hash reservations, outage policies) | Multi-region distributed Redis active-active replication not configured |
| **Event Bus & Worker** | **Complete** (NATS JetStream, Consumer Groups, deduplication) | Multi-cluster cross-datacenter event mirroring not tested locally |
| **Observability & Analytics** | **Complete** (Redis telemetry, usage rollups, dashboard widgets) | Remote OpenTelemetry OTLP collector export not active in default profile |

---

## 3. Detailed Technical Limitations

### 3.1 Live External Cloud Provider Calls
- **Current State**: Sentinel includes production-grade provider adapters for Anthropic, OpenAI, Google Gemini, and Hugging Face (`backend/internal/infrastructure/provider/*`). Deterministic mock providers are used in automated test suites and local CI.
- **Limitation**: Direct live inference against external cloud LLM APIs requires a valid external API key (e.g., `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `HUGGINGFACE_API_KEY`). Without these secrets configured in `.env`, upstream network calls are blocked at the provider boundary.
- **Impact**: Hermetic local test suites execute against mock providers and pass 100%. Live provider calls are cleanly skipped in offline or unconfigured environments.

### 3.2 Live Payment Processor Settlements
- **Current State**: Dual-gateway billing supports `disabled`, `simulation`, and `provider_test` modes. In `provider_test` mode, user-provided test credentials (`sk_test_...`, `rzp_test_...`) interact with provider sandbox APIs. Webhook signatures and idempotency are fully verified.
- **Limitation**: Production credit card processing, automated ACH debit collections, and live merchant payouts require live merchant accounts (`sk_live_...`) and production domain webhooks.
- **Impact**: Billing operates safely in test and simulation modes locally without moving real currency.

### 3.3 Enterprise SSO & Federated Identity
- **Current State**: Standard email/password authentication with PBKDF2-HMAC-SHA256 (600,000 iterations), constant-time verification, session rotation, and CSRF protection is fully implemented.
- **Limitation**: Enterprise Single Sign-On (SAML 2.0, Okta, Azure AD OIDC) is not implemented in the local development profile.

---

## 4. Operational Risk Summary

| Risk | Likelihood | Impact | Current Defense / Mitigation |
| :--- | :--- | :--- | :--- |
| Inadvertent master key exposure | Low | High | Strict key masking; raw keys returned once at creation; headers sanitized in logs |
| Database outage during streaming | Low | Medium | Gateway spools telemetry to disk/NATS; worker retries with exponential backoff |
| Double counting on worker replay | Medium | High | `request_id` unique constraint on `llm_requests`; atomic `ON CONFLICT DO NOTHING` |
| Upstream provider rate limits | Medium | Medium | Fallback model routing, upstream status code mapping, client retry headers |
| Redis connectivity interruption | Low | Medium | Explicit `AIGW_RATE_LIMIT_FAIL_CLOSED` and `AIGW_BUDGET_FAIL_CLOSED` flags |
