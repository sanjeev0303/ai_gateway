// Package controlapi implements the control-plane HTTP surface. It exposes
// liveness/readiness probes, and, when the identity service is wired in, the
// v1 identity API (members, invitations, projects, teams, service accounts,
// credentials) plus, with the provider service, the M4 provider
// accounts/credentials API. Tenant, policy, billing and audit routes land with
// their milestones.
package controlapi

import (
	"log/slog"
	"net/http"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// Deps carries the dependencies the control-plane surface needs. Identity must
// be non-nil for the v1 routes to be registered; Providers adds the M4 provider
// accounts/credentials routes behind the same authentication.
type Deps struct {
	Log             *slog.Logger
	Identity        *identity.Service
	Providers       *provider.Service
	DevLoginEnabled bool

	// SessionCookieName names the HttpOnly auth cookie for the web UI
	// session flow; empty disables cookie authentication and /v1/auth/logout.
	SessionCookieName   string
	SessionCookieSecure bool
	CSRFEnabled         bool
}

// NewHandler assembles the control-api HTTP handler: health probes always,
// plus the v1 identity routes when deps.Identity is non-nil and the M4
// provider routes when deps.Providers is also non-nil, all wrapped in the
// shared middleware chain.
func NewHandler(log *slog.Logger, mw *httpx.Middleware, reg *health.Registry, deps Deps) http.Handler {
	mux := http.NewServeMux()
	registerHealthRoutes(mux, reg)
	if deps.Identity != nil {
		registerV1Routes(mux, deps)
		if deps.Providers != nil {
			registerProviderRoutes(mux, deps)
		}
	}
	return mw.Chain(mux)
}

func registerHealthRoutes(mux *http.ServeMux, reg *health.Registry) {
	mux.Handle("GET /livez", health.Handler(reg))
	mux.Handle("GET /readyz", health.Handler(reg))
	mux.Handle("GET /healthz", health.Handler(reg))
}
