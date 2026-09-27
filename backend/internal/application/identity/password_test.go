package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

const testPassword = "correct horse battery staple"

const testDigest = "$pbkdf2-sha256$600000$c2FsdA$a2V5"

func activeUser() domain.User {
	return domain.User{
		ID:          testUserID,
		Email:       "owner@example.com",
		DisplayName: "Owner",
		Status:      domain.UserStatusActive,
	}
}

// registerWiring stubs only what Register needs: the access-token issuer, the
// refresh digest, and the audit sink. It deliberately leaves
// refreshTokens.Create unstubbed, so a stray call fails the test.
func registerWiring(st *testStubs, captured *string) {
	st.issuer.issue = func(context.Context, domain.TokenClaims) (string, error) {
		return "access-token", nil
	}
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		*captured = secret
		return "digest:" + secret, nil
	}
	st.audits.appendFn = func(context.Context, uuid.UUID, domain.NewAuditLog) (domain.AuditLog, error) {
		return domain.AuditLog{}, nil
	}
}

// issueSessionWiring registers the token ports a successful session needs and
// reports the raw value the service handed back for cookieing.
func issueSessionWiring(st *testStubs, captured *string) {
	st.issuer.issue = func(context.Context, domain.TokenClaims) (string, error) {
		return "access-token", nil
	}
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		*captured = secret
		return "digest:" + secret, nil
	}
	st.refresh.create = func(context.Context, uuid.UUID, string, time.Time) (domain.RefreshToken, error) {
		return domain.RefreshToken{ID: testInviteID, UserID: testUserID}, nil
	}
	st.audits.appendFn = func(context.Context, uuid.UUID, domain.NewAuditLog) (domain.AuditLog, error) {
		return domain.AuditLog{}, nil
	}
}

// TestAuthenticateRejectionsAreIndistinguishable pins the enumeration defence:
// an unknown email, an account with no password, a malformed stored hash, and
// a wrong password must all cost a real verification and all return the same
// error. A path that skipped Verify would answer measurably faster.
func TestAuthenticateRejectionsAreIndistinguishable(t *testing.T) {
	unknown := repositories.ErrNotFound
	noPassword := domain.UserCredential{User: activeUser()}
	broken := "not-a-digest"
	stored := testDigest
	cases := []struct {
		name       string
		credential domain.UserCredential
		err        error
		wantHash   string
	}{
		{"unknown email falls back to the decoy", domain.UserCredential{}, unknown, "decoy-digest"},
		{"null password falls back to the decoy", noPassword, nil, "decoy-digest"},
		{"malformed stored hash is still verified", domain.UserCredential{User: activeUser(), PasswordHash: &broken}, nil, broken},
		{"wrong password is verified against the stored hash", domain.UserCredential{User: activeUser(), PasswordHash: &stored}, nil, stored},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, st := newTestService()
			st.passwords.decoy = "decoy-digest"
			st.auth.byEmail = func(context.Context, string) (domain.UserCredential, error) {
				return tc.credential, tc.err
			}
			var verifiedAgainst string
			st.passwords.verify = func(_ context.Context, encoded, _ string) error {
				verifiedAgainst = encoded
				return errors.New("mismatch")
			}
			_, err := svc.Authenticate(context.Background(), "owner@example.com", "whatever-long-enough")
			requireErrIs(t, err, ErrUnauthorized)
			requireEqual(t, verifiedAgainst, tc.wantHash)
		})
	}
}

// TestAuthenticateDisabledAccountStaysUniform covers a correct password on an
// unusable account: it must be indistinguishable from a wrong password.
func TestAuthenticateDisabledAccountStaysUniform(t *testing.T) {
	svc, st := newTestService()
	disabled := activeUser()
	disabled.Status = domain.UserStatusDisabled
	stored := testDigest
	st.passwords.decoy = "decoy-digest"
	st.auth.byEmail = func(context.Context, string) (domain.UserCredential, error) {
		return domain.UserCredential{User: disabled, PasswordHash: &stored}, nil
	}
	st.passwords.verify = func(context.Context, string, string) error { return nil }
	st.auth.touchLogin = func(context.Context, uuid.UUID) error {
		t.Fatal("a disabled account must not be able to record a login")
		return nil
	}
	_, err := svc.Authenticate(context.Background(), "owner@example.com", testPassword)
	requireErrIs(t, err, ErrUnauthorized)
}

func TestAuthenticateSuccessIssuesSession(t *testing.T) {
	svc, st := newTestService()
	stored := testDigest
	st.passwords.decoy = "decoy-digest"
	st.auth.byEmail = func(context.Context, string) (domain.UserCredential, error) {
		return domain.UserCredential{User: activeUser(), PasswordHash: &stored}, nil
	}
	st.passwords.verify = func(context.Context, string, string) error { return nil }
	touched := false
	st.auth.touchLogin = func(context.Context, uuid.UUID) error {
		touched = true
		return nil
	}
	st.users.listOrgs = func(context.Context, uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{
			Organization: domain.Organization{ID: testOrgID, Name: "Acme", Slug: "acme"},
			Role:         domain.RoleOwner,
		}}, nil
	}
	var rawRefresh string
	issueSessionWiring(st, &rawRefresh)

	session, err := svc.Authenticate(context.Background(), "owner@example.com", testPassword)
	requireNoErr(t, err)
	requireEqual(t, session.Token, "access-token")
	if !touched {
		t.Fatal("a successful authentication must record last login")
	}
	if session.RefreshToken == "" {
		t.Fatal("a session must carry a raw refresh token for the caller to cookie")
	}
	requireEqual(t, session.Claims.OrganizationID, testOrgID)
	requireEqual(t, session.Claims.Role, domain.RoleOwner)
}

func TestRegisterRejectsWeakPasswords(t *testing.T) {
	for _, pw := range []string{"", "short", strings.Repeat("a", maxPasswordLength+1)} {
		svc, st := newTestService()
		st.auth.register = func(context.Context, domain.Registration) (domain.User, domain.Organization, error) {
			t.Fatalf("weak password %q reached persistence", pw)
			return domain.User{}, domain.Organization{}, nil
		}
		st.passwords.hash = func(context.Context, string) (string, error) {
			t.Fatal("a weak password must be rejected before hashing")
			return "", nil
		}
		_, err := svc.Register(context.Background(), "owner@example.com", "Owner", "", pw)
		requireErrIs(t, err, repositories.ErrInvalid)
	}
}

func TestRegisterAcceptsBoundaryLengths(t *testing.T) {
	for _, pw := range []string{strings.Repeat("a", minPasswordLength), strings.Repeat("a", maxPasswordLength)} {
		svc, st := newTestService()
		st.passwords.hash = func(context.Context, string) (string, error) { return testDigest, nil }
		var seen domain.Registration
		st.auth.register = func(_ context.Context, reg domain.Registration) (domain.User, domain.Organization, error) {
			seen = reg
			return activeUser(), domain.Organization{ID: testOrgID}, nil
		}
		var rawRefresh string
		registerWiring(st, &rawRefresh)
		_, err := svc.Register(context.Background(), "owner@example.com", "Owner", "", pw)
		requireNoErr(t, err)
		requireEqual(t, seen.PasswordHash, testDigest)
	}
}

func TestRegisterSurfacesDuplicateEmail(t *testing.T) {
	svc, st := newTestService()
	st.passwords.hash = func(context.Context, string) (string, error) { return testDigest, nil }
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "digest:" + secret, nil
	}
	attempts := 0
	st.auth.register = func(context.Context, domain.Registration) (domain.User, domain.Organization, error) {
		attempts++
		return domain.User{}, domain.Organization{}, repositories.ErrConflict
	}
	_, err := svc.Register(context.Background(), "owner@example.com", "Owner", "", testPassword)
	requireErrIs(t, err, repositories.ErrConflict)
	if attempts != maxRegistrationAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, maxRegistrationAttempts)
	}
}

// TestRegisterRetriesSlugCollision checks a slug clash is retried under a new
// suffix instead of failing the signup.
func TestRegisterRetriesSlugCollision(t *testing.T) {
	svc, st := newTestService()
	st.passwords.hash = func(context.Context, string) (string, error) { return testDigest, nil }
	var slugs []string
	st.auth.register = func(_ context.Context, reg domain.Registration) (domain.User, domain.Organization, error) {
		slugs = append(slugs, reg.OrganizationSlug)
		if len(slugs) == 1 {
			return domain.User{}, domain.Organization{}, repositories.ErrConflict
		}
		return activeUser(), domain.Organization{ID: testOrgID}, nil
	}
	var rawRefresh string
	registerWiring(st, &rawRefresh)
	_, err := svc.Register(context.Background(), "owner@example.com", "Acme", "", testPassword)
	requireNoErr(t, err)
	if len(slugs) != 2 {
		t.Fatalf("slugs = %v, want 2 attempts", slugs)
	}
	if slugs[0] == slugs[1] {
		t.Fatalf("the retry reused slug %q", slugs[1])
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	svc, st := newTestService()
	st.users.getByID = func(context.Context, uuid.UUID) (domain.User, error) {
		return activeUser(), nil
	}
	st.users.listOrgs = func(context.Context, uuid.UUID) ([]domain.UserOrganization, error) {
		return []domain.UserOrganization{{
			Organization: domain.Organization{ID: testOrgID},
			Role:         domain.RoleOwner,
		}}, nil
	}
	st.refresh.getByHash = func(context.Context, string) (domain.RefreshToken, error) {
		return domain.RefreshToken{ID: testAcctID, UserID: testUserID, ExpiresAt: testNow.Add(time.Hour)}, nil
	}
	revoked := false
	st.refresh.revoke = func(context.Context, string) error {
		revoked = true
		return nil
	}
	var rawRefresh string
	issueSessionWiring(st, &rawRefresh)

	session, err := svc.Refresh(context.Background(), "presented-token")
	requireNoErr(t, err)
	if !revoked {
		t.Fatal("rotation must retire the presented token")
	}
	if session.RefreshToken == "" || session.RefreshToken == "presented-token" {
		t.Fatalf("rotation must issue a new refresh token, got %q", session.RefreshToken)
	}
}

// TestRefreshReuseRevokesEverywhere is the theft-response path: a retired token
// coming back means the cookie leaked, so the user is signed out completely.
func TestRefreshReuseRevokesEverywhere(t *testing.T) {
	svc, st := newTestService()
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "digest:" + secret, nil
	}
	retired := testNow.Add(-time.Hour)
	st.refresh.getByHash = func(context.Context, string) (domain.RefreshToken, error) {
		return domain.RefreshToken{ID: testAcctID, UserID: testUserID, RevokedAt: &retired}, nil
	}
	revokedAll := false
	st.refresh.revokeAll = func(_ context.Context, userID uuid.UUID) error {
		revokedAll = true
		requireEqual(t, userID, testUserID)
		return nil
	}
	st.refresh.revoke = func(context.Context, string) error {
		t.Fatal("a retired token must not be revoked a second time")
		return nil
	}
	_, err := svc.Refresh(context.Background(), "stolen-token")
	requireErrIs(t, err, ErrUnauthorized)
	if !revokedAll {
		t.Fatal("reuse must revoke every active token for the user")
	}
}

// TestRefreshRaceLoserGetsNothing covers two concurrent exchanges of one
// token: the loser must not mint a second session.
func TestRefreshRaceLoserGetsNothing(t *testing.T) {
	svc, st := newTestService()
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "digest:" + secret, nil
	}
	st.users.getByID = func(context.Context, uuid.UUID) (domain.User, error) {
		return activeUser(), nil
	}
	st.refresh.getByHash = func(context.Context, string) (domain.RefreshToken, error) {
		return domain.RefreshToken{ID: testAcctID, UserID: testUserID, ExpiresAt: testNow.Add(time.Hour)}, nil
	}
	st.refresh.revoke = func(context.Context, string) error {
		return repositories.ErrNotFound
	}
	st.refresh.create = func(context.Context, uuid.UUID, string, time.Time) (domain.RefreshToken, error) {
		t.Fatal("the losing request must not persist a new session")
		return domain.RefreshToken{}, nil
	}
	_, err := svc.Refresh(context.Background(), "presented-token")
	requireErrIs(t, err, ErrUnauthorized)
}

func TestRefreshRejectsExpired(t *testing.T) {
	svc, st := newTestService()
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "digest:" + secret, nil
	}
	st.refresh.getByHash = func(context.Context, string) (domain.RefreshToken, error) {
		return domain.RefreshToken{ID: testAcctID, UserID: testUserID, ExpiresAt: testNow.Add(-time.Second)}, nil
	}
	st.refresh.revoke = func(context.Context, string) error {
		t.Fatal("an expired token must not be retired as if it were live")
		return nil
	}
	_, err := svc.Refresh(context.Background(), "presented-token")
	requireErrIs(t, err, ErrUnauthorized)
}

func TestRefreshRejectsEmptyToken(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Refresh(context.Background(), "")
	requireErrIs(t, err, ErrUnauthorized)
}

// TestRevokeRefreshIsIdempotent: a client that lost its session still needs
// logout to succeed.
func TestRevokeRefreshIsIdempotent(t *testing.T) {
	svc, st := newTestService()
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "digest:" + secret, nil
	}
	st.refresh.revoke = func(context.Context, string) error { return repositories.ErrNotFound }
	requireNoErr(t, svc.RevokeRefresh(context.Background(), "already-gone"))
	requireNoErr(t, svc.RevokeRefresh(context.Background(), ""))
}

func TestRevokeRefreshPropagatesRealFailure(t *testing.T) {
	svc, st := newTestService()
	st.hasher.hash = func(context.Context, string) (string, error) { return "h", nil }
	boom := errors.New("db down")
	st.refresh.revoke = func(context.Context, string) error { return boom }
	if err := svc.RevokeRefresh(context.Background(), "token"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}

// TestSessionPersistsOnlyTheDigest pins both halves of the refresh contract:
// the transaction receives the digest and its expiry, and the raw value comes
// back to the caller for cookieing instead of reaching storage.
func TestSessionPersistsOnlyTheDigest(t *testing.T) {
	svc, st := newTestService()
	st.passwords.hash = func(context.Context, string) (string, error) { return testDigest, nil }
	var seen domain.Registration
	st.auth.register = func(_ context.Context, reg domain.Registration) (domain.User, domain.Organization, error) {
		seen = reg
		return activeUser(), domain.Organization{ID: testOrgID}, nil
	}
	var rawRefresh string
	registerWiring(st, &rawRefresh)
	session, err := svc.Register(context.Background(), "owner@example.com", "Owner", "", testPassword)
	requireNoErr(t, err)
	if rawRefresh == "" {
		t.Fatal("Register must mint a refresh token")
	}
	if seen.RefreshTokenHash != "digest:"+rawRefresh {
		t.Fatalf("transaction got hash %q, want the digest of the issued token", seen.RefreshTokenHash)
	}
	if want := testNow.Add(30 * 24 * time.Hour); !seen.RefreshTokenExpiresAt.Equal(want) {
		t.Fatalf("expiresAt = %v, want %v", seen.RefreshTokenExpiresAt, want)
	}
	if session.RefreshToken != rawRefresh {
		t.Fatalf("session carries %q, want the raw token %q", session.RefreshToken, rawRefresh)
	}
}
