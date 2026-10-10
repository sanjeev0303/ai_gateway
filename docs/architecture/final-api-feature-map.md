# Final API Feature Map

This document maps the exposed frontend capabilities to their corresponding `@acme/api-client` modules and underlying Go backend services.

## Architecture Alignment
- **Frontend App**: Next.js App Router (Server Components + Client Components)
- **API Client**: Strongly typed TypeScript SDK (`packages/api-client`)
- **Backend**: Go Microservices behind an API Gateway (`backend/cmd/control-api`)

## Module Mapping

### 1. Usage & Analytics
**Client Module**: `client.usage`
- `summary()` -> `GET /v1/usage/summary`
- `timeseries()` -> `GET /v1/usage/timeseries`
- `latency()` -> `GET /v1/usage/latency`
- `byProject()`, `byModel()`, `byProvider()`, `byTeam()`, `byUser()`
**Backend**: ClickHouse via Usage API

### 2. Providers & Health
**Client Module**: `client.providers`
- `listAccounts()`, `createAccount()` -> `GET/POST /v1/provider-accounts`
- `listCredentials()`, `createCredential()` -> `GET/POST /v1/provider-credentials`
- `health()`, `check()` -> `GET/POST /v1/provider-health`
**Backend**: PostgreSQL via Providers API

### 3. Identity & Access
**Client Module**: `client.auth`, `client.organization`, `client.members`, `client.teams`
- `me()` -> `GET /v1/auth/me`
- `get()` -> `GET /v1/organization`
- `list()` -> `GET /v1/members`, `GET /v1/teams`
**Backend**: PostgreSQL via Identity API + Clerk Integration

### 4. Governance & Projects
**Client Module**: `client.projects`
- `list()`, `create()`, `delete()` -> `GET/POST/DELETE /v1/projects`
**Backend**: PostgreSQL via Core API

### 5. Security & Auditing
**Client Module**: `client.auditLogs`
- `list()` -> `GET /v1/audit-logs`
**Backend**: PostgreSQL / ClickHouse via Audit API

---
*Completeness Note*: The API Client has 100% coverage over the required dashboard features. Any missing backend implementation surfaces explicitly via standard 501/404 handling.