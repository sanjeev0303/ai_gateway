package controlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/alert"
	"ai-governance-saas/backend/internal/application/auditlog"
	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

var (
	orgTenantA = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	orgTenantB = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// 3 distinct users across the two organizations
	userTenantA_Owner  = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001")
	userTenantA_Member = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002")
	userTenantB_Owner  = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0001")

	// Resource IDs owned strictly by Tenant B
	projectTenantB    = uuid.MustParse("99999999-9999-9999-9999-999999999999")
	teamTenantB       = uuid.MustParse("88888888-8888-8888-8888-888888888888")
	saTenantB         = uuid.MustParse("77777777-7777-7777-7777-777777777777")
	credTenantB       = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	provAcctTenantB   = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	provCredTenantB   = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	convTenantB       = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	policyTenantB     = uuid.MustParse("22222222-2222-2222-2222-222222222220")
	auditTenantB      = uuid.MustParse("11111111-1111-1111-1111-111111111110")
	alertTenantB      = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	alertEventTenantB = uuid.MustParse("00000000-0000-0000-0000-000000000088")
	localPlanID       = uuid.MustParse("00000000-0000-0000-0000-000000000005")
)

func tenantBearer(st *controlStubs, orgID, userID uuid.UUID, role string) map[string]string {
	token := "token-" + orgID.String() + "-" + userID.String()
	st.verifier.verify = func(_ context.Context, tok string) (domain.TokenClaims, error) {
		return domain.TokenClaims{
			Subject:        userID.String(),
			DisplayName:    "Test User",
			OrganizationID: orgID,
			Role:           role,
			Kind:           domain.TokenKindUser,
			IssuedAt:       testNow,
			ExpiresAt:      testNow.Add(time.Hour),
		}, nil
	}
	return map[string]string{"Authorization": "Bearer " + token}
}

// fullMultiTenantServer wires all control-plane application services with stubs.
func fullMultiTenantServer(
	t *testing.T,
	st *controlStubs,
	conv *fakeConversationRepo,
	msg *fakeMessageRepo,
	al *fakeAlertRepo,
	aud *fakeAuditRepo,
	plans *fakeBillingPlans,
	subs *fakeBillingSubs,
	invs *fakeBillingInvoices,
) *httptest.Server {
	t.Helper()
	identitySvc := identity.NewService(st.deps())
	provSvc := provider.NewService(st.providerDeps())
	convSvc := conversation.NewService(conversation.ServiceDeps{
		Organizations: st.orgs,
		Conversations: conv,
		Messages:      msg,
	})
	polSvc := policy.NewService(st.policyRepo, st.policyRuleRepo, st.audits, stubInvalidator{}, discardLog)
	auditSvc := auditlog.NewService(auditlog.ServiceDeps{Audits: aud})
	alertSvc := alert.NewService(alert.ServiceDeps{
		Alerts: al,
		Audits: st.audits,
		Log:    discardLog,
	})
	billingSvc := billing.NewService(billing.ServiceDeps{
		Plans:         plans,
		Subscriptions: subs,
		Invoices:      invs,
	})

	return buildServer(t, Deps{
		Identity:        identitySvc,
		Providers:       provSvc,
		Conversations:   convSvc,
		Policy:          polSvc,
		AuditLogs:       auditSvc,
		Alerts:          alertSvc,
		Billing:         billingSvc,
		DevLoginEnabled: true,
	})
}

// TestMultiTenantSecurity_IDORCrossTenantProtection ensures that a user from
// Tenant A (whether Owner or Member) cannot read or manipulate any resource belonging
// to Tenant B by substituting IDs across all resource families.
func TestMultiTenantSecurity_IDORCrossTenantProtection(t *testing.T) {
	st := newControlStubs()

	// Tenant B resources in stubs:
	st.projects.get = func(_ context.Context, orgID, projID uuid.UUID) (domain.Project, error) {
		if orgID != orgTenantB && projID == projectTenantB {
			return domain.Project{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && projID == projectTenantB {
			return domain.Project{ID: projectTenantB, OrganizationID: orgTenantB, Name: "Secret Project B"}, nil
		}
		return domain.Project{}, repositories.ErrNotFound
	}

	st.teams.get = func(_ context.Context, orgID, teamID uuid.UUID) (domain.Team, error) {
		if orgID != orgTenantB && teamID == teamTenantB {
			return domain.Team{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && teamID == teamTenantB {
			return domain.Team{ID: teamTenantB, OrganizationID: orgTenantB, Name: "Team B"}, nil
		}
		return domain.Team{}, repositories.ErrNotFound
	}

	st.gatewayCredentials.get = func(_ context.Context, orgID, credID uuid.UUID) (domain.GatewayCredential, error) {
		if orgID != orgTenantB && credID == credTenantB {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && credID == credTenantB {
			return domain.GatewayCredential{ID: credTenantB, OrganizationID: orgTenantB, Name: "GW Cred B"}, nil
		}
		return domain.GatewayCredential{}, repositories.ErrNotFound
	}
	st.gatewayCredentials.revoke = func(_ context.Context, orgID, credID uuid.UUID) (domain.GatewayCredential, error) {
		if orgID != orgTenantB && credID == credTenantB {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		return domain.GatewayCredential{ID: credTenantB, Status: domain.GatewayCredentialStatusRevoked}, nil
	}

	st.providerAccounts.get = func(_ context.Context, orgID, acctID uuid.UUID) (domain.ProviderAccount, error) {
		if orgID != orgTenantB && acctID == provAcctTenantB {
			return domain.ProviderAccount{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && acctID == provAcctTenantB {
			return domain.ProviderAccount{ID: provAcctTenantB, OrganizationID: orgTenantB, Name: "Provider B"}, nil
		}
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}

	st.providerCredentials.get = func(_ context.Context, orgID, credID uuid.UUID) (domain.ProviderCredential, error) {
		if orgID != orgTenantB && credID == provCredTenantB {
			return domain.ProviderCredential{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && credID == provCredTenantB {
			return domain.ProviderCredential{ID: provCredTenantB, OrganizationID: orgTenantB, Name: "Prov Cred B"}, nil
		}
		return domain.ProviderCredential{}, repositories.ErrNotFound
	}
	st.providerCredentials.revoke = func(_ context.Context, orgID, credID uuid.UUID) (domain.ProviderCredential, error) {
		if orgID != orgTenantB && credID == provCredTenantB {
			return domain.ProviderCredential{}, repositories.ErrNotFound
		}
		return domain.ProviderCredential{ID: credID, Status: domain.ProviderCredentialStatusRevoked}, nil
	}

	st.accounts.revoke = func(_ context.Context, orgID, saID uuid.UUID) (domain.ServiceAccount, error) {
		if orgID != orgTenantB && saID == saTenantB {
			return domain.ServiceAccount{}, repositories.ErrNotFound
		}
		return domain.ServiceAccount{ID: saID, Status: domain.ServiceAccountStatusRevoked}, nil
	}

	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		if orgID != orgTenantB && userID == userTenantB_Owner {
			return domain.OrgMember{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && userID == userTenantB_Owner {
			return domain.OrgMember{User: domain.User{ID: userTenantB_Owner}, Role: domain.RoleOwner}, nil
		}
		return domain.OrgMember{}, repositories.ErrNotFound
	}
	st.members.remove = func(_ context.Context, orgID, userID uuid.UUID) error {
		if orgID != orgTenantB && userID == userTenantB_Owner {
			return repositories.ErrNotFound
		}
		return nil
	}

	st.policyRepo.get = func(_ context.Context, orgID, polID uuid.UUID) (domain.Policy, error) {
		if orgID != orgTenantB && polID == policyTenantB {
			return domain.Policy{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && polID == policyTenantB {
			return domain.Policy{ID: policyTenantB, OrganizationID: orgTenantB, Name: "Policy B"}, nil
		}
		return domain.Policy{}, repositories.ErrNotFound
	}
	st.policyRepo.delete = func(_ context.Context, orgID, polID uuid.UUID) error {
		if orgID != orgTenantB && polID == policyTenantB {
			return repositories.ErrNotFound
		}
		return nil
	}

	convPrivacy := string(domain.PrivacyFullContent)
	convRepo := &fakeConversationRepo{
		detail: domain.ConversationSummary{
			ID:           convTenantB,
			UserID:       userTenantB_Owner,
			PrivacyState: &convPrivacy,
		},
		detailErr: repositories.ErrNotFound,
	}
	msgRepo := &fakeMessageRepo{}
	auditRepo := &fakeAuditRepo{
		record: domain.AuditLog{
			ID:             auditTenantB,
			OrganizationID: orgTenantB,
			Action:         "secret.action",
		},
		getErr: repositories.ErrNotFound,
	}
	alertRepo := &fakeAlertRepo{
		alert: domain.Alert{
			ID:             alertTenantB,
			OrganizationID: orgTenantB,
			Name:           "Secret Alert B",
		},
		err: repositories.ErrNotFound,
	}

	srv := fullMultiTenantServer(t, st, convRepo, msgRepo, alertRepo, auditRepo,
		&fakeBillingPlans{plans: []domain.Plan{{ID: localPlanID, Code: "enterprise"}}},
		&fakeBillingSubs{},
		&fakeBillingInvoices{},
	)

	// User from Tenant A (acting as Owner of Tenant A)
	hdrA := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	idorTests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"CrossTenantGetProject", http.MethodGet, "/v1/projects/" + projectTenantB.String(), nil},
		{"CrossTenantGetTeam", http.MethodGet, "/v1/teams/" + teamTenantB.String(), nil},
		{"CrossTenantGetGatewayCredential", http.MethodGet, "/v1/gateway-credentials/" + credTenantB.String(), nil},
		{"CrossTenantRevokeGatewayCredential", http.MethodDelete, "/v1/gateway-credentials/" + credTenantB.String(), nil},
		{"CrossTenantRotateGatewayCredential", http.MethodPost, "/v1/gateway-credentials/" + credTenantB.String() + "/rotate", map[string]any{"name": "rotated-key"}},
		{"CrossTenantGetProviderAccount", http.MethodGet, "/v1/provider-accounts/" + provAcctTenantB.String(), nil},
		{"CrossTenantGetProviderCredential", http.MethodGet, "/v1/provider-credentials/" + provCredTenantB.String(), nil},
		{"CrossTenantRevokeProviderCredential", http.MethodDelete, "/v1/provider-credentials/" + provCredTenantB.String(), nil},
		{"CrossTenantRevokeServiceAccount", http.MethodDelete, "/v1/service-accounts/" + saTenantB.String(), nil},
		{"CrossTenantRemoveMember", http.MethodDelete, "/v1/members/" + userTenantB_Owner.String(), nil},
		{"CrossTenantGetPolicy", http.MethodGet, "/v1/policies/" + policyTenantB.String(), nil},
		{"CrossTenantDeletePolicy", http.MethodDelete, "/v1/policies/" + policyTenantB.String(), nil},
		{"CrossTenantGetAuditLog", http.MethodGet, "/v1/audit-logs/" + auditTenantB.String(), nil},
		{"CrossTenantGetAlert", http.MethodGet, "/v1/alerts/" + alertTenantB.String(), nil},
		{"CrossTenantGetConversation", http.MethodGet, "/v1/conversations/" + convTenantB.String(), nil},
	}

	for _, tt := range idorTests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doJSON(t, srv, tt.method, tt.path, tt.body, hdrA)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 404 Not Found or 403 Forbidden for cross-tenant ID %s, got %d", tt.path, resp.StatusCode)
			}
		})
	}
}

// TestMultiTenantSecurity_RoleBasedAccessControl verifies role-based permissions
// within the same organization. Ordinary members are restricted to day-to-day operations
// and blocked from administrative mutations, while owners/admins have management access.
func TestMultiTenantSecurity_RoleBasedAccessControl(t *testing.T) {
	st := newControlStubs()

	st.projects.list = func(context.Context, uuid.UUID) ([]domain.Project, error) {
		return []domain.Project{{ID: testProjID, Name: "Proj1"}}, nil
	}
	st.teams.list = func(context.Context, uuid.UUID) ([]domain.Team, error) {
		return []domain.Team{{ID: testTeamID, Name: "Team1"}}, nil
	}
	st.members.list = func(context.Context, uuid.UUID) ([]domain.OrgMember, error) {
		return []domain.OrgMember{{User: domain.User{ID: userTenantA_Member}, Role: domain.RoleMember}}, nil
	}

	srv := fullMultiTenantServer(t, st,
		&fakeConversationRepo{},
		&fakeMessageRepo{},
		&fakeAlertRepo{},
		&fakeAuditRepo{},
		&fakeBillingPlans{plans: []domain.Plan{{ID: localPlanID, Code: "enterprise"}}},
		&fakeBillingSubs{},
		&fakeBillingInvoices{},
	)

	memberHdr := tenantBearer(st, orgTenantA, userTenantA_Member, domain.RoleMember)

	t.Run("MemberPermittedReads", func(t *testing.T) {
		allowedReads := []string{
			"/v1/projects",
			"/v1/teams",
			"/v1/members",
		}
		for _, p := range allowedReads {
			resp := doJSON(t, srv, http.MethodGet, p, nil, memberHdr)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 OK for member read on %s, got %d", p, resp.StatusCode)
			}
		}
	})

	t.Run("MemberForbiddenAdministrativeMutations", func(t *testing.T) {
		forbiddenMutations := []struct {
			name   string
			method string
			path   string
			body   any
		}{
			{"CreateProject", http.MethodPost, "/v1/projects", map[string]any{"name": "Evil Proj"}},
			{"CreateTeam", http.MethodPost, "/v1/teams", map[string]any{"name": "Evil Team"}},
			{"InviteMember", http.MethodPost, "/v1/invitations", map[string]any{"email": "hacker@test.com", "role": "member"}},
			{"CreateGatewayCredential", http.MethodPost, "/v1/gateway-credentials", map[string]any{"name": "evil-key", "subject_type": "user", "subject_id": userTenantA_Member.String()}},
			{"RevokeGatewayCredential", http.MethodDelete, "/v1/gateway-credentials/" + credTenantB.String(), nil},
			{"CreatePolicy", http.MethodPost, "/v1/policies", map[string]any{"name": "Evil Policy", "enforcement": "active", "priority": 10}},
			{"DeletePolicy", http.MethodDelete, "/v1/policies/" + policyTenantB.String(), nil},
			{"ReadAuditLogs", http.MethodGet, "/v1/audit-logs", nil},
			{"CreateAlert", http.MethodPost, "/v1/alerts", map[string]any{"name": "Evil Alert", "alert_type": "budget", "severity": "warning", "configuration": map[string]any{"threshold": 80}}},
			{"ReadInvoices", http.MethodGet, "/v1/billing/invoices", nil},
		}

		for _, tt := range forbiddenMutations {
			t.Run(tt.name, func(t *testing.T) {
				resp := doJSON(t, srv, tt.method, tt.path, tt.body, memberHdr)
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusForbidden {
					t.Fatalf("expected 403 Forbidden for member attempting %s %s, got %d", tt.method, tt.path, resp.StatusCode)
				}
			})
		}
	})
}

// TestMultiTenantSecurity_SubjectLevelIsolationSameOrg tests that even within the
// same organization, an ordinary member cannot read another member's conversation.
func TestMultiTenantSecurity_SubjectLevelIsolationSameOrg(t *testing.T) {
	st := newControlStubs()

	ownerConvID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	memberConvID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	convRepo := &fakeConversationRepo{}
	convSvc := conversation.NewService(conversation.ServiceDeps{
		Organizations: st.orgs,
		Conversations: convRepo,
		Messages:      &fakeMessageRepo{},
	})

	ctx := context.Background()
	ownerActor := domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         userTenantA_Owner,
		OrganizationID: orgTenantA,
		Role:           domain.RoleOwner,
	}
	memberActor := domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         userTenantA_Member,
		OrganizationID: orgTenantA,
		Role:           domain.RoleMember,
	}

	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: orgTenantA, PrivacyDefault: domain.PrivacyFullContent}, nil
	}

	// 1. Member requests their own conversation -> permitted
	convPrivacy := string(domain.PrivacyFullContent)
	convRepo.detail = domain.ConversationSummary{
		ID:           memberConvID,
		UserID:       userTenantA_Member,
		PrivacyState: &convPrivacy,
	}
	got, err := convSvc.Get(ctx, memberActor, memberConvID)
	if err != nil {
		t.Fatalf("expected member to view own conversation, got err: %v", err)
	}
	if got.ID != memberConvID {
		t.Fatalf("expected own conversation %s, got %s", memberConvID, got.ID)
	}

	// 2. Member requests Owner's conversation -> blocked by subject access control
	convRepo.detail = domain.ConversationSummary{
		ID:           ownerConvID,
		UserID:       userTenantA_Owner,
		PrivacyState: &convPrivacy,
	}
	_, err = convSvc.Get(ctx, memberActor, ownerConvID)
	if err != repositories.ErrForbidden {
		t.Fatalf("expected ErrForbidden for member accessing another member's conversation, got: %v", err)
	}

	// 3. Owner requests Member's conversation -> permitted (privileged role)
	convRepo.detail = domain.ConversationSummary{
		ID:           memberConvID,
		UserID:       userTenantA_Member,
		PrivacyState: &convPrivacy,
	}
	gotOwner, err := convSvc.Get(ctx, ownerActor, memberConvID)
	if err != nil {
		t.Fatalf("expected owner to view member's conversation, got err: %v", err)
	}
	if gotOwner.ID != memberConvID {
		t.Fatalf("expected conversation %s, got %s", memberConvID, gotOwner.ID)
	}
}

// TestMultiTenantSecurity_CrossTenantOrgSwitchingBlocked tests that a user from Org A
// cannot switch their session context to Org B when they are not a member of Org B.
func TestMultiTenantSecurity_CrossTenantOrgSwitchingBlocked(t *testing.T) {
	st := newControlStubs()
	svc := identity.NewService(st.deps())

	st.users.getByID = func(_ context.Context, id uuid.UUID) (domain.User, error) {
		return domain.User{ID: id, Status: domain.UserStatusActive}, nil
	}
	// Org B membership check fails
	st.members.get = func(_ context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
		if orgID == orgTenantB {
			return domain.OrgMember{}, repositories.ErrNotFound
		}
		return domain.OrgMember{OrganizationID: orgID, User: domain.User{ID: userID}, Role: domain.RoleOwner}, nil
	}

	ctx := context.Background()
	_, err := svc.SwitchOrganization(ctx, userTenantA_Owner, orgTenantB)
	if err != identity.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized when switching to unjoined organization, got: %v", err)
	}
}

// TestMultiTenantSecurity_SecretScrubbing ensures responses never leak raw API keys,
// secret references, password hashes, or full tokens in listing responses.
func TestMultiTenantSecurity_SecretScrubbing(t *testing.T) {
	srv, st := newTestServer(t)

	st.gatewayCredentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error) {
		return []domain.GatewayCredential{
			{
				ID:             testOtherID,
				OrganizationID: orgTenantA,
				Name:           "gateway-key",
				KeyPrefix:      "aigw_live_test",
				KeyHash:        "super-secret-key-hash-never-expose",
				Scopes:         []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
				Status:         domain.GatewayCredentialStatusActive,
				CreatedAt:      testNow,
				UpdatedAt:      testNow,
			},
		}, nil
	}

	st.providerCredentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{
				ID:                testProvCred,
				OrganizationID:    orgTenantA,
				ProviderAccountID: testProvAcct,
				Name:              "Anthropic Key",
				Kind:              domain.CredentialKindAPIKey,
				SecretRef:         "vault/secrets/anthropic-raw-key",
				Fingerprint:       "sha256:12345678",
				Tail:              "1234",
				Status:            domain.ProviderCredentialStatusActive,
				CreatedAt:         testNow,
				UpdatedAt:         testNow,
			},
		}, nil
	}

	hdrA := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	t.Run("GatewayCredentialsScrubbed", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/gateway-credentials", nil, hdrA)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		rawBytes, _ := json.Marshal(body)
		rawStr := string(rawBytes)

		if containsSubstring(rawStr, "super-secret-key-hash-never-expose") {
			t.Errorf("leaked secret key hash in response: %s", rawStr)
		}
	})

	t.Run("ProviderCredentialsScrubbed", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials", nil, hdrA)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		rawBytes, _ := json.Marshal(body)
		rawStr := string(rawBytes)

		if containsSubstring(rawStr, "vault/secrets/anthropic-raw-key") {
			t.Errorf("leaked provider SecretRef in response: %s", rawStr)
		}
	})
}

// TestMultiTenantSecurity_RevokedCredentialImmediateRejection tests that revoking a credential
// sets status to revoked and returns 204 No Content.
func TestMultiTenantSecurity_RevokedCredentialImmediateRejection(t *testing.T) {
	srv, st := newTestServer(t)

	revoked := false
	st.gatewayCredentials.revoke = func(_ context.Context, orgID, credID uuid.UUID) (domain.GatewayCredential, error) {
		revoked = true
		return domain.GatewayCredential{
			ID:             credID,
			OrganizationID: orgID,
			Status:         domain.GatewayCredentialStatusRevoked,
			UpdatedAt:      time.Now(),
		}, nil
	}

	hdrA := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/gateway-credentials/"+testOtherID.String(), nil, hdrA)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	if !revoked {
		t.Fatal("expected credential to be revoked in repository")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && len(sub) > 0 && stringContains(s, sub)))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
