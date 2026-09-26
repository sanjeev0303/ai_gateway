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

const ctxKeyActor ctxKey = iota

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

// requireAuth authenticates the caller and stashes its actor in the request
// context before delegating to next. JWTs arrive as
// "Authorization: Bearer <jwt>"; machine identities arrive as
// "X-API-Key: <key>"; browser sessions arrive as the session cookie. Any
// failure to map the presented credential to an active principal responds 401
// without distinguishing the cause; a failed CSRF check on a cookie session
// responds 403.
func (a *api) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, method, err := a.authenticate(r)
		if err != nil {
			writeServiceError(w, r, identity.ErrUnauthorized)
			return
		}
		if !a.csrfAllowed(r, method) {
			writeServiceError(w, r, perrors.New(perrors.KindPermissionDenied, "csrf_token_invalid", "missing or invalid CSRF token"))
			return
		}
		next.ServeHTTP(w, r.WithContext(withActor(r.Context(), actor)))
	})
}

// authenticate resolves the caller from the Authorization header (JWT), the
// X-API-Key header (API key), or, when sessions are enabled, the session
// cookie (browser). The identity service maps every invalid credential to
// identity.ErrUnauthorized.
func (a *api) authenticate(r *http.Request) (domain.Actor, authMethod, error) {
	ctx := r.Context()
	if bearer := r.Header.Get("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
		actor, err := a.svc.AuthenticateToken(ctx, strings.TrimPrefix(bearer, "Bearer "))
		return actor, authMethodBearer, err
	}
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		actor, err := a.svc.AuthenticateAPIKey(ctx, key)
		return actor, authMethodAPIKey, err
	}
	if a.sessionCookieName != "" {
		if c, err := r.Cookie(a.sessionCookieName); err == nil {
			actor, err := a.svc.AuthenticateToken(ctx, c.Value)
			return actor, authMethodCookie, err
		}
	}
	return domain.Actor{}, 0, identity.ErrUnauthorized
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

// setSessionCookies issues the HttpOnly session cookie (the JWT) and the
// readable CSRF cookie; the browser echoes both on subsequent requests.
func (a *api) setSessionCookies(w http.ResponseWriter, token, csrfToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     a.csrfCookieName(),
		Value:    csrfToken,
		Path:     "/",
		Secure:   a.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookies expires the session and CSRF cookies.
func (a *api) clearSessionCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: a.sessionCookieName, Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: a.csrfCookieName(), Path: "/", MaxAge: -1})
}

// newCSRFToken returns a 256-bit random hex token for the double-submit flow.
func newCSRFToken() string {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}