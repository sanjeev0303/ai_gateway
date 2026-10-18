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

type multiTenantBillingInvoices struct {
	repositories.InvoiceRepository
	invoices []domain.Invoice
}

func (m *multiTenantBillingInvoices) List(_ context.Context, orgID uuid.UUID) ([]domain.Invoice, error) {
	var res []domain.Invoice
	for _, inv := range m.invoices {
		if inv.OrganizationID == orgID {
			res = append(res, inv)
		}
	}
	return res, nil
}

type multiTenantBillingSubs struct {
	repositories.SubscriptionRepository
	subs map[uuid.UUID]domain.OrganizationPlan
}

func (m *multiTenantBillingSubs) ActiveWithPlan(_ context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error) {
	if p, ok := m.subs[orgID]; ok {
		return p, nil
	}
	return domain.OrganizationPlan{}, repositories.ErrNotFound
}

func TestSecurity_TenantIsolation_ProviderCredentials_CrossTenantAccess(t *testing.T) {
	st := newControlStubs()

	// Org A and Org B accounts and credentials
	acctA := domain.ProviderAccount{
		ID:             uuid.New(),
		OrganizationID: orgTenantA,
		ProviderKind:   domain.ProviderKindAnthropic,
		Name:           "Anthropic Org A",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	credA := domain.ProviderCredential{
		ID:                uuid.New(),
		OrganizationID:    orgTenantA,
		ProviderAccountID: acctA.ID,
		Name:              "Org A Key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "provider/creds/org-a-secret",
		Fingerprint:       "fp-org-a",
		Tail:              "1111",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	acctB := domain.ProviderAccount{
		ID:             uuid.New(),
		OrganizationID: orgTenantB,
		ProviderKind:   domain.ProviderKindOpenAI,
		Name:           "OpenAI Org B",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	credB := domain.ProviderCredential{
		ID:                uuid.New(),
		OrganizationID:    orgTenantB,
		ProviderAccountID: acctB.ID,
		Name:              "Org B Key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "provider/creds/org-b-secret",
		Fingerprint:       "fp-org-b",
		Tail:              "2222",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	st.providerAccounts.get = func(_ context.Context, orgID, acctID uuid.UUID) (domain.ProviderAccount, error) {
		if orgID != orgTenantB && acctID == acctB.ID {
			return domain.ProviderAccount{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && acctID == acctB.ID {
			return acctB, nil
		}
		if orgID == orgTenantA && acctID == acctA.ID {
			return acctA, nil
		}
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}
	st.providerAccounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
		if orgID == orgTenantA {
			return []domain.ProviderAccount{acctA}, nil
		}
		if orgID == orgTenantB {
			return []domain.ProviderAccount{acctB}, nil
		}
		return nil, nil
	}
	st.providerAccounts.delete = func(_ context.Context, orgID, acctID uuid.UUID) error {
		if orgID != orgTenantB && acctID == acctB.ID {
			return repositories.ErrNotFound
		}
		return nil
	}

	st.providerCredentials.get = func(_ context.Context, orgID, credID uuid.UUID) (domain.ProviderCredential, error) {
		if orgID != orgTenantB && credID == credB.ID {
			return domain.ProviderCredential{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && credID == credB.ID {
			return credB, nil
		}
		if orgID == orgTenantA && credID == credA.ID {
			return credA, nil
		}
		return domain.ProviderCredential{}, repositories.ErrNotFound
	}
	st.providerCredentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
		if orgID == orgTenantA {
			return []domain.ProviderCredential{credA}, nil
		}
		if orgID == orgTenantB {
			return []domain.ProviderCredential{credB}, nil
		}
		return nil, nil
	}
	st.providerCredentials.revoke = func(_ context.Context, orgID, credID uuid.UUID) (domain.ProviderCredential, error) {
		if orgID != orgTenantB && credID == credB.ID {
			return domain.ProviderCredential{}, repositories.ErrNotFound
		}
		return domain.ProviderCredential{ID: credID, Status: domain.ProviderCredentialStatusRevoked}, nil
	}

	_ = st.providerSecrets.Put(context.Background(), credA.SecretRef, "sk-ant-org-a-secret-key-1111")
	_ = st.providerSecrets.Put(context.Background(), credB.SecretRef, "sk-proj-org-b-secret-key-2222")

	provSvc := provider.NewService(st.providerDeps())
	identitySvc := identity.NewService(st.deps())

	srv := buildServer(t, Deps{
		Identity:  identitySvc,
		Providers: provSvc,
	})

	hdrA := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	t.Run("Org A owner cannot GET Org B provider account", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-accounts/"+acctB.ID.String(), nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for cross-tenant provider account GET, got %d", resp.StatusCode)
		}
	})

	t.Run("Org A owner cannot DELETE Org B provider account", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodDelete, "/v1/provider-accounts/"+acctB.ID.String(), nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for cross-tenant provider account DELETE, got %d", resp.StatusCode)
		}
	})

	t.Run("Org A owner cannot GET Org B provider credential", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials/"+credB.ID.String(), nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for cross-tenant provider credential GET, got %d", resp.StatusCode)
		}
	})

	t.Run("Org A owner cannot REVOKE Org B provider credential", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodDelete, "/v1/provider-credentials/"+credB.ID.String(), nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for cross-tenant provider credential DELETE, got %d", resp.StatusCode)
		}
	})

	t.Run("Org A owner cannot ROTATE Org B provider credential", func(t *testing.T) {
		body := map[string]any{"secret": "sk-dummy-new-key"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/provider-credentials/"+credB.ID.String()+"/rotate", body, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for cross-tenant provider credential ROTATE, got %d", resp.StatusCode)
		}
	})

	t.Run("Org A owner cannot attach credential to Org B provider account", func(t *testing.T) {
		body := map[string]any{"name": "Tampered Key", "kind": "api_key", "secret": "sk-dummy-key"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+acctB.ID.String()+"/credentials", body, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 404 or 403 for cross-tenant credential creation under foreign account, got %d", resp.StatusCode)
		}
	})

	t.Run("Org A listing provider accounts only sees Org A accounts", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-accounts", nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}
		var list struct {
			Items []providerAccountDTO `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		for _, item := range list.Items {
			if item.ID == acctB.ID.String() {
				t.Fatalf("security violation: Org A account list leaked Org B account %s", acctB.ID)
			}
		}
	})

	t.Run("Org A listing provider credentials only sees Org A credentials", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials", nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}
		var list struct {
			Items []providerCredentialDTO `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		for _, item := range list.Items {
			if item.ID == credB.ID.String() {
				t.Fatalf("security violation: Org A credential list leaked Org B credential %s", credB.ID)
			}
		}
	})
}

func TestSecurity_TenantIsolation_BillingAndInvoices(t *testing.T) {
	st := newControlStubs()

	subA := domain.OrganizationPlan{
		PlanCode: "pro",
		PlanName: "Pro Plan",
		Status:   domain.SubscriptionStatusActive,
	}
	subB := domain.OrganizationPlan{
		PlanCode: "enterprise",
		PlanName: "Enterprise Plan",
		Status:   domain.SubscriptionStatusActive,
	}
	subs := &multiTenantBillingSubs{
		subs: map[uuid.UUID]domain.OrganizationPlan{
			orgTenantA: subA,
			orgTenantB: subB,
		},
	}

	invA := domain.Invoice{
		ID:             uuid.New(),
		OrganizationID: orgTenantA,
		Number:         "INV-ORG-A-001",
		Status:         "paid",
		Currency:       "usd",
		TotalMicros:    49000000,
	}
	invB := domain.Invoice{
		ID:             uuid.New(),
		OrganizationID: orgTenantB,
		Number:         "INV-ORG-B-999",
		Status:         "paid",
		Currency:       "usd",
		TotalMicros:    499000000,
	}
	invs := &multiTenantBillingInvoices{
		invoices: []domain.Invoice{invA, invB},
	}

	billingSvc := billing.NewService(billing.ServiceDeps{
		Plans:         &fakeBillingPlans{plans: []domain.Plan{{ID: localPlanID, Code: "pro"}, {ID: localPlanID, Code: "enterprise"}}},
		Subscriptions: subs,
		Invoices:      invs,
	})
	identitySvc := identity.NewService(st.deps())

	srv := buildServer(t, Deps{
		Identity: identitySvc,
		Billing:  billingSvc,
	})

	hdrA := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	t.Run("Org A subscription endpoint returns only Org A subscription", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/billing/subscription", nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}
		var sub subscriptionDTO
		if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
			t.Fatalf("decode subscription: %v", err)
		}
		if sub.PlanCode != "pro" {
			t.Fatalf("expected Org A pro plan, got %s", sub.PlanCode)
		}
	})

	t.Run("Org A invoice list only returns Org A invoices and excludes Org B", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}
		var res struct {
			Items []invoiceDTO `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("decode invoices: %v", err)
		}
		for _, inv := range res.Items {
			if inv.Number == "INV-ORG-B-999" || inv.ID == invB.ID.String() {
				t.Fatalf("security violation: Org A saw Org B invoice %s", inv.Number)
			}
		}
	})
}

func TestSecurity_WebhookForgeryRejection(t *testing.T) {
	st := newControlStubs()

	stripeSecret := "whsec_secure_stripe_test_12345"
	razorpaySecret := "rzp_secure_webhook_test_12345"

	stripeGW := stripe.NewCustomGateway("dummy-key", stripeSecret, "")
	razorpayGW := razorpay.NewCustomGateway("dummy-id", "dummy-secret", razorpaySecret, "")

	billingSvc := billing.NewService(billing.ServiceDeps{
		Plans:         &fakeBillingPlans{plans: []domain.Plan{{ID: localPlanID, Code: "pro"}}},
		Subscriptions: &fakeBillingSubs{},
		Invoices:      &fakeBillingInvoices{},
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGW,
			"razorpay": razorpayGW,
		},
	})
	identitySvc := identity.NewService(st.deps())

	srv := buildServer(t, Deps{
		Identity: identitySvc,
		Billing:  billingSvc,
	})

	t.Run("Stripe webhook rejects request with missing signature header", func(t *testing.T) {
		payload := []byte(`{"id":"evt_forged","type":"checkout.session.completed"}`)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(payload))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing Stripe signature, got %d", resp.StatusCode)
		}
	})

	t.Run("Stripe webhook rejects request with forged signature header", func(t *testing.T) {
		payload := []byte(`{"id":"evt_forged","type":"checkout.session.completed"}`)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", "t=1700000000,v1=bad_forged_signature_value")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for forged Stripe signature, got %d", resp.StatusCode)
		}
	})

	t.Run("Razorpay webhook rejects request with missing signature header", func(t *testing.T) {
		payload := []byte(`{"event":"payment.captured"}`)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/razorpay", bytes.NewReader(payload))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for missing Razorpay signature, got %d", resp.StatusCode)
		}
	})

	t.Run("Razorpay webhook rejects request with forged signature header", func(t *testing.T) {
		payload := []byte(`{"event":"payment.captured"}`)
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/razorpay", bytes.NewReader(payload))
		req.Header.Set("X-Razorpay-Signature", "bad_forged_razorpay_signature")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for forged Razorpay signature, got %d", resp.StatusCode)
		}
	})

	t.Run("Authentic Stripe signature succeeds", func(t *testing.T) {
		payload := []byte(fmt.Sprintf(`{
			"id": "evt_valid_stripe",
			"type": "checkout.session.completed",
			"data": {
				"object": {
					"id": "cs_valid_100",
					"customer": "cus_valid_100",
					"subscription": "sub_valid_100",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "pro"
					}
				}
			}
		}`, orgTenantA.String()))

		timestamp := time.Now().Unix()
		signed := fmt.Sprintf("%d.%s", timestamp, string(payload))
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signed))
		sig := hex.EncodeToString(mac.Sum(nil))
		header := fmt.Sprintf("t=%d,v1=%s", timestamp, sig)

		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", header)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for authentic Stripe signature, got %d", resp.StatusCode)
		}
	})
}

func TestSecurity_RBAC_ProviderAndBillingManagement(t *testing.T) {
	st := newControlStubs()

	provSvc := provider.NewService(st.providerDeps())
	billingSvc := billing.NewService(billing.ServiceDeps{
		Plans:         &fakeBillingPlans{plans: []domain.Plan{{ID: localPlanID, Code: "pro"}}},
		Subscriptions: &fakeBillingSubs{},
		Invoices:      &fakeBillingInvoices{},
	})
	identitySvc := identity.NewService(st.deps())

	srv := buildServer(t, Deps{
		Identity:  identitySvc,
		Providers: provSvc,
		Billing:   billingSvc,
	})

	memberHdr := tenantBearer(st, orgTenantA, userTenantA_Member, domain.RoleMember)
	auditorHdr := tenantBearer(st, orgTenantA, userTenantA_Member, domain.RoleAuditor)

	t.Run("RoleMember cannot create provider account", func(t *testing.T) {
		body := map[string]any{"name": "Evil Provider", "provider_kind": "anthropic"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", body, memberHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for member creating provider account, got %d", resp.StatusCode)
		}
	})

	t.Run("RoleMember cannot create provider credential", func(t *testing.T) {
		body := map[string]any{"name": "Evil Key", "kind": "api_key", "secret": "sk-dummy"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", body, memberHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for member creating provider credential, got %d", resp.StatusCode)
		}
	})

	t.Run("RoleMember cannot create billing checkout session", func(t *testing.T) {
		body := map[string]any{"provider": "stripe", "plan_code": "pro"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, memberHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for member initiating checkout, got %d", resp.StatusCode)
		}
	})

	t.Run("RoleMember cannot update subscription plan", func(t *testing.T) {
		body := map[string]any{"plan_code": "pro"}
		resp := doJSON(t, srv, http.MethodPut, "/v1/billing/subscription", body, memberHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for member updating subscription, got %d", resp.StatusCode)
		}
	})

	t.Run("RoleAuditor cannot initiate checkout or manage subscription", func(t *testing.T) {
		body := map[string]any{"provider": "stripe", "plan_code": "pro"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, auditorHdr)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for auditor initiating checkout, got %d", resp.StatusCode)
		}
	})
}

func TestSecurity_SecretNonLeakageInAPIResponses(t *testing.T) {
	st := newControlStubs()

	secretKey := "sk-ant-sensitive-api-key-9999999999"
	acct := domain.ProviderAccount{
		ID:             testProvAcct,
		OrganizationID: orgTenantA,
		ProviderKind:   domain.ProviderKindAnthropic,
		Name:           "Anthropic Prod",
		Status:         domain.ProviderStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
	cred := domain.ProviderCredential{
		ID:                testProvCred,
		OrganizationID:    orgTenantA,
		ProviderAccountID: acct.ID,
		Name:              "Production Key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "provider/creds/test-sensitive-ref",
		Fingerprint:       "fp:999999",
		Tail:              "9999",
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}

	st.providerAccounts.get = func(_ context.Context, orgID, acctID uuid.UUID) (domain.ProviderAccount, error) {
		if orgID == orgTenantA && acctID == acct.ID {
			return acct, nil
		}
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}
	st.providerCredentials.get = func(_ context.Context, orgID, credID uuid.UUID) (domain.ProviderCredential, error) {
		if orgID == orgTenantA && credID == cred.ID {
			return cred, nil
		}
		return domain.ProviderCredential{}, repositories.ErrNotFound
	}

	_ = st.providerSecrets.Put(context.Background(), cred.SecretRef, secretKey)

	provSvc := provider.NewService(st.providerDeps())
	identitySvc := identity.NewService(st.deps())

	srv := buildServer(t, Deps{
		Identity:  identitySvc,
		Providers: provSvc,
	})

	hdrA := tenantBearer(st, orgTenantA, userTenantA_Owner, domain.RoleOwner)

	t.Run("GET provider credential response never contains secret or secret_ref", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials/"+cred.ID.String(), nil, hdrA)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if _, exists := raw["secret"]; exists {
			t.Fatal("security violation: raw secret exposed in provider credential response")
		}
		if _, exists := raw["secret_ref"]; exists {
			t.Fatal("security violation: internal secret_ref exposed in provider credential response")
		}
		if raw["fingerprint"] != "fp:999999" || raw["tail"] != "9999" {
			t.Fatalf("expected masked fingerprint and tail, got: %+v", raw)
		}
	})
}
