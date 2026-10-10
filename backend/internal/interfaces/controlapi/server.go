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

	"ai-governance-saas/backend/internal/application/activity"
	"ai-governance-saas/backend/internal/application/alert"
	"ai-governance-saas/backend/internal/application/auditlog"
	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/budget"
	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/guardrails"
	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/application/usage"
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
	Usage           *usage.Service
	Activity        *activity.Service
	Conversations   *conversation.Service
	AuditLogs       *auditlog.Service
	Alerts          *alert.Service
	SecurityEvents  *securityevent.Service
	Billing         *billing.Service
	Budget          *budget.Service
	Policy          *policy.Service
	Guardrails      *guardrails.Service
	GuardrailsEval  *guardrails.Evaluator
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
		if deps.Usage != nil {
			registerUsageRoutes(mux, deps)
		}
		if deps.Activity != nil {
			registerActivityRoutes(mux, deps)
		}
		if deps.Conversations != nil {
			registerConversationRoutes(mux, deps)
		}
		if deps.AuditLogs != nil {
			registerAuditLogRoutes(mux, deps)
		}
		if deps.Alerts != nil {
			registerAlertRoutes(mux, deps)
		}
		if deps.SecurityEvents != nil {
			registerSecurityEventRoutes(mux, deps)
		}
		if deps.Billing != nil {
			registerBillingRoutes(mux, deps)
		}
		if deps.Guardrails != nil {
			registerGuardrailRoutes(mux, deps)
		}
		if deps.Budget != nil {
			registerBudgetRoutes(mux, deps)
		}
	}
	return mw.Chain(mux)
}

func registerHealthRoutes(mux *http.ServeMux, reg *health.Registry) {
	mux.Handle("GET /livez", health.Handler(reg))
	mux.Handle("GET /readyz", health.Handler(reg))
	mux.Handle("GET /healthz", health.Handler(reg))
}
