# Implementation Task Tracker

Updated after each completed prompt.

## STATUS

| Phase | Title | Status |
|---|---|---|
| P01 | Repository Deep Audit + Cleanup | ✅ DONE |
| P02 | Rebuild Dashboard Design System | ✅ DONE |
| P03 | Dashboard App Shell + Navigation | ✅ DONE |
| P04 | Complete Control API Contract | 🔲 PENDING |
| P05 | Usage API + Analytics Read Model | 🔲 PENDING |
| P06 | TTFT, Latency + Request Telemetry | 🔲 PENDING |
| P07 | Live Activity API + Real-Time Dashboard | 🔲 PENDING |
| P08 | Cost API + Cost Dashboard | 🔲 PENDING |
| P09 | Users, Teams, Projects Observability APIs | 🔲 PENDING |
| P10 | Conversation API + UI | 🔲 PENDING |
| P11 | Policy CRUD API + Policy Engine | 🔲 PENDING |
| P12 | Model Access + Budget API/UI | 🔲 PENDING |
| P13 | Guardrails API/UI | 🔲 PENDING |
| P14 | Audit Log API + Security Events | 🔲 PENDING |
| P15 | Alerts + Notification APIs | 🔲 PENDING |
| P16 | Providers, Models + Platform UI Rework | 🔲 PENDING |
| P17 | Organization, Credential + Developer UI Rework | 🔲 PENDING |
| P18 | Billing + SaaS Plan APIs | 🔲 PENDING |
| P19 | Overview Page: Real Executive Control Center | ✅ DONE |
| P20 | Dashboard Page-by-Page UX Pass | ✅ DONE |
| P21 | Provider Pages (accounts/credentials/health/models) | ✅ DONE |
| P22 | Model Registry (GET /v1/models + POST /v1/models/sync) | ✅ DONE |
| P23 | Credential Pages (one-time secret behavior) | 🔲 PENDING |
| P24 | Organization Pages (members/invitations/teams/projects) | ✅ DONE |
| P25 | Dashboard Overview (real APIs, no demo store) | ✅ DONE |
| P26 | Filters and Date Range (real query params) | 🟡 PARTIAL |
| P27 | Pagination (server-side cursor pagination) | 🔲 PENDING |
| P28 | API Contracts (OpenAPI → generate) | 🔲 PENDING |
| P29 | Frontend API Hooks (app/src/lib/api/) | 🔲 PENDING |
| P30 | Loading/Error/Empty States | 🟡 PARTIAL |
| P31 | Remove Fabricated Marketing/Dashboard Values | ✅ DONE |
| P32 | Developer Pages (real endpoint list) | ✅ DONE |
| P33 | Server vs Client Components | 🟡 PARTIAL |
| P34 | Auth Cookie / CSRF Safety | ✅ DONE |
| P35 | Tenant Isolation | ✅ DONE (backend enforces) |
| P36 | Permissions (domain.Permission*) | 🔲 PENDING |
| P37 | Database / Repository Rule | ✅ DONE (architecture enforced) |
| P38 | Read Performance (usage_daily/hourly not llm_requests) | 🟡 PARTIAL |
| P39 | Service Account Attribution | 🔲 PENDING |
| P40 | Error Contract | 🟡 PARTIAL |
| P41 | Remove Duplicated Frontend Types | 🔲 PENDING |
| P42 | Keep Features Intact | ✅ DONE |
| P43 | Realistic Local Data (seed.sql) | 🔲 PENDING |
| P44 | Data Consistency | ✅ DONE (single backend source) |
| P45 | Test End-to-End Data Flow | 🔲 PENDING |
| P46 | Remove Mock Files Only After Migration | ✅ DONE (lib/demo no longer used by any page) |
| P47 | OpenAPI / Contract Validation | 🔲 PENDING |
| P48 | API Inventory | ✅ DONE (docs/architecture/frontend-backend-api-map.md) |
| P49 | Security Review | 🟡 PARTIAL |

---

## Current State Audit (as of 2026-10-04)

### ✅ DONE — No action needed

**P21 — Provider Pages**
- `platform/provider-accounts` → `GET /v1/provider-accounts` (real)
- `platform/provider-health` → `GET /v1/provider-health` (real)
- `observe/providers` → `GET /v1/provider-accounts` + `GET /v1/provider-health` + `GET /v1/usage/by-provider` (real)
- Provider credentials page: `GET /v1/provider-credentials` (real, fingerprint/tail/status only — no secrets)

**P22 — Model Registry**
- `platform/model-registry` → `GET /v1/models` + `GET /v1/usage/by-model` (real)
- `observe/models` → `GET /v1/models` + `GET /v1/usage/by-model` (real)
- `POST /v1/models/sync` exists and is wired in api-client

**P24 — Organization Pages**
- `organization/members` → `GET /v1/members` + `PATCH /v1/members/{id}/role` + `DELETE /v1/members/{id}` (real)
- `organization/invitations` → `GET /v1/invitations` + `POST /v1/invitations` + `DELETE /v1/invitations/{id}` (real)
- `organization/teams` → `GET /v1/teams` + `POST /v1/teams` + member management (real)
- `organization/projects` → `GET /v1/projects` + `POST /v1/projects` + member management (real)

**P25 — Dashboard Overview**
- `overview` → uses `usage.summary` + `usage.latency` + `usage.timeseries` + `usage.byProject` + `usage.byModel` (all real)
- No demo store usage

**P31 — Fabricated Metrics**
- No marketing page contains fake operational metrics (96.4%, $112k, etc.)
- `lib/demo/` is completely unused by any app page (only governance.ts in demo had SOC 2 string)
- `model-access` page now shows honest empty state

**P32 — Developer Pages**
- `developer/api-docs` lists only real gateway endpoints from `gatewayv2/server.go`
- `developer/quick-start` uses only real gateway calls (`/v1/messages`)
- `developer/sdks` lists only real workspace packages
- `developer/cli` lists only real CLI commands from `packages/cli/cmd`

**P34 — CSRF Safety**
- `session.ts` correctly reads and forwards session cookie + CSRF token
- Double-submit CSRF pattern in place

**P35 — Tenant Isolation**
- All backend handlers derive org from session, never accept org_id from frontend

**P46 — Mock Files**
- `lib/demo/` is fully unused by any dashboard page (0 imports)
- Safe to delete but kept for now since governance.ts defines demo types for reference

**P48 — API Inventory**
- `docs/architecture/frontend-backend-api-map.md` created

---

### 🟡 PARTIAL — Needs work

**P23 — Credential Pages (one-time secret behavior)**
- Current read pages for credentials work (show prefix/status, never secret)
- MISSING: Create dialog with one-time secret display/copy/dismiss
- MISSING: Provider credential rotation form
- MISSING: Gateway credential create form

**P26 — Filters and Date Range**
- Overview page: date range filter works via URL searchParams (?range=7d)
- Usage/Costs/Models pages: hardcoded 30-day window, no UI filter control
- MISSING: Date range picker on most observe pages
- MISSING: model/provider/project filter inputs

**P27 — Pagination**
- DataTable has frontend pagination (slicing client-side rows)
- Backend supports limit param on some endpoints
- MISSING: cursor/page-based server-side pagination for audit logs, activity, conversations

**P30 — Loading/Error/Empty States**
- All pages have proper empty states
- Server Components do not show loading skeleton (Next.js Suspense not wired)
- MISSING: Error boundaries for API failures
- MISSING: Suspense streaming skeletons

**P33 — Server vs Client Components**
- Most pages are Server Components (correct)
- RangeTabs is a Client Component (correct)
- MISSING: Client-side filter components for interact pages
- DataTable is Client Component (correct)

**P38 — Read Performance**
- Overview uses usage_daily/hourly (correct)
- Usage, Costs pages use pre-aggregated endpoints (correct)
- Live activity scans llm_requests — bounded by limit

**P40 — Error Contract**
- `ApiError` from api-client propagates to pages
- Pages crash on API error rather than showing error boundary
- MISSING: Error.tsx boundary file in app

**P49 — Security Review**
- Provider credentials: show fingerprint/tail only ✅
- Gateway credentials: show prefix only ✅
- One-time secret: NOT YET implemented for create actions
- Session token: HttpOnly cookie, never in localStorage ✅

---

### 🔲 PENDING — Not yet started

**P23 — One-time secret create dialogs**
**P27 — Server-side cursor pagination**
**P28 — OpenAPI regeneration**
**P29 — app/src/lib/api/ hook layer**
**P36 — Permission enforcement in UI**
**P39 — Service account attribution**
**P41 — Dedup frontend types vs API DTOs**
**P43 — Seed realistic local data**
**P45 — E2E data flow test**
**P47 — OpenAPI contract validation**

---

## Next Actions (Priority Order)

1. **P23** — Add error boundaries (`app/error.tsx`, `app/(dashboard)/error.tsx`)
2. **P23** — Create credential create/rotate dialogs with one-time secret reveal
3. **P26** — Add date range picker to Usage/Costs/Models pages  
4. **P27** — Wire server-side pagination for audit logs and activity
5. **P43** — Extend seed.sql with realistic dev records

