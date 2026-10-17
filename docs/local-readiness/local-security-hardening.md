# Local Environment Security Hardening & Safe Setup Runbook

## 1. Overview & Security Boundary

Sentinel AI Gateway is designed for strict local developer security without introducing cloud-specific or production deployment infrastructure. This runbook details how configuration validation, secret handling, development authentication bypasses, CSRF defense, and database lifecycle management are hardened.

---

## 2. Configuration Validation & Secret Hardening

### Placeholder Secret Detection
The system automatically inspects configuration at boot via [`backend/internal/platform/config/config.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/platform/config/config.go).
- **Prohibited Patterns**: `change-me`, `changeme`, `placeholder`, `replace-me`, `replace_me`, `example`, `secret`, `password`, `123456`, `default`.
- **Environment Rules**:
  - `production` / `staging`:
    - `AUTH_HMAC_SIGNING_KEY` is strictly required (>= 32 bytes) and cannot be a placeholder.
    - `SECRETS_KEY` is strictly required when `SECRETS_PROVIDER=env` and cannot be a placeholder.
    - `REDIS_PASSWORD` is strictly required when `REDIS_ENABLED=true` and cannot be a placeholder.
    - `DATABASE_URL` cannot contain known placeholder passwords.
    - `AUTH_DEV_LOGIN_ENABLED` is strictly forbidden.
    - `AIGW_ALLOW_INSECURE_DEV` is strictly forbidden.
    - `AUTH_SESSION_COOKIE_SECURE` must be `true`.
    - `AUTH_CSRF_ENABLED` must be `true`.
  - `development`:
    - Placeholder secrets in `AUTH_HMAC_SIGNING_KEY`, `REDIS_PASSWORD`, `SECRETS_KEY`, or `DATABASE_URL` are blocked **unless** `AIGW_ALLOW_INSECURE_DEV=true` is explicitly set by the developer.

### Diagnostic Secrecy (Zero Secret Leakage)
- **Validation Messages**: Startup error messages only state the name of the failing configuration key (e.g., `DATABASE_URL contains a known placeholder secret; set a secure password or set AIGW_ALLOW_INSECURE_DEV=true for local development`) and never echo the raw password or secret string.
- **Redacted Configuration**: All logging and diagnostic commands (`cmd/config-validate`) use `Config.Redacted()`, which masks secret strings (`Database.URL`, `Redis.Password`, `Auth.HMACSigningKey`, `Auth.HMACPreviousKeys`, `Secrets.Key`, `Secrets.Token`) with `<redacted>`.
- **CLI & Script Execution**: Redis commands in [`scripts/verify-local.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/verify-local.sh) authenticate using container environment variables (`REDISCLI_AUTH`) rather than CLI flags (`-a`), preventing credentials from appearing in process tables (`ps aux`).

---

## 3. Authentication, Session & CSRF Lifecycle

| Security Mechanism | Implementation | Enforcement Details |
|---|---|---|
| **Development Login Bypass** | `POST /v1/auth/dev-login` | Only mounted when `AIGW_AUTH_DEV_LOGIN_ENABLED=true`. Requires `AIGW_ALLOW_INSECURE_DEV=true` in development; strictly prohibited in production/staging. |
| **CSRF Defense** | Double-Submit Cookie Check | Compares `X-CSRF-Token` header against `csrf_token` cookie for cookie-authenticated state changes (`/v1/auth/refresh`, `/v1/auth/logout`, admin operations). |
| **Session Cookies** | HttpOnly, SameSite | Access token: `HttpOnly`, `SameSite=Lax`. Refresh token: `HttpOnly`, `SameSite=Strict`. CSRF token: Readable by client JS, `SameSite=Lax`. |
| **Session Expiry** | JWT TTL & Refresh Rotation | Access token (24h default), Session (12h default), Refresh token (720h default). Refresh tokens rotate on every invocation. |
| **Logout & Revocation** | Database + Cookie Clearance | `POST /v1/auth/logout` explicitly deletes the database session record and expires all browser cookies (`MaxAge: -1`). |
| **Credential Revocation** | Stage 1 Gateway Pipeline | Revoked gateway API keys return HTTP 401 `credential_revoked` at Stage 1 before executing rate limits, budgets, or upstream proxies. |

---

## 4. Safe Setup & Script Guarding

### Preservation of Developer Configurations
- Both [`scripts/dev-ctl.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/dev-ctl.sh) and [`scripts/verify-local.sh`](file:///home/sanju/ai-project/ai_gateway/scripts/verify-local.sh) inspect whether `backend/.env` and `deploy/compose/.env` already exist.
- Existing developer `.env` files are **never overwritten**. Templates (`.env.example`) are copied only on initial setup when files are completely missing.

### Guarded Destructive Reset Operations
- `dev-ctl.sh reset` and `make dev-reset` destroy persistent Docker volumes (`pgdata`, `redisdata`, `natsdata`, `secrets`).
- **Interactive Guard**: In an interactive terminal, the user must explicitly type `'RESET'` in capital letters to proceed.
- **Non-Interactive Guard**: When run non-interactively or in automation without `--confirm` or `-f`, the script aborts immediately with exit code 1 to protect existing test data.

### Functional Readiness & Verification
- All service containers provide `/readyz` endpoints:
  - Control API: `http://localhost:8081/readyz` (verifies DB, Redis, and Eventbus connections)
  - Gateway v2: `http://localhost:8084/readyz` (verifies upstream resolution, rate limiter, and budget guard)
  - Worker: `http://localhost:8082/readyz` (verifies background event processing)
  - NATS JetStream: `http://localhost:8222/healthz`
  - PostgreSQL: `pg_isready` and `SELECT 1;`
  - Redis: `redis-cli ping` via `REDISCLI_AUTH`
- Database migrations (`cmd/migrate`) and seed operations (`cmd/seed`) are strictly idempotent.
