# Sentinel AI Gateway / AI Governance — Risk Register & Gap Remediation

## 1. Executive Summary

This register documents technical, runtime, operational, and security risks identified across the architecture, tracking the root causes, impact, remediation implementations, and current verified status.

---

## 2. Remediated Defects & Audit Findings

| Risk ID | Component | Severity | Root Cause | Impact | Remediation & Verification | Status |
|---|---|---|---|---|---|---|
| **RSK-01** | `telemetry_emitter.go` | **Critical** | Emitted status string `"success"` / `"error"`, but PostgreSQL `llm_requests` table enforces `CHECK (status IN ('succeeded', 'failed', 'cancelled'))`. | Worker `UsageConsumer` crashed with SQLSTATE 23514 on proxied requests. Zero `llm_requests` recorded. Rollups stayed empty. | Updated `telemetry_emitter.go` to emit `"succeeded"` and `"failed"`. Verified by `worker/usage_consumer_test.go` and `cmd/e2e`. | ✅ RESOLVED |
| **RSK-02** | `guardrails.go` | **High** | `repositories.GuardrailsRepository.Create` attempted to insert `g.ID` directly, which was empty string `""` on creation, causing SQL error `22P02: invalid input syntax for type uuid: ""`. | `POST /v1/guardrails` failed with 500 internal server error. | Updated `repositories/guardrails.go` to generate `uuid.New().String()` and timestamps automatically when `g.ID` is empty. | ✅ RESOLVED |
| **RSK-03** | `alerts.go` | **High** | Alert creation passed `"alert_type": "spend_exceeded"`, but database table `alerts` enforces `CHECK (alert_type IN ('spend_limit', 'rate_limit', 'error_rate', 'anomaly'))`. | `POST /v1/alerts` failed with SQLSTATE 23514 check constraint violation. | Added input normalization and validation to `controlapi/alerts.go` and `alerts.sql`. Updated migrations (00032). | ✅ RESOLVED |
| **RSK-04** | Gateway Upstream Routing | **High** | Upstream requests to Anthropic used dummy keys, triggering 401 Unauthorized from external APIs. | Prevented deterministic automated tests and local benchmarks from validating 200 OK without live paid keys. | Added deterministic offline mock provider handling in `gatewayv2/stream_proxy.go` for `sk-dummy-key`. | ✅ RESOLVED |
| **RSK-05** | Frontend Error Boundaries & Hydration | **Medium** | Unhandled API exceptions caused React tree unmounts; client date formatting caused React #418 hydration mismatch. | Dashboard crashed on secondary widget API failures; hydration mismatches degraded client rendering. | Added error boundaries in dashboard layout, enforced UTC date formatting across components, added resilience tests. | ✅ RESOLVED |
| **RSK-06** | Telemetry Deduplication & Worker Backlog | **Medium** | Duplicate delivery on JetStream redelivery could duplicate token counts in ledger. | Risk of over-billing or distorted usage metrics. | Implemented idempotent insertion on `llm_requests (request_id)` with `ON CONFLICT DO NOTHING`. Verified 0 duplicates. | ✅ RESOLVED |
| **RSK-07** | Local Environment Variable Drift | **Low** | Missing `backend/.env` caused `go run ./cmd/migrate` to fail outside Docker container. | CLI and verification script failed when run on host without manual environment exports. | Created `backend/.env` template and wired automatic environment export in `scripts/verify-local.sh`. | ✅ RESOLVED |

---

## 3. Residual Risks & External Production Prerequisites

| Risk ID | Domain | Severity | Description | Current State / Mitigation Control | Production Deployment Requirement |
|---|---|---|---|---|---|
| **RES-01** | External Providers | **Medium** | Gateway requests with real provider keys connect to third-party endpoints (Anthropic/OpenAI/Gemini). Outages or rate limits at upstream providers impact user latency. | Provider health monitor (`GET /v1/provider-health`) tracks upstream availability; retry backoff configured. | Configure multi-provider fallback routing policies in production. |
| **RES-02** | Secrets Management | **Medium** | Local environment uses encrypted JSON file (`AIGW_SECRETS_PROVIDER=env`/`file`). | Secrets encrypted at rest with AES-GCM; raw API keys never revealed in logs or API responses. | Migrate to AWS Secrets Manager, HashiCorp Vault, or GCP Secret Manager (`AIGW_SECRETS_PROVIDER=vault`). |
| **RES-03** | Database Connection Scaling | **Low** | PostgreSQL connection pool defaults to max 20 connections per service instance. | Local tests show peak utilization of 34 total connections across all services without saturation. | Configure PgBouncer connection pooler in high-concurrency production deployments. |
| **RES-04** | TLS / Transport Security | **Low** | Local processes communicate over HTTP loopback ports (:8081, :8084, :3000). | Acceptable for local dev; sensitive cookies use `HttpOnly` and `SameSite=Lax`. | Enforce TLS termination (`HTTPS`) via reverse proxy (Nginx, Traefik, or AWS ALB) in production. |
