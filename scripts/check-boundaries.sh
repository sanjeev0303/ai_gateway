#!/usr/bin/env bash
# check-boundaries: fail the build if an internal layer imports a higher one.
#
# Dependency direction (each layer may import only itself and everything
# below it):
#
#   domain -> application -> interfaces -> infrastructure
#   ports   - owned by application/domain; implemented by infrastructure
#   platform - shared low-level plumbing; imports nothing from the layers above
#   bootstrap - composition root; nothing under internal/ may import it
#   cmd/*   - entry points; nothing under internal/ may import them
#
# Layers that do not exist yet are skipped by go list, so this file grows
# teeth exactly when the corresponding packages appear.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../backend"

MOD="$(go list -m)"
violations=0

check() {
    local layer="$1"
    shift
    local -a forbidden=("$@")
    local root="${MOD}/internal/${layer}"

    local pkg
    while IFS= read -r pkg; do
        [[ "$pkg" == "$root" ]] && continue
        local dep
        while IFS= read -r dep; do
            [[ "$dep" == "$pkg" ]] && continue
            local f
            for f in "${forbidden[@]}"; do
                case "$dep" in
                    "${MOD}/internal/${f}" | "${MOD}/internal/${f}/"*)
                        echo "VIOLATION: ${pkg#"${MOD}/"} imports ${dep#"${MOD}/"} (${layer} must not depend on ${f})" >&2
                        violations=1
                        ;;
                esac
            done
        done < <(go list -deps "$pkg")
    done < <(go list "./internal/${layer}/..." 2>/dev/null || true)
}

# domain is self-contained: no imports from the rest of the stack.
check domain application interfaces infrastructure ports bootstrap
# application depends on domain + ports only.
check application interfaces infrastructure bootstrap
# ports are plain interface definitions.
check ports application interfaces infrastructure bootstrap
# inbound interfaces implement application ports; never touch infra.
check interfaces infrastructure bootstrap
# infrastructure implements ports; never calls application/adapters.
check infrastructure application interfaces bootstrap
# platform is shared plumbing below everything.
check platform domain application interfaces infrastructure ports bootstrap

# Nothing under internal/ may import a cmd/* entry point.
if go list ./cmd/... 2>/dev/null | grep -q .; then
    while IFS= read -r entry; do
        if go list -deps ./internal/... 2>/dev/null | grep -q "^${entry}$"; then
            echo "VIOLATION: internal package imports entry point ${entry#"${MOD}/"}" >&2
            violations=1
        fi
    done < <(go list ./cmd/...)
fi

exit "$violations"