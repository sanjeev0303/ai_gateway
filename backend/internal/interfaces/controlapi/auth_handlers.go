package controlapi

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	perrors "ai-governance-saas/backend/internal/platform/errors"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports/repositories"
)

const (
	// signInAttempts and signInWindowDuration bound password guessing per client IP
	// plus email. Ten tries a minute is generous for a human and useless to an
	// offline guesser, whose cost is dominated by the KDF rather than by this
	// limit.
	signInAttempts       = 10
	signInWindowDuration = time.Minute

	// signInThrottleMaxWindows caps the limiter's memory. Reaching it means
	// many distinct keys are in flight, which is itself an attack signal, so
	// the expired ones are swept before any new key is admitted.
	signInThrottleMaxWindows = 4096
)

type signUpRequest struct {
	Email            string `json:"email"`
	DisplayName      string `json:"display_name"`
	OrganizationName string `json:"organization_name"`
	Password         string `json:"password"`
}

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// signup creates the account and returns a session. The password is never
// logged or echoed; only the resulting identity crosses the wire.
func (a *api) signup(w http.ResponseWriter, r *http.Request) {
	var req signUpRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	session, err := a.svc.Register(r.Context(), req.Email, req.DisplayName, req.OrganizationName, req.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	a.writeSession(w, session, http.StatusCreated)
}

// signin exchanges a password for a session, rate limited per client IP plus
// email so neither a single host nor a single account can be ground down.
func (a *api) signin(w http.ResponseWriter, r *http.Request) {
	var req signInRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	if retry, ok := a.signInLimiter.allow(r, req.Email); !ok {
		w.Header().Set("Retry-After", retrySeconds(retry))
		httpx.WriteError(w, http.StatusTooManyRequests, httpx.RequestIDFrom(r.Context()),
			perrors.New(perrors.KindRateLimited, "rate_limited", "too many attempts, retry later"))
		return
	}
	session, err := a.svc.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	a.writeSession(w, session, http.StatusOK)
}

// refresh rotates the refresh cookie. It is cookie-authenticated, so it requires
// the double-submit CSRF check even though it has no access token to fall back
// on: a cross-site POST would otherwise mint a fresh session from the victim's
// long-lived cookie.
func (a *api) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(a.refreshCookieName())
	if err != nil || c.Value == "" {
		writeServiceError(w, r, errMissingRefreshToken)
		return
	}
	session, err := a.svc.Refresh(r.Context(), c.Value)
	if err != nil {
		a.clearSessionCookies(w)
		writeServiceError(w, r, err)
		return
	}
	a.writeSession(w, session, http.StatusOK)
}

// session reports the caller's verified session so a browser can render a
// signed-in state and obtain the CSRF token it must echo on later state
// changes. It never re-issues access or refresh cookies: this is a read, not a
// renewal, so calling it cannot extend a session's life.
func (a *api) session(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeServiceError(w, r, identity.ErrUnauthorized)
		return
	}
	dto := sessionDTO{Claims: toClaimsDTO(claims)}
	if a.sessionCookieName != "" {
		if c, err := r.Cookie(a.csrfCookieName()); err == nil {
			dto.CSRFToken = c.Value
		} else {
			dto.CSRFToken = newCSRFToken()
			a.setCSRFCookie(w, dto.CSRFToken)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, dto)
}

// writeSession emits the session and, when the deployment uses cookies, the
// cookie set and CSRF token. Credentials exist only in the cookies.
func (a *api) writeSession(w http.ResponseWriter, session identity.Session, status int) {
	dto := sessionDTO{Claims: toClaimsDTO(session.Claims)}
	if a.sessionCookieName != "" {
		csrf := newCSRFToken()
		a.setSessionCookies(w, session, csrf)
		dto.CSRFToken = csrf
	}
	httpx.WriteJSON(w, status, dto)
}

// ---- sign-in throttle -----------------------------------------------------

// errMissingRefreshToken is returned when /refresh is called without the
// cookie. It is an authentication failure, not a bad request, so it must not
// imply the caller's cookie was malformed.
var errMissingRefreshToken = perrors.New(perrors.KindUnauthenticated, "missing_refresh_token", "missing refresh token")

// signInThrottle is a fixed-window counter. A fixed window can admit up to
// twice the limit across a boundary, which is an acceptable trade for keeping
// this dependency-free and bounded; the KDF cost per attempt is the real
// brake on distributed guessing anyway.
type signInThrottle struct {
	mu      sync.Mutex
	windows map[string]signInWindow
	limit   int
	span    time.Duration
	now     func() time.Time
}

type signInWindow struct {
	count   int
	resetAt time.Time
}

// allow records an attempt and reports whether it may proceed, along with how
// long the caller should wait when it may not. The key combines client IP and
// the email being tried: keying on IP alone lets one attacker lock out a
// legitimate user, and keying on email alone lets a bot spray one account
// from any host.
func (t *signInThrottle) allow(r *http.Request, email string) (retry time.Duration, ok bool) {
	now := t.now()
	key := clientIP(r) + "\x00" + strings.ToLower(strings.TrimSpace(email))

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.windows == nil {
		t.windows = make(map[string]signInWindow)
	}
	if len(t.windows) >= signInThrottleMaxWindows {
		for k, w := range t.windows {
			if !now.Before(w.resetAt) {
				delete(t.windows, k)
			}
		}
	}

	w := t.windows[key]
	if !now.Before(w.resetAt) {
		w = signInWindow{count: 0, resetAt: now.Add(t.span)}
	}
	if w.count >= t.limit {
		return w.resetAt.Sub(now), false
	}
	w.count++
	t.windows[key] = w
	return 0, true
}

func newSignInThrottle() *signInThrottle {
	return &signInThrottle{
		windows: make(map[string]signInWindow),
		limit:   signInAttempts,
		span:    signInWindowDuration,
		now:     time.Now,
	}
}

// clientIP reads the peer address. A forwarded header is deliberately not
// trusted: it is client-controlled, so honouring it would let an attacker
// mint an unlimited number of throttle buckets. Behind a proxy this collapses
// to the shared proxy address, which throttles more conservatively rather than
// less.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// retrySeconds renders a duration as whole seconds, rounded up so a client
// never retries a fraction of a second early and is rejected again.
func retrySeconds(d time.Duration) string {
	secs := int(d / time.Second)
	if d%time.Second != 0 {
		secs++
	}
	if secs < 1 {
		secs = 1
	}
	return strconv.Itoa(secs)
}

type switchOrganizationRequest struct {
	OrganizationID string `json:"organization_id"`
}

func (a *api) switchOrg(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	if actor.Kind != domain.ActorKindUser {
		writeServiceError(w, r, identity.ErrUnauthorized)
		return
	}
	var req switchOrganizationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		writeServiceError(w, r, repositories.ErrInvalid)
		return
	}
	session, err := a.svc.SwitchOrganization(r.Context(), actor.UserID, orgID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	a.writeSession(w, session, http.StatusOK)
}
