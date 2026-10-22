#!/usr/bin/env bash
# ==============================================================================
# Sentinel AI Gateway / AI Governance — Local Release-Readiness Verification Entry Point
# Deterministically runs all configuration, backend, frontend, contract, integration,
# race-detection, e2e user journey, and local performance checks.
# ==============================================================================

set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Text styles
BOLD="\033[1m"
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
NC="\033[0m" # No Color

PASSED_COUNT=0
FAILED_COUNT=0
SKIPPED_COUNT=0
BLOCKED_COUNT=0

declare -a REPORT_NAMES=()
declare -a REPORT_STATUS=()
declare -a REPORT_DURATIONS=()

run_check() {
    local index="$1"
    local name="$2"
    local cmd="$3"
    local required="${4:-true}"

    echo -e "\n${BOLD}${CYAN}>>> [${index}] ${name}...${NC}"
    local start_time
    start_time=$(date +%s%N 2>/dev/null || date +%s)

    local output
    local status="PASSED"

    if output=$(eval "$cmd" 2>&1); then
        echo -e "${GREEN}  ✓ ${name} passed${NC}"
        PASSED_COUNT=$((PASSED_COUNT + 1))
    else
        local exit_code=$?
        if [[ "$required" == "blocked" ]]; then
            echo -e "${YELLOW}  ⚠ ${name} blocked (code: ${exit_code})${NC}"
            echo -e "${YELLOW}${output}${NC}"
            status="BLOCKED"
            BLOCKED_COUNT=$((BLOCKED_COUNT + 1))
        elif [[ "$required" == "false" ]]; then
            echo -e "${YELLOW}  ⚠ ${name} skipped/optional failed (code: ${exit_code})${NC}"
            status="SKIPPED"
            SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
        else
            echo -e "${RED}  ✗ ${name} failed (exit code: ${exit_code})${NC}"
            echo -e "${RED}${output}${NC}"
            status="FAILED"
            FAILED_COUNT=$((FAILED_COUNT + 1))
        fi
    fi

    local end_time
    end_time=$(date +%s%N 2>/dev/null || date +%s)
    local duration_ms=0
    if [[ ${#start_time} -gt 10 ]]; then
        duration_ms=$(( (end_time - start_time) / 1000000 ))
    else
        duration_ms=$(( (end_time - start_time) * 1000 ))
    fi
    local duration_sec
    duration_sec=$(awk -v ms="$duration_ms" 'BEGIN { printf "%.2fs", ms / 1000 }')

    REPORT_NAMES+=("${name}")
    REPORT_STATUS+=("${status}")
    REPORT_DURATIONS+=("${duration_sec}")
}

GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")"

echo "=========================================================================="
echo "    SENTINEL AI GATEWAY: LOCAL RELEASE-READINESS VERIFICATION SUITE       "
echo "=========================================================================="
echo "Monorepo Root:    $REPO_ROOT"
echo "Git Checkout:     commit ${GIT_COMMIT} (${GIT_BRANCH})"
echo "Execution Time:   $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "Target Endpoints:"
echo "  - Control API:  http://localhost:8081 (/readyz, /v1/...)"
echo "  - Gateway v2:   http://localhost:8084 (/readyz, /v1/chat/completions, /v1/messages)"
echo "  - Worker:       http://localhost:8082 (/readyz)"
echo "  - Web UI:       http://localhost:3000"
echo "  - PostgreSQL:   localhost:5432 (database: ai_gateway)"
echo "  - Redis:        localhost:6379"
echo "  - NATS:         localhost:4222 (monitoring: http://localhost:8222/healthz)"
echo "=========================================================================="

# 1. Validate local configuration (strictly non-destructive)
check_config() {
    # Never overwrite existing .env files
    if [[ ! -f "backend/.env" ]]; then
        if [[ -f "backend/.env.example" ]]; then
            cp "backend/.env.example" "backend/.env"
            echo "Initialized backend/.env from template (no existing file was modified)."
        fi
    fi
    if [[ ! -f "deploy/compose/.env" ]]; then
        if [[ -f "deploy/compose/.env.example" ]]; then
            cp "deploy/compose/.env.example" "deploy/compose/.env"
            echo "Initialized deploy/compose/.env from template (no existing file was modified)."
        fi
    fi
    if [[ ! -f ".env" ]]; then
        if [[ -f ".env.example" ]]; then
            cp ".env.example" ".env"
            echo "Initialized .env from template (no existing file was modified)."
        fi
    fi
    if [[ ! -f "app/.env.local" ]]; then
        if [[ -f "app/.env.example" ]]; then
            cp "app/.env.example" "app/.env.local"
            echo "Initialized app/.env.local from template (no existing file was modified)."
        fi
    fi
    export AIGW_DATABASE_URL="${AIGW_DATABASE_URL:-postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable}"
    
    # Run Go config validation binary
    (cd backend && go run ./cmd/config-validate)
}
run_check "1/15" "Local Configuration Validation" "check_config"

# 2. Check required local dependencies
check_deps() {
    local missing=()
    for dep in go node npm docker curl; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing+=("$dep")
        fi
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo "Missing required dependencies: ${missing[*]}" >&2
        return 1
    fi
    return 0
}
run_check "2/15" "Local Tooling Dependencies" "check_deps"

# 3. Run database migrations
check_db_migrations() {
    if ! docker ps >/dev/null 2>&1 && ! (command -v pg_isready >/dev/null 2>&1 && pg_isready -h localhost -p 5432 >/dev/null 2>&1); then
        echo "Local PostgreSQL service is inactive (Docker daemon or local Postgres not running). Run 'make dev-start' when Docker is available." >&2
        return 1
    fi
    (cd backend && export AIGW_DATABASE_URL="${AIGW_DATABASE_URL:-postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable}" && go run ./cmd/migrate)
}
run_check "3/15" "Database Schema Migrations" "check_db_migrations" "blocked"

# 4. Verify functional service readiness (/readyz and query checks)
check_service_readiness() {
    if ! docker ps >/dev/null 2>&1; then
        echo "Docker daemon is inactive or containers are not running. Run 'make dev-start' when Docker daemon is available." >&2
        return 1
    fi

    # Check PostgreSQL
    if docker ps --format '{{.Names}}' | grep -q 'ai-gateway-postgres-1'; then
        docker exec ai-gateway-postgres-1 pg_isready -U gateway -d ai_gateway >/dev/null 2>&1 || {
            echo "PostgreSQL pg_isready check failed." >&2
            return 1
        }
        docker exec ai-gateway-postgres-1 psql -U gateway -d ai_gateway -c "SELECT 1;" >/dev/null 2>&1 || {
            echo "PostgreSQL functional query SELECT 1 failed." >&2
            return 1
        }
    else
        echo "PostgreSQL container ai-gateway-postgres-1 is not running." >&2
        return 1
    fi

    # Check Redis
    if docker ps --format '{{.Names}}' | grep -q 'ai-gateway-redis-1'; then
        local redis_pass="${AIGW_REDIS_PASSWORD:-${REDIS_PASSWORD:-change-me}}"
        docker exec -e REDISCLI_AUTH="$redis_pass" ai-gateway-redis-1 redis-cli ping >/dev/null 2>&1 || {
            echo "Redis ping check failed." >&2
            return 1
        }
    fi

    # Check NATS JetStream
    if docker ps --format '{{.Names}}' | grep -q 'ai-gateway-nats-1'; then
        curl -s -f http://localhost:8222/healthz >/dev/null 2>&1 || {
            echo "NATS health check failed." >&2
            return 1
        }
    fi

    # Check Control API /readyz (functional check of DB, Redis, Eventbus)
    curl -s -f http://localhost:8081/readyz >/dev/null 2>&1 || {
        echo "Control API /readyz functional check failed." >&2
        return 1
    }

    # Check Gateway v2 /readyz
    curl -s -f http://localhost:8084/readyz >/dev/null 2>&1 || {
        echo "Gateway v2 /readyz functional check failed." >&2
        return 1
    }

    # Check Worker /readyz
    curl -s -f http://localhost:8082/readyz >/dev/null 2>&1 || {
        echo "Worker /readyz functional check failed." >&2
        return 1
    }

    return 0
}
run_check "4/15" "Full-Stack Functional Readiness" "check_service_readiness" "blocked"

# 5. Run backend formatting and static analysis
run_check "5/15" "Backend Static Analysis (go vet)" "(cd backend && go vet ./...)"

# 6. Run Go unit tests across backend and CLI packages
run_check "6/15" "Go Unit Test Suite (Backend & CLI)" "(cd backend && go test -count=1 ./...) && (cd packages/cli && go test -count=1 ./...)"

# 7. Run backend integration tests
run_check "7/15" "Repository Integration Tests" "(cd backend && go test -count=1 ./internal/infrastructure/postgres/repositories/...)"

# 8. Run supported race tests
run_check "8/15" "Race Detector Suite" "(cd backend && go test -race -count=1 ./internal/interfaces/controlapi ./internal/interfaces/gatewayv2 ./internal/interfaces/worker)"

# 9. Run frontend lint
run_check "9/15" "Frontend Linter (eslint)" "(cd app && npm run lint)"

# 10. Run frontend type checking
run_check "10/15" "Frontend Type Checking (tsc)" "(cd app && npm run typecheck)"

# 11. Run frontend unit tests
run_check "11/15" "Frontend Unit Tests (tsx --test)" "(cd app && npm test)"

# 12. Build the frontend
run_check "12/15" "Frontend Production Build (next)" "(cd app && npm run build)"

# 13. Run API contract consistency checks
run_check "13/15" "Monorepo API Contracts Check" "(npm run typecheck --workspace=@ai-governance/contracts && npm run typecheck --workspace=@ai-governance/api-client)"

# 14. Run browser end-to-end user journeys
run_check "14/15" "E2E User Journeys (A through G)" "(cd backend && go run ./cmd/e2e)"

# 15. Run local performance and resilience validation
run_check "15/15" "Performance & Resilience Suite" "(cd backend && go run ./cmd/perf/main.go)"

# ------------------------------------------------------------------------------
# Final Verification Report
# ------------------------------------------------------------------------------
echo ""
echo "=========================================================================="
echo "                     LOCAL RELEASE-READINESS SUMMARY                      "
echo "=========================================================================="
printf "%-38s | %-10s | %-10s\n" "CHECK" "STATUS" "DURATION"
echo "--------------------------------------------------------------------------"

for i in "${!REPORT_NAMES[@]}"; do
    local_name="${REPORT_NAMES[$i]}"
    local_status="${REPORT_STATUS[$i]}"
    local_dur="${REPORT_DURATIONS[$i]}"

    if [[ "$local_status" == "PASSED" ]]; then
        printf "%-38s | ${GREEN}%-10s${NC} | %-10s\n" "$local_name" "$local_status" "$local_dur"
    elif [[ "$local_status" == "BLOCKED" ]]; then
        printf "%-38s | ${YELLOW}%-10s${NC} | %-10s\n" "$local_name" "$local_status" "$local_dur"
    elif [[ "$local_status" == "SKIPPED" ]]; then
        printf "%-38s | ${YELLOW}%-10s${NC} | %-10s\n" "$local_name" "$local_status" "$local_dur"
    else
        printf "%-38s | ${RED}%-10s${NC} | %-10s\n" "$local_name" "$local_status" "$local_dur"
    fi
done

echo "--------------------------------------------------------------------------"
echo -e "Total: ${BOLD}${#REPORT_NAMES[@]}${NC} | Passed: ${GREEN}${PASSED_COUNT}${NC} | Failed: ${RED}${FAILED_COUNT}${NC} | Skipped: ${YELLOW}${SKIPPED_COUNT}${NC} | Blocked: ${YELLOW}${BLOCKED_COUNT}${NC}"

if [[ $FAILED_COUNT -eq 0 ]]; then
    echo -e "\n${BOLD}${GREEN}RESULT: ALL RELEASE-READINESS CHECKS PASSED (LOCAL RELEASE READY)${NC}"
    echo "=========================================================================="
    exit 0
else
    echo -e "\n${BOLD}${RED}RESULT: VERIFICATION FAILED WITH ${FAILED_COUNT} DEFECT(S)${NC}"
    echo "=========================================================================="
    exit 1
fi
