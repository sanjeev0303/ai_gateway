package controlapi

import "net/http"

// registerV1Routes wires the v1 identity surface onto mux. Every v1 route runs
// behind requireAuth; the auth entry points are the exceptions. They register
// only when their feature flag is on, so a disabled deployment simply 404s
// them.
func registerV1Routes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)

	if deps.SessionCookieName != "" {
		// signup and signin gate on cookies because credentials are never
		// returned in a response body: without cookies a caller would create an
		// account it has no way to authenticate as. refresh and logout read the
		// cookies directly and so need the double-submit CSRF check rather than
		// requireAuth, which authenticates on the access token alone.
		mux.Handle("POST /v1/auth/signup", http.HandlerFunc(a.signup))
		mux.Handle("POST /v1/auth/signin", http.HandlerFunc(a.signin))
		mux.Handle("POST /v1/auth/refresh", a.requireCSRF(a.refresh))
		mux.Handle("POST /v1/auth/logout", a.requireCSRF(a.logout))
		// session runs behind requireAuth because the caller already holds a
		// valid access cookie; a GET needs no CSRF token, which csrfAllowed
		// exempts for safe methods.
		mux.Handle("GET /v1/auth/session", a.requireAuth(a.session))
	}
	if deps.DevLoginEnabled {
		mux.Handle("POST /v1/auth/dev-login", http.HandlerFunc(a.devLogin))
	}

	// organization
	mux.Handle("GET /v1/organizations/me", a.requireAuth(a.getOrganization))
	mux.Handle("GET /v1/organization", a.requireAuth(a.getOrganization))
	mux.Handle("PATCH /v1/organization", a.requireAuth(a.updateOrganization))

	// members
	mux.Handle("GET /v1/members", a.requireAuth(a.listMembers))
	mux.Handle("PATCH /v1/members/{userID}/role", a.requireAuth(a.setMemberRole))
	mux.Handle("DELETE /v1/members/{userID}", a.requireAuth(a.removeMember))

	// invitations
	mux.Handle("POST /v1/invitations", a.requireAuth(a.invite))
	mux.Handle("POST /v1/invitations/accept", a.requireAuth(a.acceptInvitation))
	mux.Handle("GET /v1/invitations", a.requireAuth(a.listInvitations))
	mux.Handle("DELETE /v1/invitations/{id}", a.requireAuth(a.revokeInvitation))

	// projects
	mux.Handle("POST /v1/projects", a.requireAuth(a.createProject))
	mux.Handle("GET /v1/projects", a.requireAuth(a.listProjects))
	mux.Handle("GET /v1/projects/{id}", a.requireAuth(a.getProject))
	mux.Handle("PATCH /v1/projects/{id}", a.requireAuth(a.updateProject))
	mux.Handle("PATCH /v1/projects/{id}/status", a.requireAuth(a.setProjectStatus))
	mux.Handle("POST /v1/projects/{id}/members", a.requireAuth(a.addProjectMember))
	mux.Handle("GET /v1/projects/{id}/members", a.requireAuth(a.listProjectMembers))
	mux.Handle("DELETE /v1/projects/{id}/members/{userID}", a.requireAuth(a.removeProjectMember))

	// teams
	mux.Handle("POST /v1/teams", a.requireAuth(a.createTeam))
	mux.Handle("GET /v1/teams", a.requireAuth(a.listTeams))
	mux.Handle("GET /v1/teams/{id}", a.requireAuth(a.getTeam))
	mux.Handle("PATCH /v1/teams/{id}", a.requireAuth(a.updateTeam))
	mux.Handle("POST /v1/teams/{id}/members", a.requireAuth(a.addTeamMember))
	mux.Handle("GET /v1/teams/{id}/members", a.requireAuth(a.listTeamMembers))
	mux.Handle("DELETE /v1/teams/{id}/members/{userID}", a.requireAuth(a.removeTeamMember))

	// service accounts
	mux.Handle("POST /v1/service-accounts", a.requireAuth(a.createServiceAccount))
	mux.Handle("GET /v1/service-accounts", a.requireAuth(a.listServiceAccounts))
	mux.Handle("DELETE /v1/service-accounts/{id}", a.requireAuth(a.revokeServiceAccount))

	// credentials
	mux.Handle("POST /v1/credentials", a.requireAuth(a.createCredential))
	mux.Handle("GET /v1/credentials", a.requireAuth(a.listCredentials))
	mux.Handle("DELETE /v1/credentials/{id}", a.requireAuth(a.revokeCredential))

	// gateway credentials
	registerGatewayCredentialRoutes(mux, deps)

	// policies
	if deps.Policy != nil {
		registerPolicyRoutes(mux, deps)
	}

	// model access
	if deps.Policy != nil {
		registerModelAccessRoutes(mux, deps)
	}

	// search
	mux.Handle("GET /v1/search", a.requireAuth(a.search))
}

// registerGatewayCredentialRoutes wires the gateway credential surface onto mux.
func registerGatewayCredentialRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("POST /v1/gateway-credentials", a.requireAuth(a.createGatewayCredential))
	mux.Handle("GET /v1/gateway-credentials", a.requireAuth(a.listGatewayCredentials))
	mux.Handle("GET /v1/gateway-credentials/{id}", a.requireAuth(a.getGatewayCredential))
	mux.Handle("DELETE /v1/gateway-credentials/{id}", a.requireAuth(a.revokeGatewayCredential))
	mux.Handle("POST /v1/gateway-credentials/{id}/rotate", a.requireAuth(a.rotateGatewayCredential))
	mux.Handle("POST /v1/gateway-credentials/{id}/expire", a.requireAuth(a.expireGatewayCredential))
	mux.Handle("POST /v1/gateway-credentials/{id}/restore", a.requireAuth(a.restoreGatewayCredential))
}

func registerPolicyRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("POST /v1/policies", a.requireAuth(a.createPolicy))
	mux.Handle("GET /v1/policies", a.requireAuth(a.listPolicies))
	mux.Handle("GET /v1/policies/{id}", a.requireAuth(a.getPolicy))
	mux.Handle("PATCH /v1/policies/{id}", a.requireAuth(a.updatePolicy))
	mux.Handle("DELETE /v1/policies/{id}", a.requireAuth(a.deletePolicy))
	mux.Handle("POST /v1/policies/{id}/rules", a.requireAuth(a.createPolicyRule))
	mux.Handle("GET /v1/policies/{id}/rules", a.requireAuth(a.listPolicyRules))
	mux.Handle("PATCH /v1/policies/{id}/rules/{ruleId}", a.requireAuth(a.updatePolicyRule))
	mux.Handle("DELETE /v1/policies/{id}/rules/{ruleId}", a.requireAuth(a.deletePolicyRule))
	mux.Handle("POST /v1/policies/preview", a.requireAuth(a.previewPolicy))
	mux.Handle("POST /v1/policies/{id}/validate", a.requireAuth(a.validatePolicy))
}

// registerProviderRoutes wires the M4 provider surface onto mux. Every
// provider route runs behind requireAuth and is registered only when both the
// identity and provider services are wired in.
func registerProviderRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)

	// provider accounts
	mux.Handle("POST /v1/provider-accounts", a.requireAuth(a.createProviderAccount))
	mux.Handle("GET /v1/provider-accounts", a.requireAuth(a.listProviderAccounts))
	mux.Handle("GET /v1/provider-accounts/{id}", a.requireAuth(a.getProviderAccount))

	// provider credentials, created under their provider account
	mux.Handle("POST /v1/provider-accounts/{id}/credentials", a.requireAuth(a.createProviderCredential))
	mux.Handle("GET /v1/provider-credentials", a.requireAuth(a.listProviderCredentials))
	mux.Handle("GET /v1/provider-credentials/{id}", a.requireAuth(a.getProviderCredential))
	mux.Handle("DELETE /v1/provider-credentials/{id}", a.requireAuth(a.revokeProviderCredential))
	mux.Handle("POST /v1/provider-credentials/{id}/rotate", a.requireAuth(a.rotateProviderCredential))

	// provider model catalog
	mux.Handle("GET /v1/models", a.requireAuth(a.listModels))
	mux.Handle("POST /v1/models/sync", a.requireAuth(a.syncProviderModels))

	// provider connectivity. The probe endpoints take a secret in the body for
	// the upstream call only; it is never stored, returned, or logged.
	mux.Handle("POST /v1/provider-credentials/validate", a.requireAuth(a.validateProviderCredential))
	mux.Handle("POST /v1/provider-health/check", a.requireAuth(a.checkProviderHealth))
	mux.Handle("GET /v1/provider-health", a.requireAuth(a.listProviderHealth))
}

func registerModelAccessRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/model-access", a.requireAuth(a.listModelAccess))
	mux.Handle("PUT /v1/model-access", a.requireAuth(a.updateModelAccess))
	mux.Handle("POST /v1/model-access/preview", a.requireAuth(a.previewModelAccess))
}

func registerBudgetRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/budgets", a.requireAuth(a.listBudgets))
	mux.Handle("POST /v1/budgets", a.requireAuth(a.createBudget))
	mux.Handle("GET /v1/budgets/{id}", a.requireAuth(a.getBudget))
	mux.Handle("PATCH /v1/budgets/{id}", a.requireAuth(a.updateBudget))
	mux.Handle("DELETE /v1/budgets/{id}", a.requireAuth(a.deleteBudget))
}
