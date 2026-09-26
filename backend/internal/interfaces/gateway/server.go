// Package gateway implements the data-plane HTTP surface: the health probes
// and the provider streaming proxy that fronts tenant Anthropic upstreams.
package gateway

import (
	"log/slog"
	"net/http"

	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// Deps carries optional data-plane dependencies; routes register only when
// their dependency is non-nil.
type Deps struct {
	Proxy *Proxy
}

// NewHandler assembles the gateway HTTP handler: health probes, plus the
// provider streaming proxy when wired, wrapped in the shared middleware chain.
func NewHandler(log *slog.Logger, mw *httpx.Middleware, reg *health.Registry, deps Deps) http.Handler {
	mux := http.NewServeMux()
	registerHealthRoutes(mux, reg)
	if deps.Proxy != nil {
		registerProxyRoutes(mux, deps.Proxy)
	}
	return mw.Chain(mux)
}

func registerHealthRoutes(mux *http.ServeMux, reg *health.Registry) {
	mux.Handle("GET /livez", health.Handler(reg))
	mux.Handle("GET /readyz", health.Handler(reg))
	mux.Handle("GET /healthz", health.Handler(reg))
}

func registerProxyRoutes(mux *http.ServeMux, p *Proxy) {
	// Go 1.22+ method patterns: non-POST requests get an automatic 405.
	mux.Handle("POST /v1/messages", p)
	mux.Handle("POST /v1/messages/count_tokens", p)
}
