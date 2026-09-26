package controlapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/platform/health"
)

type healthBody struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// TestHealthProbesReportOK exercises all three probe paths with an empty
// registry: an empty registry is healthy.
func TestHealthProbesReportOK(t *testing.T) {
	srv := buildServer(t, Deps{})
	for _, path := range []string{"/livez", "/readyz", "/healthz"} {
		resp, err := srv.Client().Get(srv.URL + path)
		requireNoErr(t, err)
		requireEqual(t, resp.StatusCode, http.StatusOK)

		ct := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("%s: Content-Type = %q, want application/json", path, ct)
		}
		if resp.Header.Get("X-Request-Id") == "" {
			t.Fatalf("%s: missing X-Request-Id trace header", path)
		}
		var body healthBody
		decodeBody(t, resp, &body)
		requireEqual(t, body.Status, "ok")
		if len(body.Checks) != 0 {
			t.Fatalf("%s: checks = %v, want empty", path, body.Checks)
		}
	}
}

// TestHealthDegradedWhenCheckFails proves a failing dependency check flips the
// probe to 503 with the check name and failure surfaced in the body.
func TestHealthDegradedWhenCheckFails(t *testing.T) {
	reg := health.NewRegistry()
	reg.Register("db", func(context.Context) error { return errors.New("down") })
	srv := buildServerReg(t, Deps{}, reg)

	resp, err := srv.Client().Get(srv.URL + "/readyz")
	requireNoErr(t, err)
	requireEqual(t, resp.StatusCode, http.StatusServiceUnavailable)

	var body healthBody
	decodeBody(t, resp, &body)
	requireEqual(t, body.Status, "degraded")
	requireEqual(t, body.Checks["db"], "down")
}

// TestV1RoutesAbsentWithoutIdentity proves the v1 surface is not wired in at
// all when deps.Identity is nil: requests 404 instead of hitting handlers.
func TestV1RoutesAbsentWithoutIdentity(t *testing.T) {
	srv := buildServer(t, Deps{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, nil)
	requireEqual(t, resp.StatusCode, http.StatusNotFound)
	resp.Body.Close()

	resp, err := srv.Client().Get(srv.URL + "/livez")
	requireNoErr(t, err)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body healthBody
	decodeBody(t, resp, &body)
	requireEqual(t, body.Status, "ok")
}

// TestProviderRoutesAbsentWithoutService proves the M4 surface is not wired
// in when deps.Providers is nil: requests 404 instead of hitting handlers.
func TestProviderRoutesAbsentWithoutService(t *testing.T) {
	st := newControlStubs()
	svc := identity.NewService(st.deps())
	srv := buildServer(t, Deps{Identity: svc})

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "accounts", path: "/v1/provider-accounts"},
		{name: "credentials", path: "/v1/provider-credentials"},
		{name: "models", path: "/v1/models"},
	} {
		resp := doJSON(t, srv, http.MethodGet, tc.path, nil, nil)
		requireEqual(t, resp.StatusCode, http.StatusNotFound)
		resp.Body.Close()
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]string{"name": "x"}, nil)
	requireEqual(t, resp.StatusCode, http.StatusNotFound)
	resp.Body.Close()

	resp, err := srv.Client().Get(srv.URL + "/livez")
	requireNoErr(t, err)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
}

// TestProviderRoutesRequireAuth proves every M4 route sits behind the same
// authentication as the rest of the v1 surface.
func TestProviderRoutesRequireAuth(t *testing.T) {
	srv, _ := newTestServer(t)

	for _, tc := range []struct {
		name string
		meth string
		path string
	}{
		{name: "create account", meth: http.MethodPost, path: "/v1/provider-accounts"},
		{name: "list accounts", meth: http.MethodGet, path: "/v1/provider-accounts"},
		{name: "create credential", meth: http.MethodPost, path: "/v1/provider-accounts/" + testProvAcct.String() + "/credentials"},
		{name: "list credentials", meth: http.MethodGet, path: "/v1/provider-credentials"},
		{name: "revoke credential", meth: http.MethodDelete, path: "/v1/provider-credentials/" + testProvCred.String()},
		{name: "models", meth: http.MethodGet, path: "/v1/models"},
	} {
		resp := doJSON(t, srv, tc.meth, tc.path, map[string]string{"name": "x"}, nil)
		status, code := decodeError(t, resp)
		requireEqual(t, status, http.StatusUnauthorized)
		requireEqual(t, code, "unauthenticated")
	}
}

// TestDevLoginRouteAbsentWhenDisabled proves dev-login is wired only when
// DevLoginEnabled is set; when disabled the route 404s while the rest of the
// v1 surface (and its auth) still works.
func TestDevLoginRouteAbsentWhenDisabled(t *testing.T) {
	st := newControlStubs()
	svc := identity.NewService(st.deps())
	srv := buildServer(t, Deps{Identity: svc, DevLoginEnabled: false})

	resp := doJSON(t, srv, http.MethodPost, "/v1/auth/dev-login", map[string]string{"email": "a@b.c"}, nil)
	requireEqual(t, resp.StatusCode, http.StatusNotFound)
	resp.Body.Close()

	resp = doJSON(t, srv, http.MethodGet, "/v1/members", nil, nil)
	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusUnauthorized)
	requireEqual(t, code, "unauthenticated")
}
