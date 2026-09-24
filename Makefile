.PHONY: build test vet lint dev-up dev-down tidy

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

# --- Local services (Postgres + Redis) ---
dev-up:
	docker compose -f deploy/compose/docker-compose.yml up -d

dev-down:
	docker compose -f deploy/compose/docker-compose.yml down