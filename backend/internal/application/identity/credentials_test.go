package identity

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// Credential lifecycle: service accounts and API keys, including the
// scope/service-account binding rules and the prefixing of stored keys.

func TestCreateCredentialValidation(t *testing.T) {
	ctx := context.Background()
	saID := testAcctID
	cases := []struct {
		name    string
		actor   domain.Actor
		keyName string
		scope   string
		sa      *uuid.UUID
		stub    func(*testStubs)
		want    error
	}{
		{"member lacks manage permission", userActor(domain.RoleMember), "k", string(domain.APICredentialScopeGateway), nil, nil, repositories.ErrForbidden},
		{"blank name is invalid", userActor(domain.RoleOwner), "  ", string(domain.APICredentialScopeGateway), nil, nil, repositories.ErrInvalid},
		{"unknown scope is invalid", userActor(domain.RoleOwner), "k", "superuser", nil, nil, repositories.ErrInvalid},
		{"control scope requires a service account", userActor(domain.RoleOwner), "k", string(domain.APICredentialScopeControl), nil, nil, repositories.ErrInvalid},
		{"control scope rejects a revoked service account", userActor(domain.RoleOwner), "k", string(domain.APICredentialScopeControl), &saID, func(st *testStubs) {
			st.accounts.get = func(_ context.Context, _, _ uuid.UUID) (domain.ServiceAccount, error) {
				return domain.ServiceAccount{ID: testAcctID, OrganizationID: testOrgID, Status: domain.ServiceAccountStatusRevoked}, nil
			}
		}, repositories.ErrInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, st := newTestService()
			if c.stub != nil {
				c.stub(st)
			}
			_, _, err := svc.CreateCredential(ctx, c.actor, c.keyName, c.scope, c.sa)
			requireErrIs(t, err, c.want)
		})
	}
}

func TestCreateCredentialGatewayHappyPath(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	var got domain.NewAPICredential
	st.hasher.hash = func(_ context.Context, secret string) (string, error) { return "hash:" + secret, nil }
	st.credentials.create = func(_ context.Context, orgID uuid.UUID, in domain.NewAPICredential) (domain.APICredential, error) {
		requireEqual(t, orgID, testOrgID)
		got = in
		return domain.APICredential{
			ID: testOtherID, OrganizationID: testOrgID, Name: in.Name, KeyPrefix: in.KeyPrefix,
			KeyHash: in.KeyHash, Scope: in.Scope, Status: domain.APICredentialStatusActive, ServiceAccountID: in.ServiceAccountID,
		}, nil
	}
	saID := testAcctID
	cred, raw, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), "gateway key", string(domain.APICredentialScopeGateway), &saID)
	requireNoErr(t, err)
	requireEqual(t, cred.ID, testOtherID)
	requireEqual(t, got.Scope, string(domain.APICredentialScopeGateway))
	if got.ServiceAccountID != nil {
		t.Errorf("gateway scope must drop the service account, got %v", *got.ServiceAccountID)
	}
	if !strings.HasPrefix(raw, apiKeyPrefix) {
		t.Errorf("raw key %q lacks the %q prefix", raw, apiKeyPrefix)
	}
	requireEqual(t, got.KeyPrefix, raw[:len(apiKeyPrefix)+7])
	requireEqual(t, got.KeyHash, "hash:"+raw)
	requireAudit(t, st, "api_credential.create")
}

func TestCreateCredentialControlHappyPath(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	saID := testAcctID
	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testAcctID)
		return domain.ServiceAccount{ID: testAcctID, OrganizationID: testOrgID, Name: "ci-bot", Status: domain.ServiceAccountStatusActive}, nil
	}
	var got domain.NewAPICredential
	st.hasher.hash = func(_ context.Context, secret string) (string, error) { return "hash:" + secret, nil }
	st.credentials.create = func(_ context.Context, _ uuid.UUID, in domain.NewAPICredential) (domain.APICredential, error) {
		got = in
		return domain.APICredential{ID: testOtherID, OrganizationID: testOrgID, Scope: in.Scope, Status: domain.APICredentialStatusActive, ServiceAccountID: in.ServiceAccountID}, nil
	}
	_, raw, err := svc.CreateCredential(ctx, userActor(domain.RoleOwner), "control key", string(domain.APICredentialScopeAll), &saID)
	requireNoErr(t, err)
	requireEqual(t, got.Scope, string(domain.APICredentialScopeAll))
	if got.ServiceAccountID == nil || *got.ServiceAccountID != testAcctID {
		t.Fatalf("ServiceAccountID = %v, want %v", got.ServiceAccountID, testAcctID)
	}
	requireEqual(t, got.KeyPrefix, raw[:len(apiKeyPrefix)+7])
	requireAudit(t, st, "api_credential.create")
}

func TestCreateServiceAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("member lacks manage permission", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.CreateServiceAccount(ctx, userActor(domain.RoleMember), domain.NewServiceAccount{Name: "sa"})
		requireErrIs(t, err, repositories.ErrForbidden)
	})

	t.Run("blank name is invalid", func(t *testing.T) {
		svc, _ := newTestService()
		_, err := svc.CreateServiceAccount(ctx, userActor(domain.RoleOwner), domain.NewServiceAccount{Name: "  "})
		requireErrIs(t, err, repositories.ErrInvalid)
	})

	t.Run("stores the account as given", func(t *testing.T) {
		svc, st := newTestService()
		var got domain.NewServiceAccount
		st.accounts.create = func(_ context.Context, orgID uuid.UUID, in domain.NewServiceAccount) (domain.ServiceAccount, error) {
			requireEqual(t, orgID, testOrgID)
			got = in
			return domain.ServiceAccount{ID: testAcctID, OrganizationID: testOrgID, Name: in.Name, Description: in.Description, Status: domain.ServiceAccountStatusActive}, nil
		}
		account, err := svc.CreateServiceAccount(ctx, userActor(domain.RoleOwner), domain.NewServiceAccount{Name: "ci-bot", Description: "CI"})
		requireNoErr(t, err)
		requireEqual(t, account.ID, testAcctID)
		requireEqual(t, got.Name, "ci-bot")
		requireEqual(t, got.Description, "CI")
		requireAudit(t, st, "service_account.create")
	})
}
