package controlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// credentialFor returns the stored-credential fixture the signin path reads:
// an active user plus a password hash, so Verify runs against a real digest
// instead of the decoy.
func credentialFor(user domain.User) domain.UserCredential {
	hash := testPasswordDigest
	return domain.UserCredential{User: user, PasswordHash: &hash}
}

// signupRequest is the body the browser form posts.
func signupRequest() map[string]any {
	return map[string]any{
		"email":             "alice@example.com",
		"display_name":      "Alice",
		"organization_name": "Acme",
		"password":          testPassword,
	}
}

func TestSignUpCreatesAccountAndSetsCookies(t *testing.T) {
	srv, st := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signup", signupRequest(), nil)

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	raw := readBody(t, resp)
	assertNoCredentialInBody(t, raw)

	cookies := cookiesByName(resp)
	session := cookies["session"]
	if session == nil {
		t.Fatalf("signup must set the session cookie, got %v", cookies)
	}
	if !session.HttpOnly {
		t.Fatal("session cookie must be HttpOnly so scripts cannot read the access token")
	}
	if !session.Secure {
		t.Fatal("session cookie must be Secure when the deployment sets it")
	}
	refresh := cookies["session_refresh"]
	if refresh == nil {
		t.Fatalf("signup must set the refresh cookie, got %v", cookies)
	}
	if !refresh.HttpOnly {
		t.Fatal("refresh cookie must be HttpOnly")
	}
	// The refresh cookie is the durable credential, so it must not ride along
	// on a top-level cross-site navigation the way the access cookie does.
	if refresh.SameSite != http.SameSiteStrictMode {
		t.Fatalf("refresh cookie SameSite is %v, want Strict", refresh.SameSite)
	}
	csrf := cookies["session_csrf"]
	if csrf == nil {
		t.Fatalf("signup must set the CSRF cookie, got %v", cookies)
	}
	if csrf.HttpOnly {
		t.Fatal("CSRF cookie must be readable by the browser so JS can echo it in a header")
	}

	var out sessionDTO
	requireNoErr(t, json.Unmarshal(raw, &out))
	requireEqual(t, out.Claims.Subject, testUserID.String())
	if out.CSRFToken == "" {
		t.Fatal("signup must return a CSRF token for the browser to echo")
	}
	// The digest must travel inside the atomic registration write rather than a
	// separate insert, so a failure cannot leave a user with no session.
	requireEqual(t, st.auth.capturedRegistration.RefreshTokenHash, "digest:"+refresh.Value)
	if st.refresh.createdHash != "" {
		t.Fatal("signup must persist the refresh digest in the registration transaction, not separately")
	}
	// The organization is named from organization_name, not the person's name.
	requireEqual(t, st.auth.capturedRegistration.OrganizationName, "Acme")
}

func TestSignUpRejectsWeakPassword(t *testing.T) {
	srv, st := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signup", map[string]any{
		"email":        "alice@example.com",
		"display_name": "Alice",
		"password":     "short",
	}, nil)

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
	if st.refresh.createdHash != "" {
		t.Fatal("a rejected password must not leave a refresh token behind")
	}
}

func TestSignUpRejectsDuplicateEmail(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.auth.registerFn = func(_ context.Context, _ domain.Registration) (domain.User, domain.Organization, error) {
		return domain.User{}, domain.Organization{}, repositories.ErrEmailTaken
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signup", signupRequest(), nil)

	code, errCode := decodeError(t, resp)
	requireEqual(t, code, http.StatusConflict)
	requireEqual(t, errCode, "email_taken")
}

func TestSignUpRequiresCookieDeployment(t *testing.T) {
	// Without cookies there is no way to receive credentials, so the route is
	// not registered at all rather than returning a session nobody can use.
	srv, _ := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signup", signupRequest(), nil)
	resp.Body.Close()
	requireEqual(t, resp.StatusCode, http.StatusNotFound)
}

func TestSignInIssuesSessionAndRefreshCookies(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.auth.credentialByEmail = func(_ context.Context, _ string) (domain.UserCredential, error) {
		return credentialFor(testUser), nil
	}
	st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signin", map[string]any{
		"email": "alice@example.com", "password": testPassword,
	}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := readBody(t, resp)
	assertNoCredentialInBody(t, raw)

	cookies := cookiesByName(resp)
	if cookies["session"] == nil || cookies["session_refresh"] == nil {
		t.Fatalf("signin must set both cookies, got %v", cookies)
	}
	if st.refresh.createdHash != "digest:"+cookies["session_refresh"].Value {
		t.Fatalf("signin stored digest %q, want the digest of the cookie value",
			st.refresh.createdHash)
	}
}

func TestSignInRejectsWrongPasswordUniformly(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.auth.credentialByEmail = func(_ context.Context, _ string) (domain.UserCredential, error) {
		return credentialFor(testUser), nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signin", map[string]any{
		"email": "alice@example.com", "password": "wrong-password-here",
	}, nil)

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
	if cookiesByName(resp)["session"] != nil {
		t.Fatal("a failed signin must not set a session cookie")
	}
}

func TestSignInUnknownEmailLooksIdenticalToWrongPassword(t *testing.T) {
	// The unknown-email path must be indistinguishable on the wire, or the
	// endpoint becomes an account-existence oracle.
	srv, st := newAuthServer(t, true, true)
	st.auth.credentialByEmail = func(_ context.Context, _ string) (domain.UserCredential, error) {
		return domain.UserCredential{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signin", map[string]any{
		"email": "nobody@example.com", "password": testPassword,
	}, nil)

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
	// The decoy keeps the KDF cost equal on both paths.
	if st.passwords.decoyHits == 0 {
		t.Fatal("an unknown email must still be verified against the decoy digest")
	}
}

func TestSignInRateLimitsByIPAndEmail(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.auth.credentialByEmail = func(_ context.Context, _ string) (domain.UserCredential, error) {
		return credentialFor(testUser), nil
	}
	body := map[string]any{"email": "alice@example.com", "password": "wrong-password-here"}

	for i := range signInAttempts {
		resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signin", body, nil)
		code, _ := decodeError(t, resp)
		requireEqual(t, code, http.StatusUnauthorized)
		_ = i
	}
	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/signin", body, nil)
	defer resp.Body.Close()
	requireEqual(t, resp.StatusCode, http.StatusTooManyRequests)
	if resp.Header.Get("Retry-After") == "" {
		t.Fatal("a throttled signin must tell the client when to retry")
	}

	// A different email from the same client gets its own budget, so one noisy
	// account cannot exhaust the limit for every other user behind that IP.
	other := doJSON(t, srv, http.MethodPost, "/v1/auth/signin",
		map[string]any{"email": "bob@example.com", "password": "wrong-password-here"}, nil)
	code, _ := decodeError(t, other)
	requireEqual(t, code, http.StatusUnauthorized)
}

func TestRefreshRotatesCookies(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.refresh.getByHashFn = func(_ context.Context, _ string) (domain.RefreshToken, error) {
		return domain.RefreshToken{UserID: testUserID, ExpiresAt: testNow.Add(testRefreshTTL)}, nil
	}
	st.users.getByID = func(_ context.Context, _ uuid.UUID) (domain.User, error) { return testUser, nil }
	st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/refresh", nil, map[string]string{
		"Cookie":       "session_refresh=refresh-1; session_csrf=csrf-1",
		"X-CSRF-Token": "csrf-1",
	})

	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := readBody(t, resp)
	assertNoCredentialInBody(t, raw)

	// The presented token is retired and a different one issued, so a leaked
	// token is usable at most once.
	requireEqual(t, st.refresh.revokedHash, "digest:refresh-1")
	cookies := cookiesByName(resp)
	if cookies["session_refresh"] == nil || cookies["session_refresh"].Value == "refresh-1" {
		t.Fatal("refresh must issue a new refresh token rather than echo the old one")
	}
}

func TestRefreshWithoutCSRFIsRejected(t *testing.T) {
	srv, st := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/refresh", nil, map[string]string{
		"Cookie": "session_refresh=refresh-1",
	})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "csrf_token_invalid")
	if st.refresh.revokedHash != "" {
		t.Fatal("a cross-site refresh must not retire the victim's token")
	}
}

func TestRefreshWithoutCookieIsUnauthorized(t *testing.T) {
	srv, _ := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/refresh", nil, map[string]string{
		"Cookie":       "session_csrf=csrf-1",
		"X-CSRF-Token": "csrf-1",
	})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "missing_refresh_token")
}

func TestRefreshRejectsUnknownTokenAndClearsCookies(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.refresh.getByHashFn = func(_ context.Context, _ string) (domain.RefreshToken, error) {
		return domain.RefreshToken{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/refresh", nil, map[string]string{
		"Cookie":       "session_refresh=bogus; session_csrf=csrf-1",
		"X-CSRF-Token": "csrf-1",
	})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	requireEqual(t, msg, "unauthenticated")
	// A dead refresh token leaves the browser holding a session it can never
	// renew, so the cookies are cleared to force a clean signin.
	cookies := cookiesByName(resp)
	if cookies["session"] == nil || cookies["session"].MaxAge >= 0 {
		t.Fatal("a rejected refresh must clear the session cookies")
	}
}

func TestRefreshRejectsExpiredToken(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.refresh.getByHashFn = func(_ context.Context, _ string) (domain.RefreshToken, error) {
		return domain.RefreshToken{UserID: testUserID, ExpiresAt: testNow.Add(-time.Hour)}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/refresh", nil, map[string]string{
		"Cookie":       "session_refresh=refresh-1; session_csrf=csrf-1",
		"X-CSRF-Token": "csrf-1",
	})

	code, _ := decodeError(t, resp)
	requireEqual(t, code, http.StatusUnauthorized)
	if st.refresh.revokeCalls != 0 {
		t.Fatal("an expired token is already dead; revoking it is not the reuse response")
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	srv, st := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/logout", nil, map[string]string{
		"Cookie":       "session_refresh=refresh-1; session_csrf=csrf-1",
		"X-CSRF-Token": "csrf-1",
	})

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
	// Clearing the cookie alone would leave a stolen copy of the token usable
	// until it expired, so logout has to retire it server-side.
	requireEqual(t, st.refresh.revokedHash, "digest:refresh-1")
}

func TestDevLoginIssuesNoRefreshCookie(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	st.users.getByEmail = func(_ context.Context, _ string) (domain.User, error) { return testUser, nil }
	st.users.listOrgs = func(_ context.Context, _ uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{Organization: testOrg, Role: domain.RoleAdmin}}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login",
		map[string]any{"email": "alice@example.com"}, nil)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
	// Dev-login is passwordless, so it must not hand out the 30-day durable
	// credential a real signin earns.
	if cookiesByName(resp)["session_refresh"] != nil {
		t.Fatal("dev-login must not issue a refresh cookie")
	}
	if st.refresh.createdHash != "" {
		t.Fatal("dev-login must not persist a refresh token")
	}
}

func TestSessionReturnsClaimsAndCSRFToken(t *testing.T) {
	srv, st := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodGet, "/v1/auth/session", nil,
		sessionCookie(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := readBody(t, resp)
	assertNoCredentialInBody(t, raw)

	var got sessionDTO
	requireNoErr(t, json.Unmarshal(raw, &got))
	if got.Claims.Email != testUser.Email {
		t.Fatalf("session email = %q, want %q", got.Claims.Email, testUser.Email)
	}
	if got.Claims.OrganizationID != testOrgID.String() {
		t.Fatalf("session org = %q, want %q", got.Claims.OrganizationID, testOrgID)
	}
	if got.Claims.Role != domain.RoleAdmin {
		t.Fatalf("session role = %q, want %q", got.Claims.Role, domain.RoleAdmin)
	}
	// The browser cannot read the HttpOnly cookies, so the session read is the
	// only place it can obtain the token it must echo to change state.
	if got.CSRFToken == "" {
		t.Fatal("session must return a CSRF token")
	}
	csrf := cookiesByName(resp)["session_csrf"]
	if csrf == nil || csrf.Value != got.CSRFToken {
		t.Fatalf("session CSRF cookie %v does not match returned token %q", csrf, got.CSRFToken)
	}
}

func TestSessionRejectsAnonymousCaller(t *testing.T) {
	srv, _ := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodGet, "/v1/auth/session", nil, nil)

	requireEqual(t, resp.StatusCode, http.StatusUnauthorized)
	resp.Body.Close()
}

func TestSessionDoesNotReissueCredentials(t *testing.T) {
	srv, st := newAuthServer(t, true, true)

	resp := doJSON(t, srv, http.MethodGet, "/v1/auth/session", nil,
		sessionCookie(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
	// A dashboard polls this endpoint on every render. Minting a fresh session
	// here would silently extend a session's life with no user action.
	cookies := cookiesByName(resp)
	if cookies["session"] != nil {
		t.Fatal("session read must not reissue the access cookie")
	}
	if cookies["session_refresh"] != nil {
		t.Fatal("session read must not reissue the refresh cookie")
	}
}

func TestSessionReusesExistingCSRFCookie(t *testing.T) {
	srv, st := newAuthServer(t, true, true)
	hdr := sessionCookie(st, domain.RoleAdmin)
	hdr["Cookie"] += "; session_csrf=csrf-existing"

	resp := doJSON(t, srv, http.MethodGet, "/v1/auth/session", nil, hdr)

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var got sessionDTO
	decodeBody(t, resp, &got)
	// Rotating on every read would invalidate a token the page is still using.
	if got.CSRFToken != "csrf-existing" {
		t.Fatalf("session rotated an in-use CSRF token to %q", got.CSRFToken)
	}
}
