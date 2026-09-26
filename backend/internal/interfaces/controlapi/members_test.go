package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- list members --------------------------------------------------------

func TestListMembersHappyPath(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.list = func(_ context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
		return []domain.OrgMember{
			{User: testUser, Role: domain.RoleOwner, MemberSince: testNow},
			{User: testOtherUser, Role: domain.RoleMember, MemberSince: testNow},
		}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, bearerHeader(st, domain.RoleOwner))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var out struct {
		Items []orgMemberDTO `json:"items"`
	}
	decodeBody(t, resp, &out)

	requireEqual(t, len(out.Items), 2)
	requireEqual(t, out.Items[0].User.ID, testUserID.String())
	requireEqual(t, out.Items[0].Role, domain.RoleOwner)
	requireEqual(t, out.Items[1].User.ID, testOtherID.String())
	requireEqual(t, out.Items[1].Role, domain.RoleMember)
}

func TestListMembersAsMember(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.list = func(_ context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
		return []domain.OrgMember{{User: testUser, Role: domain.RoleMember, MemberSince: testNow}}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()
}

func TestListMembersStoreFailure(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.list = func(_ context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
		return nil, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/members", nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

// ---- set member role -----------------------------------------------------

func TestSetMemberRoleHappyPath(t *testing.T) {
	srv, st := newTestServer(t)
	var getCalls int
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		getCalls++
		if getCalls == 1 {
			return otherMember(domain.RoleMember), nil
		}
		return otherMember(domain.RoleAdmin), nil
	}
	var capturedRole string
	var capturedUserID uuid.UUID
	st.members.setRole = func(_ context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
		capturedRole = role
		capturedUserID = userID
		return domain.OrganizationMember{OrganizationID: orgID, UserID: userID, Role: role, CreatedAt: testNow}, nil
	}

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": domain.RoleAdmin}, bearerHeader(st, domain.RoleOwner))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var out orgMemberDTO
	decodeBody(t, resp, &out)

	requireEqual(t, out.User.ID, testOtherID.String())
	requireEqual(t, out.Role, domain.RoleAdmin)
	requireEqual(t, getCalls, 2)
	requireEqual(t, capturedRole, domain.RoleAdmin)
	requireEqual(t, capturedUserID, testOtherID)
	requireAudit(t, st, "member.role.update")
}

func TestSetMemberRoleRejectsInvalidRole(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": "superuser"}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestSetMemberRoleForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": domain.RoleAdmin}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestSetMemberRoleForbiddenForSelf(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testUserID.String()+"/role",
		map[string]any{"role": domain.RoleAdmin}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestSetMemberRoleForbiddenForAdminGrantingOwner(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": domain.RoleOwner}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestSetMemberRolePreservesLastOwner(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return otherMember(domain.RoleOwner), nil
	}
	st.members.countByRole = func(_ context.Context, orgID uuid.UUID, role string) (int64, error) {
		return 1, nil
	}

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": domain.RoleMember}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestSetMemberRoleMemberNotFound(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": domain.RoleAdmin}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestSetMemberRoleRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/"+testOtherID.String()+"/role",
		map[string]any{"role": 42}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

func TestSetMemberRoleRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/members/not-a-uuid/role",
		map[string]any{"role": domain.RoleAdmin}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

// ---- remove member -------------------------------------------------------

func TestRemoveMemberHappyPath(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return otherMember(domain.RoleMember), nil
	}
	st.members.remove = func(_ context.Context, orgID, userID uuid.UUID) error {
		return nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleOwner))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
	requireAudit(t, st, "member.remove")
}

func TestRemoveMemberForbiddenForSelf(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/members/"+testUserID.String(), nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestRemoveMemberPreservesLastOwner(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return otherMember(domain.RoleOwner), nil
	}
	st.members.countByRole = func(_ context.Context, orgID uuid.UUID, role string) (int64, error) {
		return 1, nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestRemoveMemberNotFound(t *testing.T) {
	srv, st := newTestServer(t)
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestRemoveMemberRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/members/not-a-uuid", nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}
