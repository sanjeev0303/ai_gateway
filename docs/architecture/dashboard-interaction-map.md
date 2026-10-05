# Dashboard Interaction Map

This document outlines the API coverage and interaction patterns required to transform the Sentinel dashboard from a read-only analytics view into a fully operational control plane.

## Observe

| Page | Existing APIs | Missing APIs | Existing Actions | Missing Actions | Detail Route | Permissions |
|---|---|---|---|---|---|---|
| `/app/observe/conversations` | `GET /v1/conversations` | `GET /v1/conversations/:id/messages` | None | None | Transcript Drawer | `usage.read` |
| `/app/observe/costs` | `GET /v1/usage/summary`, `timeseries`, `by-team`, `by-project`, `by-provider`, `by-model`, `by-user` | None | None | None | None | `usage.read` |
| `/app/observe/live-activity` | `GET /v1/activity/stream` | None | None | None | Request Inspector | `usage.read` |
| `/app/observe/models` | `GET /v1/usage/...` | None | None | None | None | `usage.read` |
| `/app/observe/projects` | `GET /v1/usage/by-project` | None | None | New project | Linked to `/app/organization/projects/:id` | `usage.read` |
| `/app/observe/providers` | `GET /v1/usage/by-provider`, `GET /v1/provider-accounts`, `GET /v1/provider-health` | None | None | None | None | `usage.read`, `providers.read` |
| `/app/observe/teams` | `GET /v1/usage/by-team`, `GET /v1/teams` | None | None | New team | Linked to `/app/organization/teams/:id` | `usage.read` |
| `/app/observe/usage` | `GET /v1/usage/...` | None | None | None | None | `usage.read` |
| `/app/observe/users` | `GET /v1/usage/by-user`, `GET /v1/members` | None | None | None | Linked to `/app/organization/members` | `usage.read`, `members.read` |

## Govern

| Page | Existing APIs | Missing APIs | Existing Actions | Missing Actions | Detail Route | Permissions |
|---|---|---|---|---|---|---|
| `/app/govern/policies` | None | `GET/POST/PUT/DELETE /v1/policies` | None | Create, Edit, Delete Policy | Policy Drawer | `org.manage` |
| `/app/govern/model-access` | None | `GET/POST/DELETE /v1/model-access-grants` | None | Grant, Revoke Access | None | `org.manage` |
| `/app/govern/budgets` | None | `GET/POST/PUT/DELETE /v1/budgets` | None | Create, Edit, Delete Budget | Budget Drawer | `org.manage` |
| `/app/govern/guardrails` | None | `GET/POST/PUT/DELETE /v1/guardrails` | None | Configure Guardrail | Guardrail Drawer | `org.manage` |
| `/app/govern/credentials` | `GET/POST/DELETE /v1/credentials` | None | None | Create, Revoke Credential | None | `credentials.read`, `credentials.manage` |

## Organization

| Page | Existing APIs | Missing APIs | Existing Actions | Missing Actions | Detail Route | Permissions |
|---|---|---|---|---|---|---|
| `/app/organization/invitations` | `GET/POST/DELETE /v1/invitations`, `POST /v1/invitations/accept` | None | None | Invite, Revoke | None | `members.read`, `members.manage` |
| `/app/organization/members` | `GET /v1/members`, `PATCH /v1/members/:id/role`, `DELETE /v1/members/:id` | None | None | Change role, Remove | None | `members.read`, `members.manage` |
| `/app/organization/projects` | `GET/POST /v1/projects`, `GET/POST/DELETE /v1/projects/:id/members` | `PUT/DELETE /v1/projects/:id` | None | New project, Manage members, Edit, Delete | Project Drawer/Page | `projects.read`, `projects.manage` |
| `/app/organization/settings` | `GET/PATCH /v1/organizations/me` | None | `updateOrganization` | None | None | `org.manage` |
| `/app/organization/teams` | `GET/POST /v1/teams`, `GET/POST/DELETE /v1/teams/:id/members` | `PUT/DELETE /v1/teams/:id` | None | New team, Manage members, Edit, Delete | Team Drawer/Page | `teams.read`, `teams.manage` |

## Platform

| Page | Existing APIs | Missing APIs | Existing Actions | Missing Actions | Detail Route | Permissions |
|---|---|---|---|---|---|---|
| `/app/platform/gateway-credentials` | `GET/POST/DELETE /v1/gateway-credentials`, `rotate`, `expire`, `restore` | None | None | Create, Rotate, Expire, Restore, Revoke | Credential Drawer | `credentials.read`, `credentials.manage` |
| `/app/platform/model-registry` | `GET /v1/models`, `POST /v1/models/sync` | None | None | Sync Models | None | `providers.read`, `providers.manage` |
| `/app/platform/provider-accounts` | `GET/POST /v1/provider-accounts`, `GET /v1/provider-accounts/:id` | `PUT/DELETE /v1/provider-accounts/:id` | None | Add, Edit, Delete Account | Account Drawer/Page | `providers.read`, `providers.manage` |
| `/app/platform/provider-health` | `GET /v1/provider-health`, `POST /v1/provider-health/check` | None | None | Trigger health check | None | `providers.read`, `providers.manage` |
| `/app/platform/service-accounts` | `GET/POST/DELETE /v1/service-accounts` | None | None | Create, Delete | Service Account Drawer | `org.manage` |

## Security

| Page | Existing APIs | Missing APIs | Existing Actions | Missing Actions | Detail Route | Permissions |
|---|---|---|---|---|---|---|
| `/app/security/alerts` | `GET /v1/alerts`, `GET/PUT/DELETE /v1/alerts/:id`, `POST /v1/alerts/:id/test` | `POST /v1/alerts` | None | Create, Edit, Test, Mute, Delete | Alert Drawer | `alerts.read`, `alerts.manage` |
| `/app/security/audit-logs` | `GET /v1/audit-logs` | None | None | None | None | `audit.read` |
| `/app/security/events` | `GET /v1/security-events` | None | None | None | None | `security_events.read` |

## Billing

| Page | Existing APIs | Missing APIs | Existing Actions | Missing Actions | Detail Route | Permissions |
|---|---|---|---|---|---|---|
| `/app/billing/invoices` | `GET /v1/billing/invoices` | None | None | None | Invoice Drawer | `billing.read` |
| `/app/billing/subscription` | `GET /v1/billing/subscription`, `GET /v1/billing/plans` | `POST /v1/billing/subscription/cancel` | None | Manage/Cancel | None | `billing.read`, `billing.manage` |
| `/app/billing/usage` | `GET /v1/billing/usage` | None | None | None | None | `billing.read` |
