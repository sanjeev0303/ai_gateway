# Safe Repository Cleanup Report

## Executive Summary
This cleanup phase targeted dead code, obsolete generated audit dumps, abandoned one-off scripts, and accidental tracked executable binaries without altering application runtime behavior, public APIs, or database migrations.

---

## 1. Metrics: Before vs. After

| Metric | Before Cleanup | After Cleanup | Net Difference | Notes |
|---|---|---|---|---|
| **Tracked Git Files** | 890 files | 880 files | **-10 files** | 10 obsolete/dead files removed |
| **Tracked Repository Size** | 51 MB | 13 MB | **-38 MB (-74.5%)** | Measured via `du --files0-from` across `git ls-files` |
| **Workspace Disk Size (with artifacts)** | 1.3 GB | 1.3 GB | -38 MB source | Includes 644 MB `node_modules`, 444 MB `app/.next` |

---

## 2. Removed Artifacts

### A. Obsolete Generated Audit Dumps
* `audit_results.txt` (12.2 MB): Unreferenced raw text dump from historical grep runs.
* `audit_results_clean.txt` (240 KB): Unreferenced filtered text dump from historical grep runs.
* `filtered_audit.txt` (41 KB): Unreferenced filtered text dump from historical grep runs.

### B. Accidental Tracked Executable Binaries
* `backend/control-api` (21.0 MB): 64-bit ELF executable compiled locally and accidentally tracked in git. Deployment and Docker environments build directly from source (`cmd/control-api`).
* `packages/cli/aigw` (6.3 MB): 64-bit ELF executable compiled locally and accidentally tracked in git. CLI builds from `packages/cli/main.go`.

### C. Redundant Package Manager Lockfile
* `packages/contracts/pnpm-lock.yaml` (6.0 KB): Obsolete pnpm lockfile. The monorepo has standardized on `npm@11.12.1` with root `package-lock.json` and npm workspaces.

### D. Abandoned One-Off Patch and Recovery Scripts
* `fix.py` (543 B): One-off script previously used to patch service account creation in `backend/cmd/e2e/main.go`.
* `fix_e2e.py` (478 B): Incomplete 17-line scratch script ending in a comment.
* `patch_e2e.py` (793 B): One-off script previously used to patch `backend/cmd/e2e/main.go`.
* `recover.sh` (14.6 KB): Shell script containing an outdated stub of `backend/cmd/e2e/main.go` that was superseded by the production-ready E2E suite.

---

## 3. Code Symbols & Dependencies Removed
* **Dependencies Removed**: None removed from `package.json` or `go.mod` in this pass. `go mod tidy` in both `backend` and `packages/cli` confirmed zero unused Go modules.
* **Code Symbols Removed**: 0 application symbols removed. All removals were external to active compilation units.

---

## 4. Retained Candidates (Preserved for Safety & Maintenance)
* `audit.sh` (257 B): Preserved as an ad-hoc developer utility for pattern checking.
* `app/src/lib/demo/` (108 KB): Preserved per P46 architecture record as a schema and mock reference, despite zero runtime dashboard imports.
* `scripts/check-boundaries.sh` and `scripts/verify-local.sh`: Preserved as active architectural boundary check and local release-readiness verification scripts.

---

## 5. Configuration Improvements
* Updated [.gitignore](file:///home/sanju/ai-project/ai_gateway/.gitignore) to:
  * Ignore `backend/control-api` and `packages/cli/aigw` to prevent future commits of compiled binaries.
  * Explicitly ignore `app/.next/` and `app/out/` aligned with the canonical `app` folder structure.

---

## 6. Verification and Test Results

All verification suites ran post-cleanup and passed completely:

1. **Go Compilation**:
   * `backend`: `go build ./...` — **PASS (exit 0)**
   * `packages/cli`: `go build ./...` — **PASS (exit 0)**
   * `backend/cmd/e2e`: `go build ./cmd/e2e` — **PASS (exit 0)**

2. **Go Test Suite**:
   * `backend`: `go test ./...` — **PASS (all packages pass, exit 0)**
   * `backend/internal/platform/...`: `go test -race` — **PASS (exit 0, 0 data races)**

3. **Frontend Tests**:
   * `app`: `npm run test` — **24 passed, 0 failed (exit 0)**

4. **Frontend TypeScript & Typecheck**:
   * `app`: `npx tsc --noEmit` — **0 errors (exit 0)**
   * Workspace: `npm run typecheck` across `@ai-governance/api-client` and `@ai-governance/contracts` — **PASS (exit 0)**

5. **OpenAPI Type Generation**:
   * `npm run generate --workspace=@ai-governance/contracts` — **PASS (exit 0)**

6. **Frontend Lint**:
   * `app`: `npm run lint` — **0 errors (388 existing warnings, exit 0)**

7. **Docker Services**:
   * All containers (`control-api`, `gateway-v2`, `worker`, `app`, `postgres`, `redis`, `nats`) healthy and active.

---

## 7. Remaining Opportunities
* Consider removing `app/src/lib/demo/` once all legacy reference types are fully mirrored in domain types.
* Clean up untracked local PNG screenshots in `.gitignore`'d `scratch/` during regular disk maintenance.
