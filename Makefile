.PHONY: build test vet lint tidy format generate db-migrate db-seed dev-up dev-down dev-seed dev-logs dev-gateway dev-gateway-v1 dev-control-api dev-worker check-boundaries

# --- Go backend ---
build:
	cd backend && go build ./...

test:
	cd backend && go test -race -shuffle=on -count=1 ./...

vet:
	cd backend && go vet ./...

lint:
	cd backend && gofumpt -l . && golangci-lint run ./... && nilaway ./...

tidy:
	cd backend && go mod tidy

format:
	cd backend && gofumpt -w .

# --- Dependency direction gate (domain must not import up-stack layers) ---
check-boundaries:
	bash scripts/check-boundaries.sh

# --- OpenAPI -> TypeScript contract (packages/contracts) ---
generate:
	npm run generate --workspace=@ai-governance/contracts

# --- Local database lifecycle (migrations + seed; AIGW_DATABASE_URL from backend/.env) ---
db-migrate:
	cd backend && go run ./cmd/migrate

db-seed:
	cd backend && go run ./cmd/seed

# --- Full stack (Postgres + Redis + NATS + backend processes + web), via docker compose ---
# Reads deploy/compose/.env for credentials (see deploy/compose/.env.example).
dev-up:
	docker compose -f deploy/compose/docker-compose.yml up -d --build

dev-down:
	docker compose -f deploy/compose/docker-compose.yml down

# One-shot seed (behind the "seed" profile so plain `dev-up` skips it).
dev-seed:
	docker compose -f deploy/compose/docker-compose.yml --profile seed up seed

dev-logs:
	docker compose -f deploy/compose/docker-compose.yml logs -f

# --- Run each process with its own listen port ---
# gateway-v2 is the canonical data plane (ADR-5); dev-gateway starts it.
dev-gateway:
	cd backend && AIGW_SERVER_ADDR=:8080 go run ./cmd/gateway-v2

# Deprecated: v1 data plane, frozen and not deployed (ADR-5). Kept for one
# release as a rollback path; remove once no deployment references it.
dev-gateway-v1:
	cd backend && AIGW_SERVER_ADDR=:8080 go run ./cmd/gateway

dev-control-api:
	cd backend && AIGW_SERVER_ADDR=:8081 go run ./cmd/control-api

dev-worker:
	cd backend && AIGW_SERVER_ADDR=:8082 go run ./cmd/worker