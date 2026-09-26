// Package worker implements the async-worker surface. The worker owns no
// user-facing routes; it exposes the same health probes so orchestrators can
// liveness/readiness-check it uniformly with the other processes.
package worker

import (
	"log/slog"
	"net/http"

	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// NewHandler assembles the worker HTTP handler: health probes wrapped in the
// shared middleware chain.
func NewHandler(log *slog.Logger, mw *httpx.Middleware, reg *health.Registry) http.Handler {
	mux := http.NewServeMux()
	registerHealthRoutes(mux, reg)
	return mw.Chain(mux)
}

func registerHealthRoutes(mux *http.ServeMux, reg *health.Registry) {
	mux.Handle("GET /livez", health.Handler(reg))
	mux.Handle("GET /readyz", health.Handler(reg))
	mux.Handle("GET /healthz", health.Handler(reg))
}
