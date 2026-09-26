package controlapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- invite --------------------------------------------------------------

func TestInviteHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var capturedNew domain.NewInvitation
	var capturedRaw string
	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		capturedRaw = raw
		return "hash:" + raw, nil
	}
	st.invitations.create = func(_ context.Context, orgID uuid.UUID, in domain.NewInvitation) (domain.Invitation, error) {
		capturedNew = in
		return domain.Invitation{
			ID:             testInviteID,
			OrganizationID: orgID,
			Email:          in.Email,
			Role:           in.Role,
			TokenHash:      in.TokenHash,
			Status:         domain.InvitationStatusPending,
			ExpiresAt:      in.Expires,
			InvitedBy:      in.InvitedBy,
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "  carol@example.com ",
		"role":  "member",
	}, bearerHeader(st, domain.RoleOwner))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body inviteResponse
	decodeBody(t, resp, &body)

	requireEqual(t, len(capturedRaw), 64)
	requireEqual(t, body.Token, capturedRaw)
	requireEqual(t, body.Invitation.OrganizationID, testOrgID.String())
	requireEqual(t, body.Invitation.Email, "carol@example.com")
	requireEqual(t, body.Invitation.Role, "member")
	requireEqual(t, body.Invitation.Status, domain.InvitationStatusPending)
	requireEqual(t, body.Invitation.ExpiresAt, ts(testNow.Add(testInviteTTL)))
	if body.Invitation.InvitedBy == nil || *body.Invitation.InvitedBy != testUserID.String() {
		t.Fatalf("InvitedBy = %v, want %s", body.Invitation.InvitedBy, testUserID.String())
	}

	requireEqual(t, capturedNew.Email, "carol@example.com")
	requireEqual(t, capturedNew.Role, "member")
	requireEqual(t, capturedNew.TokenHash, "hash:"+capturedRaw)
	requireEqual(t, capturedNew.Expires, testNow.Add(testInviteTTL))
	if capturedNew.InvitedBy == nil || *capturedNew.InvitedBy != testUserID {
		t.Fatalf("NewInvitation.InvitedBy = %v, want %s", capturedNew.InvitedBy, testUserID)
	}
	requireAudit(t, st, "invitation.create")
}

func TestInviteRejectsBlankEmail(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "   ",
		"role":  "member",
	}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
	if st.hasher.hash != nil {
		t.Fatal("hasher.hash was installed; invite should fail before hashing")
	}
}

func TestInviteRejectsInvalidRole(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "carol@example.com",
		"role":  "supreme-leader",
	}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
	if st.hasher.hash != nil {
		t.Fatal("hasher.hash was installed; invite should fail before hashing")
	}
}

func TestInviteRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "carol@example.com",
		"role":  42,
	}, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

func TestInviteForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "carol@example.com",
		"role":  "member",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

func TestInviteForbiddenForAdminInvitingOwner(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "carol@example.com",
		"role":  "owner",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
	if st.hasher.hash != nil {
		t.Fatal("hasher.hash was installed; invite should fail before hashing")
	}
}

func TestInviteForbiddenForAdminInvitingAdmin(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", map[string]any{
		"email": "carol@example.com",
		"role":  "admin",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- accept invitation ----------------------------------------------------

func TestAcceptInvitationHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	memberGetCalls := 0
	var addedUserID uuid.UUID
	var addedRole string
	var acceptedBy uuid.UUID
	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		return "hash:" + raw, nil
	}
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		return testInvitation, nil
	}
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		return testUser, nil
	}
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		memberGetCalls++
		if memberGetCalls == 1 {
			return domain.OrgMember{}, repositories.ErrNotFound
		}
		return testMember(domain.RoleMember), nil
	}
	st.members.add = func(_ context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
		addedUserID = userID
		addedRole = role
		return domain.OrganizationMember{OrganizationID: orgID, UserID: userID, Role: role, CreatedAt: testNow}, nil
	}
	st.invitations.markAccepted = func(_ context.Context, orgID, id, by uuid.UUID) (domain.Invitation, error) {
		acceptedBy = by
		return testInvitation, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "raw-tok",
	}, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body orgMemberDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.User.ID, testUserID.String())
	requireEqual(t, body.Role, "member")
	requireEqual(t, body.MemberSince, ts(testNow))

	requireEqual(t, memberGetCalls, 2)
	requireEqual(t, addedUserID, testUserID)
	requireEqual(t, addedRole, "member")
	requireEqual(t, acceptedBy, testUserID)
	requireAudit(t, st, "invitation.accept")
}

func TestAcceptInvitationAlreadyMember(t *testing.T) {
	srv, st := newTestServer(t)

	markAcceptedCalled := false
	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		return "hash:" + raw, nil
	}
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		return testInvitation, nil
	}
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		return testUser, nil
	}
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return testMember(domain.RoleMember), nil
	}
	st.invitations.markAccepted = func(_ context.Context, orgID, id, by uuid.UUID) (domain.Invitation, error) {
		markAcceptedCalled = true
		return testInvitation, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "raw-tok",
	}, bearerHeader(st, domain.RoleMember))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	requireEqual(t, markAcceptedCalled, false)
	requireEqual(t, len(st.audits.actions), 0)
}

func TestAcceptInvitationRejectsEmailMismatch(t *testing.T) {
	srv, st := newTestServer(t)

	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		return "hash:" + raw, nil
	}
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		return testInvitation, nil
	}
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		return testOtherUser, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "raw-tok",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invitation_email_mismatch")
}

func TestAcceptInvitationRejectsExpired(t *testing.T) {
	srv, st := newTestServer(t)

	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		return "hash:" + raw, nil
	}
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		inv := testInvitation
		inv.ExpiresAt = testNow.Add(-1 * time.Hour)
		return inv, nil
	}
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		return testUser, nil
	}
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "raw-tok",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invitation_expired")
}

func TestAcceptInvitationRejectsRevoked(t *testing.T) {
	srv, st := newTestServer(t)

	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		return "hash:" + raw, nil
	}
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		inv := testInvitation
		inv.Status = domain.InvitationStatusRevoked
		return inv, nil
	}
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		return testUser, nil
	}
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "raw-tok",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_invitation")
}

func TestAcceptInvitationRejectsUnknownToken(t *testing.T) {
	srv, st := newTestServer(t)

	st.hasher.hash = func(_ context.Context, raw string) (string, error) {
		return "hash:" + raw, nil
	}
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		return domain.Invitation{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "unknown-tok",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestAcceptInvitationForbiddenForServiceActor(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": "raw-tok",
	}, map[string]string{"X-API-Key": apiKeyToken(st, controlCred())})

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
	if st.invitations.resolveByHash != nil {
		t.Fatal("resolveByHash was installed; service should reject before token lookup")
	}
}

func TestAcceptInvitationRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations/accept", map[string]any{
		"token": 42,
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

// ---- list invitations -----------------------------------------------------

func TestListInvitationsHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.invitations.list = func(_ context.Context, orgID uuid.UUID) ([]domain.Invitation, error) {
		return []domain.Invitation{testInvitation}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/invitations", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []invitationDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ID, testInviteID.String())
	requireEqual(t, body.Items[0].OrganizationID, testOrgID.String())
	requireEqual(t, body.Items[0].Email, "alice@example.com")
	requireEqual(t, body.Items[0].Role, "member")
	requireEqual(t, body.Items[0].Status, domain.InvitationStatusPending)
}

func TestListInvitationsStoreFailure(t *testing.T) {
	srv, st := newTestServer(t)

	st.invitations.list = func(_ context.Context, orgID uuid.UUID) ([]domain.Invitation, error) {
		return nil, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/invitations", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

// ---- revoke invitation ----------------------------------------------------

func TestRevokeInvitationHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.invitations.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.Invitation, error) {
		return testInvitation, nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/invitations/"+testInviteID.String(), nil, bearerHeader(st, domain.RoleOwner))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
	requireAudit(t, st, "invitation.revoke")
}

func TestRevokeInvitationNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.invitations.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.Invitation, error) {
		return domain.Invitation{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/invitations/"+testInviteID.String(), nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestRevokeInvitationRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/invitations/not-a-uuid", nil, bearerHeader(st, domain.RoleOwner))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestRevokeInvitationForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/invitations/"+testInviteID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}
