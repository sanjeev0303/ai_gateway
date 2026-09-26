package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- create project ------------------------------------------------------

func TestCreateProjectHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var captured *domain.NewProject
	st.projects.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProject) (domain.Project, error) {
		captured = &domain.NewProject{Name: in.Name, Slug: in.Slug, Description: in.Description}
		return testProject, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects", map[string]any{
		"name":        "gateway",
		"slug":        "gateway",
		"description": "the gateway",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body projectDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ID, testProjID.String())
	requireEqual(t, body.OrganizationID, testOrgID.String())
	requireEqual(t, body.Name, "gateway")
	requireEqual(t, body.Slug, "gateway")
	requireEqual(t, body.Description, "the gateway")
	requireEqual(t, body.Status, domain.ProjectStatusActive)
	requireEqual(t, body.CreatedAt, ts(testNow))

	requireEqual(t, captured.Name, "gateway")
	requireEqual(t, captured.Slug, "gateway")
	requireEqual(t, captured.Description, "the gateway")
	requireAudit(t, st, "project.create")
}

func TestCreateProjectRejectsBlankName(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects", map[string]any{
		"name": "   ",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateProjectRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects", map[string]any{
		"name": 42,
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

func TestCreateProjectForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects", map[string]any{
		"name": "gateway",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- list projects --------------------------------------------------------

func TestListProjectsHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.projects.list = func(_ context.Context, orgID uuid.UUID) ([]domain.Project, error) {
		return []domain.Project{testProject}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/projects", nil, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []projectDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ID, testProjID.String())
	requireEqual(t, body.Items[0].Name, "gateway")
	requireEqual(t, body.Items[0].Slug, "gateway")
}

func TestListProjectsStoreFailure(t *testing.T) {
	srv, st := newTestServer(t)

	st.projects.list = func(_ context.Context, orgID uuid.UUID) ([]domain.Project, error) {
		return nil, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/projects", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

// ---- get project -----------------------------------------------------------

func TestGetProjectHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.projects.get = func(_ context.Context, orgID, id uuid.UUID) (domain.Project, error) {
		return testProject, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/projects/"+testProjID.String(), nil, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body projectDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ID, testProjID.String())
	requireEqual(t, body.Name, "gateway")
}

func TestGetProjectNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.projects.get = func(_ context.Context, orgID, id uuid.UUID) (domain.Project, error) {
		return domain.Project{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/projects/"+testProjID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestGetProjectRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodGet, "/v1/projects/not-a-uuid", nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

// ---- add project member ----------------------------------------------------

func TestAddProjectMemberHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var addedProjID, addedUserID uuid.UUID
	var addedRole string
	st.projMembers.add = func(_ context.Context, orgID, projectID, userID uuid.UUID, role string) (domain.ProjectMember, error) {
		addedProjID = projectID
		addedUserID = userID
		addedRole = role
		return domain.ProjectMember{
			ProjectID: projectID, OrganizationID: orgID, UserID: userID, Role: role, CreatedAt: testNow,
		}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects/"+testProjID.String()+"/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "member",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body projectMemberRowDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ProjectID, testProjID.String())
	requireEqual(t, body.UserID, testOtherID.String())
	requireEqual(t, body.Role, "member")
	requireEqual(t, body.MemberSince, ts(testNow))

	requireEqual(t, addedProjID, testProjID)
	requireEqual(t, addedUserID, testOtherID)
	requireEqual(t, addedRole, "member")
	requireAudit(t, st, "project_member.add")
}

func TestAddProjectMemberRejectsOwnerRole(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects/"+testProjID.String()+"/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "owner",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestAddProjectMemberRejectsInvalidUserID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects/"+testProjID.String()+"/members", map[string]any{
		"user_id": "not-a-uuid",
		"role":    "member",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestAddProjectMemberRejectsInvalidProjectID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects/not-a-uuid/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "member",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestAddProjectMemberForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/projects/"+testProjID.String()+"/members", map[string]any{
		"user_id": testOtherID.String(),
		"role":    "member",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- list project members --------------------------------------------------

func TestListProjectMembersHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.projMembers.list = func(_ context.Context, orgID, projectID uuid.UUID) ([]domain.ProjectMemberView, error) {
		return []domain.ProjectMemberView{{User: testUser, Role: "member", MemberSince: testNow}}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/projects/"+testProjID.String()+"/members", nil, bearerHeader(st, domain.RoleMember))

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

// ---- remove project member -------------------------------------------------

func TestRemoveProjectMemberHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.projMembers.remove = func(_ context.Context, orgID, projectID, userID uuid.UUID) error {
		return nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/projects/"+testProjID.String()+"/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
}

func TestRemoveProjectMemberNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.projMembers.remove = func(_ context.Context, orgID, projectID, userID uuid.UUID) error {
		return repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/projects/"+testProjID.String()+"/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestRemoveProjectMemberForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/projects/"+testProjID.String()+"/members/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}
