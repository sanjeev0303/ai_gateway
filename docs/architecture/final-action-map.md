# Final Action Map

| Feature | UI Route | API Endpoint | Read/Write |
|---------|----------|--------------|------------|
| **Auth** | `/login`, `/register` | `/v1/auth/signup`, `/v1/auth/login` | Write |
| **Overview Dashboard** | `app/overview` | `/v1/usage/summary` | Read |
| **Live Activity** | `app/observe/live-activity` | `/v1/messages` (SSE via NATS) | Read |
| **Usage Analytics** | `app/observe/usage`, `app/observe/costs` | `/v1/usage/summary` | Read |
| **Projects** | `app/organization/projects` | `/v1/projects` | Read/Write |
| **Teams** | `app/organization/teams` | `/v1/teams` | Read/Write |
| **Members** | `app/organization/members` | `/v1/members`, `/v1/invitations` | Read/Write |
| **Service Accounts** | `app/platform/service-accounts` | `/v1/service-accounts` | Read/Write |
| **Provider Accounts** | `app/platform/provider-accounts` | `/v1/provider-accounts` | Read/Write |
| **Provider Credentials** | `app/platform/provider-accounts/[id]` | `/v1/provider-accounts/{id}/credentials` | Read/Write |
| **Gateway Credentials** | `app/platform/gateway-credentials` | `/v1/gateway-credentials` | Read/Write |
| **Policies** | `app/govern/policies` | `/v1/policies` | Read/Write |
| **Security Events** | `app/security/events` | `/v1/security-events` | Read |
| **Audit Logs** | `app/security/audit-logs` | `/v1/audit-logs` | Read |

*Note: All endpoints are protected by authentication cookies and CSRF tokens.*
