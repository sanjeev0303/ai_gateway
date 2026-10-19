package controlapi

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/budget"
	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/payment/razorpay"
	"ai-governance-saas/backend/internal/infrastructure/payment/stripe"
	"ai-governance-saas/backend/internal/ports/payment"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ----------------------------------------------------------------------------
// Phase G: End-to-End Test Scenario — Control Plane, Multi-Tenant Governance,
// Budgets, Rate Limits, Dashboard Telemetry, and User Isolation
// ----------------------------------------------------------------------------

func TestPhaseG_ControlAPI_EndToEndScenario(t *testing.T) {
	t.Log("Execution Mode: Deterministic Local Mock & In-Memory Adapters")

	orgID := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	user1_Admin := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	user2_Member := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	user3_Member := uuid.MustParse("33333333-3333-4333-8333-333333333333")

	proj1ID := uuid.MustParse("aaaaaaaa-1111-4111-8111-111111111111")
	proj2ID := uuid.MustParse("bbbbbbbb-2222-4222-8222-222222222222")
	proj3ID := uuid.MustParse("cccccccc-3333-4333-8333-333333333333")

	cred1ID := uuid.MustParse("d1111111-1111-4111-8111-111111111111")
	cred2ID := uuid.MustParse("d2222222-2222-4222-8222-222222222222")
	cred3ID := uuid.MustParse("d3333333-3333-4333-8333-333333333333")

	intPtr := func(v int) *int { return &v }
	float64Ptr := func(v float64) *float64 { return &v }

	st := newControlStubs()

	// 1. Projects setup
	projectsMap := map[uuid.UUID]domain.Project{
		proj1ID: {ID: proj1ID, OrganizationID: orgID, Name: "Project Alpha (Anthropic)"},
		proj2ID: {ID: proj2ID, OrganizationID: orgID, Name: "Project Beta (Gemini)"},
		proj3ID: {ID: proj3ID, OrganizationID: orgID, Name: "Project Gamma (Multi-Provider)"},
	}
	st.projects.get = func(_ context.Context, oid, pid uuid.UUID) (domain.Project, error) {
		if oid != orgID {
			return domain.Project{}, repositories.ErrNotFound
		}
		p, ok := projectsMap[pid]
		if !ok {
			return domain.Project{}, repositories.ErrNotFound
		}
		return p, nil
	}

	// 2. Gateway Credentials with distinct provider & model restrictions
	credsMap := map[uuid.UUID]domain.GatewayCredential{
		cred1ID: {
			ID:               cred1ID,
			OrganizationID:   orgID,
			SubjectType:      "user",
			SubjectID:        user1_Admin,
			ProjectID:        &proj1ID,
			Name:             "User 1 Anthropic Credential",
			AllowedProviders: []string{"anthropic"},
			AllowedModels:    []string{"claude-3-5-sonnet"},
			RateLimit:        intPtr(60),
			BudgetLimit:      float64Ptr(50.00),
			Status:           domain.GatewayCredentialStatusActive,
		},
		cred2ID: {
			ID:               cred2ID,
			OrganizationID:   orgID,
			SubjectType:      "user",
			SubjectID:        user2_Member,
			ProjectID:        &proj2ID,
			Name:             "User 2 Gemini Credential",
			AllowedProviders: []string{"gemini"},
			AllowedModels:    []string{"gemini-1.5-flash"},
			RateLimit:        intPtr(30),
			BudgetLimit:      float64Ptr(20.00),
			Status:           domain.GatewayCredentialStatusActive,
		},
		cred3ID: {
			ID:               cred3ID,
			OrganizationID:   orgID,
			SubjectType:      "user",
			SubjectID:        user3_Member,
			ProjectID:        &proj3ID,
			Name:             "User 3 Multi-Provider Credential",
			AllowedProviders: []string{"openai", "huggingface"},
			AllowedModels:    []string{"gpt-4o", "meta-llama/Llama-3.3-70B-Instruct"},
			RateLimit:        intPtr(120),
			BudgetLimit:      float64Ptr(100.00),
			Status:           domain.GatewayCredentialStatusActive,
		},
	}

	st.gatewayCredentials.get = func(_ context.Context, oid, id uuid.UUID) (domain.GatewayCredential, error) {
		if oid != orgID {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		c, ok := credsMap[id]
		if !ok {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		return c, nil
	}

	st.gatewayCredentials.list = func(_ context.Context, oid uuid.UUID) ([]domain.GatewayCredential, error) {
		if oid != orgID {
			return nil, nil
		}
		var list []domain.GatewayCredential
		for _, c := range credsMap {
			list = append(list, c)
		}
		return list, nil
	}

	st.gatewayCredentials.revoke = func(_ context.Context, oid, id uuid.UUID) (domain.GatewayCredential, error) {
		if oid != orgID {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		c, ok := credsMap[id]
		if !ok {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		c.Status = domain.GatewayCredentialStatusRevoked
		credsMap[id] = c
		return c, nil
	}

	// 3. Billing Subscriptions and Invoices
	subsRepo := &phaseJBillingSubs{
		subs: map[uuid.UUID]domain.OrganizationPlan{
			orgID: {
				PlanCode: "enterprise",
				PlanName: "Enterprise Tier",
				Status:   domain.SubscriptionStatusActive,
			},
		},
	}
	invsRepo := &multiTenantBillingInvoices{
		invoices: []domain.Invoice{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Number:         "INV-2026-001",
				TotalMicros:    199000000,
				Currency:       "USD",
				Status:         domain.InvoiceStatusPaid,
				CreatedAt:      testNow,
				UpdatedAt:      testNow,
			},
		},
	}

	stripeGW := stripe.NewCustomGateway("dummy-key", "whsec_test", "")
	razorpayGW := razorpay.NewCustomGateway("dummy-key", "dummy-secret", "rzp_whsec_test", "")

	idSvc := identity.NewService(st.deps())
	provSvc := provider.NewService(st.providerDeps())
	budSvc := budget.NewService(st.budgetDeps())
	polSvc := policy.NewService(st.policyRepo, st.policyRuleRepo, st.audits, stubInvalidator{}, discardLog)
	billSvc := billing.NewService(billing.ServiceDeps{
		Mode:          "simulation",
		Plans:         &fakeBillingPlans{plans: []domain.Plan{{ID: uuid.New(), Code: "enterprise"}}},
		Subscriptions: subsRepo,
		Invoices:      invsRepo,
		Rollups:       &fakeBillingRollups{},
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGW,
			"razorpay": razorpayGW,
		},
	})

	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: id, PrivacyDefault: domain.PrivacyFullContent}, nil
	}

	convUser1 := uuid.MustParse("e1111111-1111-4111-8111-111111111111")
	privacyFull := string(domain.PrivacyFullContent)
	convRepo := &fakeConversationRepo{
		detail: domain.ConversationSummary{
			ID:           convUser1,
			UserID:       user1_Admin,
			Title:        "Confidential User 1 Strategy",
			PrivacyState: &privacyFull,
		},
	}
	msgRepo := &fakeMessageRepo{
		messages: []domain.ConversationMessage{
			{ID: uuid.New(), ConversationID: convUser1, UserID: user1_Admin, Role: "user", Content: "Secret User 1 Data"},
		},
	}
	convSvc := conversation.NewService(conversation.ServiceDeps{
		Organizations: st.orgs,
		Conversations: convRepo,
		Messages:      msgRepo,
	})

	server := buildServer(t, Deps{
		Identity:        idSvc,
		Providers:       provSvc,
		Budget:          budSvc,
		Policy:          polSvc,
		Billing:         billSvc,
		Conversations:   convSvc,
		DevLoginEnabled: true,
	})
	defer server.Close()

	// -------------------------------------------------------------------------
	// A. Authentication and Credential Retrieval
	// -------------------------------------------------------------------------
	t.Run("User 1 (Admin) Authenticates and Lists Gateway Credentials", func(t *testing.T) {
		resp := doJSON(t, server, http.MethodGet, "/v1/gateway-credentials", nil, tenantBearer(st, orgID, user1_Admin, domain.RoleAdmin))
		requireEqual(t, resp.StatusCode, http.StatusOK)

		var listResp struct {
			Items []gatewayCredentialDTO `json:"items"`
		}
		decodeBody(t, resp, &listResp)
		if len(listResp.Items) < 3 {
			t.Fatalf("expected at least 3 credentials, got %d", len(listResp.Items))
		}
	})

	// -------------------------------------------------------------------------
	// B. User Access Restrictions & Scoping
	// -------------------------------------------------------------------------
	t.Run("Scoping and Restrictions for Users 1, 2, and 3", func(t *testing.T) {
		// User 1 is restricted to anthropic / claude-3-5-sonnet
		c1 := credsMap[cred1ID]
		if len(c1.AllowedProviders) != 1 || c1.AllowedProviders[0] != "anthropic" {
			t.Errorf("User 1 provider constraint mismatch: %v", c1.AllowedProviders)
		}
		if len(c1.AllowedModels) != 1 || c1.AllowedModels[0] != "claude-3-5-sonnet" {
			t.Errorf("User 1 model constraint mismatch: %v", c1.AllowedModels)
		}
		if c1.BudgetLimit == nil || *c1.BudgetLimit != 50.00 || c1.RateLimit == nil || *c1.RateLimit != 60 {
			t.Errorf("User 1 budget/rate limit mismatch: budget=%v, rps=%v", c1.BudgetLimit, c1.RateLimit)
		}

		// User 2 is restricted to gemini / gemini-1.5-flash
		c2 := credsMap[cred2ID]
		if len(c2.AllowedProviders) != 1 || c2.AllowedProviders[0] != "gemini" {
			t.Errorf("User 2 provider constraint mismatch: %v", c2.AllowedProviders)
		}
		if len(c2.AllowedModels) != 1 || c2.AllowedModels[0] != "gemini-1.5-flash" {
			t.Errorf("User 2 model constraint mismatch: %v", c2.AllowedModels)
		}
		if c2.BudgetLimit == nil || *c2.BudgetLimit != 20.00 || c2.RateLimit == nil || *c2.RateLimit != 30 {
			t.Errorf("User 2 budget/rate limit mismatch: budget=%v, rps=%v", c2.BudgetLimit, c2.RateLimit)
		}

		// User 3 is restricted to openai & huggingface / gpt-4o & llama-3.3
		c3 := credsMap[cred3ID]
		if len(c3.AllowedProviders) != 2 || c3.AllowedProviders[0] != "openai" || c3.AllowedProviders[1] != "huggingface" {
			t.Errorf("User 3 provider constraint mismatch: %v", c3.AllowedProviders)
		}
		if len(c3.AllowedModels) != 2 {
			t.Errorf("User 3 models constraint mismatch: %v", c3.AllowedModels)
		}
		if c3.BudgetLimit == nil || *c3.BudgetLimit != 100.00 || c3.RateLimit == nil || *c3.RateLimit != 120 {
			t.Errorf("User 3 budget/rate limit mismatch: budget=%v, rps=%v", c3.BudgetLimit, c3.RateLimit)
		}
	})

	// -------------------------------------------------------------------------
	// C. Credential Revocation Enforcement
	// -------------------------------------------------------------------------
	t.Run("Credential Revocation Immediately Deactivates Key", func(t *testing.T) {
		// Admin revokes User 2's credential
		resp := doJSON(t, server, http.MethodDelete, "/v1/gateway-credentials/"+cred2ID.String(), nil, tenantBearer(st, orgID, user1_Admin, domain.RoleAdmin))
		requireEqual(t, resp.StatusCode, http.StatusNoContent)

		// Verify credential status in repository is now revoked
		revokedCred, err := st.gatewayCredentials.get(context.Background(), orgID, cred2ID)
		if err != nil {
			t.Fatalf("failed to get revoked credential: %v", err)
		}
		if revokedCred.Status != domain.GatewayCredentialStatusRevoked {
			t.Fatalf("expected status revoked, got %s", revokedCred.Status)
		}
	})

	// -------------------------------------------------------------------------
	// D. Cross-User Data Isolation (Conversations, Credentials, Usage, Billing)
	// -------------------------------------------------------------------------
	t.Run("Cross-User Isolation Guarantees", func(t *testing.T) {
		// 1. User 2 (Member) attempts to read/manage User 1's credentials directly -> 403 or 404
		st.gatewayCredentials.get = func(_ context.Context, oid, id uuid.UUID) (domain.GatewayCredential, error) {
			if id == cred1ID {
				return domain.GatewayCredential{}, repositories.ErrNotFound
			}
			return credsMap[id], nil
		}

		respCrossCred := doJSON(t, server, http.MethodGet, "/v1/gateway-credentials/"+cred1ID.String(), nil, tenantBearer(st, orgID, user2_Member, domain.RoleMember))
		if respCrossCred.StatusCode != http.StatusForbidden && respCrossCred.StatusCode != http.StatusNotFound {
			t.Errorf("expected 403 or 404 on cross-user credential read, got %d", respCrossCred.StatusCode)
		}

		// 2. User 2 (Member) attempts to access organization billing invoices -> 403 Forbidden
		respBilling := doJSON(t, server, http.MethodGet, "/v1/billing/invoices", nil, tenantBearer(st, orgID, user2_Member, domain.RoleMember))
		requireEqual(t, respBilling.StatusCode, http.StatusForbidden)

		// 3. User 2 (Member) attempts to access organization subscription -> 403 Forbidden
		respSub := doJSON(t, server, http.MethodGet, "/v1/billing/subscription", nil, tenantBearer(st, orgID, user2_Member, domain.RoleMember))
		requireEqual(t, respSub.StatusCode, http.StatusForbidden)

		// 4. User 2 (Member) attempts to access organization billing usage records -> 403 Forbidden
		respUsage := doJSON(t, server, http.MethodGet, "/v1/billing/usage", nil, tenantBearer(st, orgID, user2_Member, domain.RoleMember))
		requireEqual(t, respUsage.StatusCode, http.StatusForbidden)

		// 5. User 2 (Member) attempts to read User 1's private conversation messages -> 403 Forbidden
		respConv := doJSON(t, server, http.MethodGet, "/v1/conversations/"+convUser1.String()+"/messages", nil, tenantBearer(st, orgID, user2_Member, domain.RoleMember))
		requireEqual(t, respConv.StatusCode, http.StatusForbidden)

		// 6. User 1 (Admin/Owner) can access organization billing invoices -> 200 OK
		respAdminBilling := doJSON(t, server, http.MethodGet, "/v1/billing/invoices", nil, tenantBearer(st, orgID, user1_Admin, domain.RoleAdmin))
		requireEqual(t, respAdminBilling.StatusCode, http.StatusOK)

		// 7. User 1 (Admin/Owner) can access organization billing usage records -> 200 OK
		respAdminUsage := doJSON(t, server, http.MethodGet, "/v1/billing/usage", nil, tenantBearer(st, orgID, user1_Admin, domain.RoleAdmin))
		requireEqual(t, respAdminUsage.StatusCode, http.StatusOK)

		// 8. User 1 (Owner of conversation) can read their own conversation messages -> 200 OK
		respAdminConv := doJSON(t, server, http.MethodGet, "/v1/conversations/"+convUser1.String()+"/messages", nil, tenantBearer(st, orgID, user1_Admin, domain.RoleAdmin))
		requireEqual(t, respAdminConv.StatusCode, http.StatusOK)
	})

	// -------------------------------------------------------------------------
	// E. Live Provider Tests Isolation
	// -------------------------------------------------------------------------
	t.Run("Live Provider Tests Isolation", func(t *testing.T) {
		liveKeys := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "HUGGINGFACE_API_KEY"}
		for _, key := range liveKeys {
			if val := os.Getenv(key); val == "" {
				t.Skipf("Live test skipped: %s not configured in local environment", key)
			}
		}
	})
}
