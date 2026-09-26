package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- bearer / API-key authentication -------------------------------------

func TestAuthenticateMissingCredentials(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, nil)

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateRejectsMalformedAuthorizationHeader(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"Authorization": "Basic abc"})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateRejectsInvalidBearerToken(t *testing.T) {
	srv, st := newTestServer(t)
	st.verifier.verify = func(_ context.Context, _ string) (domain.TokenClaims, error) {
		return domain.TokenClaims{}, identity.ErrUnauthorized
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"Authorization": "Bearer bogus"})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateAcceptsBearerToken(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.list = func(_ context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
		return []domain.OrgMember{}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, bearerHeader(st, domain.RoleOwner))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
}

// TestAuthenticateAPIKeyProofOfAuth proves an API key authenticated as a
// service actor: the service role has no members.read, so a valid key reaches
// the authorization layer (403) rather than being rejected as unauthenticated.
func TestAuthenticateAPIKeyProofOfAuth(t *testing.T) {
	srv, st := newTestServer(t)
	raw := apiKeyToken(st, controlCred())

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestAuthenticateAPIKeyRejectsUnknownKey(t *testing.T) {
	srv, st := newTestServer(t)
	raw := apiKeyToken(st, controlCred())
	st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) {
		return domain.APICredential{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateAPIKeyRejectsRevokedCredential(t *testing.T) {
	srv, st := newTestServer(t)
	cred := controlCred()
	cred.Status = domain.APICredentialStatusRevoked
	raw := apiKeyToken(st, cred)

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateAPIKeyRejectsGatewayScope(t *testing.T) {
	srv, st := newTestServer(t)
	raw := apiKeyToken(st, testCredential) // gateway-scoped fixture

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateAPIKeyRejectsMissingServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)
	cred := controlCred()
	cred.ServiceAccountID = nil
	raw := apiKeyToken(st, cred)

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateAPIKeyRejectsUnknownServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)
	raw := apiKeyToken(st, controlCred())
	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		return domain.ServiceAccount{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

func TestAuthenticateAPIKeyRejectsRevokedServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)
	raw := apiKeyToken(st, controlCred())
	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		sa := testServiceAccount
		sa.Status = domain.ServiceAccountStatusRevoked
		return sa, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil,
		map[string]string{"X-API-Key": raw})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
}

// ---- developer login -----------------------------------------------------

func TestDevLoginReturnsSessionToken(t *testing.T) {
	srv, st := newTestServer(t)
	st.users.getByEmail = func(_ context.Context, email string) (domain.User, error) {
		return testUser, nil
	}
	st.users.listOrgs = func(_ context.Context, id uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}
	var gotClaims domain.TokenClaims
	st.issuer.issue = func(_ context.Context, claims domain.TokenClaims) (string, error) {
		gotClaims = claims
		return "token-1", nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "alice@example.com"}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var out sessionDTO
	decodeBody(t, resp, &out)

	requireEqual(t, out.Token, "token-1")
	requireEqual(t, out.Claims.Subject, testUserID.String())
	requireEqual(t, out.Claims.Email, testUser.Email)
	requireEqual(t, out.Claims.DisplayName, testUser.DisplayName)
	requireEqual(t, out.Claims.OrganizationID, testOrgID.String())
	requireEqual(t, out.Claims.Role, domain.RoleAdmin)
	requireEqual(t, out.Claims.Kind, string(domain.TokenKindUser))

	requireEqual(t, gotClaims.Subject, testUserID.String())
	requireEqual(t, gotClaims.Email, testUser.Email)
	requireEqual(t, gotClaims.DisplayName, testUser.DisplayName)
	requireEqual(t, gotClaims.OrganizationID, testOrgID)
	requireEqual(t, gotClaims.Role, domain.RoleAdmin)
	requireEqual(t, gotClaims.Kind, domain.TokenKindUser)
	if !gotClaims.IssuedAt.IsZero() || !gotClaims.ExpiresAt.IsZero() {
		t.Fatal("dev-login claims must not carry fixed issue/expiry timestamps")
	}
	// Zero claims render as the zero RFC3339 instant on the wire.
	requireEqual(t, out.Claims.IssuedAt, "0001-01-01T00:00:00Z")
	requireEqual(t, out.Claims.ExpiresAt, "0001-01-01T00:00:00Z")
}

func TestDevLoginRejectsBlankEmail(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "   "}, nil)

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestDevLoginProvisionsUserAndOrganization(t *testing.T) {
	srv, st := newTestServer(t)
	st.users.getByEmail = func(_ context.Context, email string) (domain.User, error) {
		return domain.User{}, repositories.ErrNotFound
	}
	var createdUser domain.NewUser
	st.users.create = func(_ context.Context, in domain.NewUser) (domain.User, error) {
		createdUser = in
		return testUser, nil
	}
	st.users.listOrgs = func(_ context.Context, id uuid.UUID) ([]domain.UserOrganization, error) {
		return nil, nil
	}
	var createdOrg domain.NewOrganization
	st.orgs.create = func(_ context.Context, in domain.NewOrganization) (domain.Organization, error) {
		createdOrg = in
		return testOrg, nil
	}
	var addedRole string
	st.members.add = func(_ context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
		addedRole = role
		return domain.OrganizationMember{OrganizationID: orgID, UserID: userID, Role: role, CreatedAt: testNow}, nil
	}
	var gotClaims domain.TokenClaims
	st.issuer.issue = func(_ context.Context, claims domain.TokenClaims) (string, error) {
		gotClaims = claims
		return "token-1", nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "alice@example.com", "display_name": "Alice"}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var out sessionDTO
	decodeBody(t, resp, &out)

	requireEqual(t, createdUser.Email, "alice@example.com")
	requireEqual(t, createdUser.DisplayName, "Alice")
	requireEqual(t, createdOrg.Name, "Alice")
	if createdOrg.Slug == "" {
		t.Fatal("provisioned organization must have a slug")
	}
	requireEqual(t, addedRole, domain.RoleOwner)
	requireEqual(t, gotClaims.Role, domain.RoleOwner)
	requireEqual(t, out.Claims.Role, domain.RoleOwner)
	requireEqual(t, out.Claims.OrganizationID, testOrgID.String())
}

// ---- browser sessions: session cookie + CSRF -------------------------------

func TestDevLoginSetsSessionCookies(t *testing.T) {
	srv, st := newSessionServer(t, true, false)
	st.users.getByEmail = func(_ context.Context, email string) (domain.User, error) {
		return testUser, nil
	}
	st.users.listOrgs = func(_ context.Context, id uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}
	st.issuer.issue = func(_ context.Context, _ domain.TokenClaims) (string, error) {
		return "token-1", nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "alice@example.com", "display_name": "Alice"}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var out sessionDTO
	decodeBody(t, resp, &out)

	if out.CSRFToken == "" {
		t.Fatal("session login must return a CSRF token")
	}
	cookies := map[string]*http.Cookie{}
	for _, c := range resp.Cookies() {
		cookies[c.Name] = c
	}
	session, ok := cookies["session"]
	if !ok {
		t.Fatalf("expected session cookie, got %v", resp.Cookies())
	}
	requireEqual(t, session.Value, "token-1")
	requireEqual(t, session.HttpOnly, true)
	if session.Path != "/" {
		t.Fatalf("session cookie path = %q, want /", session.Path)
	}
	csrfCookie, ok := cookies["session_csrf"]
	if !ok {
		t.Fatalf("expected session_csrf cookie, got %v", resp.Cookies())
	}
	requireEqual(t, csrfCookie.Value, out.CSRFToken)
	if csrfCookie.HttpOnly {
		t.Fatal("CSRF cookie must be readable by client JS")
	}
}

func TestDevLoginWithoutSessionsSetsNoCookies(t *testing.T) {
	srv, st := newTestServer(t)
	st.users.getByEmail = func(_ context.Context, email string) (domain.User, error) {
		return testUser, nil
	}
	st.users.listOrgs = func(_ context.Context, id uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}
	st.issuer.issue = func(_ context.Context, _ domain.TokenClaims) (string, error) {
		return "token-1", nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "alice@example.com"}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var out sessionDTO
	decodeBody(t, resp, &out)
	requireEqual(t, out.CSRFToken, "")
	if n := len(resp.Cookies()); n != 0 {
		t.Fatalf("cookie-less server must not set cookies, got %d", n)
	}
}

func TestSessionCookieAuthenticatesGet(t *testing.T) {
	srv, st := newSessionServer(t, true, false)
	st.members.list = func(_ context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
		return []domain.OrgMember{}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, sessionCookie(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
}

func TestSessionCookieStateChangeRejectsMissingCSRF(t *testing.T) {
	srv, st := newSessionServer(t, true, false)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams",
		map[string]any{"name": "core"}, sessionCookie(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "csrf_token_invalid")
}

func TestSessionCookieStateChangeRejectsMismatchedCSRF(t *testing.T) {
	srv, st := newSessionServer(t, true, false)
	hdr := sessionCSRFHeaders(st, domain.RoleAdmin, "expected")
	hdr["X-CSRF-Token"] = "attacker-token"

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams",
		map[string]any{"name": "core"}, hdr)

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "csrf_token_invalid")
}

func TestSessionCookieStateChangeAcceptsMatchingCSRF(t *testing.T) {
	srv, st := newSessionServer(t, true, false)
	st.teams.create = func(_ context.Context, orgID uuid.UUID, in domain.NewTeam) (domain.Team, error) {
		return testTeam, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams",
		map[string]any{"name": "core"}, sessionCSRFHeaders(st, domain.RoleAdmin, "expected"))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	resp.Body.Close()
	requireAudit(t, st, "team.create")
}

func TestCSRFRequiredOnlyForCookieSessions(t *testing.T) {
	srv, st := newSessionServer(t, true, false)
	st.teams.create = func(_ context.Context, orgID uuid.UUID, in domain.NewTeam) (domain.Team, error) {
		return testTeam, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams",
		map[string]any{"name": "core"}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	resp.Body.Close()
}

func TestSessionCookieStateChangeWithoutCSRFEnabled(t *testing.T) {
	srv, st := newSessionServer(t, false, false)
	st.teams.create = func(_ context.Context, orgID uuid.UUID, in domain.NewTeam) (domain.Team, error) {
		return testTeam, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams",
		map[string]any{"name": "core"}, sessionCookie(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	resp.Body.Close()
}

func TestLogoutClearsSessionCookies(t *testing.T) {
	srv, _ := newSessionServer(t, true, false)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/logout", nil, nil)

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	cookies := map[string]*http.Cookie{}
	for _, c := range resp.Cookies() {
		cookies[c.Name] = c
	}
	for _, name := range []string{"session", "session_csrf"} {
		c, ok := cookies[name]
		if !ok {
			t.Fatalf("logout must clear %q cookie, got %v", name, resp.Cookies())
		}
		if c.MaxAge > 0 {
			t.Fatalf("cleared %q cookie has MaxAge %d, want expired", name, c.MaxAge)
		}
	}
}

func TestSecureSessionCookiesWhenConfigured(t *testing.T) {
	srv, st := newSessionServer(t, true, true)
	st.users.getByEmail = func(_ context.Context, email string) (domain.User, error) {
		return testUser, nil
	}
	st.users.listOrgs = func(_ context.Context, id uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}
	st.issuer.issue = func(_ context.Context, _ domain.TokenClaims) (string, error) {
		return "token-1", nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "alice@example.com"}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
	for _, c := range resp.Cookies() {
		if !c.Secure {
			t.Fatalf("cookie %q must be Secure when configured, got %+v", c.Name, c)
		}
	}
}
