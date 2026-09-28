package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

var testGatewayCredentialID = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

func testGatewayCredential() domain.GatewayCredential {
	return domain.GatewayCredential{
		ID:             testGatewayCredentialID,
		OrganizationID: testOrgID,
		Name:           "ci-gateway",
		KeyPrefix:      "gk_live_abcd",
		SubjectType:    domain.SubjectTypeServiceAccount,
		SubjectID:      testAcctID,
		Scopes:         []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
		Status:         domain.GatewayCredentialStatusActive,
	}
}

// TestNewTestServiceWiresGatewayCredentials guards the regression where
// ServiceDeps.GatewayCredentials was left unset, leaving the live
// GET /v1/gateway-credentials route dereferencing a nil repository.
// Every other required dependency is wired here, so an omission stands out.
func TestNewTestServiceWiresGatewayCredentials(t *testing.T) {
	svc, _ := newTestService()
	if svc.gatewayCredentials == nil {
		t.Fatal("newTestService: GatewayCredentials is nil; gateway-credential routes would panic")
	}
}

func TestListGatewayCredentials(t *testing.T) {
	svc, st := newTestService()
	want := []domain.GatewayCredential{testGatewayCredential()}

	var gotOrg uuid.UUID
	st.gatewayCreds.list = func(_ context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error) {
		gotOrg = orgID
		return want, nil
	}

	got, err := svc.ListGatewayCredentials(context.Background(), userActor(domain.RoleAdmin))
	if err != nil {
		t.Fatalf("ListGatewayCredentials: %v", err)
	}
	if gotOrg != testOrgID {
		t.Errorf("repo org = %v, want %v", gotOrg, testOrgID)
	}
	if len(got) != 1 || got[0].ID != want[0].ID {
		t.Fatalf("got %+v, want 1 credential %v", got, want[0].ID)
	}
}

func TestListGatewayCredentialsPropagatesRepoError(t *testing.T) {
	svc, st := newTestService()
	sentinel := errors.New("db down")
	st.gatewayCreds.list = func(context.Context, uuid.UUID) ([]domain.GatewayCredential, error) {
		return nil, sentinel
	}

	if _, err := svc.ListGatewayCredentials(context.Background(), userActor(domain.RoleAdmin)); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
}

func TestListGatewayCredentialsRequiresPermission(t *testing.T) {
	svc, st := newTestService()
	// A member has no credentials.read; the permission check must reject
	// before the repository is reached.
	st.gatewayCreds.list = func(context.Context, uuid.UUID) ([]domain.GatewayCredential, error) {
		t.Error("repository must not be reached without credentials.read")
		return nil, nil
	}

	if _, err := svc.ListGatewayCredentials(context.Background(), userActor(domain.RoleMember)); err == nil {
		t.Fatal("member: want permission error, got nil")
	}
}

func TestGetGatewayCredential(t *testing.T) {
	svc, st := newTestService()
	want := testGatewayCredential()

	var gotOrg, gotID uuid.UUID
	st.gatewayCreds.get = func(_ context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
		gotOrg, gotID = orgID, id
		return want, nil
	}

	got, err := svc.GetGatewayCredential(context.Background(), userActor(domain.RoleAdmin), testGatewayCredentialID)
	if err != nil {
		t.Fatalf("GetGatewayCredential: %v", err)
	}
	if gotOrg != testOrgID || gotID != testGatewayCredentialID {
		t.Errorf("repo args = (%v, %v), want (%v, %v)", gotOrg, gotID, testOrgID, testGatewayCredentialID)
	}
	if got.ID != want.ID {
		t.Errorf("got ID %v, want %v", got.ID, want.ID)
	}
}

func TestRevokeGatewayCredential(t *testing.T) {
	svc, st := newTestService()

	var gotOrg, gotID uuid.UUID
	st.gatewayCreds.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
		gotOrg, gotID = orgID, id
		revoked := testGatewayCredential()
		revoked.Status = domain.GatewayCredentialStatusRevoked
		return revoked, nil
	}

	if err := svc.RevokeGatewayCredential(context.Background(), userActor(domain.RoleAdmin), testGatewayCredentialID); err != nil {
		t.Fatalf("RevokeGatewayCredential: %v", err)
	}
	if gotOrg != testOrgID || gotID != testGatewayCredentialID {
		t.Errorf("repo args = (%v, %v), want (%v, %v)", gotOrg, gotID, testOrgID, testGatewayCredentialID)
	}
	if !containsAction(st.audits.actions, "gateway_credential.revoke") {
		t.Errorf("audits = %v, want a gateway_credential.revoke entry", st.audits.actions)
	}
}

func TestRevokeGatewayCredentialRequiresManage(t *testing.T) {
	svc, st := newTestService()
	// An auditor holds credentials.read but not credentials.manage.
	st.gatewayCreds.revoke = func(context.Context, uuid.UUID, uuid.UUID) (domain.GatewayCredential, error) {
		t.Error("repository must not be reached without credentials.manage")
		return domain.GatewayCredential{}, nil
	}

	err := svc.RevokeGatewayCredential(context.Background(), userActor(domain.RoleAuditor), testGatewayCredentialID)
	if err == nil {
		t.Fatal("auditor: want permission error, got nil")
	}
}

func TestGatewayCredentialMethodsRequireCredentialPermissions(t *testing.T) {
	// The read-only member role must be rejected by every gateway-credential
	// entry point that gates on a credential permission.
	svc, _ := newTestService()
	ctx := context.Background()
	actor := userActor(domain.RoleMember)

	cases := []struct {
		name string
		call func() error
	}{
		{"list", func() error {
			_, err := svc.ListGatewayCredentials(ctx, actor)
			return err
		}},
		{"get", func() error {
			_, err := svc.GetGatewayCredential(ctx, actor, testGatewayCredentialID)
			return err
		}},
		{"revoke", func() error {
			return svc.RevokeGatewayCredential(ctx, actor, testGatewayCredentialID)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err == nil {
				t.Fatalf("%s: want permission error for member, got nil", tc.name)
			}
		})
	}
}

func containsAction(actions []string, want string) bool {
	for _, a := range actions {
		if a == want {
			return true
		}
	}
	return false
}
