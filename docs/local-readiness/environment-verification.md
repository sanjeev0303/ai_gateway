# Sentinel AI Governance Gateway — Environment Verification & Local Readiness Report

**Document Version:** 1.0.0  
**Repository:** `ai_gateway`  
**Execution Timestamp:** October 2026  
**Status:** FULLY VERIFIED & HARDENED

---

## 1. Executive Summary

This report establishes the verified baseline and security guardrails for Sentinel's local development environment under **Prompt 1: Secure and Reproducible Local Development Environment**. 

All application services, configuration loaders, local lifecycle scripts, dependency orderings, and readiness probes have been audited and hardened to ensure:
1. **Zero Accidental Secret Exposure**: Strict redaction of credentials, authorization headers, API keys, and bearer tokens in logs and reports.
2. **Deterministic Startup Validation**: Detection and rejection of known placeholder secrets in inappropriate environments (`production`, `staging`, or unconfirmed `development`).
3. **Non-Destructive Configuration**: Zero silent overwrites or replacement of existing `.env` files.
4. **Explicit Development Bypasses**: Development conveniences (such as `dev-login`, insecure cookies, and disabled CSRF) require an explicit, documented opt-in (`AIGW_ALLOW_INSECURE_DEV=true`).
5. **Functional Dependency Readiness**: Health probes query `/readyz` (which verifies live PostgreSQL connection pools, Redis pings, and NATS JetStream channels) rather than mere process liveness (`/livez`).
6. **Safe Lifecycle Controls**: Complete lifecycle commands (`dev-start`, `dev-stop`, `dev-status`, `dev-reset`) with mandatory interactive confirmation for destructive operations.

---

## 2. Environment & Service Identity

The local development verification suite targets the exact local checkout and service topology:

| Attribute | Verified Value |
|---|---|
| **Monorepo Root** | `/home/sanju/ai-project/ai_gateway` |
| **Git Branch / Checkout** | `main` |
| **Control Plane API** | `http://localhost:8081` (`/readyz`, `/v1/auth/...`, `/v1/tenants/...`) |
| **Data Plane Gateway (v2)** | `http://localhost:8084` (`/readyz`, `/v1/chat/completions`, `/v1/messages`) |
| **Background Worker** | `http://localhost:8082` (`/readyz`) |
| **Web UI Dashboard** | `http://localhost:3000` (Next.js 16 App Router) |
| **PostgreSQL** | `localhost:5432` (`ai_gateway` database, user: `gateway`) |
| **Redis** | `localhost:6379` (cache & Lua rate-limiting) |
| **NATS JetStream** | `localhost:4222` (client), `http://localhost:8222/healthz` (monitoring) |

---

## 3. Configuration Validation & Secret Safety Standards

### 3.1 Environment Partitioning
The backend configuration loader (`backend/internal/platform/config/config.go`) validates the operating environment via `AIGW_ENV`:
- `development` (default for local development)
- `staging` (production-like test environment)
- `production` (production deployment)
- `test` (automated test runner)

### 3.2 Known Placeholder Secret Detection
The loader incorporates pattern matching for known development placeholder values:
```go
knownPlaceholderPatterns = []string{
    "change-me",
    "changeme",
    "replace-me",
    "replaceme",
    "your-secret-here",
    "your-signing-key-here",
    "password",
    "secret",
    "placeholder",
}
```

#### Enforcement Rules:
1. **Production and Staging (`AIGW_ENV=production` | `staging`)**:
   - Known placeholders in `AUTH_HMAC_SIGNING_KEY`, `AUTH_HMAC_PREVIOUS_KEYS`, `SECRETS_KEY`, or `REDIS_PASSWORD` are **strictly rejected** at startup with actionable error messages.
   - `AUTH_DEV_LOGIN_ENABLED` is forbidden.
   - `AUTH_SESSION_COOKIE_SECURE` must be `true`.
   - `AUTH_CSRF_ENABLED` must be `true`.
   - `ALLOW_INSECURE_DEV` cannot be enabled.
2. **Development (`AIGW_ENV=development`)**:
   - If placeholder secrets or development bypasses (`AUTH_DEV_LOGIN_ENABLED=true`) are present, they are rejected unless the developer explicitly opted in via `AIGW_ALLOW_INSECURE_DEV=true`.
   - Actionable error messages guide the developer to either supply a real secret or acknowledge local development mode.
3. **Actionable Errors Without Value Leakage**:
   - Error messages state the failing variable name and the remediation step, **never printing the actual variable value or partial secret characters**.

### 3.3 Non-Destructive Environment Loading
Both `scripts/dev-ctl.sh` and `scripts/verify-local.sh` adhere to strict non-destructive policies:
- If `backend/.env` or `deploy/compose/.env` already exists, it is **never overwritten** or truncated.
- If missing, safe template defaults are copied from `.env.example` with clear notices.

---

## 4. Secret Redaction & Logging Safeguards

### 4.1 Structured Logging Redaction (`backend/internal/platform/logging`)
Every structured log event emitted through `slog` passes through a custom `ReplaceAttr` hook:
- **Sensitive Key Redaction**: Any attribute matching `authorization`, `auth`, `token`, `password`, `secret`, `api_key`, `apikey`, `signing_key`, `hmac_key`, `cookie`, or `credential` has its value replaced with `<redacted>`.
- **Bearer Token Redaction**: Any string value starting with `Bearer ` has the token payload stripped and replaced with `Bearer <redacted>`.

### 4.2 Configuration Redaction
Calling `cfg.Redacted()` produces a sanitised copy masking:
- `Database.URL` -> `<redacted>`
- `Redis.Password` -> `<redacted>`
- `Auth.HMACSigningKey` -> `<redacted>`
- `Auth.HMACPreviousKeys` -> `["<redacted>", ...]`
- `Secrets.Key` -> `<redacted>`
- `Secrets.Token` -> `<redacted>`

---

## 5. Service Topology & Functional Readiness

### 5.1 Startup Dependency Graph
In `deploy/compose/docker-compose.yml`:
- PostgreSQL, Redis, and NATS start first and undergo container healthchecks.
- Migrate runs to apply Goose migrations, then exits successfully.
- Control API, Gateway-v2, and Worker wait for Migrate to complete and Redis/NATS/Postgres to be healthy.
- Web UI (Next.js) starts once Control API is healthy.

### 5.2 Functional Readiness Probing (`/readyz`)
Rather than relying on process liveness (`/livez`), Docker Compose and verification scripts query the functional `/readyz` endpoint on all backend services:
- **Control API `/readyz`**: Verifies PostgreSQL connection pool ping and Redis ping.
- **Gateway-v2 `/readyz`**: Verifies PostgreSQL connection pool ping, Redis ping, and NATS JetStream connectivity.
- **Worker `/readyz`**: Verifies PostgreSQL connection pool ping and NATS JetStream channel connectivity.

---

## 6. Local Lifecycle Management Commands

The lifecycle script `scripts/dev-ctl.sh` and top-level `Makefile` targets provide safe, verified service management:

| Command | Action | Safety Guarantee |
|---|---|---|
| `make dev-start` | Starts Docker Compose stack in dependency order | Preserves existing `.env`; waits for `/readyz` 200 OK |
| `make dev-stop` | Gracefully stops containers | Preserves all container states and volume data |
| `make dev-down` | Removes containers from network | Keeps persistent volumes (`pgdata`, `redisdata`, `natsdata`, `secrets`) intact |
| `make dev-status` | Displays checkout commit, host URLs, container status, and `/readyz` health | Read-only inspection |
| `make dev-reset` | Wipes volumes and restarts from pristine state | **Requires explicit confirmation** (interactive `RESET` or `--confirm` flag) |
| `make verify` | Runs all 15 local release-readiness verification checks | Deterministic, non-destructive execution |

---

## 7. Verification Results & Evidence

| Check # | Check Name | Command / Target | Status |
|---|---|---|---|
| **1/15** | Local Configuration Validation | `go run ./cmd/config-validate` | **PASSED** |
| **2/15** | Tooling Dependencies | `go`, `node`, `npm`, `docker`, `curl` | **PASSED** |
| **3/15** | Database Schema Migrations | `go run ./cmd/migrate` | **PASSED** |
| **4/15** | Full-Stack Functional Readiness | `/readyz`, `pg_isready`, `SELECT 1`, Redis, NATS | **PASSED** |
| **5/15** | Backend Static Analysis | `go vet ./...` | **PASSED** |
| **6/15** | Go Unit Test Suite | `backend` & `packages/cli` test suites | **PASSED** |
| **7/15** | Repository Integration Tests | `postgres/repositories/...` | **PASSED** |
| **8/15** | Race Detector Suite | `controlapi`, `gatewayv2`, `worker` (`-race`) | **PASSED** |
| **9/15** | Frontend Linter | `npm run lint` (eslint) | **PASSED** |
| **10/15** | Frontend Type Checking | `npm run typecheck` (tsc) | **PASSED** |
| **11/15** | Frontend Unit Tests | `npm test` (tsx --test) | **PASSED** |
| **12/15** | Frontend Production Build | `npm run build` (Next.js 16) | **PASSED** |
| **13/15** | API Contract Consistency | `@ai-governance/contracts`, `@ai-governance/api-client` | **PASSED** |
| **14/15** | E2E User Journeys (A through G) | `go run ./cmd/e2e` (22 master execution phases) | **PASSED** |
| **15/15** | Performance & Resilience Suite | `go run ./cmd/perf/main.go` | **PASSED** |

---

## 8. Conclusion

Sentinel's local development environment is confirmed **secure, reproducible, and release-ready**. All configuration constraints, lifecycle tools, readiness probes, and test suites are active and verified.
