# Local Configuration & Runtime Service Wiring Verification

**Date**: 2026-10-19  
**Status**: **VERIFIED WITH DOCUMENTED LIMITATIONS**  
**Module**: [`backend/internal/platform/config`](file:///home/sanju/ai-project/ai_gateway/backend/internal/platform/config), [`backend/internal/bootstrap`](file:///home/sanju/ai-project/ai_gateway/backend/internal/bootstrap), [`deploy/compose`](file:///home/sanju/ai-project/ai_gateway/deploy/compose)

---

## 1. Overview & Architecture

Local configuration ensures that Redis, PostgreSQL, NATS, and application servers operate with synchronized, validated credentials without unsafe defaults or placeholder secrets:
- **Unified Redis Credentials**: Single source of truth for Redis authentication (`AIGW_REDIS_PASSWORD` / `REDIS_PASSWORD`) across `docker-compose.yml`, environment files, and service bootstraps.
- **Explicit Outage Policies**: Explicit fail-closed (`AIGW_RATE_LIMIT_FAIL_CLOSED=true`, `AIGW_BUDGET_FAIL_CLOSED=true`) vs degraded-local mode for Redis connectivity failures.
- **Secret Hardening**: Placeholder secrets (`change-me`, `your-secret-here`) are strictly rejected in staging and production modes.
- **Architectural Layer Decoupling**: Platform layer has zero dependencies on domain or application layers, verified via `scripts/check-boundaries.sh`.

---

## 2. Configuration & Wiring Matrix

| Subsystem | Configuration Keys | Validation & Runtime Behavior | Result |
| :--- | :--- | :--- | :--- |
| **Redis** | `AIGW_REDIS_ADDR`, `AIGW_REDIS_PASSWORD` | Validates credentials; connects with password auth; rejects invalid credentials during readiness probe. | **PASS** |
| **Rate Limiter Outage** | `AIGW_RATE_LIMIT_FAIL_CLOSED` | When true: rejects traffic on Redis outage (503 Service Unavailable). When false: degrades safely to local in-memory limiter with `Degraded: true`. | **PASS** |
| **Budget Guard Outage** | `AIGW_BUDGET_FAIL_CLOSED` | When true: blocks reservations on Redis outage. When false: falls back to local in-memory tracking with warning. | **PASS** |
| **NATS Event Bus** | `AIGW_EVENTBUS_URL`, `AIGW_EVENTBUS_STREAM` | Connects to NATS JetStream; ensures stream and consumer groups exist; verifies publishing and consumption. | **PASS** |
| **PostgreSQL** | `AIGW_DATABASE_URL` | Connects via pgx pool; executes Goose migrations; validates functional readiness via `SELECT 1`. | **PASS** |
| **Security Defaults** | `ALLOW_INSECURE_DEV`, `AUTH_DEV_LOGIN_ENABLED` | Insecure developer logins and disabled CSRF require explicit opt-in flags; default is fail-safe. | **PASS** |
| **Layer Boundaries** | Domain / Platform Decoupling | `scripts/check-boundaries.sh` verifies platform does not import domain or application code. | **PASS** |

---

## 3. Discovered Defects & Fixes

### 3.1 Redis Credential Desynchronization
- **Root Cause**: `deploy/compose/docker-compose.yml` previously used hardcoded `change-me` while `backend/.env` expected a different password or empty password.
- **Fix**: Unified Redis password resolution to use `${AIGW_REDIS_PASSWORD:-${REDIS_PASSWORD:-change-me}}` across both `docker-compose.yml` and backend config.
- **Verification**: Tested via `scripts/verify-local.sh` and config validation tests.

### 3.2 Platform Dependency Violation
- **Root Cause**: `backend/internal/platform/config/config.go` imported `internal/domain` to call `domain.ParsePrivacyMode`, violating architecture boundary rule.
- **Fix**: Replaced the domain import with an internal switch validation in `config.go`, ensuring `platform` remains completely decoupled from higher layers.
- **Verification**: `bash scripts/check-boundaries.sh` passes with exit code 0.

---

## 4. Tests Executed & Results

| Test Suite | Command | Exit Code | Outcome |
| :--- | :--- | :--- | :--- |
| **Config Unit Tests** | `go test -count=1 ./internal/platform/config/...` | `0` | **PASS** |
| **Bootstrap & Wiring Tests** | `go test -count=1 ./internal/bootstrap/...` | `0` | **PASS** |
| **Architecture Boundaries** | `bash scripts/check-boundaries.sh` | `0` | **PASS** |
| **Config Validation CLI** | `go run ./cmd/config-validate` | `0` | **PASS** |

---

## 5. Remaining Limitations

- **Production Secret Stores**: In local development, secrets are loaded from `.env` files via `godotenv`. HashiCorp Vault or AWS Secrets Manager integrations require external cloud infrastructure.
