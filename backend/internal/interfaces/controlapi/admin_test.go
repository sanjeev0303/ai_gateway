package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/platformadmin"
	"ai-governance-saas/backend/internal/domain"
)

func adminServerFor(t *testing.T, st *controlStubs, adminEmails ...string) func(method, path string) int {
	t.Helper()
	padmin := platformadmin.NewService(platformadmin.ServiceDeps{
		Users:       st.users,
		AdminEmails: adminEmails,
	})
	srv := buildServer(t, Deps{
		Identity:      identity.NewService(st.deps()),
		PlatformAdmin: padmin,
	})
	t.Cleanup(srv.Close)
	return func(method, path string) int {
		resp := doJSON(t, srv, method, path, nil, bearerHeader(st, domain.RoleOwner))
		resp.Body.Close()
		return resp.StatusCode
	}
}

func TestAdminRoutes_HiddenFromNonAdmins(t *testing.T) {
	st := newControlStubs()
	call := adminServerFor(t, st, "someone-else@example.com")

	if got := call(http.MethodGet, "/v1/admin/me"); got != http.StatusNotFound {
		t.Fatalf("non-admin owner got %d on /v1/admin/me, want 404", got)
	}
}

func TestAdminRoutes_OpenToAllowlistedAdmin(t *testing.T) {
	st := newControlStubs()
	call := adminServerFor(t, st, "ALICE@example.com") // the fixture user, case differs

	if got := call(http.MethodGet, "/v1/admin/me"); got != http.StatusOK {
		t.Fatalf("allowlisted admin got %d on /v1/admin/me, want 200", got)
	}
}

func TestSuspendedOrganization_BlocksConsoleButNotAdmin(t *testing.T) {
	st := newControlStubs()
	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		org := testOrg
		org.ID = id
		org.Status = domain.OrgStatusSuspended
		return org, nil
	}
	call := adminServerFor(t, st, "alice@example.com")

	if got := call(http.MethodGet, "/v1/members"); got != http.StatusForbidden {
		t.Fatalf("member of a suspended org got %d on /v1/members, want 403", got)
	}
	// The admin's own organization being suspended must not lock them out of
	// the console they would use to reactivate it.
	if got := call(http.MethodGet, "/v1/admin/me"); got != http.StatusOK {
		t.Fatalf("admin in a suspended org got %d on /v1/admin/me, want 200", got)
	}
}

func TestDisabledUser_RefusedWithValidToken(t *testing.T) {
	st := newControlStubs()
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		user := testUser
		user.ID = id
		user.Status = domain.UserStatusDisabled
		return user, nil
	}
	call := adminServerFor(t, st)

	if got := call(http.MethodGet, "/v1/members"); got != http.StatusUnauthorized {
		t.Fatalf("disabled user got %d on /v1/members, want 401", got)
	}
}
