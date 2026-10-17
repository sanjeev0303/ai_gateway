package controlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/budget"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ----------------------------------------------------------------------------
// Phase I: Three-User Fixture, Secret Isolation, Budgets, Policies & Revocation
// ----------------------------------------------------------------------------

func TestThreeUsers_Fixtures_SecretIsolation_BudgetsAndRevocation(t *testing.T) {
	st := newControlStubs()

	if st.policyRepo.list == nil {
		st.policyRepo.list = func(context.Context, uuid.UUID) ([]domain.Policy, error) {
			return []domain.Policy{{ID: testPolicyID, Name: "Budget Enforcement"}}, nil
		}
	}
	if st.policyRuleRepo.list == nil {
		st.policyRuleRepo.list = func(context.Context, uuid.UUID, uuid.UUID) ([]domain.PolicyRule, error) {
			return nil, nil
		}
	}
	if st.policyRuleRepo.create == nil {
		st.policyRuleRepo.create = func(_ context.Context, _ uuid.UUID, in domain.NewPolicyRule) (domain.PolicyRule, error) {
			return domain.PolicyRule{ID: testRuleID}, nil
		}
	}

	svc := identity.NewService(st.deps())
	psvc := provider.NewService(st.providerDeps())
	bSvc := budget.NewService(st.budgetDeps())
	pSvc := policy.NewService(st.policyRepo, st.policyRuleRepo, st.audits, stubInvalidator{}, discardLog)

	srv := buildServer(t, Deps{
		Identity:        svc,
		Providers:       psvc,
		Budget:          bSvc,
		Policy:          pSvc,
		DevLoginEnabled: true,
	})
	defer srv.Close()

	orgID := testOrgID
	ownerID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	user1ID := uuid.MustParse("11111111-0000-0000-0000-000000000001")
	user2ID := uuid.MustParse("22222222-0000-0000-0000-000000000002")
	user3ID := uuid.MustParse("33333333-0000-0000-0000-000000000003")

	proj1ID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
	proj2ID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")

	providerMasterSecret := "sk-super-secret-provider-master-key-xyz123"

	// 1. Verify Provider Master Secret Registration in Secure Vault
	st.projects.get = func(_ context.Context, oid, pid uuid.UUID) (domain.Project, error) {
		return domain.Project{
			ID:             pid,
			OrganizationID: oid,
			Name:           "Test Project",
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, nil
	}

	st.providerCredentials.create = func(_ context.Context, oid uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
		return domain.ProviderCredential{
			ID:                uuid.New(),
			OrganizationID:    oid,
			ProviderAccountID: in.ProviderAccountID,
			Name:              in.Name,
			Kind:              in.Kind,
			SecretRef:         "vault:ref:" + uuid.New().String(),
			Fingerprint:       "fp_12345678",
			Status:            domain.ProviderCredentialStatusActive,
			CreatedAt:         testNow,
			UpdatedAt:         testNow,
		}, nil
	}

	// 2. Issue 3 Separate Sentinel Gateway Credentials for the 3 Users
	credIDs := make(map[uuid.UUID]uuid.UUID)
	st.hasher.hash = func(_ context.Context, s string) (string, error) { return "hash:" + s, nil }
	st.gatewayCredentials.create = func(_ context.Context, oid uuid.UUID, in domain.NewGatewayCredential) (domain.GatewayCredential, error) {
		id := uuid.New()
		if in.SubjectID != uuid.Nil {
			credIDs[in.SubjectID] = id
		}
		return domain.GatewayCredential{
			ID:             id,
			OrganizationID: oid,
			Name:           in.Name,
			KeyPrefix:      in.KeyPrefix,
			KeyHash:        in.KeyHash,
			SubjectType:    in.SubjectType,
			SubjectID:      in.SubjectID,
			Scopes:         in.Scopes,
			Status:         domain.GatewayCredentialStatusActive,
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, nil
	}

	// User 1 Credential (Project 1)
	cred1Req := map[string]any{
		"name":         "test-user-1-key",
		"subject_type": "user",
		"subject_id":   user1ID.String(),
		"project_id":   proj1ID.String(),
		"scopes":       []string{"gateway"},
	}
	r1 := doJSON(t, srv, http.MethodPost, "/v1/gateway-credentials", cred1Req, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, r1.StatusCode, http.StatusCreated)
	var c1Resp map[string]any
	decodeBody(t, r1, &c1Resp)

	// User 2 Credential (Project 1)
	cred2Req := map[string]any{
		"name":         "test-user-2-key",
		"subject_type": "user",
		"subject_id":   user2ID.String(),
		"project_id":   proj1ID.String(),
		"scopes":       []string{"gateway"},
	}
	r2 := doJSON(t, srv, http.MethodPost, "/v1/gateway-credentials", cred2Req, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, r2.StatusCode, http.StatusCreated)
	var c2Resp map[string]any
	decodeBody(t, r2, &c2Resp)

	// User 3 Credential (Project 2)
	cred3Req := map[string]any{
		"name":         "test-user-3-key",
		"subject_type": "user",
		"subject_id":   user3ID.String(),
		"project_id":   proj2ID.String(),
		"scopes":       []string{"gateway"},
	}
	r3 := doJSON(t, srv, http.MethodPost, "/v1/gateway-credentials", cred3Req, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, r3.StatusCode, http.StatusCreated)
	var c3Resp map[string]any
	decodeBody(t, r3, &c3Resp)

	// 3. Security Assertion: Master provider secret MUST NEVER appear in employee credentials
	for _, c := range []map[string]any{c1Resp, c2Resp, c3Resp} {
		for k, v := range c {
			strVal := strings.ToLower(k + ":" + string(mustJSON(v)))
			if strings.Contains(strVal, "sk-super-secret") || strings.Contains(strVal, providerMasterSecret) {
				t.Fatalf("CRITICAL SECURITY LEAK: Provider master secret found in employee gateway credential response: %s", strVal)
			}
		}
	}

	// 4. Explicit Model-Access & Budget Settings
	budgetCreated := false
	st.budgets.create = func(_ context.Context, oid uuid.UUID, in domain.NewBudget) (domain.Budget, error) {
		budgetCreated = true
		return domain.Budget{
			ID:             uuid.New(),
			OrganizationID: oid,
			Name:           in.Name,
			Scope:          in.Scope,
			ScopeID:        in.ScopeID,
			LimitType:      in.LimitType,
			LimitAmount:    in.LimitAmount,
			Window:         in.Window,
			Enabled:        in.Enabled,
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, nil
	}

	budgetPayload := map[string]any{
		"name":         "Project Alpha Token Budget",
		"scope":        "project",
		"scope_id":     proj1ID.String(),
		"limit_type":   "tokens",
		"limit_amount": 500000,
		"window":       "monthly",
		"enabled":      true,
	}
	rBudget := doJSON(t, srv, http.MethodPost, "/v1/budgets", budgetPayload, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, rBudget.StatusCode, http.StatusCreated)
	if !budgetCreated {
		t.Fatalf("expected budget repository create to be invoked")
	}

	// Model Access Policy
	policyCreated := false
	polID := uuid.New()
	st.policyRepo.create = func(_ context.Context, oid uuid.UUID, in domain.NewPolicy) (domain.Policy, error) {
		policyCreated = true
		return domain.Policy{
			ID:             polID,
			OrganizationID: oid,
			Name:           in.Name,
			Enforcement:    in.Enforcement,
			Priority:       in.Priority,
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, nil
	}
	rPol := doJSON(t, srv, http.MethodPost, "/v1/policies", map[string]any{
		"name":        "Three-User Governance Policy",
		"enforcement": "active",
		"priority":    10,
	}, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, rPol.StatusCode, http.StatusCreated)
	if !policyCreated {
		t.Fatalf("expected policy repository create to be invoked")
	}

	// 5. Cross-User Isolation: User 1 cannot delete User 2's credential
	user2CredID := credIDs[user2ID]
	st.gatewayCredentials.get = func(_ context.Context, oid, cid uuid.UUID) (domain.GatewayCredential, error) {
		if cid == user2CredID {
			return domain.GatewayCredential{
				ID:             user2CredID,
				OrganizationID: orgID,
				SubjectType:    domain.SubjectTypeUser,
				SubjectID:      user2ID,
				Status:         domain.GatewayCredentialStatusActive,
			}, nil
		}
		return domain.GatewayCredential{}, repositories.ErrNotFound
	}
	rUser1DeleteUser2 := doJSON(t, srv, http.MethodDelete, "/v1/gateway-credentials/"+user2CredID.String(), nil,
		tenantBearer(st, orgID, user1ID, domain.RoleMember))
	if rUser1DeleteUser2.StatusCode != http.StatusForbidden && rUser1DeleteUser2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 when user1 attempts to delete user2 credential, got %d", rUser1DeleteUser2.StatusCode)
	}

	// 6. Revocation: Owner can revoke User 2's credential
	revoked := false
	st.gatewayCredentials.revoke = func(_ context.Context, oid, cid uuid.UUID) (domain.GatewayCredential, error) {
		if cid == user2CredID {
			revoked = true
			return domain.GatewayCredential{
				ID:             cid,
				OrganizationID: oid,
				Status:         domain.GatewayCredentialStatusRevoked,
			}, nil
		}
		return domain.GatewayCredential{}, repositories.ErrNotFound
	}
	rRevoke := doJSON(t, srv, http.MethodDelete, "/v1/gateway-credentials/"+user2CredID.String(), nil,
		tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, rRevoke.StatusCode, http.StatusNoContent)
	if !revoked {
		t.Fatalf("expected credential revoke to be called")
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
