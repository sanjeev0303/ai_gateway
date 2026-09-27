package controlapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	perrors "ai-governance-saas/backend/internal/platform/errors"
)

type ctxKey int

const (
	ctxKeyActor ctxKey = iota
	ctxKeyClaims
)

// authMethod records how a caller was authenticated so CSRF enforcement can
// be scoped to browser-origin (cookie) sessions.
type authMethod int

const (
	authMethodBearer authMethod = iota
	authMethodAPIKey
	authMethodCookie
)

// csrfTokenBytes is the CSRF token entropy: 32 random bytes, 64 hex chars.
const csrfTokenBytes = 32

// withActor stores the authenticated actor in the request context.
func withActor(ctx context.Context, actor domain.Actor) context.Context {
	return context.WithValue(ctx, ctxKeyActor, actor)
}

// actorFrom returns the actor stored by requireAuth. Every v1 route runs
// behind requireAuth, so the actor is always present there; dev-login and
// logout are the only routes that skip it.
func actorFrom(ctx context.Context) (domain.Actor, bool) {
	actor, ok := ctx.Value(ctxKeyActor).(domain.Actor)
	return actor, ok
}

// withClaims stores the verified token claims alongside the actor. Claims are
// absent for API-key callers, which authenticate a machine rather than a user
// session and have no claim set to report.
func withClaims(ctx context.Context, claims domain.TokenClaims) context.Context {
	return context.WithValue(ctx, ctxKeyClaims, claims)
}

// claimsFrom returns the claims requireAuth verified for this request.
func claimsFrom(ctx context.Context) (domain.TokenClaims, bool) {
	claims, ok := ctx.Value(ctxKeyClaims).(domain.TokenClaims)
	return claims, ok
}

// requireAuth authenticates the caller and stashes its actor in the request
// context before delegating to next. JWTs arrive as
// "Authorization: Bearer <jwt>"; machine identities arrive as
// "X-API-Key: <key>"; browser sessions arrive as the session cookie. Any
// failure to map the presented credential to an active principal responds 401
// without distinguishing the cause; a failed CSRF check on a cookie session
// responds 403.
func (a *api) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, claims, method, err := a.authenticate(r)
		if err != nil {
			writeServiceError(w, r, identity.ErrUnauthorized)
			return
		}
		if !a.csrfAllowed(r, method) {
			writeServiceError(w, r, perrors.New(perrors.KindPermissionDenied, "csrf_token_invalid", "missing or invalid CSRF token"))
			return
		}
		ctx := withActor(r.Context(), actor)
		if claims.Subject != "" {
			ctx = withClaims(ctx, claims)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticate resolves the caller from the Authorization header (JWT), the
// X-API-Key header (API key), or, when sessions are enabled, the session
// cookie (browser). The identity service maps every invalid credential to
// identity.ErrUnauthorized.
func (a *api) authenticate(r *http.Request) (domain.Actor, domain.TokenClaims, authMethod, error) {
	ctx := r.Context()
	if bearer := r.Header.Get("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
		claims, err := a.svc.VerifyToken(ctx, strings.TrimPrefix(bearer, "Bearer "))
		return claims.Actor(), claims, authMethodBearer, err
	}
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		actor, err := a.svc.AuthenticateAPIKey(ctx, key)
		return actor, domain.TokenClaims{}, authMethodAPIKey, err
	}
	if a.sessionCookieName != "" {
		if c, err := r.Cookie(a.sessionCookieName); err == nil {
			claims, err := a.svc.VerifyToken(ctx, c.Value)
			return claims.Actor(), claims, authMethodCookie, err
		}
	}
	return domain.Actor{}, domain.TokenClaims{}, 0, identity.ErrUnauthorized
}

// csrfAllowed enforces the double-submit pattern for cookie-authenticated
// state changes: the X-CSRF-Token header must match the CSRF cookie issued at
// login. Bearer/API-key callers are not browser-origin-exposed and read-only
// methods cannot be CSRF-driven, so both skip the check.
func (a *api) csrfAllowed(r *http.Request, method authMethod) bool {
	if method != authMethodCookie || !a.csrfEnabled {
		return true
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	cookie, err := r.Cookie(a.csrfCookieName())
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(cookie.Value)) == 1
}

func (a *api) csrfCookieName() string {
	return a.sessionCookieName + "_csrf"
}

func (a *api) refreshCookieName() string {
	return a.sessionCookieName + "_refresh"
}

// requireCSRF enforces the double-submit check for cookie-authenticated state
// changes that have no actor to authenticate, such as refresh and logout. It
// applies unconditionally when CSRF is enabled: a browser is the only thing
// that can present these cookies, so there is no non-browser caller to exempt.
func (a *api) requireCSRF(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.csrfEnabled && !a.csrfAllowed(r, authMethodCookie) {
			writeServiceError(w, r, perrors.New(perrors.KindPermissionDenied, "csrf_token_invalid", "missing or invalid CSRF token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// setSessionCookies issues the HttpOnly access and refresh cookies plus the
// readable CSRF cookie; the browser echoes them back on later requests. The
// access token is a signed JWT, the refresh token is an opaque credential
// rotated on every use.
func (a *api) setSessionCookies(w http.ResponseWriter, session identity.Session, csrfToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.sessionCookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	if session.RefreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     a.refreshCookieName(),
			Value:    session.RefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   a.sessionCookieSecure,
			SameSite: http.SameSiteStrictMode,
		})
	}
	a.setCSRFCookie(w, csrfToken)
}

// setCSRFCookie issues the JS-readable CSRF cookie. It is deliberately not
// HttpOnly: the browser must copy this one value into the X-CSRF-Token header
// to satisfy the double-submit check. Kept separate from setSessionCookies so
// the session endpoint can issue it without re-issuing session cookies.
func (a *api) setCSRFCookie(w http.ResponseWriter, csrfToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.csrfCookieName(),
		Value:    csrfToken,
		Path:     "/",
		Secure:   a.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookies expires the access, refresh, and CSRF cookies.
func (a *api) clearSessionCookies(w http.ResponseWriter) {
	secure := a.sessionCookieSecure
	for _, c := range []struct {
		name string
		// The refresh cookie is Strict: it is only ever sent to this origin's
		// own /refresh route, so it never needs the Lax top-level-navigation
		// allowance the access cookie gets.
		sameSite http.SameSite
	}{
		{a.sessionCookieName, http.SameSiteLaxMode},
		{a.refreshCookieName(), http.SameSiteStrictMode},
		{a.csrfCookieName(), http.SameSiteLaxMode},
	} {
		http.SetCookie(w, &http.Cookie{
			Name: c.name, Path: "/", MaxAge: -1, Secure: secure, SameSite: c.sameSite,
		})
	}
}

// newCSRFToken returns a 256-bit random hex token for the double-submit flow.
func newCSRFToken() string {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
