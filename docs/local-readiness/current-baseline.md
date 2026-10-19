# Sentinel AI Gateway — Current Local Baseline Assessment

## 1. Environment & Toolchains Baseline

- **Evaluation Date**: 2026-10-19
- **Repository Checkout**: Commit `c64ef69` on branch `main`
- **Go Toolchain**: `go version go1.25.10 X:nodwarf5 linux/amd64` (meets repository requirement `go 1.25.7` in `backend/go.mod`)
- **Node.js**: `v24.15.0` (meets `package.json` requirement `node >= 20`)
- **Package Manager**: `npm 11.12.1` (matches declared `packageManager: npm@11.12.1`)
- **Docker**: `Docker version 29.5.3, build d1c06ef` (daemon socket inactive in current shell; hermetic test suites execute without daemon dependency)
- **Host OS**: Linux fedora (x86_64, 12 cores, 15.3 GiB RAM)

---

## 2. Command Execution & Baseline Status

| Layer / Verification Gate | Command | Status | Notes |
| :--- | :--- | :--- | :--- |
| **Go Formatting** | `gofmt -l backend packages/cli` | **PASS** | 0 unformatted Go files |
| **Backend & CLI Vetting** | `go vet ./...` (backend & cli) | **PASS** | 0 static analysis defects |
| **Architecture Boundaries** | `bash scripts/check-boundaries.sh` | **PASS** | 0 domain/platform layer violations |
| **Backend Race Detector** | `go test -race ./internal/interfaces/...` | **PASS** | 0 data races in controlapi, gatewayv2, worker |
| **Go Unit & Integration** | `go test -count=1 ./...` (backend) | **PASS** | All 26 packages pass |
| **CLI Test Suite** | `go test -v ./...` (packages/cli) | **PASS** | REPL, slash commands, SSRF defense pass |
| **TypeScript SDK** | `npm test --workspace=@ai-governance/api-client` | **PASS** | 15/15 tests pass (streaming SSE, 503 retry, timeout) |
| **Contracts Typecheck** | `npm run typecheck --workspace=@ai-governance/contracts` | **PASS** | Generated types consistent with OpenAPI contract |
| **Frontend Linter** | `npm run lint` (app) | **PASS** | 0 errors, 489 warnings |
| **Frontend Typecheck** | `npm run typecheck` (app) | **PASS** | 0 TypeScript errors |
| **Frontend Unit Tests** | `npm test` (app) | **PASS** | 34/34 tests pass |
| **Frontend Build** | `npm run build` (app) | **PASS** | 49/49 static & dynamic routes compiled |

---

## 3. Baseline Environment Limitations & Boundaries

1. **Docker Daemon Socket**:
   - `docker ps` returns connection failure because the Docker Desktop daemon socket (`~/.docker/desktop/docker.sock`) is not running.
   - All acceptance tests execute using embedded PostgreSQL (`testutil.StartPostgres`), in-memory Redis, and mock event buses.
2. **Live Third-Party Cloud LLM APIs**:
   - Live upstream provider tests require real external API keys (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `HUGGINGFACE_API_KEY`).
   - In accordance with repository safety standards, live tests cleanly skip when keys are absent; hermetic mock provider suites run 100%.
3. **Live Payment Settlements**:
   - Live merchant charges require production Stripe / Razorpay merchant accounts.
   - Test mode and simulation mode are verified end-to-end.
