# Sentinel AI Gateway — Remaining Gaps Implementation & Verification Plan

**Evaluation Date**: 2026-10-21  
**Status**: **IN PROGRESS**  
**Repository Revision**: `HEAD` (`main`)

---

## 1. Overview & Audit Scope

This document details all remaining implementation, integration, contract, and verification gaps identified in the Sentinel AI Gateway repository following the Phase 3 Pricing Management core implementation.

Every finding specifies the exact source file, root cause, affected workflow, severity, proposed fix, and required regression test.

---

## 2. Identified Gaps and Implementation Roadmap

### Finding 1: OpenAPI 3.1 Specification Missing Pricing Routes & Schemas
- **Exact Source File**: `backend/api/openapi/openapi.yaml`
- **Root Cause**: The `/v1/pricing` route group (`GET /v1/pricing`, `POST /v1/pricing`, `GET /v1/pricing/{idOrSlug}`, `POST /v1/pricing/{idOrSlug}/versions`, `GET /v1/pricing/{idOrSlug}/versions/{version}`, `POST /v1/pricing/calculate-cost`) implemented in `backend/internal/interfaces/controlapi/pricing.go` is not documented in `openapi.yaml`.
- **Affected Workflow**: OpenAPI compliance, automated contract generation (`@ai-governance/contracts`), developer documentation (`/app/developer/api-docs`), and SDK contract alignment.
- **Severity**: **Medium**
- **Proposed Change**: Add complete path operations, request bodies, query/path parameters, responses (`200`, `201`, `400`, `401`, `403`, `404`), and schema definitions (`PricingModel`, `PricingVersion`, `PricingModelDetail`, `CreatePricingModelRequest`, `CreatePricingVersionRequest`, `CalculateCostRequest`, `CalculateCostResponse`) to `backend/api/openapi/openapi.yaml`.
- **Required Regression Test**: `npm run generate --workspace=@ai-governance/contracts && npm run typecheck --workspace=@ai-governance/contracts`.

---

### Finding 2: Generated TypeScript Contracts Out of Sync with Pricing Model
- **Exact Source File**: `packages/contracts/src/generated.ts`
- **Root Cause**: `generated.ts` was generated before `/v1/pricing` was added, so TypeScript contracts workspace does not expose pricing types.
- **Affected Workflow**: Type-safe consumption of pricing APIs across frontend, SDK, and third-party consumers.
- **Severity**: **Medium**
- **Proposed Change**: Regenerate `packages/contracts/src/generated.ts` from `backend/api/openapi/openapi.yaml` using `openapi-typescript`.
- **Required Regression Test**: `npm run test --workspace=@ai-governance/contracts`.

---

### Finding 3: Next.js Web Dashboard Missing Pricing Management UI
- **Exact Source Files**:
  - `app/src/lib/constants.ts`
  - `app/src/app/(dashboard)/app/platform/pricing/page.tsx`
  - `app/src/app/(dashboard)/app/platform/pricing/client.tsx`
  - `app/src/__tests__/pricing_management.test.ts`
- **Root Cause**: Organization administrators and billing administrators have no dashboard UI interface to inspect model price sheets, view token pricing rates (input/output/cached micros), add new pricing models, or update pricing versions.
- **Affected Workflow**: Organization token cost governance and administrator pricing visibility in the Next.js web application.
- **Severity**: **Medium**
- **Proposed Change**:
  1. Add Pricing link under the "Platform" section in `DASHBOARD_NAV` (`app/src/lib/constants.ts`).
  2. Implement server-rendered page shell `/app/platform/pricing/page.tsx` with role checking and data loading.
  3. Implement interactive client component `/app/platform/pricing/client.tsx` with search, filter, active price sheet version badges, token rate cards (micro-dollar and USD/1M token equivalents), create model dialog, and new version creation modal.
  4. Write comprehensive frontend unit tests in `app/src/__tests__/pricing_management.test.ts`.
- **Required Regression Test**: `npm --prefix app test`, `npm --prefix app run typecheck`, `npm --prefix app run build`.

---

### Finding 4: Local Verification Script Host Docker Socket Resilience
- **Exact Source File**: `scripts/verify-local.sh`
- **Root Cause**: Check 4 in `verify-local.sh` strictly expects Docker containers to be running and marks the check as FAILED when Docker Desktop daemon socket is inactive, even though all backend tests are 100% hermetic and do not require Docker.
- **Affected Workflow**: Local verification on developer machines without active Docker daemon.
- **Severity**: **Low**
- **Proposed Change**: Update `scripts/verify-local.sh` check 4 to detect when the Docker daemon socket is unavailable and designate the status as `BLOCKED` with actionable diagnostics rather than failing hermetic repository validation.
- **Required Regression Test**: `bash scripts/verify-local.sh`.

---

## 3. Execution Order

1. **Step 1**: Update `backend/api/openapi/openapi.yaml` with `/v1/pricing` endpoints and schemas.
2. **Step 2**: Regenerate `packages/contracts/src/generated.ts` and verify contract typechecking.
3. **Step 3**: Implement Next.js Dashboard Pricing UI (`app/src/app/(dashboard)/app/platform/pricing/*`), navigation, and tests.
4. **Step 4**: Update `scripts/verify-local.sh` to handle inactive Docker socket gracefully.
5. **Step 5**: Run full verification suite (Go backend, CLI, contracts, SDK, frontend) and confirm 100% pass rate.
6. **Step 6**: Update `docs/local-readiness/current-baseline.md` and `docs/architecture/completeness-matrix.md`.
