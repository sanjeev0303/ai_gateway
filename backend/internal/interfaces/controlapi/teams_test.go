package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- create team ----------------------------------------------------------

func TestCreateTeamHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var captured domain.NewTeam
	st.teams.create = func(_ context.Context, orgID uuid.UUID, in domain.NewTeam) (domain.Team, error) {
		captured = in
		return testTeam, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams", map[string]any{
		"name":        "core",
		"description": "core team",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body teamDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ID, testTeamID.String())
	requireEqual(t, body.OrganizationID, testOrgID.String())
	requireEqual(t, body.Name, "core")
	requireEqual(t, body.Description, "core team")
	requireEqual(t, body.CreatedAt, ts(testNow))

	requireEqual(t, captured.Name, "core")
	requireEqual(t, captured.Description, "core team")
	requireAudit(t, st, "team.create")
}

func TestCreateTeamRejectsBlankName(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams", map[string]any{
		"name": "   ",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateTeamRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams", map[string]any{
		"name": 42,
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

func TestCreateTeamForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams", map[string]any{
		"name": "core",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- list teams ------------------------------------------------------------

func TestListTeamsHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.teams.list = func(_ context.Context, orgID uuid.UUID) ([]domain.Team, error) {
		return []domain.Team{testTeam}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/teams", nil, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []teamDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ID, testTeamID.String())
	requireEqual(t, body.Items[0].Name, "core")
}

func TestListTeamsStoreFailure(t *testing.T) {
	srv, st := newTestServer(t)

	st.teams.list = func(_ context.Context, orgID uuid.UUID) ([]domain.Team, error) {
		return nil, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/teams", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

// ---- get team ---------------------------------------------------------------

func TestGetTeamHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.teams.get = func(_ context.Context, orgID, id uuid.UUID) (domain.Team, error) {
		return testTeam, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/teams/"+testTeamID.String(), nil, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body teamDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ID, testTeamID.String())
	requireEqual(t, body.Name, "core")
}

func TestGetTeamNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.teams.get = func(_ context.Context, orgID, id uuid.UUID) (domain.Team, error) {
		return domain.Team{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/teams/"+testTeamID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestGetTeamRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodGet, "/v1/teams/not-a-uuid", nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

// ---- add team member --------------------------------------------------------

func TestAddTeamMemberHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var addedTeamID, addedUserID uuid.UUID
	var addedRole string
	st.teamMembers.add = func(_ context.Context, orgID, teamID, userID uuid.UUID, role string) (domain.TeamMember, error) {
		addedTeamID = teamID
		addedUserID = userID
		addedRole = role
		return domain.TeamMember{
			TeamID: teamID, OrganizationID: orgID, UserID: userID, Role: role, CreatedAt: testNow,
		}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams/"+testTeamID.String()+"/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "member",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body teamMemberRowDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.TeamID, testTeamID.String())
	requireEqual(t, body.UserID, testOtherID.String())
	requireEqual(t, body.Role, "member")
	requireEqual(t, body.MemberSince, ts(testNow))

	requireEqual(t, addedTeamID, testTeamID)
	requireEqual(t, addedUserID, testOtherID)
	requireEqual(t, addedRole, "member")
	requireAudit(t, st, "team_member.add")
}

func TestAddTeamMemberRejectsOwnerRole(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams/"+testTeamID.String()+"/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "owner",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestAddTeamMemberRejectsInvalidUserID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams/"+testTeamID.String()+"/members", map[string]any{
		"user_id": "not-a-uuid",
		"role":    "member",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestAddTeamMemberRejectsInvalidTeamID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams/not-a-uuid/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "member",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestAddTeamMemberForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/teams/"+testTeamID.String()+"/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "member",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- list team members ------------------------------------------------------

func TestListTeamMembersHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.teamMembers.list = func(_ context.Context, orgID, teamID uuid.UUID) ([]domain.TeamMemberView, error) {
		return []domain.TeamMemberView{{User: testUser, Role: "member", MemberSince: testNow}}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/teams/"+testTeamID.String()+"/members", nil, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []membershipDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].User.ID, testUserID.String())
	requireEqual(t, body.Items[0].Role, "member")
	requireEqual(t, body.Items[0].MemberSince, ts(testNow))
}

// ---- remove team member -----------------------------------------------------

func TestRemoveTeamMemberHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.teamMembers.remove = func(_ context.Context, orgID, teamID, userID uuid.UUID) error {
		return nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/teams/"+testTeamID.String()+"/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
}

func TestRemoveTeamMemberNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.teamMembers.remove = func(_ context.Context, orgID, teamID, userID uuid.UUID) error {
		return repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/teams/"+testTeamID.String()+"/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestRemoveTeamMemberForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/teams/"+testTeamID.String()+"/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}
