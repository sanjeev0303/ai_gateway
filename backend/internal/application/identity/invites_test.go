package identity

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// Invitation lifecycle: who may invite for which roles, the single-visible
// raw token, and the acceptance guards (email match, status, expiry, and the
// concurrent-accept race).

func TestInviteValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		actor domain.Actor
		email string
		role  string
		want  error
	}{
		{"member lacks manage permission", userActor(domain.RoleMember), "new@example.com", domain.RoleMember, repositories.ErrForbidden},
		{"admin cannot grant admin", userActor(domain.RoleAdmin), "new@example.com", domain.RoleAdmin, repositories.ErrForbidden},
		{"service actor is forbidden", testServiceActor(), "new@example.com", domain.RoleMember, repositories.ErrForbidden},
		{"blank email is invalid", userActor(domain.RoleOwner), "  ", domain.RoleMember, repositories.ErrInvalid},
		{"non-org role is invalid", userActor(domain.RoleOwner), "new@example.com", domain.RoleService, repositories.ErrInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, _ := newTestService()
			_, _, err := svc.Invite(ctx, c.actor, c.email, c.role)
			requireErrIs(t, err, c.want)
		})
	}
}

func TestInviteHappyPath(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	var got domain.NewInvitation
	st.hasher.hash = func(_ context.Context, secret string) (string, error) { return "hash:" + secret, nil }
	st.invitations.create = func(_ context.Context, orgID uuid.UUID, in domain.NewInvitation) (domain.Invitation, error) {
		requireEqual(t, orgID, testOrgID)
		got = in
		return domain.Invitation{
			ID:             testInviteID,
			OrganizationID: testOrgID,
			Email:          in.Email,
			Role:           in.Role,
			TokenHash:      in.TokenHash,
			Status:         domain.InvitationStatusPending,
			ExpiresAt:      in.Expires,
		}, nil
	}
	invitation, raw, err := svc.Invite(ctx, userActor(domain.RoleOwner), "  NEW@Example.COM \n", domain.RoleMember)
	requireNoErr(t, err)
	requireEqual(t, invitation.ID, testInviteID)
	requireEqual(t, got.Email, "new@example.com")
	requireEqual(t, got.Role, domain.RoleMember)
	requireEqual(t, got.Expires, testNow.Add(testInviteTTL))
	requireEqual(t, got.TokenHash, "hash:"+raw)
	if got.InvitedBy == nil || *got.InvitedBy != testUserID {
		t.Fatalf("InvitedBy = %v, want %v", got.InvitedBy, testUserID)
	}
	if len(raw) != 64 {
		t.Errorf("raw token length = %d, want 64", len(raw))
	}
	if _, err := hex.DecodeString(raw); err != nil {
		t.Errorf("raw token %q is not hex: %v", raw, err)
	}
	requireAudit(t, st, "invitation.create")
}

func TestInviteAuditFailureIsBestEffort(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.hasher.hash = func(_ context.Context, secret string) (string, error) { return "hash:" + secret, nil }
	st.invitations.create = func(_ context.Context, _ uuid.UUID, in domain.NewInvitation) (domain.Invitation, error) {
		return domain.Invitation{ID: testInviteID, OrganizationID: testOrgID, Status: domain.InvitationStatusPending, ExpiresAt: in.Expires}, nil
	}
	st.audits.appendFn = func(_ context.Context, _ uuid.UUID, _ domain.NewAuditLog) (domain.AuditLog, error) {
		return domain.AuditLog{}, errors.New("audit down")
	}
	_, _, err := svc.Invite(ctx, userActor(domain.RoleOwner), "new@example.com", domain.RoleMember)
	requireNoErr(t, err)
}

func TestAcceptInvitationValidation(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("boom")
	pending := domain.Invitation{
		ID:             testInviteID,
		OrganizationID: testOrgID,
		Email:          "new@example.com",
		Role:           domain.RoleMember,
		Status:         domain.InvitationStatusPending,
		ExpiresAt:      testNow.Add(time.Hour),
	}
	matchingUser := func(_ context.Context, _ uuid.UUID) (domain.User, error) {
		return domain.User{ID: testUserID, Email: "new@example.com"}, nil
	}
	hash := func(_ context.Context, _ string) (string, error) { return "h", nil }
	cases := []struct {
		name  string
		actor domain.Actor
		stub  func(*testStubs)
		want  error
	}{
		{"service actor is forbidden", testServiceActor(), nil, repositories.ErrForbidden},
		{"hasher failures propagate", userActor(domain.RoleMember), func(st *testStubs) {
			st.hasher.hash = func(_ context.Context, _ string) (string, error) { return "", boom }
		}, boom},
		{"unknown token passes through not found", userActor(domain.RoleMember), func(st *testStubs) {
			st.hasher.hash = hash
			st.invitations.resolveByHash = func(_ context.Context, _ string) (domain.Invitation, error) {
				return domain.Invitation{}, repositories.ErrNotFound
			}
		}, repositories.ErrNotFound},
		{"email mismatch is rejected", userActor(domain.RoleMember), func(st *testStubs) {
			st.hasher.hash = hash
			st.invitations.resolveByHash = func(_ context.Context, _ string) (domain.Invitation, error) { return pending, nil }
			st.users.getByID = func(_ context.Context, _ uuid.UUID) (domain.User, error) {
				return domain.User{ID: testUserID, Email: "other@example.com"}, nil
			}
		}, ErrInvitationEmailMismatch},
		{"non-pending invitation is invalid", userActor(domain.RoleMember), func(st *testStubs) {
			revoked := pending
			revoked.Status = domain.InvitationStatusRevoked
			st.hasher.hash = hash
			st.invitations.resolveByHash = func(_ context.Context, _ string) (domain.Invitation, error) { return revoked, nil }
			st.users.getByID = matchingUser
			st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
				return domain.OrgMember{}, repositories.ErrNotFound
			}
		}, ErrInvalidInvitation},
		{"expired invitation is rejected", userActor(domain.RoleMember), func(st *testStubs) {
			stale := pending
			stale.ExpiresAt = testNow.Add(-time.Hour)
			st.hasher.hash = hash
			st.invitations.resolveByHash = func(_ context.Context, _ string) (domain.Invitation, error) { return stale, nil }
			st.users.getByID = matchingUser
			st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
				return domain.OrgMember{}, repositories.ErrNotFound
			}
		}, ErrInvitationExpired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, st := newTestService()
			if c.stub != nil {
				c.stub(st)
			}
			_, err := svc.AcceptInvitation(ctx, c.actor, "raw-token")
			requireErrIs(t, err, c.want)
		})
	}
}

func TestAcceptInvitationAlreadyMember(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.hasher.hash = func(_ context.Context, _ string) (string, error) { return "h", nil }
	st.invitations.resolveByHash = func(_ context.Context, _ string) (domain.Invitation, error) {
		return domain.Invitation{ID: testInviteID, OrganizationID: testOrgID, Email: "new@example.com", Status: domain.InvitationStatusPending, ExpiresAt: testNow.Add(time.Hour)}, nil
	}
	st.users.getByID = func(_ context.Context, _ uuid.UUID) (domain.User, error) {
		return domain.User{ID: testUserID, Email: "new@example.com"}, nil
	}
	want := domain.OrgMember{User: domain.User{ID: testUserID, Email: "new@example.com"}, Role: domain.RoleMember}
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) { return want, nil }
	got, err := svc.AcceptInvitation(ctx, userActor(domain.RoleMember), "raw-token")
	requireNoErr(t, err)
	requireEqual(t, got.Role, domain.RoleMember)
	if len(st.audits.actions) != 0 {
		t.Errorf("unexpected audits for an existing member: %v", st.audits.actions)
	}
}

func TestAcceptInvitationHappyPath(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testUserID, Email: "new@example.com", DisplayName: "New"}
	pending := domain.Invitation{
		ID: testInviteID, OrganizationID: testOrgID, Email: "new@example.com",
		Role: domain.RoleMember, Status: domain.InvitationStatusPending, ExpiresAt: testNow.Add(time.Hour),
	}
	st.hasher.hash = func(_ context.Context, secret string) (string, error) { return "hash:" + secret, nil }
	st.invitations.resolveByHash = func(_ context.Context, hash string) (domain.Invitation, error) {
		requireEqual(t, hash, "hash:raw-token")
		return pending, nil
	}
	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		requireEqual(t, id, testUserID)
		return user, nil
	}
	membersGet := 0
	st.members.get = func(_ context.Context, orgID, memberID uuid.UUID) (domain.OrgMember, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, memberID, testUserID)
		membersGet++
		if membersGet == 1 {
			return domain.OrgMember{}, repositories.ErrNotFound
		}
		return domain.OrgMember{User: user, Role: domain.RoleMember}, nil
	}
	st.members.add = func(_ context.Context, orgID, memberID uuid.UUID, role string) (domain.OrganizationMember, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, memberID, testUserID)
		requireEqual(t, role, domain.RoleMember)
		return domain.OrganizationMember{OrganizationID: orgID, UserID: memberID, Role: role}, nil
	}
	accepted := false
	st.invitations.markAccepted = func(_ context.Context, orgID, id, acceptedBy uuid.UUID) (domain.Invitation, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testInviteID)
		requireEqual(t, acceptedBy, testUserID)
		accepted = true
		return pending, nil
	}
	got, err := svc.AcceptInvitation(ctx, userActor(domain.RoleMember), "raw-token")
	requireNoErr(t, err)
	requireEqual(t, got.User.Email, "new@example.com")
	if !accepted {
		t.Error("MarkAccepted was not called")
	}
	requireAudit(t, st, "invitation.accept")
}

func TestAcceptInvitationConcurrentAcceptRace(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testUserID, Email: "new@example.com"}
	st.hasher.hash = func(_ context.Context, _ string) (string, error) { return "h", nil }
	st.invitations.resolveByHash = func(_ context.Context, _ string) (domain.Invitation, error) {
		return domain.Invitation{ID: testInviteID, OrganizationID: testOrgID, Email: "new@example.com", Role: domain.RoleMember, Status: domain.InvitationStatusPending, ExpiresAt: testNow.Add(time.Hour)}, nil
	}
	st.users.getByID = func(_ context.Context, _ uuid.UUID) (domain.User, error) { return user, nil }
	membersGet := 0
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
		membersGet++
		if membersGet == 1 {
			return domain.OrgMember{}, repositories.ErrNotFound
		}
		return domain.OrgMember{User: user, Role: domain.RoleMember}, nil
	}
	st.members.add = func(_ context.Context, _, _ uuid.UUID, _ string) (domain.OrganizationMember, error) {
		return domain.OrganizationMember{}, repositories.ErrConflict
	}
	got, err := svc.AcceptInvitation(ctx, userActor(domain.RoleMember), "raw-token")
	requireNoErr(t, err)
	requireEqual(t, got.Role, domain.RoleMember)
	// The race-return path skips MarkAccepted and the audit.
	if len(st.audits.actions) != 0 {
		t.Errorf("unexpected audits on a lost accept race: %v", st.audits.actions)
	}
}
