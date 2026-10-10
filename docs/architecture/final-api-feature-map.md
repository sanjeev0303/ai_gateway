# Final API Feature Map

This document maps out the available features exposed through the Control API and implemented across the system.

## 1. Identity & Access Management
- **Auth**: Email/Password signup, Session handling via Cookies (JWT), CSRF Protection.
- **Organizations**: Multi-tenant isolation at the DB and API layer.
- **Projects**: Logical grouping for environments/workloads.
- **Teams**: Group-based access control.
- **Service Accounts**: Non-human identities for automation.
- **Roles & Permissions**: Fine-grained RBAC for Owners, Admins, Members, and Viewers.

## 2. Platform Connectivity
- **Provider Accounts**: Connect to external LLM providers (Anthropic, OpenAI, etc.).
- **Provider Credentials**: Store secrets securely with `argon2id` hashes and AES-256 encryption via Vault/Env.
- **Gateway Credentials**: Scoped API keys for client routing through the gateway (`gateway` / `control` scopes).
- **Model Registry**: Provider models synchronization.

## 3. Governance
- **Policies**: Define routing, fallbacks, and blocking rules.
- **Model Access**: Control which identities can invoke specific models.
- **Budgets**: Set spending limits on Projects/Teams.
- **Guardrails**: (Stubbed) Output evaluation logic.

## 4. Telemetry & Observation
- **Live Activity**: NATS-driven real-time request streaming.
- **Usage Rollups**: Worker-processed token and request aggregation for dashboards.
- **Security Events**: Policy violation detection and auditing.
- **Audit Logs**: Traceability for all mutation actions on platform resources.

## 5. Gateway Data Plane
- **Authentication**: Validates Gateway Credentials via Redis caching / DB fallback.
- **Tenant Resolution**: Maps incoming requests to the configured Provider Account / Upstream.
- **Policy Enforcement**: Applies rate limits, budgets, and access rules in the critical path.
- **Streaming Proxy**: Forwards requests to upstream models and records usage synchronously.