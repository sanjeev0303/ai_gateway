# Final Action Map

| Feature | UI route | Action | HTTP method | Endpoint | Application service | Repository | Database | Permission | Audit event | Gateway effect |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Provider Accounts** | `/app/platform/provider-accounts` | Create Account | POST | `/v1/provider-accounts` | ProvidersAPI | Postgres | `provider_accounts` | `provider.create` | `provider.created` | Enables downstream proxying to provider |
| **Provider Accounts** | `/app/platform/provider-accounts` | Add Credential | POST | `/v1/provider-credentials` | ProvidersAPI | Postgres | `provider_credentials` | `credential.create` | `credential.created` | Gateway authenticates requests |
| **Provider Health** | `/app/platform/provider-health` | Probe Provider | POST | `/v1/provider-health/check` | ProvidersAPI | Memory/Transient | None | `provider.check` | `provider.checked` | None |
| **Projects** | `/app/organization/projects` | Create Project | POST | `/v1/projects` | CoreAPI | Postgres | `projects` | `project.create` | `project.created` | Requests scoped by ProjectID |
| **Projects** | `/app/organization/projects` | Delete Project | DELETE | `/v1/projects/:id` | CoreAPI | Postgres | `projects` | `project.delete` | `project.deleted` | Rejects requests with ProjectID |
| **Gateway Credentials** | `/app/platform/gateway-credentials` | Generate Key | POST | `/v1/gateway-credentials` | CoreAPI | Postgres | `api_keys` | `apikey.create` | `apikey.created` | Authenticates incoming gateway traffic |
| **Service Accounts** | `/app/platform/service-accounts` | Create SA | POST | `/v1/service-accounts` | IdentityAPI | Postgres | `service_accounts` | `sa.create` | `sa.created` | Maps to headless access tokens |
| **Team Management** | `/app/organization/teams` | Create Team | POST | `/v1/teams` | IdentityAPI | Postgres | `teams` | `team.create` | `team.created` | None |
| **Members** | `/app/organization/members` | Update Role | PATCH | `/v1/members/:id` | IdentityAPI | Postgres | `members` | `member.update` | `member.updated` | None |
| **Invitations** | `/app/organization/invitations` | Invite User | POST | `/v1/invitations` | IdentityAPI | Postgres | `invitations` | `invite.create` | `invite.created` | None |
| **Policies** | `/app/govern/policies` | Create Policy | POST | `/v1/policies` | GovernAPI | Postgres | `policies` | `policy.create` | `policy.created` | Applies caching/retry logic on Gateway |
| **Policies** | `/app/govern/policies` | Delete Policy | DELETE | `/v1/policies/:id` | GovernAPI | Postgres | `policies` | `policy.delete` | `policy.deleted` | Removes logic from Gateway |
| **Budgets** | `/app/govern/budgets` | Set Limit | POST | `/v1/budgets` | GovernAPI | Postgres | `budgets` | `budget.create` | `budget.created` | Gateway rejects traffic if cap exceeded |
| **Model Access** | `/app/govern/model-access` | Toggle Access | PATCH | `/v1/model-access` | GovernAPI | Postgres | `model_rules` | `access.update` | `access.updated` | Gateway blocks banned model names |
| **Audit Logs** | `/app/security/audit-logs` | Export Logs | GET | `/v1/audit-logs/export` | AuditAPI | ClickHouse | `audit_events` | `audit.read` | `audit.exported` | None |
| **Usage** | `/app/observe/usage` | View Dashboard | GET | `/v1/usage/timeseries` | UsageAPI | ClickHouse | `usage_logs` | `usage.read` | None | None |
