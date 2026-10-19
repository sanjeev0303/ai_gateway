#!/usr/bin/env bash
# ==============================================================================
# Sentinel AI Gateway — Local Lifecycle Management Control Script
# Provides safe start, stop, inspection, and explicitly-confirmed reset.
# ==============================================================================

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

COMPOSE_FILE="deploy/compose/docker-compose.yml"
COMPOSE_ENV="deploy/compose/.env"
COMPOSE_ENV_EXAMPLE="deploy/compose/.env.example"
BACKEND_ENV="backend/.env"
BACKEND_ENV_EXAMPLE="backend/.env.example"

BOLD="\033[1m"
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
NC="\033[0m"

log_info() { echo -e "${CYAN}ℹ ${NC}$*"; }
log_ok()   { echo -e "${GREEN}✓ ${NC}$*"; }
log_warn() { echo -e "${YELLOW}⚠ ${NC}$*"; }
log_err()  { echo -e "${RED}✗ ${NC}$*" >&2; }

# Non-destructive environment bootstrap
ensure_env_files() {
    if [[ ! -f "$COMPOSE_ENV" ]]; then
        if [[ -f "$COMPOSE_ENV_EXAMPLE" ]]; then
            cp "$COMPOSE_ENV_EXAMPLE" "$COMPOSE_ENV"
            log_ok "Created $COMPOSE_ENV from template (preserved all defaults)"
        fi
    else
        log_info "Preserved existing $COMPOSE_ENV"
    fi

    if [[ ! -f "$BACKEND_ENV" ]]; then
        if [[ -f "$BACKEND_ENV_EXAMPLE" ]]; then
            cp "$BACKEND_ENV_EXAMPLE" "$BACKEND_ENV"
            log_ok "Created $BACKEND_ENV from template (preserved all defaults)"
        fi
    else
        log_info "Preserved existing $BACKEND_ENV"
    fi

    if [[ ! -f ".env" ]]; then
        if [[ -f ".env.example" ]]; then
            cp ".env.example" ".env"
            log_ok "Created .env from template (preserved all defaults)"
        fi
    else
        log_info "Preserved existing .env"
    fi

    if [[ ! -f "app/.env.local" ]]; then
        if [[ -f "app/.env.example" ]]; then
            cp "app/.env.example" "app/.env.local"
            log_ok "Created app/.env.local from template (preserved all defaults)"
        fi
    else
        log_info "Preserved existing app/.env.local"
    fi
}

cmd_start() {
    log_info "Verifying local environment configuration..."
    ensure_env_files

    local build_flag=""
    if [[ "${1:-}" == "--build" ]]; then
        build_flag="--build"
    elif ! docker image inspect ai-gateway-backend:dev >/dev/null 2>&1 || ! docker image inspect ai-gateway-web:dev >/dev/null 2>&1; then
        build_flag="--build"
    fi

    log_info "Starting Sentinel local service containers in ordered topology..."
    if [[ -n "$build_flag" ]]; then
        docker compose -f "$COMPOSE_FILE" up -d --build
    else
        docker compose -f "$COMPOSE_FILE" up -d
    fi

    log_info "Awaiting service functional readiness (/readyz)..."
    local retries=30
    local ready=false
    for i in $(seq 1 "$retries"); do
        if curl -s -f http://localhost:8081/readyz >/dev/null 2>&1 && \
           curl -s -f http://localhost:8084/readyz >/dev/null 2>&1 && \
           curl -s -f http://localhost:8082/readyz >/dev/null 2>&1 && \
           curl -s -f http://localhost:3000/ >/dev/null 2>&1; then
            ready=true
            break
        fi
        sleep 1
    done

    if [[ "$ready" == "true" ]]; then
        log_ok "All Sentinel local services are healthy and ready!"
        cmd_status
    else
        log_warn "Some services are still initializing or degraded. Current status:"
        cmd_status
    fi
}

cmd_stop() {
    log_info "Stopping Sentinel local services gracefully..."
    docker compose -f "$COMPOSE_FILE" stop
    log_ok "Services stopped without data destruction."
}

cmd_down() {
    log_info "Tearing down Sentinel service containers (retaining persistent volumes)..."
    docker compose -f "$COMPOSE_FILE" down
    log_ok "Containers removed; database, redis, and nats volumes preserved."
}

cmd_status() {
    local git_commit
    git_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local git_branch
    git_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

    echo ""
    echo "=========================================================================="
    echo "              SENTINEL AI GATEWAY: LOCAL RUNTIME STATUS                   "
    echo "=========================================================================="
    echo "Repository:   $REPO_ROOT"
    echo "Git Checkout: commit $git_commit ($git_branch)"
    echo "Time:         $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "--------------------------------------------------------------------------"
    printf "%-18s | %-24s | %-12s | %-12s\n" "SERVICE" "HOST ENDPOINT" "CONTAINER" "READINESS"
    echo "--------------------------------------------------------------------------"

    check_endpoint() {
        local svc="$1"
        local url="$2"
        local container="$3"
        local readyz_url="$4"

        local c_state="STOPPED"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container"; then
            c_state="RUNNING"
        fi

        local r_state="N/A"
        if [[ -n "$readyz_url" ]]; then
            local r_code
            r_code=$(curl -s -o /dev/null -w "%{http_code}" "$readyz_url" 2>/dev/null || echo "000")
            if [[ "$r_code" == "200" ]]; then
                r_state="READY (200)"
            elif [[ "$r_code" == "000" ]]; then
                r_state="DOWN"
            else
                r_state="DEGRADED ($r_code)"
            fi
        fi

        printf "%-18s | %-24s | %-12s | %-12s\n" "$svc" "$url" "$c_state" "$r_state"
    }

    check_endpoint "Control Plane" "http://localhost:8081" "ai-gateway-control-api-1" "http://localhost:8081/readyz"
    check_endpoint "Data Plane (v2)" "http://localhost:8084" "ai-gateway-gateway-v2-1" "http://localhost:8084/readyz"
    check_endpoint "Worker Service" "http://localhost:8082" "ai-gateway-worker-1" "http://localhost:8082/readyz"
    check_endpoint "Web Dashboard" "http://localhost:3000" "ai-gateway-app-1" "http://localhost:3000/"
    
    # Infrastructure services
    local pg_ready="DOWN"
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "ai-gateway-postgres-1"; then
        if docker exec ai-gateway-postgres-1 pg_isready -U gateway -d ai_gateway >/dev/null 2>&1; then
            pg_ready="READY (5432)"
        else
            pg_ready="DEGRADED"
        fi
        printf "%-18s | %-24s | %-12s | %-12s\n" "PostgreSQL (16)" "localhost:5432" "RUNNING" "$pg_ready"
    else
        printf "%-18s | %-24s | %-12s | %-12s\n" "PostgreSQL (16)" "localhost:5432" "STOPPED" "DOWN"
    fi

    local redis_ready="DOWN"
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "ai-gateway-redis-1"; then
        local rpass="${AIGW_REDIS_PASSWORD:-${REDIS_PASSWORD:-change-me}}"
        if docker exec -e REDISCLI_AUTH="$rpass" ai-gateway-redis-1 redis-cli ping >/dev/null 2>&1; then
            redis_ready="READY (PONG)"
        else
            redis_ready="DEGRADED"
        fi
        printf "%-18s | %-24s | %-12s | %-12s\n" "Redis (7)" "localhost:6379" "RUNNING" "$redis_ready"
    else
        printf "%-18s | %-24s | %-12s | %-12s\n" "Redis (7)" "localhost:6379" "STOPPED" "DOWN"
    fi

    local nats_ready="DOWN"
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "ai-gateway-nats-1"; then
        if curl -s -f http://localhost:8222/healthz >/dev/null 2>&1; then
            nats_ready="READY (8222)"
        else
            nats_ready="DEGRADED"
        fi
        printf "%-18s | %-24s | %-12s | %-12s\n" "NATS JetStream" "localhost:4222 / :8222" "RUNNING" "$nats_ready"
    else
        printf "%-18s | %-24s | %-12s | %-12s\n" "NATS JetStream" "localhost:4222 / :8222" "STOPPED" "DOWN"
    fi

    echo "=========================================================================="
    echo ""
}

cmd_reset() {
    local force=false
    for arg in "$@"; do
        if [[ "$arg" == "--confirm" || "$arg" == "-f" ]]; then
            force=true
        fi
    done

    echo -e "${RED}${BOLD}==========================================================================${NC}"
    echo -e "${RED}${BOLD}  DANGER: DESTRUCTIVE LOCAL DATA RESET REQUESTED                         ${NC}"
    echo -e "${RED}${BOLD}==========================================================================${NC}"
    echo "This operation will completely destroy:"
    echo "  - PostgreSQL persistent volume (pgdata): all organizations, audit logs, models, users"
    echo "  - Redis persistent volume (redisdata): rate-limits and caches"
    echo "  - NATS JetStream persistent volume (natsdata): queued telemetry"
    echo "  - Local encrypted secrets store volume (secrets)"
    echo ""

    if [[ "$force" != "true" ]]; then
        if [[ -t 0 ]]; then
            read -r -p "To confirm data destruction, type 'RESET' exactly: " confirmation
            if [[ "$confirmation" != "RESET" ]]; then
                log_warn "Reset aborted by user. No data was destroyed."
                exit 0
            fi
        else
            log_err "Destructive reset requires interactive terminal or explicit --confirm flag."
            log_err "Aborting to prevent accidental data loss."
            exit 1
        fi
    fi

    log_warn "Destroying all local service containers and persistent storage volumes..."
    docker compose -f "$COMPOSE_FILE" down -v

    log_info "Restarting services from pristine baseline..."
    ensure_env_files
    docker compose -f "$COMPOSE_FILE" up -d --build

    log_info "Waiting for database and running schema migrations..."
    docker compose -f "$COMPOSE_FILE" run --rm migrate

    log_ok "Local environment reset completed safely."
    cmd_status
}

cmd_logs() {
    docker compose -f "$COMPOSE_FILE" logs -f "$@"
}

cmd_setup() {
    log_info "Running local development setup..."
    ensure_env_files
    log_info "Verifying required local dependencies..."
    local missing=()
    for dep in go node npm docker curl; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing+=("$dep")
        fi
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_err "Missing required local dependencies: ${missing[*]}"
        exit 1
    fi
    log_ok "Toolchains detected (Go $(go version | awk '{print $3}'), Node $(node -v), npm $(npm -v), $(docker --version | awk '{print $1,$2,$3}'))"
    log_ok "Local development environment is configured and ready."
}

cmd_migrate() {
    log_info "Running database schema migrations..."
    ensure_env_files
    if docker compose -f "$COMPOSE_FILE" ps postgres 2>/dev/null | grep -q "Up"; then
        docker compose -f "$COMPOSE_FILE" run --rm migrate
    else
        log_info "Postgres container not running in compose; attempting local backend migration..."
        (cd "$REPO_ROOT/backend" && export AIGW_DATABASE_URL="${AIGW_DATABASE_URL:-postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable}" && go run ./cmd/migrate)
    fi
    log_ok "Database migrations completed successfully."
}

cmd_seed() {
    log_info "Running database demo seed..."
    ensure_env_files
    if docker compose -f "$COMPOSE_FILE" ps postgres 2>/dev/null | grep -q "Up"; then
        docker compose -f "$COMPOSE_FILE" --profile seed run --rm seed
    else
        log_info "Postgres container not running in compose; attempting local backend seed..."
        (cd "$REPO_ROOT/backend" && export AIGW_DATABASE_URL="${AIGW_DATABASE_URL:-postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable}" && go run ./cmd/seed)
    fi
    log_ok "Database seed completed successfully."
}

cmd_health() {
    log_info "Running functional health checks on Sentinel services..."
    local failed=0

    # PostgreSQL
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'ai-gateway-postgres-1'; then
        if docker exec ai-gateway-postgres-1 pg_isready -U gateway -d ai_gateway >/dev/null 2>&1; then
            log_ok "PostgreSQL: healthy"
        else
            log_err "PostgreSQL: pg_isready failed"
            failed=$((failed + 1))
        fi
    else
        log_warn "PostgreSQL: container not running"
    fi

    # Redis
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'ai-gateway-redis-1'; then
        local rpass="${AIGW_REDIS_PASSWORD:-${REDIS_PASSWORD:-change-me}}"
        if docker exec -e REDISCLI_AUTH="$rpass" ai-gateway-redis-1 redis-cli ping >/dev/null 2>&1; then
            log_ok "Redis: healthy"
        else
            log_err "Redis: ping failed"
            failed=$((failed + 1))
        fi
    else
        log_warn "Redis: container not running"
    fi

    # NATS
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q 'ai-gateway-nats-1'; then
        if curl -s -f http://localhost:8222/healthz >/dev/null 2>&1; then
            log_ok "NATS JetStream: healthy"
        else
            log_err "NATS JetStream: healthz probe failed"
            failed=$((failed + 1))
        fi
    else
        log_warn "NATS JetStream: container not running"
    fi

    # Control API
    if curl -s -f http://localhost:8081/readyz >/dev/null 2>&1; then
        log_ok "Control API (/readyz): healthy"
    else
        log_warn "Control API (/readyz): not reachable at http://localhost:8081"
    fi

    # Gateway v2
    if curl -s -f http://localhost:8084/readyz >/dev/null 2>&1; then
        log_ok "Gateway v2 (/readyz): healthy"
    else
        log_warn "Gateway v2 (/readyz): not reachable at http://localhost:8084"
    fi

    # Worker
    if curl -s -f http://localhost:8082/readyz >/dev/null 2>&1; then
        log_ok "Worker (/readyz): healthy"
    else
        log_warn "Worker (/readyz): not reachable at http://localhost:8082"
    fi

    if [[ $failed -gt 0 ]]; then
        log_err "Health check completed with $failed failure(s)"
        return 1
    fi
    log_ok "Health check evaluation complete."
    return 0
}

usage() {
    echo "Usage: $0 {setup|start|stop|down|status|reset [--confirm]|migrate|seed|health|logs [service]}"
    echo ""
    echo "Commands:"
    echo "  setup     Initialize local development environment and check toolchains"
    echo "  start     Safely start all local containers and verify functional readiness"
    echo "  stop      Stop containers gracefully without destroying data"
    echo "  down      Tear down containers while preserving persistent volumes"
    echo "  status    Show checkout commit, endpoints, container status, and /readyz health"
    echo "  reset     Destructive reset: wipe volumes and recreate (requires explicit confirmation)"
    echo "  migrate   Run database schema migrations"
    echo "  seed      Seed local database with demo records"
    echo "  health    Execute functional health probe suite against running services"
    echo "  logs      Follow container logs"
    exit 1
}

case "${1:-status}" in
    setup)
        cmd_setup
        ;;
    start)
        shift
        cmd_start "$@"
        ;;
    stop)
        cmd_stop
        ;;
    down)
        cmd_down
        ;;
    status)
        cmd_status
        ;;
    reset)
        shift
        cmd_reset "$@"
        ;;
    migrate)
        cmd_migrate
        ;;
    seed)
        cmd_seed
        ;;
    health)
        cmd_health
        ;;
    logs)
        shift
        cmd_logs "$@"
        ;;
    *)
        usage
        ;;
esac

