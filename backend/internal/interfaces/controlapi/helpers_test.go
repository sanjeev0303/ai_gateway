package controlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/budget"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
)

// discardLog silences application logs during tests; TestMain also silences
// the package-level default so writeServiceError's default branch stays quiet.
var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.Run()
}

// Fixture identifiers and times, mirroring the identity package conventions.
var (
	testUserID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testOrgID    = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testAcctID   = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	testTeamID   = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	testProjID   = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	testInviteID = uuid.MustParse("00000000-0000-0000-0000-000000000006")
	testOtherID  = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	testProvAcct = uuid.MustParse("00000000-0000-0000-0000-000000000008")
	testProvCred = uuid.MustParse("00000000-0000-0000-0000-000000000009")
	testModelID  = uuid.MustParse("00000000-0000-0000-0000-00000000000a")

	testNow       = time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	testInviteTTL = 24 * time.Hour
)

var (
	testUser = domain.User{
		ID: testUserID, Email: "alice@example.com", DisplayName: "Alice",
		Status: domain.UserStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
	}
	testOtherUser = domain.User{
		ID: testOtherID, Email: "bob@example.com", DisplayName: "Bob",
		Status: domain.UserStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
	}
	testOrg = domain.Organization{
		ID: testOrgID, Name: "Acme", Slug: "acme",
		Status: domain.OrgStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
	}
	// testInvitation's email matches testUser so AcceptInvitation's email
	// guard passes; vary it in specific tests.
	testInvitation = domain.Invitation{
		ID: testInviteID, OrganizationID: testOrgID, Email: "alice@example.com",
		Role: domain.RoleMember, TokenHash: "hash:raw-tok", Status: domain.InvitationStatusPending,
		ExpiresAt: testNow.Add(testInviteTTL), CreatedAt: testNow, UpdatedAt: testNow,
	}
	testProject = domain.Project{
		ID: testProjID, OrganizationID: testOrgID, Name: "gateway", Slug: "gateway",
		Description: "the gateway", Status: domain.ProjectStatusActive,
		CreatedAt: testNow, UpdatedAt: testNow,
	}
	testTeam = domain.Team{
		ID: testTeamID, OrganizationID: testOrgID, Name: "core", Description: "core team",
		CreatedAt: testNow, UpdatedAt: testNow,
	}
	testServiceAccount = domain.ServiceAccount{
		ID: testAcctID, OrganizationID: testOrgID, Name: "ci-bot", Description: "CI",
		Status: domain.ServiceAccountStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
	}
	testCredential = domain.APICredential{
		ID: testOtherID, OrganizationID: testOrgID, Name: "gateway key",
		KeyPrefix: "aigw_ab12cd", KeyHash: "hash:aigw_ab12cdef0123456789abcdef0123456789",
		Scope: string(domain.APICredentialScopeGateway), Status: domain.APICredentialStatusActive,
		CreatedAt: testNow, UpdatedAt: testNow,
	}
	testProviderAccount = domain.ProviderAccount{
		ID: testProvAcct, OrganizationID: testOrgID,
		ProviderKind: domain.ProviderKindAnthropic, Name: "Anthropic Prod",
		BaseURL: "https://api.anthropic.com", Status: domain.ProviderStatusActive,
		CreatedAt: testNow, UpdatedAt: testNow,
	}
	testProviderCredential = domain.ProviderCredential{
		ID: testProvCred, OrganizationID: testOrgID, ProviderAccountID: testProvAcct,
		Name: "prod key", Kind: domain.CredentialKindAPIKey,
		SecretRef:   "provider/creds/00000000-0000-0000-0000-000000000009",
		Fingerprint: "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		Tail:        "0a08",
		Status:      domain.ProviderCredentialStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
	}
	testModel = domain.Model{
		ID: testModelID, ProviderKind: domain.ProviderKindAnthropic,
		ModelSlug: "claude-sonnet-4-5", DisplayName: "Claude Sonnet 4.5",
		IsActive: true, CreatedAt: testNow, UpdatedAt: testNow,
	}
)

func testMember(role string) domain.OrgMember {
	return domain.OrgMember{User: testUser, Role: role, MemberSince: testNow}
}

func otherMember(role string) domain.OrgMember {
	return domain.OrgMember{User: testOtherUser, Role: role, MemberSince: testNow}
}

func userActor(role string) domain.Actor {
	return domain.Actor{
		Kind: domain.ActorKindUser, UserID: testUserID, OrganizationID: testOrgID,
		Role: role, DisplayName: testUser.DisplayName,
	}
}

// controlCred returns an authenticatable control-plane credential bound to the
// fixture service account.
func controlCred() domain.APICredential {
	cred := testCredential
	cred.Scope = string(domain.APICredentialScopeControl)
	cred.Status = domain.APICredentialStatusActive
	cred.ServiceAccountID = &testAcctID
	return cred
}

// controlStubs bundles every dependency the control plane can reach, plus the
// per-dependency func fields installed per test. Stub functions run inside the
// HTTP server's goroutine, so they must never call t.Fatalf.
type controlStubs struct {
	orgs               *stubOrgs
	users              *stubUsers
	members            *stubMembers
	invitations        *stubInvitations
	projects           *stubProjects
	teams              *stubTeams
	teamMembers        *stubTeamMembers
	projMembers        *stubProjMembers
	accounts           *stubAccounts
	credentials        *stubCredentials
	gatewayCredentials *stubGatewayCredentials
	audits             *stubAudits
	issuer             *stubIssuer
	verifier           *stubVerifier
	hasher             *stubHasher
	auth               *stubAuth
	refresh            *stubRefreshTokens
	passwords          *stubPasswords
	subs               *stubSubscriptions

	budgets *stubBudgets
	policy  *stubPolicy

	providerAccounts    *stubProviderAccounts
	providerCredentials *stubProviderCredentials
	providerModels      *stubModels
	providerHealth      *stubProviderHealth
	providerSecrets     *stubSecrets
	providerClient      *stubProvider

	policyRepo     *stubPolicyRepo
	policyRuleRepo *stubPolicyRuleRepo
}

func newControlStubs() *controlStubs {
	return &controlStubs{
		orgs:               &stubOrgs{},
		users:              &stubUsers{},
		members:            &stubMembers{},
		invitations:        &stubInvitations{},
		projects:           &stubProjects{},
		teams:              &stubTeams{},
		teamMembers:        &stubTeamMembers{},
		projMembers:        &stubProjMembers{},
		accounts:           &stubAccounts{},
		credentials:        &stubCredentials{},
		gatewayCredentials: &stubGatewayCredentials{},
		audits:             &stubAudits{},
		issuer:             &stubIssuer{},
		verifier:           &stubVerifier{},
		// hasher.hash stays nil so tests can assert a code path never reached
		// TokenHasher.Hash; the auth helper below installs one on demand.
		hasher:    &stubHasher{},
		auth:      &stubAuth{},
		refresh:   &stubRefreshTokens{},
		passwords: &stubPasswords{},
		subs:      &stubSubscriptions{},

		budgets:        &stubBudgets{},
		policyRepo:     &stubPolicyRepo{},
		policyRuleRepo: &stubPolicyRuleRepo{},

		providerAccounts:    &stubProviderAccounts{},
		providerCredentials: &stubProviderCredentials{},
		providerModels:      &stubModels{},
		providerHealth:      &stubProviderHealth{},
		providerSecrets:     &stubSecrets{values: map[string]string{}},
		providerClient:      &stubProvider{},
	}
}

// deps wires the stub bundle into the identity service with a fixed clock.
func (st *controlStubs) deps() identity.ServiceDeps {
	return identity.ServiceDeps{
		Orgs:               st.orgs,
		Users:              st.users,
		Members:            st.members,
		Invitations:        st.invitations,
		Projects:           st.projects,
		Teams:              st.teams,
		TeamMembers:        st.teamMembers,
		ProjMembers:        st.projMembers,
		Accounts:           st.accounts,
		Credentials:        st.credentials,
		GatewayCredentials: st.gatewayCredentials,
		Audits:             st.audits,
		Auth:               st.auth,
		RefreshTokens:      st.refresh,
		Subs:               st.subs,
		Issuer:             st.issuer,
		Verifier:           st.verifier,
		Hasher:             st.hasher,
		Passwords:          st.passwords,
		RefreshTokenTTL:    testRefreshTTL,
		Now:                func() time.Time { return testNow },
		InviteTTL:          testInviteTTL,
	}
}

// providerDeps wires the provider stubs into the provider service with the
// same fixed clock; the Anthropic client (the only M4 provider) is registered
// by default.
func (st *controlStubs) providerDeps() provider.ServiceDeps {
	return provider.ServiceDeps{
		Accounts:    st.providerAccounts,
		Credentials: st.providerCredentials,
		Models:      st.providerModels,
		Health:      st.providerHealth,
		Audits:      st.audits,
		Secrets:     st.providerSecrets,
		Providers: map[string]providerclient.Provider{
			domain.ProviderKindAnthropic: st.providerClient,
		},
		Now: func() time.Time { return testNow },
	}
}

// budgetDeps wires the budget stubs into the budget service with a fixed clock.
func (st *controlStubs) budgetDeps() budget.ServiceDeps {
	return budget.ServiceDeps{
		Budgets:     st.budgets,
		Policies:    st.policyRepo,
		Rules:       st.policyRuleRepo,
		Audits:      st.audits,
		Invalidator: stubInvalidator{},
	}
}

// buildServer assembles the full handler stack (middleware, health registry)
// around the given deps and serves it over real HTTP.
func buildServer(t *testing.T, deps Deps) *httptest.Server {
	t.Helper()
	return buildServerReg(t, deps, health.NewRegistry())
}

// newBudgetServer returns a server with the budget routes enabled.
func newBudgetServer(t *testing.T, st *controlStubs) *httptest.Server {
	t.Helper()

	// Default mocks for budget-policy syncing to avoid panics
	if st.policyRepo.list == nil {
		st.policyRepo.list = func(context.Context, uuid.UUID) ([]domain.Policy, error) {
			return []domain.Policy{{ID: testPolicyID, Name: "Budget Enforcement"}}, nil
		}
	}
	if st.policyRuleRepo.list == nil {
		st.policyRuleRepo.list = func(context.Context, uuid.UUID, uuid.UUID) ([]domain.PolicyRule, error) {
			return nil, nil // No rules found
		}
	}
	if st.policyRuleRepo.create == nil {
		st.policyRuleRepo.create = func(_ context.Context, _ uuid.UUID, in domain.NewPolicyRule) (domain.PolicyRule, error) {
			return domain.PolicyRule{ID: testRuleID}, nil
		}
	}
	if st.policyRuleRepo.update == nil {
		st.policyRuleRepo.update = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID, in domain.UpdatePolicyRule) (domain.PolicyRule, error) {
			return domain.PolicyRule{ID: testRuleID}, nil
		}
	}
	if st.policyRuleRepo.delete == nil {
		st.policyRuleRepo.delete = func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
			return nil
		}
	}

	svc := identity.NewService(st.deps())
	bSvc := budget.NewService(st.budgetDeps())
	return buildServer(t, Deps{
		Identity:        svc,
		Budget:          bSvc,
		DevLoginEnabled: true,
	})
}

// newPolicyServer returns a server with the model-access (policy) routes enabled.
func newPolicyServer(t *testing.T, st *controlStubs) *httptest.Server {
	t.Helper()
	svc := identity.NewService(st.deps())
	pSvc := policy.NewService(st.policyRepo, st.policyRuleRepo, st.audits, stubInvalidator{}, discardLog)
	return buildServer(t, Deps{
		Identity:        svc,
		Policy:          pSvc,
		DevLoginEnabled: true,
	})
}

// buildServerReg is buildServer with an externally supplied health registry,
// for tests that must register failing checks.
func buildServerReg(t *testing.T, deps Deps, reg *health.Registry) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := NewHandler(logger, httpx.NewMiddleware(logger, httpx.DefaultOptions()), reg, deps)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// newTestServer returns a server with a default stubbed identity and provider
// service (dev login enabled) plus the stubs for per-test configuration.
func newTestServer(t *testing.T) (*httptest.Server, *controlStubs) {
	t.Helper()
	st := newControlStubs()
	svc := identity.NewService(st.deps())
	psvc := provider.NewService(st.providerDeps())
	return buildServer(t, Deps{Identity: svc, Providers: psvc, DevLoginEnabled: true}), st
}

// newSessionServer builds a server with the browser session flow enabled: a
// session cookie named "session" (optionally Secure) and, when csrfEnabled,
// the double-submit CSRF guard on cookie-authenticated state changes.
func newSessionServer(t *testing.T, csrfEnabled, secure bool) (*httptest.Server, *controlStubs) {
	t.Helper()
	st := newControlStubs()
	svc := identity.NewService(st.deps())
	psvc := provider.NewService(st.providerDeps())
	return buildServer(t, Deps{
		Identity:            svc,
		Providers:           psvc,
		DevLoginEnabled:     true,
		SessionCookieName:   "session",
		SessionCookieSecure: secure,
		CSRFEnabled:         csrfEnabled,
	}), st
}

// newAuthServer builds the browser session server with the credential
// dependencies the signup, signin, and refresh routes need: a token hasher for
// refresh digests, a password hasher, and the auth/refresh repositories. The
// session cookie is named "session" and CSRF follows the flag.
func newAuthServer(t *testing.T, csrfEnabled, secure bool) (*httptest.Server, *controlStubs) {
	t.Helper()
	st := newControlStubs()
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "digest:" + secret, nil
	}
	// A fixed access token keeps the cookie assertions readable; cases that
	// need a distinct value override this.
	st.issuer.issue = func(_ context.Context, _ domain.TokenClaims) (string, error) {
		return "access-token-1", nil
	}
	svc := identity.NewService(st.deps())
	psvc := provider.NewService(st.providerDeps())
	return buildServer(t, Deps{
		Identity:            svc,
		Providers:           psvc,
		DevLoginEnabled:     true,
		SessionCookieName:   "session",
		SessionCookieSecure: secure,
		CSRFEnabled:         csrfEnabled,
	}), st
}

// tokenClaimsFor returns fixture claims for the fixture user acting under the
// given role; shared by the bearer and session-cookie auth helpers.
func tokenClaimsFor(role string) domain.TokenClaims {
	return domain.TokenClaims{
		Subject:        testUserID.String(),
		Email:          testUser.Email,
		DisplayName:    testUser.DisplayName,
		OrganizationID: testOrgID,
		Role:           role,
		Kind:           domain.TokenKindUser,
		IssuedAt:       testNow,
		ExpiresAt:      testNow.Add(time.Hour),
	}
}

// bearerHeader serves an authenticated user actor for the given role via a
// token the verifier stub resolves. The token value is fixed; claims come from
// the fixtures.
func bearerHeader(st *controlStubs, role string) map[string]string {
	st.verifier.verify = func(_ context.Context, _ string) (domain.TokenClaims, error) {
		return tokenClaimsFor(role), nil
	}
	return map[string]string{"Authorization": "Bearer any-token"}
}

// sessionCookie authenticates the fixture user via a session cookie the
// verifier stub resolves to claims for the given role.
func sessionCookie(st *controlStubs, role string) map[string]string {
	st.verifier.verify = func(_ context.Context, _ string) (domain.TokenClaims, error) {
		return tokenClaimsFor(role), nil
	}
	return map[string]string{"Cookie": "session=any-token"}
}

// sessionCSRFHeaders combines the session cookie with a matching CSRF cookie +
// X-CSRF-Token header pair for cookie-authenticated state changes.
func sessionCSRFHeaders(st *controlStubs, role, csrfToken string) map[string]string {
	hdr := sessionCookie(st, role)
	hdr["Cookie"] += "; session_csrf=" + csrfToken
	hdr["X-CSRF-Token"] = csrfToken
	return hdr
}

// apiKeyToken returns a raw API key and installs the hasher/resolver stubs so
// the key authenticates as cred's service account. Callers override
// st.accounts.get to vary the account outcome.
func apiKeyToken(st *controlStubs, cred domain.APICredential) string {
	raw := "aigw_ab12cdef0123456789abcdef0123456789"
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "hash:" + secret, nil
	}
	st.credentials.resolveByHash = func(_ context.Context, hash string) (domain.APICredential, error) {
		return cred, nil
	}
	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		return testServiceAccount, nil
	}
	return raw
}

// doJSON sends an HTTP request against the test server. A nil body sends no
// content; headers are merged over the defaults.
func doJSON(t *testing.T, srv *httptest.Server, method, path string, body any, hdr map[string]string) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		requireNoErr(t, err)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, srv.URL+path, rdr)
	requireNoErr(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := srv.Client().Do(req)
	requireNoErr(t, err)
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	requireNoErr(t, json.NewDecoder(resp.Body).Decode(dst))
}

// errorEnvelope matches the wire error shape ({error:{code,message}}).
type errorEnvelope struct {
	Err struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeError(t *testing.T, resp *http.Response) (int, string) {
	t.Helper()
	defer resp.Body.Close()
	var env errorEnvelope
	requireNoErr(t, json.NewDecoder(resp.Body).Decode(&env))
	return resp.StatusCode, env.Err.Code
}

// testPassword and testPasswordDigest stand in for a real credential pair in
// the signup and signin tests.
const (
	testPassword       = "correct-horse-battery-staple"
	testPasswordDigest = "$pbkdf2-sha256$600000$c3Rlcw==$a2V5"
	testRefreshTTL     = 30 * 24 * time.Hour
)

// readBody returns the raw response bytes and closes the body, so a test can
// inspect the exact wire shape rather than only a decoded struct.
func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	requireNoErr(t, err)
	return b
}

// cookiesByName indexes the Set-Cookie headers for assertions.
func cookiesByName(resp *http.Response) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, c := range resp.Cookies() {
		out[c.Name] = c
	}
	return out
}

// assertNoCredentialInBody fails if the response body carries a credential.
// Returning the access or refresh token as JSON would put it in reach of any
// script on the page, which defeats the HttpOnly cookie.
func assertNoCredentialInBody(t *testing.T, raw []byte) {
	t.Helper()
	var fields map[string]json.RawMessage
	requireNoErr(t, json.Unmarshal(raw, &fields))
	for _, key := range []string{"token", "access_token", "refresh_token", "password"} {
		if _, leaked := fields[key]; leaked {
			t.Fatalf("response body leaked %q", key)
		}
	}
}
