# Frontend → Backend API Map

Every dashboard page listed with its real backend endpoint(s), the service and
repository that back it, the database table(s) it reads, and the current wiring
status.

Generated from: `app/src/app/(dashboard)/app/**` and
`backend/internal/interfaces/controlapi/`.

---

## Legend

| Status | Meaning |
|---|---|
| ✅ Wired | Page calls a real backend endpoint; data is live |
| ⚠️ No API | Page renders correctly but no backing endpoint exists yet — honest empty state shown |
| 🔲 Missing | Page not yet built or endpoint not registered |

---

## Overview

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/overview` | `/v1/usage/summary` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/overview` | `/v1/usage/timeseries` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_hourly` | ✅ Wired |
| `/app/overview` | `/v1/usage/by-project` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/overview` | `/v1/usage/by-model` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/overview` | `/v1/usage/latency` | GET | Existing | `usage.Service` | `UsageRepository` | `llm_requests` | ✅ Wired |

---

## Observe

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/observe/live-activity` | `/v1/activity` | GET | Existing | `activity.Service` | `ActivityRepository` | `llm_requests` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/summary` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/timeseries` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_hourly` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/by-model` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/by-provider` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/by-project` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/by-user` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/usage` | `/v1/usage/by-team` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/usage` | `/v1/projects` | GET | Existing | `identity.Service` | `ProjectRepository` | `projects` | ✅ Wired |
| `/app/observe/usage` | `/v1/teams` | GET | Existing | `identity.Service` | `TeamRepository` | `teams` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/summary` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/timeseries` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_hourly` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/by-team` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/by-project` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/by-provider` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/by-model` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/costs` | `/v1/usage/by-user` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/models` | `/v1/models` | GET | Existing | `provider.Service` | `ModelRepository` | `provider_models` | ✅ Wired |
| `/app/observe/models` | `/v1/usage/by-model` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/providers` | `/v1/provider-accounts` | GET | Existing | `provider.Service` | `ProviderAccountRepository` | `provider_accounts` | ✅ Wired |
| `/app/observe/providers` | `/v1/provider-health` | GET | Existing | `provider.Service` | `ProviderHealthRepository` | `provider_health` | ✅ Wired |
| `/app/observe/providers` | `/v1/usage/by-provider` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/providers` | `/v1/usage/timeseries` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_hourly` | ✅ Wired |
| `/app/observe/users` | `/v1/members` | GET | Existing | `identity.Service` | `MemberRepository` | `org_members` | ✅ Wired |
| `/app/observe/users` | `/v1/usage/by-user` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/teams` | `/v1/teams` | GET | Existing | `identity.Service` | `TeamRepository` | `teams` | ✅ Wired |
| `/app/observe/teams` | `/v1/usage/by-team` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/projects` | `/v1/projects` | GET | Existing | `identity.Service` | `ProjectRepository` | `projects` | ✅ Wired |
| `/app/observe/projects` | `/v1/usage/by-project` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/observe/conversations` | `/v1/conversations` | GET | Existing | `conversation.Service` | `ConversationRepository` | `conversations` | ✅ Wired |

---

## Govern

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/govern/policies` | — | — | — | — | — | `policies`, `policy_rules` | ⚠️ No API — enforced from snapshot, no read endpoint |
| `/app/govern/model-access` | — | — | — | — | — | `policies`, `policy_rules` | ⚠️ No API — grant records not exposed |
| `/app/govern/budgets` | — | — | — | — | — | `policies` | ⚠️ No API — caps enforced from snapshot only |
| `/app/govern/guardrails` | — | — | — | — | — | — | ⚠️ No API — no detector endpoint exists |
| `/app/govern/credentials` | `/v1/credentials` | GET | Existing | `identity.Service` | `CredentialRepository` | `api_credentials` | ✅ Wired |

---

## Security

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/security/audit-logs` | `/v1/audit-logs` | GET | Existing | `auditlog.Service` | `AuditLogRepository` | `audit_logs` | ✅ Wired |
| `/app/security/alerts` | `/v1/alerts` | GET | Existing | `alert.Service` | `AlertRepository` | `alerts` | ✅ Wired |
| `/app/security/events` | `/v1/security-events` | GET | Existing | `securityevent.Service` | `SecurityEventRepository` | `security_events` | ✅ Wired |

---

## Platform

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/platform/provider-health` | `/v1/provider-health` | GET | Existing | `provider.Service` | `ProviderHealthRepository` | `provider_health` | ✅ Wired |
| `/app/platform/provider-accounts` | `/v1/provider-accounts` | GET | Existing | `provider.Service` | `ProviderAccountRepository` | `provider_accounts` | ✅ Wired |
| `/app/platform/service-accounts` | `/v1/service-accounts` | GET | Existing | `identity.Service` | `ServiceAccountRepository` | `service_accounts` | ✅ Wired |
| `/app/platform/gateway-credentials` | `/v1/gateway-credentials` | GET | Existing | `identity.Service` | `GatewayCredentialRepository` | `gateway_credentials` | ✅ Wired |
| `/app/platform/model-registry` | `/v1/models` | GET | Existing | `provider.Service` | `ModelRepository` | `provider_models` | ✅ Wired |
| `/app/platform/model-registry` | `/v1/usage/by-model` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |

---

## Organization

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/organization/members` | `/v1/members` | GET | Existing | `identity.Service` | `MemberRepository` | `org_members` | ✅ Wired |
| `/app/organization/members` | `/v1/members/{id}/role` | PATCH | Existing | `identity.Service` | `MemberRepository` | `org_members` | ✅ Wired |
| `/app/organization/members` | `/v1/members/{id}` | DELETE | Existing | `identity.Service` | `MemberRepository` | `org_members` | ✅ Wired |
| `/app/organization/invitations` | `/v1/invitations` | GET | Existing | `identity.Service` | `InvitationRepository` | `invitations` | ✅ Wired |
| `/app/organization/invitations` | `/v1/invitations` | POST | Existing | `identity.Service` | `InvitationRepository` | `invitations` | ✅ Wired |
| `/app/organization/invitations` | `/v1/invitations/{id}` | DELETE | Existing | `identity.Service` | `InvitationRepository` | `invitations` | ✅ Wired |
| `/app/organization/teams` | `/v1/teams` | GET | Existing | `identity.Service` | `TeamRepository` | `teams` | ✅ Wired |
| `/app/organization/teams` | `/v1/teams` | POST | Existing | `identity.Service` | `TeamRepository` | `teams` | ✅ Wired |
| `/app/organization/teams` | `/v1/teams/{id}` | GET | Existing | `identity.Service` | `TeamRepository` | `teams` | ✅ Wired |
| `/app/organization/teams` | `/v1/teams/{id}/members` | POST | Existing | `identity.Service` | `TeamMemberRepository` | `team_members` | ✅ Wired |
| `/app/organization/teams` | `/v1/teams/{id}/members/{userID}` | DELETE | Existing | `identity.Service` | `TeamMemberRepository` | `team_members` | ✅ Wired |
| `/app/organization/projects` | `/v1/projects` | GET | Existing | `identity.Service` | `ProjectRepository` | `projects` | ✅ Wired |
| `/app/organization/projects` | `/v1/projects` | POST | Existing | `identity.Service` | `ProjectRepository` | `projects` | ✅ Wired |
| `/app/organization/projects` | `/v1/projects/{id}` | GET | Existing | `identity.Service` | `ProjectRepository` | `projects` | ✅ Wired |
| `/app/organization/projects` | `/v1/projects/{id}/members` | POST | Existing | `identity.Service` | `ProjectMemberRepository` | `project_members` | ✅ Wired |
| `/app/organization/projects` | `/v1/projects/{id}/members/{userID}` | DELETE | Existing | `identity.Service` | `ProjectMemberRepository` | `project_members` | ✅ Wired |
| `/app/organization/settings` | `/v1/organization` | GET | Existing | `identity.Service` | `OrganizationRepository` | `organizations` | ✅ Wired |
| `/app/organization/settings` | `/v1/organization` | PATCH | Existing | `identity.Service` | `OrganizationRepository` | `organizations` | ✅ Wired |

---

## Billing

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/billing/usage` | `/v1/usage/summary` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/billing/usage` | `/v1/usage/timeseries` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_hourly` | ✅ Wired |
| `/app/billing/usage` | `/v1/usage/by-project` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/billing/usage` | `/v1/usage/by-team` | GET | Existing | `usage.Service` | `UsageRepository` | `usage_daily` | ✅ Wired |
| `/app/billing/usage` | `/v1/billing/invoices` | GET | Existing | `billing.Service` | `InvoiceRepository` | `invoices` | ✅ Wired |
| `/app/billing/subscription` | `/v1/billing/plans` | GET | Existing | `billing.Service` | `PlanRepository` | `plans` | ✅ Wired |
| `/app/billing/subscription` | `/v1/billing/subscription` | GET | Existing | `billing.Service` | `SubscriptionRepository` | `subscriptions` | ✅ Wired |
| `/app/billing/invoices` | `/v1/billing/invoices` | GET | Existing | `billing.Service` | `InvoiceRepository` | `invoices` | ✅ Wired |

---

## Developer

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/app/developer/quick-start` | — | — | Static content | — | — | — | ✅ Static only |
| `/app/developer/sdks` | — | — | Static content | — | — | — | ✅ Static only |
| `/app/developer/cli` | — | — | Static content | — | — | — | ✅ Static only |
| `/app/developer/api-docs` | — | — | Static content | — | — | — | ✅ Static only |

---

## Auth

| Page | Endpoint | Method | Existing/New | Backend service | Repository | DB table | Status |
|---|---|---|---|---|---|---|---|
| `/signin` | `/v1/auth/signin` | POST | Existing | `identity.Service` | `UserRepository` | `users` | ✅ Wired |
| `/signup` | `/v1/auth/signup` | POST | Existing | `identity.Service` | `UserRepository` | `users` | ✅ Wired |
| Session | `/v1/auth/session` | GET | Existing | `identity.Service` | `SessionRepository` | `refresh_tokens` | ✅ Wired |
| Session refresh | `/v1/auth/refresh` | POST | Existing | `identity.Service` | `SessionRepository` | `refresh_tokens` | ✅ Wired |
| Logout | `/v1/auth/logout` | POST | Existing | `identity.Service` | `SessionRepository` | `refresh_tokens` | ✅ Wired |

---

## Gaps and Backlog

| Feature | Reason not yet wired | Recommended action |
|---|---|---|
| Model grants / denials | No control-plane endpoint | Add `GET /v1/model-grants`, `GET /v1/model-denials` backed by policy_rules |
| Guardrails | No guardrail endpoint or detection runtime | Add endpoints when detector runtime exists |
| Budgets authoring | Budget caps stored in policies only | Add `GET/POST /v1/budgets` backed by policy records |
| Policy authoring | Policy records exist in DB but no list/get endpoint | Add `GET /v1/policies`, `GET /v1/policy-rules` |
| Provider credential validation | Endpoint exists, UI not wired | Wire `POST /v1/provider-credentials/validate` to validation form |
| Model sync | `POST /v1/models/sync` exists | Wire to model registry page refresh action |

---

*Last updated: 2026-10-04. Update this file when adding or removing dashboard pages or backend endpoints.*
