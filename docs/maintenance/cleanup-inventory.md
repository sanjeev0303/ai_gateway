# Repository Cleanup Inventory

Comprehensive reachability and dead-code inventory for Sentinel AI Gateway codebase maintenance.

## Candidate Inventory

| Path | Category | Classification | Reason | References Found | Build/Runtime Impact | Risk Level | Proposed Action | Validation Required |
|---|---|---|---|---|---|---|---|---|
| `audit_results.txt` | Repository Artifacts | `SAFE_TO_REMOVE` | 12.2 MB raw text grep output from past audit passes. Not source of truth. | 0 references across entire repo | None | Very Low | Remove file from git | Full test suite & build validation |
| `audit_results_clean.txt` | Repository Artifacts | `SAFE_TO_REMOVE` | 240 KB filtered grep text output from past audit passes. | 0 references across entire repo | None | Very Low | Remove file from git | Full test suite & build validation |
| `filtered_audit.txt` | Repository Artifacts | `SAFE_TO_REMOVE` | 41 KB filtered grep text output from past audit passes. | 0 references across entire repo | None | Very Low | Remove file from git | Full test suite & build validation |
| `backend/control-api` | Compiled Binaries | `SAFE_TO_REMOVE` | 21 MB compiled 64-bit ELF binary accidentally committed to git. Builds compile from `cmd/control-api`. | 0 references (Dockerfile builds from source) | None | Very Low | Remove file from git, update .gitignore | Verify Docker backend build & Go tests |
| `packages/cli/aigw` | Compiled Binaries | `SAFE_TO_REMOVE` | 6.3 MB compiled 64-bit ELF binary accidentally committed to git. CLI compiles from `main.go`. | 0 references | None | Very Low | Remove file from git, update .gitignore | Verify CLI builds & tests |
| `packages/contracts/pnpm-lock.yaml` | Package Management | `SAFE_TO_REMOVE` | Redundant pnpm lockfile in contracts workspace; repository standardized on `npm@11.12.1` with root `package-lock.json`. | 0 references | None | Very Low | Remove file from git | Run `npm run generate --workspace=@ai-governance/contracts` |
| `fix.py` | Abandoned Scripts | `SAFE_TO_REMOVE` | One-off scratch script used to patch `backend/cmd/e2e/main.go`. Code already incorporated. | 0 references | None | Very Low | Remove file from git | Run E2E and Go tests |
| `fix_e2e.py` | Abandoned Scripts | `SAFE_TO_REMOVE` | Incomplete 17-line scratch script that ends in a comment. | 0 references | None | Very Low | Remove file from git | Run E2E and Go tests |
| `patch_e2e.py` | Abandoned Scripts | `SAFE_TO_REMOVE` | One-off scratch script used to patch `backend/cmd/e2e/main.go`. Code already incorporated. | 0 references | None | Very Low | Remove file from git | Run E2E and Go tests |
| `recover.sh` | Abandoned Scripts | `SAFE_TO_REMOVE` | 502-line recovery script containing obsolete draft of `backend/cmd/e2e/main.go`. Superseded by production-ready E2E suite. | 0 references | None | Very Low | Remove file from git | Run E2E and Go tests |
| `audit.sh` | Development Tooling | `KEEP` | 257-byte grep shell script for repository audits. Useful for maintenance checks. | 0 references | None | Low | Retain in repository | None |
| `.gitignore` | Configuration | `SAFE_TO_SIMPLIFY` | Still refers to legacy path `apps/web/` instead of `app/` and lacks explicit ignores for root/backend binaries. | Active in git | None | Low | Update `.gitignore` with `app/.next/`, `app/out/`, and binary patterns | Check `git status` |
| `app/src/lib/demo/` | UI Mock Code | `KEEP` | Standalone demo data generator; 0 active imports by app pages, but retained for reference types per P46 decision. | 0 imports from dashboard | None | Medium | Retain in place per safety policy | Type check |
