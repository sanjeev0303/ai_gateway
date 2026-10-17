package controlapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ----------------------------------------------------------------------------
// A. Organization Lifecycle, Invitations, Teams, Projects, & Credentials
// ----------------------------------------------------------------------------

func TestOrgLifecycle_InvitesAndDuplicatesAndRBAC(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()

	orgID := testOrgID
	ownerID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	adminID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002")
	memberID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000003")

	// 1. Owner invites a new member -> 201 Created
	invitedEmail := "developer@example.com"
	inviteID := uuid.New()
	st.hasher.hash = func(_ context.Context, s string) (string, error) { return "hash:" + s, nil }
	st.invitations.create = func(_ context.Context, oid uuid.UUID, in domain.NewInvitation) (domain.Invitation, error) {
		if oid != orgID {
			t.Fatalf("expected orgID %v, got %v", orgID, oid)
		}
		if in.Email != invitedEmail || in.Role != domain.RoleMember {
			t.Fatalf("unexpected invite input: %+v", in)
		}
		return domain.Invitation{
			ID:             inviteID,
			OrganizationID: orgID,
			Email:          in.Email,
			Role:           in.Role,
			TokenHash:      in.TokenHash,
			Status:         domain.InvitationStatusPending,
			ExpiresAt:      testNow.Add(testInviteTTL),
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, nil
	}

	invitePayload := map[string]any{"email": "developer@example.com", "role": "member"}
	resp := doJSON(t, srv, http.MethodPost, "/v1/invitations", invitePayload, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusCreated)

	var inviteResp struct {
		Token string `json:"token"`
	}
	decodeBody(t, resp, &inviteResp)
	if inviteResp.Token == "" {
		t.Fatalf("expected raw token in invite response")
	}

	// 2. Duplicate pending invitation for same email -> 409 Conflict
	st.invitations.create = func(_ context.Context, _ uuid.UUID, _ domain.NewInvitation) (domain.Invitation, error) {
		return domain.Invitation{}, repositories.ErrConflict
	}
	respDup := doJSON(t, srv, http.MethodPost, "/v1/invitations", invitePayload, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respDup.StatusCode, http.StatusConflict)

	// 3. Admin attempts to invite an Owner -> 403 Forbidden
	adminInviteOwnerPayload := map[string]any{"email": "newowner@example.com", "role": "owner"}
	respAdminOwner := doJSON(t, srv, http.MethodPost, "/v1/invitations", adminInviteOwnerPayload, tenantBearer(st, orgID, adminID, domain.RoleAdmin))
	requireEqual(t, respAdminOwner.StatusCode, http.StatusForbidden)

	// 4. Member attempts to invite anyone -> 403 Forbidden
	respMemberInvite := doJSON(t, srv, http.MethodPost, "/v1/invitations", invitePayload, tenantBearer(st, orgID, memberID, domain.RoleMember))
	requireEqual(t, respMemberInvite.StatusCode, http.StatusForbidden)

	// 5. Admin attempts to promote user to Owner via PATCH /v1/members/{id}/role -> 403 Forbidden
	respPatchOwner := doJSON(t, srv, http.MethodPatch, "/v1/members/"+memberID.String()+"/role", map[string]any{"role": "owner"}, tenantBearer(st, orgID, adminID, domain.RoleAdmin))
	requireEqual(t, respPatchOwner.StatusCode, http.StatusForbidden)

	// 6. User attempts to modify their own role -> 403 Forbidden
	respSelfRole := doJSON(t, srv, http.MethodPatch, "/v1/members/"+ownerID.String()+"/role", map[string]any{"role": "admin"}, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respSelfRole.StatusCode, http.StatusForbidden)

	// 7. Revoke invitation -> 204 No Content
	st.invitations.revoke = func(_ context.Context, oid, iid uuid.UUID) (domain.Invitation, error) {
		if oid != orgID || iid != inviteID {
			t.Fatalf("unexpected revoke args: oid=%v, iid=%v", oid, iid)
		}
		return domain.Invitation{ID: iid, OrganizationID: oid, Status: domain.InvitationStatusRevoked}, nil
	}
	respRevoke := doJSON(t, srv, http.MethodDelete, "/v1/invitations/"+inviteID.String(), nil, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respRevoke.StatusCode, http.StatusNoContent)
}

func TestOrgLifecycle_TeamsProjectsMembership(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()

	orgID := testOrgID
	ownerID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	targetUserID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002")

	teamID := uuid.New()
	st.teams.create = func(_ context.Context, oid uuid.UUID, nt domain.NewTeam) (domain.Team, error) {
		return domain.Team{ID: teamID, OrganizationID: oid, Name: nt.Name, Description: nt.Description}, nil
	}
	st.teamMembers.add = func(_ context.Context, oid, tid, uid uuid.UUID, role string) (domain.TeamMember, error) {
		return domain.TeamMember{
			TeamID:         tid,
			OrganizationID: oid,
			UserID:         uid,
			Role:           role,
		}, nil
	}
	st.teamMembers.remove = func(_ context.Context, _, _, _ uuid.UUID) error {
		return nil
	}

	// 1. Add team member
	addMemberPayload := map[string]any{"user_id": targetUserID.String(), "role": "member"}
	respAddTeam := doJSON(t, srv, http.MethodPost, "/v1/teams/"+teamID.String()+"/members", addMemberPayload, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respAddTeam.StatusCode, http.StatusCreated)

	// 2. Remove team member
	respDelTeam := doJSON(t, srv, http.MethodDelete, "/v1/teams/"+teamID.String()+"/members/"+targetUserID.String(), nil, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respDelTeam.StatusCode, http.StatusNoContent)

	// 3. Project member add and remove
	projID := uuid.New()
	st.projects.get = func(_ context.Context, oid, pid uuid.UUID) (domain.Project, error) {
		return domain.Project{ID: pid, OrganizationID: oid, Name: "AIGW", Slug: "aigw"}, nil
	}
	st.projMembers.add = func(_ context.Context, oid, pid, uid uuid.UUID, role string) (domain.ProjectMember, error) {
		if oid != orgID || pid != projID || uid != targetUserID {
			t.Fatalf("unexpected project member add args")
		}
		return domain.ProjectMember{
			ProjectID:      pid,
			OrganizationID: oid,
			UserID:         uid,
			Role:           role,
		}, nil
	}
	st.projMembers.remove = func(_ context.Context, _, _, _ uuid.UUID) error { return nil }

	respAddProj := doJSON(t, srv, http.MethodPost, "/v1/projects/"+projID.String()+"/members", addMemberPayload, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respAddProj.StatusCode, http.StatusCreated)

	respDelProj := doJSON(t, srv, http.MethodDelete, "/v1/projects/"+projID.String()+"/members/"+targetUserID.String(), nil, tenantBearer(st, orgID, ownerID, domain.RoleOwner))
	requireEqual(t, respDelProj.StatusCode, http.StatusNoContent)
}

func TestOrgLifecycle_GatewayCredentials(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()

	orgID := testOrgID
	ownerID := testUserID
	credID := uuid.New()

	st.hasher.hash = func(_ context.Context, s string) (string, error) { return "hash:" + s, nil }
	st.gatewayCredentials.create = func(_ context.Context, oid uuid.UUID, in domain.NewGatewayCredential) (domain.GatewayCredential, error) {
		return domain.GatewayCredential{
			ID:             credID,
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

	// 1. Create gateway credential
	credPayload := map[string]any{
		"name":         "Production User Key",
		"subject_type": "user",
		"subject_id":   ownerID.String(),
		"scopes":       []string{"gateway"},
	}
	resp := doJSON(t, srv, http.MethodPost, "/v1/gateway-credentials", credPayload, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusCreated)

	var credResp gatewayCredentialDTO
	decodeBody(t, resp, &credResp)
	if credResp.KeyPrefix == "" || credResp.Status != "active" {
		t.Fatalf("invalid credential response: %+v", credResp)
	}

	// 2. Rotate gateway credential
	newCredID := uuid.New()
	st.gatewayCredentials.get = func(_ context.Context, _, id uuid.UUID) (domain.GatewayCredential, error) {
		return domain.GatewayCredential{
			ID:             id,
			OrganizationID: orgID,
			Name:           "Production User Key",
			Status:         domain.GatewayCredentialStatusActive,
		}, nil
	}
	st.gatewayCredentials.rotate = func(_ context.Context, oid, id uuid.UUID, in domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
		return domain.GatewayCredential{
			ID:             newCredID,
			OrganizationID: oid,
			Name:           in.Name,
			KeyPrefix:      in.KeyPrefix,
			KeyHash:        in.KeyHash,
			Status:         domain.GatewayCredentialStatusActive,
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}, "gk_live_rotatedsecret12345678", nil
	}

	rotatePayload := map[string]any{"name": "Rotated Key", "scopes": []string{"gateway"}}
	respRotate := doJSON(t, srv, http.MethodPost, "/v1/gateway-credentials/"+credID.String()+"/rotate", rotatePayload, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, respRotate.StatusCode, http.StatusCreated)

	var rotateResp gatewayCredentialDTO
	decodeBody(t, respRotate, &rotateResp)
	if rotateResp.ID != newCredID.String() || rotateResp.KeyPrefix == "" {
		t.Fatalf("invalid rotate response: %+v", rotateResp)
	}

	// 3. Revoke credential
	st.gatewayCredentials.revoke = func(_ context.Context, oid, id uuid.UUID) (domain.GatewayCredential, error) {
		return domain.GatewayCredential{ID: id, OrganizationID: oid, Status: domain.GatewayCredentialStatusRevoked}, nil
	}
	respRevoke := doJSON(t, srv, http.MethodDelete, "/v1/gateway-credentials/"+newCredID.String(), nil, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, respRevoke.StatusCode, http.StatusNoContent)
}

// ----------------------------------------------------------------------------
// B. Conversation History, Ownership Isolation, & Privacy Redaction
// ----------------------------------------------------------------------------

func TestConversationHistory_SubjectAccessAndPrivacyModes(t *testing.T) {
	st := newControlStubs()
	convRepo := &fakeConversationRepo{}
	msgRepo := &fakeMessageRepo{}
	alRepo := &fakeAlertRepo{}
	audRepo := &fakeAuditRepo{}
	plansRepo := &fakeBillingPlans{}
	subsRepo := &fakeBillingSubs{}
	invsRepo := &fakeBillingInvoices{}

	srv := fullMultiTenantServer(t, st, convRepo, msgRepo, alRepo, audRepo, plansRepo, subsRepo, invsRepo)
	defer srv.Close()

	orgID := orgTenantA
	userAlice := userTenantA_Owner // Admin/Owner
	userBob := userTenantA_Member  // Regular Member

	convAlice := uuid.New()
	convRepo.detail = domain.ConversationSummary{
		ID:           convAlice,
		UserID:       userAlice,
		Title:        "Alice Architecture Session",
		MessageCount: 2,
		TotalTokens:  250,
		StartedAt:    testNow,
	}

	msgRepo.messages = []domain.ConversationMessage{
		{
			ID:             uuid.New(),
			ConversationID: convAlice,
			UserID:         userAlice,
			Role:           "user",
			Content:        "Can we store credentials in plaintext?",
			Tokens:         100,
			CreatedAt:      testNow,
		},
		{
			ID:             uuid.New(),
			ConversationID: convAlice,
			UserID:         userAlice,
			Role:           "assistant",
			Content:        "No, credentials must always be hashed with SHA-256.",
			Tokens:         150,
			CreatedAt:      testNow.Add(time.Second),
		},
	}

	// 1. Bob (member) requests messages of Alice's conversation -> 403 Forbidden
	respBob := doJSON(t, srv, http.MethodGet, "/v1/conversations/"+convAlice.String()+"/messages", nil, tenantBearer(st, orgID, userBob, domain.RoleMember))
	requireEqual(t, respBob.StatusCode, http.StatusForbidden)

	// 2. Alice (owner) requests conversation messages under METADATA_ONLY -> 200 OK with redacted content
	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: id, PrivacyDefault: domain.PrivacyMetadataOnly}, nil
	}

	respAlice := doJSON(t, srv, http.MethodGet, "/v1/conversations/"+convAlice.String()+"/messages", nil, tenantBearer(st, orgID, userAlice, domain.RoleOwner))
	requireEqual(t, respAlice.StatusCode, http.StatusOK)

	var metaResp struct {
		Items []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			Tokens  int32  `json:"tokens"`
		} `json:"items"`
	}
	decodeBody(t, respAlice, &metaResp)
	if len(metaResp.Items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(metaResp.Items))
	}
	// Under METADATA_ONLY, all turn content MUST be redacted to empty string while tokens remain visible
	for _, m := range metaResp.Items {
		if m.Content != "" {
			t.Errorf("expected empty content under METADATA_ONLY, got %q", m.Content)
		}
		if m.Tokens == 0 {
			t.Errorf("expected non-zero token count under METADATA_ONLY")
		}
	}

	// 3. Organization with PROMPT_ONLY -> system & user visible, assistant redacted
	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: id, PrivacyDefault: domain.PrivacyPromptOnly}, nil
	}
	respPrompt := doJSON(t, srv, http.MethodGet, "/v1/conversations/"+convAlice.String()+"/messages", nil, tenantBearer(st, orgID, userAlice, domain.RoleOwner))
	requireEqual(t, respPrompt.StatusCode, http.StatusOK)

	var promptResp struct {
		Items []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"items"`
	}
	decodeBody(t, respPrompt, &promptResp)
	if promptResp.Items[0].Content != "Can we store credentials in plaintext?" {
		t.Errorf("expected user prompt to be visible under PROMPT_ONLY, got %q", promptResp.Items[0].Content)
	}
	if promptResp.Items[1].Content != "" {
		t.Errorf("expected assistant turn to be redacted under PROMPT_ONLY, got %q", promptResp.Items[1].Content)
	}

	// 4. Organization with FULL_CONTENT -> all turns visible
	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: id, PrivacyDefault: domain.PrivacyFullContent}, nil
	}
	respFull := doJSON(t, srv, http.MethodGet, "/v1/conversations/"+convAlice.String()+"/messages", nil, tenantBearer(st, orgID, userAlice, domain.RoleOwner))
	requireEqual(t, respFull.StatusCode, http.StatusOK)

	var fullResp struct {
		Items []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"items"`
	}
	decodeBody(t, respFull, &fullResp)
	if fullResp.Items[1].Content != "No, credentials must always be hashed with SHA-256." {
		t.Errorf("expected assistant turn visible under FULL_CONTENT, got %q", fullResp.Items[1].Content)
	}
}

// ----------------------------------------------------------------------------
// C. Usage & Local Billing Accounting
// ----------------------------------------------------------------------------

func TestBilling_UsageAccountingAndInvoiceTotals(t *testing.T) {
	st := newControlStubs()
	plansRepo := &fakeBillingPlans{
		plans: []domain.Plan{
			{ID: localPlanID, Code: "developer", Name: "Developer Tier", Status: "active"},
			{ID: uuid.New(), Code: "scale", Name: "Scale Tier", Status: "active"},
		},
	}
	subsRepo := &fakeBillingSubs{
		plan: domain.OrganizationPlan{
			PlanCode: "developer",
			PlanName: "Developer Tier",
			Status:   "active",
		},
	}
	invsRepo := &fakeBillingInvoices{
		invoices: []domain.Invoice{
			{
				ID:          uuid.New(),
				Number:      "INV-2026-001",
				Status:      domain.InvoiceStatusOpen,
				Currency:    "usd",
				TotalMicros: 45000000, // $45.00
				PeriodStart: testNow.Add(-30 * 24 * time.Hour),
				PeriodEnd:   testNow,
				IssuedAt:    testNow,
				Lines: []domain.InvoiceLine{
					{
						ID:           uuid.New(),
						Description:  "Claude Sonnet 4.5 Tokens",
						Quantity:     2000000, // 2M tokens
						UnitMicros:   15,      // $15/M
						AmountMicros: 30000000,
					},
					{
						ID:           uuid.New(),
						Description:  "GPT-4o Tokens",
						Quantity:     3000000, // 3M tokens
						UnitMicros:   5,       // $5/M
						AmountMicros: 15000000,
					},
				},
			},
		},
	}

	convRepo := &fakeConversationRepo{}
	msgRepo := &fakeMessageRepo{}
	alRepo := &fakeAlertRepo{}
	audRepo := &fakeAuditRepo{}

	srv := fullMultiTenantServer(t, st, convRepo, msgRepo, alRepo, audRepo, plansRepo, subsRepo, invsRepo)
	defer srv.Close()

	ownerHeaders := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	// 1. Get Subscription
	respSub := doJSON(t, srv, http.MethodGet, "/v1/billing/subscription", nil, ownerHeaders)
	requireEqual(t, respSub.StatusCode, http.StatusOK)

	var subResp subscriptionDTO
	decodeBody(t, respSub, &subResp)
	if subResp.PlanCode != "developer" || subResp.Status != "active" {
		t.Fatalf("unexpected subscription: %+v", subResp)
	}

	// 2. Change subscription plan
	respUp := doJSON(t, srv, http.MethodPut, "/v1/billing/subscription", map[string]any{"plan_code": "scale"}, ownerHeaders)
	requireEqual(t, respUp.StatusCode, http.StatusOK)

	// 3. List Invoices and verify exact integer micros arithmetic
	respInvs := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, ownerHeaders)
	requireEqual(t, respInvs.StatusCode, http.StatusOK)

	var invsResp struct {
		Items []invoiceDTO `json:"items"`
	}
	decodeBody(t, respInvs, &invsResp)
	if len(invsResp.Items) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(invsResp.Items))
	}
	inv := invsResp.Items[0]
	if inv.Currency != "usd" || inv.TotalMicros != 45000000 {
		t.Fatalf("unexpected invoice totals: %+v", inv)
	}

	var sumLines int64
	for _, l := range inv.Lines {
		sumLines += l.AmountMicros
	}
	if sumLines != inv.TotalMicros {
		t.Fatalf("invoice lines sum (%d) does not match total_micros (%d)", sumLines, inv.TotalMicros)
	}
}
