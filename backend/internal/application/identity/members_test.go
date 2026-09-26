package identity

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// Membership management: role changes and removals, including the guards
// that protect a member's own record and the last owner of an organization.

func TestSetMemberRoleValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		actor  domain.Actor
		userID uuid.UUID
		role   string
		stub   func(*testStubs)
		want   error
	}{
		{"member lacks manage permission", userActor(domain.RoleMember), testOtherID, domain.RoleMember, nil, repositories.ErrForbidden},
		{"non-org role is invalid", userActor(domain.RoleAdmin), testOtherID, "superuser", nil, repositories.ErrInvalid},
		{"self changes are forbidden", userActor(domain.RoleAdmin), testUserID, domain.RoleMember, nil, repositories.ErrForbidden},
		{"admin cannot grant admin", userActor(domain.RoleAdmin), testOtherID, domain.RoleAdmin, nil, repositories.ErrForbidden},
		{"target must exist", userActor(domain.RoleAdmin), testOtherID, domain.RoleMember, func(st *testStubs) {
			st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
				return domain.OrgMember{}, repositories.ErrNotFound
			}
		}, repositories.ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, st := newTestService()
			if c.stub != nil {
				c.stub(st)
			}
			_, err := svc.SetMemberRole(ctx, c.actor, c.userID, c.role)
			requireErrIs(t, err, c.want)
		})
	}
}

func TestSetMemberRoleHappyPath(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testOtherID, Email: "bob@example.com", DisplayName: "Bob"}
	membersGet := 0
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
		membersGet++
		if membersGet == 1 {
			return domain.OrgMember{User: user, Role: domain.RoleMember}, nil
		}
		return domain.OrgMember{User: user, Role: domain.RoleAdmin}, nil
	}
	st.members.setRole = func(_ context.Context, orgID, memberID uuid.UUID, role string) (domain.OrganizationMember, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, memberID, testOtherID)
		requireEqual(t, role, domain.RoleAdmin)
		return domain.OrganizationMember{OrganizationID: orgID, UserID: memberID, Role: role}, nil
	}
	got, err := svc.SetMemberRole(ctx, userActor(domain.RoleOwner), testOtherID, domain.RoleAdmin)
	requireNoErr(t, err)
	requireEqual(t, got.Role, domain.RoleAdmin)
	requireAudit(t, st, "member.role.update")
}

func TestSetMemberRoleLastOwnerBlocked(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testOtherID, Email: "bob@example.com"}
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{User: user, Role: domain.RoleOwner}, nil
	}
	st.members.countByRole = func(_ context.Context, _ uuid.UUID, role string) (int64, error) {
		requireEqual(t, role, domain.RoleOwner)
		return 1, nil
	}
	_, err := svc.SetMemberRole(ctx, userActor(domain.RoleOwner), testOtherID, domain.RoleMember)
	requireErrIs(t, err, repositories.ErrForbidden)
}

func TestSetMemberRoleOwnerDemotionWithBackupOwner(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testOtherID, Email: "bob@example.com"}
	membersGet := 0
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
		membersGet++
		if membersGet == 1 {
			return domain.OrgMember{User: user, Role: domain.RoleOwner}, nil
		}
		return domain.OrgMember{User: user, Role: domain.RoleMember}, nil
	}
	st.members.countByRole = func(_ context.Context, _ uuid.UUID, _ string) (int64, error) { return 2, nil }
	st.members.setRole = func(_ context.Context, _, memberID uuid.UUID, role string) (domain.OrganizationMember, error) {
		requireEqual(t, memberID, testOtherID)
		requireEqual(t, role, domain.RoleMember)
		return domain.OrganizationMember{UserID: memberID, Role: role}, nil
	}
	got, err := svc.SetMemberRole(ctx, userActor(domain.RoleOwner), testOtherID, domain.RoleMember)
	requireNoErr(t, err)
	requireEqual(t, got.Role, domain.RoleMember)
	requireAudit(t, st, "member.role.update")
}

func TestRemoveMemberValidation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		actor  domain.Actor
		userID uuid.UUID
		stub   func(*testStubs)
		want   error
	}{
		{"member lacks manage permission", userActor(domain.RoleMember), testOtherID, nil, repositories.ErrForbidden},
		{"self removal is forbidden", userActor(domain.RoleAdmin), testUserID, nil, repositories.ErrForbidden},
		{"target must exist", userActor(domain.RoleAdmin), testOtherID, func(st *testStubs) {
			st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
				return domain.OrgMember{}, repositories.ErrNotFound
			}
		}, repositories.ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, st := newTestService()
			if c.stub != nil {
				c.stub(st)
			}
			err := svc.RemoveMember(ctx, c.actor, c.userID)
			requireErrIs(t, err, c.want)
		})
	}
}

func TestRemoveMemberLastOwnerBlocked(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testOtherID, Email: "bob@example.com"}
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{User: user, Role: domain.RoleOwner}, nil
	}
	st.members.countByRole = func(_ context.Context, _ uuid.UUID, _ string) (int64, error) { return 1, nil }
	err := svc.RemoveMember(ctx, userActor(domain.RoleAdmin), testOtherID)
	requireErrIs(t, err, repositories.ErrForbidden)
}

func TestRemoveMemberHappyPath(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	user := domain.User{ID: testOtherID, Email: "bob@example.com"}
	st.members.get = func(_ context.Context, _, _ uuid.UUID) (domain.OrgMember, error) {
		return domain.OrgMember{User: user, Role: domain.RoleMember}, nil
	}
	removed := false
	st.members.remove = func(_ context.Context, orgID, memberID uuid.UUID) error {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, memberID, testOtherID)
		removed = true
		return nil
	}
	err := svc.RemoveMember(ctx, userActor(domain.RoleAdmin), testOtherID)
	requireNoErr(t, err)
	if !removed {
		t.Error("Remove was not called")
	}
	requireAudit(t, st, "member.remove")
}

func TestListMembers(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	members := []domain.OrgMember{{
		User: domain.User{ID: testUserID, Email: "alice@example.com", DisplayName: "Alice"},
		Role: domain.RoleOwner,
	}}
	st.members.list = func(_ context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
		requireEqual(t, orgID, testOrgID)
		return members, nil
	}
	got, err := svc.ListMembers(ctx, userActor(domain.RoleMember))
	requireNoErr(t, err)
	if len(got) != 1 {
		t.Fatalf("ListMembers returned %d members, want 1", len(got))
	}
	requireEqual(t, got[0].User.ID, testUserID)
	requireEqual(t, got[0].Role, domain.RoleOwner)
}
