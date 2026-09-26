package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// Authentication and authorization: the role-to-permission matrix, JWT and
// API key credential resolution, and the permission gates that deny
// underprivileged actors before any repository call.

func TestRequirePermissionMatrix(t *testing.T) {
	perms := []string{
		domain.PermissionOrgManage, domain.PermissionOrgRead,
		domain.PermissionMembersManage, domain.PermissionMembersRead,
		domain.PermissionCredentialsManage, domain.PermissionCredentialsRead,
		domain.PermissionBillingManage, domain.PermissionUsageRead,
	}
	expected := map[string][]bool{
		domain.RoleOwner:        {true, true, true, true, true, true, true, true},
		domain.RoleAdmin:        {false, true, true, true, true, true, false, true},
		domain.RoleAuditor:      {false, true, false, true, false, true, false, true},
		domain.RoleBillingAdmin: {false, true, false, true, false, false, true, true},
		domain.RoleMember:       {false, true, false, true, false, false, false, true},
		domain.RoleService:      {false, true, false, false, false, false, false, true},
		"unknown":               {false, false, false, false, false, false, false, false},
	}
	svc, _ := newTestService()
	for role, want := range expected {
		actor := domain.Actor{Kind: domain.ActorKindUser, OrganizationID: testOrgID, Role: role}
		for i, perm := range perms {
			err := svc.RequirePermission(actor, perm)
			if want[i] && err != nil {
				t.Errorf("role %q: %s = %v, want nil", role, perm, err)
			}
			if !want[i] && !errors.Is(err, repositories.ErrForbidden) {
				t.Errorf("role %q: %s = %v, want ErrForbidden", role, perm, err)
			}
		}
	}
}

func TestAuthenticateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("materializes a user actor", func(t *testing.T) {
		svc, st := newTestService()
		st.verifier.verify = func(_ context.Context, token string) (domain.TokenClaims, error) {
			requireEqual(t, token, "raw-jwt")
			return domain.TokenClaims{
				Subject:        testUserID.String(),
				Email:          "alice@example.com",
				DisplayName:    "Alice",
				OrganizationID: testOrgID,
				Role:           domain.RoleAdmin,
				Kind:           domain.TokenKindUser,
			}, nil
		}
		actor, err := svc.AuthenticateToken(ctx, "raw-jwt")
		requireNoErr(t, err)
		requireEqual(t, actor.Kind, domain.ActorKindUser)
		requireEqual(t, actor.UserID, testUserID)
		requireEqual(t, actor.OrganizationID, testOrgID)
		requireEqual(t, actor.Role, domain.RoleAdmin)
		requireEqual(t, actor.DisplayName, "Alice")
	})

	t.Run("materializes a service actor from service-kind claims", func(t *testing.T) {
		svc, st := newTestService()
		st.verifier.verify = func(_ context.Context, _ string) (domain.TokenClaims, error) {
			return domain.TokenClaims{
				Subject:        testAcctID.String(),
				DisplayName:    "ci-bot",
				OrganizationID: testOrgID,
				Role:           domain.RoleService,
				Kind:           domain.TokenKindService,
			}, nil
		}
		actor, err := svc.AuthenticateToken(ctx, "raw-jwt")
		requireNoErr(t, err)
		requireEqual(t, actor.Kind, domain.ActorKindService)
		requireEqual(t, actor.UserID, testAcctID)
	})

	t.Run("maps any verifier failure to unauthorized", func(t *testing.T) {
		svc, st := newTestService()
		st.verifier.verify = func(_ context.Context, _ string) (domain.TokenClaims, error) {
			return domain.TokenClaims{}, errors.New("expired")
		}
		_, err := svc.AuthenticateToken(ctx, "raw-jwt")
		requireErrIs(t, err, ErrUnauthorized)
	})
}

func TestAuthenticateAPIKey(t *testing.T) {
	ctx := context.Background()
	active := domain.APICredential{
		OrganizationID:   testOrgID,
		Scope:            string(domain.APICredentialScopeControl),
		Status:           domain.APICredentialStatusActive,
		ServiceAccountID: &testAcctID,
	}
	account := domain.ServiceAccount{
		ID:             testAcctID,
		OrganizationID: testOrgID,
		Name:           "ci-bot",
		Status:         domain.ServiceAccountStatusActive,
	}
	boom := errors.New("boom")
	hash := func(_ context.Context, secret string) (string, error) {
		requireEqual(t, secret, "aigw_rawkey")
		return "hash:" + secret, nil
	}
	cases := []struct {
		name string
		stub func(*testStubs)
		want error
	}{
		{"happy path maps to a service actor", func(st *testStubs) {
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, keyHash string) (domain.APICredential, error) {
				requireEqual(t, keyHash, "hash:aigw_rawkey")
				return active, nil
			}
			st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
				requireEqual(t, orgID, testOrgID)
				requireEqual(t, id, testAcctID)
				return account, nil
			}
		}, nil},
		{"hasher failures propagate", func(st *testStubs) {
			st.hasher.hash = func(_ context.Context, _ string) (string, error) { return "", boom }
		}, boom},
		{"unknown key is unauthorized", func(st *testStubs) {
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) {
				return domain.APICredential{}, repositories.ErrNotFound
			}
		}, ErrUnauthorized},
		{"lookup failures pass through", func(st *testStubs) {
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) {
				return domain.APICredential{}, boom
			}
		}, boom},
		{"revoked credential is unauthorized", func(st *testStubs) {
			revoked := active
			revoked.Status = domain.APICredentialStatusRevoked
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) { return revoked, nil }
		}, ErrUnauthorized},
		{"gateway-scoped key is unauthorized", func(st *testStubs) {
			gateway := active
			gateway.Scope = string(domain.APICredentialScopeGateway)
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) { return gateway, nil }
		}, ErrUnauthorized},
		{"key without a service account is unauthorized", func(st *testStubs) {
			orphan := active
			orphan.ServiceAccountID = nil
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) { return orphan, nil }
		}, ErrUnauthorized},
		{"missing service account is unauthorized", func(st *testStubs) {
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) { return active, nil }
			st.accounts.get = func(_ context.Context, _, _ uuid.UUID) (domain.ServiceAccount, error) {
				return domain.ServiceAccount{}, repositories.ErrNotFound
			}
		}, ErrUnauthorized},
		{"revoked service account is unauthorized", func(st *testStubs) {
			revoked := account
			revoked.Status = domain.ServiceAccountStatusRevoked
			st.hasher.hash = hash
			st.credentials.resolveByHash = func(_ context.Context, _ string) (domain.APICredential, error) { return active, nil }
			st.accounts.get = func(_ context.Context, _, _ uuid.UUID) (domain.ServiceAccount, error) { return revoked, nil }
		}, ErrUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, st := newTestService()
			c.stub(st)
			actor, err := svc.AuthenticateAPIKey(ctx, "aigw_rawkey")
			if c.want != nil {
				requireErrIs(t, err, c.want)
				return
			}
			requireNoErr(t, err)
			requireEqual(t, actor, domain.ServiceActor(testOrgID, testAcctID, "ci-bot"))
		})
	}
}

func TestMemberActorManageOpsDenied(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	member := userActor(domain.RoleMember)
	ops := []struct {
		name string
		run  func() error
	}{
		{"Invite", func() error { _, _, err := svc.Invite(ctx, member, "x@y.z", domain.RoleMember); return err }},
		{"RevokeInvitation", func() error { return svc.RevokeInvitation(ctx, member, testInviteID) }},
		{"SetMemberRole", func() error { _, err := svc.SetMemberRole(ctx, member, testOtherID, domain.RoleMember); return err }},
		{"RemoveMember", func() error { return svc.RemoveMember(ctx, member, testOtherID) }},
		{"CreateProject", func() error { _, err := svc.CreateProject(ctx, member, domain.NewProject{Name: "p"}); return err }},
		{"CreateTeam", func() error { _, err := svc.CreateTeam(ctx, member, domain.NewTeam{Name: "t"}); return err }},
		{"AddTeamMember", func() error {
			_, err := svc.AddTeamMember(ctx, member, testTeamID, testOtherID, domain.RoleMember)
			return err
		}},
		{"RemoveTeamMember", func() error { return svc.RemoveTeamMember(ctx, member, testTeamID, testOtherID) }},
		{"AddProjectMember", func() error {
			_, err := svc.AddProjectMember(ctx, member, testProjID, testOtherID, domain.RoleMember)
			return err
		}},
		{"RemoveProjectMember", func() error { return svc.RemoveProjectMember(ctx, member, testProjID, testOtherID) }},
		{"CreateServiceAccount", func() error {
			_, err := svc.CreateServiceAccount(ctx, member, domain.NewServiceAccount{Name: "sa"})
			return err
		}},
		{"RevokeServiceAccount", func() error { return svc.RevokeServiceAccount(ctx, member, testAcctID) }},
		{"CreateCredential", func() error {
			_, _, err := svc.CreateCredential(ctx, member, "k", string(domain.APICredentialScopeGateway), nil)
			return err
		}},
		{"RevokeCredential", func() error { return svc.RevokeCredential(ctx, member, testOtherID) }},
	}
	for _, op := range ops {
		if err := op.run(); !errors.Is(err, repositories.ErrForbidden) {
			t.Errorf("%s: err = %v, want ErrForbidden", op.name, err)
		}
	}
}

func TestServiceActorReadDenied(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	svcActor := testServiceActor()
	reads := []struct {
		name string
		run  func() error
	}{
		{"ListMembers", func() error { _, err := svc.ListMembers(ctx, svcActor); return err }},
		{"ListInvitations", func() error { _, err := svc.ListInvitations(ctx, svcActor); return err }},
		{"ListProjects", func() error { _, err := svc.ListProjects(ctx, svcActor); return err }},
		{"ListTeams", func() error { _, err := svc.ListTeams(ctx, svcActor); return err }},
		{"ListCredentials", func() error { _, err := svc.ListCredentials(ctx, svcActor); return err }},
		{"ListServiceAccounts", func() error { _, err := svc.ListServiceAccounts(ctx, svcActor); return err }},
	}
	for _, read := range reads {
		if err := read.run(); !errors.Is(err, repositories.ErrForbidden) {
			t.Errorf("%s: err = %v, want ErrForbidden", read.name, err)
		}
	}
}
