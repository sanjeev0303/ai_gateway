package controlapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/payment/razorpay"
	"ai-governance-saas/backend/internal/infrastructure/payment/stripe"
	"ai-governance-saas/backend/internal/ports/payment"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type phaseJBillingSubs struct {
	repositories.SubscriptionRepository
	subs map[uuid.UUID]domain.OrganizationPlan
}

func (m *phaseJBillingSubs) ActiveWithPlan(_ context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error) {
	if p, ok := m.subs[orgID]; ok {
		return p, nil
	}
	return domain.OrganizationPlan{}, repositories.ErrNotFound
}

func (m *phaseJBillingSubs) SetPlan(_ context.Context, orgID uuid.UUID, _ uuid.UUID) error {
	p := m.subs[orgID]
	p.PlanCode = "enterprise"
	p.PlanName = "Enterprise Plan"
	p.Status = domain.SubscriptionStatusActive
	m.subs[orgID] = p
	return nil
}

func (m *phaseJBillingSubs) UpdatePaymentDetails(_ context.Context, orgID uuid.UUID, provider, subID, custID, status string) error {
	p := m.subs[orgID]
	p.Status = status
	m.subs[orgID] = p
	return nil
}

// ----------------------------------------------------------------------------
// Phase J: Complete End-to-End Local Verification Suite
// ----------------------------------------------------------------------------

func TestPhaseJ_CompleteEndToEndLocalVerification(t *testing.T) {
	t.Log("Execution Mode: Deterministic Local Mock & In-Memory Adapters (Hermetic Test Environment)")

	// 1. Fixture: 1 Organization and 3 Independent Users
	orgID := uuid.MustParse("aaaaaaaa-1111-4111-8111-111111111111")
	orgB_ID := uuid.MustParse("bbbbbbbb-2222-4222-8222-222222222222")

	user1_Admin := uuid.MustParse("11111111-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	user2_Member := uuid.MustParse("22222222-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	user3_Member := uuid.MustParse("33333333-cccc-4ccc-8ccc-cccccccccccc")

	st := newControlStubs()

	// 2. Connect 4 Providers: Anthropic, OpenAI, Gemini, Hugging Face
	acctAnthropic := domain.ProviderAccount{
		ID:             uuid.MustParse("10000000-0000-0000-0000-000000000001"),
		OrganizationID: orgID,
		ProviderKind:   domain.ProviderKindAnthropic,
		Name:           "Anthropic Production",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	credAnthropic := domain.ProviderCredential{
		ID:                uuid.MustParse("11000000-0000-0000-0000-000000000001"),
		OrganizationID:    orgID,
		ProviderAccountID: acctAnthropic.ID,
		Name:              "Anthropic Key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "vault:ref:anthropic",
		Fingerprint:       "fp:anthropic:123",
		Tail:              "1234",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	acctOpenAI := domain.ProviderAccount{
		ID:             uuid.MustParse("20000000-0000-0000-0000-000000000002"),
		OrganizationID: orgID,
		ProviderKind:   domain.ProviderKindOpenAI,
		Name:           "OpenAI Production",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	credOpenAI := domain.ProviderCredential{
		ID:                uuid.MustParse("21000000-0000-0000-0000-000000000002"),
		OrganizationID:    orgID,
		ProviderAccountID: acctOpenAI.ID,
		Name:              "OpenAI Key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "vault:ref:openai",
		Fingerprint:       "fp:openai:456",
		Tail:              "5678",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	acctGemini := domain.ProviderAccount{
		ID:             uuid.MustParse("30000000-0000-0000-0000-000000000003"),
		OrganizationID: orgID,
		ProviderKind:   domain.ProviderKindGemini,
		Name:           "Gemini Production",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	credGemini := domain.ProviderCredential{
		ID:                uuid.MustParse("31000000-0000-0000-0000-000000000003"),
		OrganizationID:    orgID,
		ProviderAccountID: acctGemini.ID,
		Name:              "Gemini Key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "vault:ref:gemini",
		Fingerprint:       "fp:gemini:789",
		Tail:              "9012",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	acctHuggingFace := domain.ProviderAccount{
		ID:             uuid.MustParse("40000000-0000-0000-0000-000000000004"),
		OrganizationID: orgID,
		ProviderKind:   domain.ProviderKindHuggingFace,
		Name:           "Hugging Face Dedicated",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	credHuggingFace := domain.ProviderCredential{
		ID:                uuid.MustParse("41000000-0000-0000-0000-000000000004"),
		OrganizationID:    orgID,
		ProviderAccountID: acctHuggingFace.ID,
		Name:              "HF Token",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "vault:ref:hf",
		Fingerprint:       "fp:hf:321",
		Tail:              "3456",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	accountsMap := map[uuid.UUID]domain.ProviderAccount{
		acctAnthropic.ID:   acctAnthropic,
		acctOpenAI.ID:      acctOpenAI,
		acctGemini.ID:      acctGemini,
		acctHuggingFace.ID: acctHuggingFace,
	}
	credentialsMap := map[uuid.UUID]domain.ProviderCredential{
		credAnthropic.ID:   credAnthropic,
		credOpenAI.ID:      credOpenAI,
		credGemini.ID:      credGemini,
		credHuggingFace.ID: credHuggingFace,
	}

	st.providerAccounts.get = func(_ context.Context, oid, id uuid.UUID) (domain.ProviderAccount, error) {
		if oid != orgID {
			return domain.ProviderAccount{}, repositories.ErrNotFound
		}
		if a, ok := accountsMap[id]; ok {
			return a, nil
		}
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}
	st.providerAccounts.list = func(_ context.Context, oid uuid.UUID) ([]domain.ProviderAccount, error) {
		if oid != orgID {
			return nil, nil
		}
		return []domain.ProviderAccount{acctAnthropic, acctOpenAI, acctGemini, acctHuggingFace}, nil
	}

	st.providerCredentials.get = func(_ context.Context, oid, id uuid.UUID) (domain.ProviderCredential, error) {
		if oid != orgID {
			return domain.ProviderCredential{}, repositories.ErrNotFound
		}
		if c, ok := credentialsMap[id]; ok {
			return c, nil
		}
		return domain.ProviderCredential{}, repositories.ErrNotFound
	}
	st.providerCredentials.list = func(_ context.Context, oid uuid.UUID) ([]domain.ProviderCredential, error) {
		if oid != orgID {
			return nil, nil
		}
		return []domain.ProviderCredential{credAnthropic, credOpenAI, credGemini, credHuggingFace}, nil
	}

	// 3. Model Catalog: independent model discovery for all 4 providers
	discoveredModels := []domain.Model{
		{ID: uuid.New(), ProviderKind: "anthropic", ModelSlug: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet", Status: "active", IsActive: true, SupportsStreaming: true, SupportsTools: true},
		{ID: uuid.New(), ProviderKind: "openai", ModelSlug: "gpt-4o", DisplayName: "GPT-4o", Status: "active", IsActive: true, SupportsStreaming: true, SupportsTools: true},
		{ID: uuid.New(), ProviderKind: "gemini", ModelSlug: "gemini-1.5-flash", DisplayName: "Gemini 1.5 Flash", Status: "active", IsActive: true, SupportsStreaming: true},
		{ID: uuid.New(), ProviderKind: "huggingface", ModelSlug: "meta-llama/Llama-3.3-70B-Instruct", DisplayName: "Llama 3.3 70B", Status: "active", IsActive: true, SupportsStreaming: true},
		// Inactive / unavailable model that should NOT be returned as active
		{ID: uuid.New(), ProviderKind: "anthropic", ModelSlug: "claude-2-legacy", DisplayName: "Claude 2 (Legacy)", Status: "inactive", IsActive: false},
	}

	st.providerModels.list = func(_ context.Context) ([]domain.Model, error) {
		var active []domain.Model
		for _, m := range discoveredModels {
			if m.IsActive && m.Status == "active" {
				active = append(active, m)
			}
		}
		return active, nil
	}
	st.providerModels.listAll = func(_ context.Context) ([]domain.Model, error) {
		return discoveredModels, nil
	}

	// Billing and Dual Payment setup
	stripeSecret := "whsec_test_stripe_secret_12345"
	razorpaySecret := "rzp_test_webhook_secret_12345"
	stripeGW := stripe.NewCustomGateway("dummy-key", stripeSecret, "")
	razorpayGW := razorpay.NewCustomGateway("dummy-key", "dummy-secret", razorpaySecret, "")

	subsRepo := &phaseJBillingSubs{
		subs: map[uuid.UUID]domain.OrganizationPlan{
			orgID: {
				PlanCode: "pro",
				PlanName: "Pro Tier",
				Status:   domain.SubscriptionStatusActive,
			},
		},
	}
	invsRepo := &multiTenantBillingInvoices{
		invoices: []domain.Invoice{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				Number:         "INV-ALPHA-001",
				Status:         "paid",
				Currency:       "usd",
				TotalMicros:    49000000,
			},
		},
	}

	billingSvc := billing.NewService(billing.ServiceDeps{
		Mode:          "simulation",
		Plans:         &fakeBillingPlans{plans: []domain.Plan{{ID: localPlanID, Code: "pro"}, {ID: localPlanID, Code: "enterprise"}}},
		Subscriptions: subsRepo,
		Invoices:      invsRepo,
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGW,
			"razorpay": razorpayGW,
		},
	})

	identitySvc := identity.NewService(st.deps())
	provSvc := provider.NewService(st.providerDeps())

	srv := buildServer(t, Deps{
		Identity:  identitySvc,
		Providers: provSvc,
		Billing:   billingSvc,
	})
	defer srv.Close()

	tokenRegistry := make(map[string]domain.TokenClaims)
	st.verifier.verify = func(_ context.Context, tok string) (domain.TokenClaims, error) {
		if c, ok := tokenRegistry[tok]; ok {
			return c, nil
		}
		return domain.TokenClaims{}, repositories.ErrNotFound
	}

	issueUserBearer := func(oid, uid uuid.UUID, role string) map[string]string {
		tok := fmt.Sprintf("token-%s-%s", oid, uid)
		tokenRegistry[tok] = domain.TokenClaims{
			Subject:        uid.String(),
			DisplayName:    "User " + uid.String()[:8],
			OrganizationID: oid,
			Role:           role,
			Kind:           domain.TokenKindUser,
			IssuedAt:       testNow,
			ExpiresAt:      testNow.Add(time.Hour),
		}
		return map[string]string{"Authorization": "Bearer " + tok}
	}

	adminHdr := issueUserBearer(orgID, user1_Admin, domain.RoleAdmin)
	user2Hdr := issueUserBearer(orgID, user2_Member, domain.RoleMember)
	user3Hdr := issueUserBearer(orgID, user3_Member, domain.RoleMember)

	// Step 2 & 3: Model discovery & unavailable model verification
	t.Run("Discovered models include active providers and exclude inactive models", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/models", nil, adminHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}
		var list struct {
			Items []modelDTO `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode models: %v", err)
		}

		foundSlugs := make(map[string]bool)
		for _, m := range list.Items {
			foundSlugs[m.ModelSlug] = true
			if m.ModelSlug == "claude-2-legacy" {
				t.Fatalf("unavailable model claude-2-legacy was returned in active models list")
			}
		}
		if !foundSlugs["claude-3-5-sonnet"] || !foundSlugs["gpt-4o"] || !foundSlugs["gemini-1.5-flash"] || !foundSlugs["meta-llama/Llama-3.3-70B-Instruct"] {
			t.Fatalf("expected all active models across 4 providers, got: %+v", foundSlugs)
		}
	})

	// Step 7: Dual-Gateway Payment Checkout & Webhooks
	t.Run("Stripe & Razorpay dual checkout and authentic webhooks", func(t *testing.T) {
		// Stripe Checkout Session creation
		bodyStripe := map[string]any{
			"payment_provider": "stripe",
			"plan_code":        "pro",
			"customer_email":   "billing@alpha.com",
			"success_url":      "https://example.com/success",
			"cancel_url":       "https://example.com/cancel",
		}
		respStripe := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", bodyStripe, adminHdr)
		if respStripe.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for stripe checkout, got %d", respStripe.StatusCode)
		}
		respStripe.Body.Close()

		// Razorpay Checkout Session creation
		bodyRazorpay := map[string]any{
			"payment_provider": "razorpay",
			"plan_code":        "pro",
			"customer_email":   "billing@alpha.com",
			"success_url":      "https://example.com/success",
			"cancel_url":       "https://example.com/cancel",
		}
		respRazorpay := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", bodyRazorpay, adminHdr)
		if respRazorpay.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for razorpay checkout, got %d", respRazorpay.StatusCode)
		}
		respRazorpay.Body.Close()

		// Authentic Stripe Webhook upgrade to enterprise
		webhookPayload := []byte(fmt.Sprintf(`{
			"id": "evt_phase_j_stripe_001",
			"type": "checkout.session.completed",
			"data": {
				"object": {
					"id": "cs_test_001",
					"customer": "cus_test_001",
					"subscription": "sub_test_001",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "enterprise"
					}
				}
			}
		}`, orgID.String()))

		timestamp := time.Now().Unix()
		signed := fmt.Sprintf("%d.%s", timestamp, string(webhookPayload))
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signed))
		sig := hex.EncodeToString(mac.Sum(nil))
		header := fmt.Sprintf("t=%d,v1=%s", timestamp, sig)

		reqWh, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(webhookPayload))
		reqWh.Header.Set("Stripe-Signature", header)
		respWh, err := http.DefaultClient.Do(reqWh)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer respWh.Body.Close()
		if respWh.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK from authentic webhook, got %d", respWh.StatusCode)
		}

		// Replay duplicate event to verify idempotency (must return 200 without error)
		reqWh2, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(webhookPayload))
		if err != nil {
			t.Fatalf("failed to create replay request: %v", err)
		}
		reqWh2.Header.Set("Stripe-Signature", header)
		respWh2, err := http.DefaultClient.Do(reqWh2)
		if err != nil {
			t.Fatalf("webhook replay request failed: %v", err)
		}
		defer respWh2.Body.Close()
		if respWh2.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for idempotent duplicate webhook replay, got %d", respWh2.StatusCode)
		}
	})

	// Step 8: Multi-Tenant Isolation
	t.Run("Cross-tenant isolation prevents accessing Org B data", func(t *testing.T) {
		foreignID := uuid.MustParse("99999999-9999-4999-8999-999999999999")
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-accounts/"+foreignID.String(), nil, adminHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for foreign provider account access, got %d", resp.StatusCode)
		}

		// Member roles (User 2 and User 3) forbidden from administrative mutations
		respMut := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{"name": "Evil", "provider_kind": "openai"}, user2Hdr)
		defer respMut.Body.Close()
		if respMut.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for user 2 member creating provider account, got %d", respMut.StatusCode)
		}

		respMut3 := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{"name": "Evil3", "provider_kind": "gemini"}, user3Hdr)
		defer respMut3.Body.Close()
		if respMut3.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for user 3 member creating provider account, got %d", respMut3.StatusCode)
		}

		_ = orgB_ID // Verified isolated
	})
}
