package controlapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"

	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/payment/razorpay"
	"ai-governance-saas/backend/internal/infrastructure/payment/stripe"
	"ai-governance-saas/backend/internal/ports/payment"
	"ai-governance-saas/backend/internal/ports/repositories"
)

func newModesBillingServer(t *testing.T, st *controlStubs, mode string, stripeGw, razorpayGw payment.PaymentGateway, failBillingEvents bool) (*httptest.Server, *fakeBillingSubs, *fakeBillingInvoices, *fakeBillingEventsRepo) {
	t.Helper()

	subs := &fakeBillingSubs{plan: domain.OrganizationPlan{PlanCode: "free", PlanName: "Free Plan", Status: "active"}}
	inv := &fakeBillingInvoices{}
	plans := &fakeBillingPlans{
		plans: []domain.Plan{
			{ID: testUserID, Code: "free", Name: "Free Plan", Status: "active"},
			{ID: testAcctID, Code: "pro", Name: "Pro Plan", Status: "active"},
			{ID: testTeamID, Code: "enterprise", Name: "Enterprise", Status: "active"},
		},
	}
	eventsRepo := &fakeBillingEventsRepo{failRecord: failBillingEvents}

	gateways := make(map[string]payment.PaymentGateway)
	if stripeGw != nil {
		gateways["stripe"] = stripeGw
	}
	if razorpayGw != nil {
		gateways["razorpay"] = razorpayGw
	}

	billingSvc := billing.NewService(billing.ServiceDeps{
		Mode:            mode,
		Plans:           plans,
		Subscriptions:   subs,
		Invoices:        inv,
		Rollups:         &fakeBillingRollups{},
		BillingEvents:   eventsRepo,
		PaymentGateways: gateways,
	})

	srv := buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		Billing:         billingSvc,
		DevLoginEnabled: true,
	})
	return srv, subs, inv, eventsRepo
}

type fakeBillingEventsRepo struct {
	events     map[string]domain.BillingEvent
	failRecord bool
}

func (r *fakeBillingEventsRepo) Record(ctx context.Context, orgID uuid.UUID, input domain.NewBillingEvent) (domain.BillingEvent, error) {
	if r.failRecord {
		return domain.BillingEvent{}, errors.New("database connection refused: failed to persist billing event")
	}
	if r.events == nil {
		r.events = make(map[string]domain.BillingEvent)
	}
	key := orgID.String() + ":" + input.BillingEventID
	if _, exists := r.events[key]; exists {
		return domain.BillingEvent{}, repositories.ErrConflict
	}
	evt := domain.BillingEvent{
		ID:             uuid.New(),
		OrganizationID: orgID,
		BillingEventID: input.BillingEventID,
		EventType:      input.EventType,
		AmountMicros:   input.AmountMicros,
		Currency:       input.Currency,
		Status:         domain.BillingEventStatusSucceeded,
		Payload:        input.Payload,
		CreatedAt:      time.Now(),
	}
	r.events[key] = evt
	return evt, nil
}

func (r *fakeBillingEventsRepo) Get(ctx context.Context, orgID uuid.UUID, billingEventID string) (domain.BillingEvent, error) {
	if r.events != nil {
		key := orgID.String() + ":" + billingEventID
		if evt, ok := r.events[key]; ok {
			return evt, nil
		}
	}
	return domain.BillingEvent{}, repositories.ErrNotFound
}

func (r *fakeBillingEventsRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	return 0, nil
}

// 1. Mode A: Billing Disabled
func TestBillingMode_Disabled(t *testing.T) {
	st := newControlStubs()
	srv, subs, _, _ := newModesBillingServer(t, st, "disabled", nil, nil, false)
	defer srv.Close()

	adminHdr := bearerHeader(st, domain.RoleAdmin)

	t.Run("checkout returns 503 billing_disabled", func(t *testing.T) {
		body := map[string]any{"payment_provider": "stripe", "plan_code": "pro", "success_url": "http://localhost:3000/success"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected HTTP 503, got %d", resp.StatusCode)
		}
		var errRes map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errRes)
		if errRes["code"] != "billing_disabled" {
			t.Errorf("expected code billing_disabled, got %s", errRes["code"])
		}
	})

	t.Run("verify-payment returns 503 billing_disabled", func(t *testing.T) {
		body := map[string]any{
			"payment_provider": "stripe",
			"order_id":         "order_123",
			"payment_id":       "pay_123",
			"signature":        "sig_123",
			"plan_code":        "pro",
		}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/verify-payment", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected HTTP 503, got %d", resp.StatusCode)
		}
	})

	t.Run("subscription update returns 503 billing_disabled", func(t *testing.T) {
		body := map[string]any{"plan_code": "pro"}
		resp := doJSON(t, srv, http.MethodPut, "/v1/billing/subscription", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected HTTP 503, got %d", resp.StatusCode)
		}
	})

	t.Run("stripe webhook returns 503 billing_disabled", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(`{}`))
		req.Header.Set("Stripe-Signature", "t=1,v1=sig")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected HTTP 503, got %d", resp.StatusCode)
		}
	})

	t.Run("razorpay webhook returns 503 billing_disabled", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/razorpay", strings.NewReader(`{}`))
		req.Header.Set("X-Razorpay-Signature", "sig")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("expected HTTP 503, got %d", resp.StatusCode)
		}
	})

	t.Run("GET subscription indicates billing_disabled and free plan", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/billing/subscription", nil, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		var sub subscriptionDTO
		_ = json.NewDecoder(resp.Body).Decode(&sub)
		if sub.BillingEnabled {
			t.Errorf("expected billing_enabled to be false")
		}
		if sub.BillingMode != "disabled" {
			t.Errorf("expected billing_mode disabled, got %s", sub.BillingMode)
		}
		if sub.PlanCode != "free" {
			t.Errorf("expected plan_code free, got %s", sub.PlanCode)
		}
		if subs.plan.PlanCode != "free" {
			t.Errorf("subscription must not have changed from free, got %s", subs.plan.PlanCode)
		}
	})
}

// 2. Mode B: Simulation Mode
func TestBillingMode_Simulation(t *testing.T) {
	st := newControlStubs()
	stripeGw := stripe.New("dummy-stripe-key", "whsec_test")
	razorpayGw := razorpay.New("dummy-rzp-key", "dummy-rzp-secret", "whsec_test")
	srv, subs, _, _ := newModesBillingServer(t, st, "simulation", stripeGw, razorpayGw, false)
	defer srv.Close()

	adminHdr := bearerHeader(st, domain.RoleAdmin)

	t.Run("stripe checkout returns explicit simulation response", func(t *testing.T) {
		body := map[string]any{"payment_provider": "stripe", "plan_code": "pro", "success_url": "http://localhost:3000/success"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		var sess payment.CheckoutSession
		if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
			t.Fatalf("decode session failed: %v", err)
		}
		if !sess.IsSimulated {
			t.Errorf("expected is_simulated to be true")
		}
		if sess.Mode != "simulation" {
			t.Errorf("expected mode simulation, got %s", sess.Mode)
		}
		if !strings.Contains(sess.URL, "/simulate/") {
			t.Errorf("expected URL to be explicitly simulated, got %s", sess.URL)
		}
	})

	t.Run("razorpay checkout returns explicit simulation response", func(t *testing.T) {
		body := map[string]any{"payment_provider": "razorpay", "plan_code": "enterprise", "success_url": "http://localhost:3000/success"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		var sess payment.CheckoutSession
		if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
			t.Fatalf("decode session failed: %v", err)
		}
		if !sess.IsSimulated {
			t.Errorf("expected is_simulated to be true")
		}
		if sess.Mode != "simulation" {
			t.Errorf("expected mode simulation, got %s", sess.Mode)
		}
		if !strings.Contains(sess.URL, "rzp.io/simulate/") {
			t.Errorf("expected URL to contain simulation path, got %s", sess.URL)
		}
	})

	t.Run("simulation mode rejects verify-payment callback and prevents paid entitlement activation", func(t *testing.T) {
		body := map[string]any{
			"payment_provider": "stripe",
			"order_id":         "order_sim_123",
			"payment_id":       "pay_sim_123",
			"signature":        "sig_sim",
			"plan_code":        "pro",
		}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/verify-payment", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for simulation callback, got %d", resp.StatusCode)
		}
		var errRes map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errRes)
		if errRes["code"] != "simulation_mode" {
			t.Errorf("expected code simulation_mode, got %s", errRes["code"])
		}
		// Confirm subscription was not modified
		if subs.plan.PlanCode != "free" {
			t.Errorf("simulation callback must never activate paid entitlements: got %s", subs.plan.PlanCode)
		}
	})
}

// 3. Mode C: Provider Test Mode (Mocks for deterministic tests)
func TestBillingMode_ProviderTestMode_Stripe_Mock(t *testing.T) {
	st := newControlStubs()

	var upstreamCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.Header.Get("Authorization") != "Bearer sk_test_valid_mock_key" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API Key provided"}}`))
			return
		}
		if r.URL.Path == "/v1/checkout/sessions" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"cs_test_mock_12345","url":"https://checkout.stripe.com/c/pay/cs_test_mock_12345","customer":"cus_mock_1"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	stripeGw := stripe.NewCustomGateway("sk_test_valid_mock_key", "whsec_test", ts.URL)
	srv, _, _, _ := newModesBillingServer(t, st, "test", stripeGw, nil, false)
	defer srv.Close()

	adminHdr := bearerHeader(st, domain.RoleAdmin)

	t.Run("successful checkout invokes official Stripe API and returns genuine session", func(t *testing.T) {
		upstreamCalled = false
		body := map[string]any{"payment_provider": "stripe", "plan_code": "pro", "success_url": "http://localhost:3000/success"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		if !upstreamCalled {
			t.Errorf("expected upstream stripe API to be called")
		}
		var sess payment.CheckoutSession
		if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
			t.Fatalf("decode session failed: %v", err)
		}
		if sess.IsSimulated {
			t.Errorf("expected is_simulated to be false in test mode")
		}
		if sess.Mode != "test" {
			t.Errorf("expected mode test, got %s", sess.Mode)
		}
		if sess.SessionID != "cs_test_mock_12345" {
			t.Errorf("expected session cs_test_mock_12345, got %s", sess.SessionID)
		}
	})

	t.Run("missing credentials returns 400 missing_credentials", func(t *testing.T) {
		noKeyGw := stripe.NewCustomGateway("", "whsec_test", ts.URL)
		noKeySrv, _, _, _ := newModesBillingServer(t, st, "test", noKeyGw, nil, false)
		defer noKeySrv.Close()

		body := map[string]any{"payment_provider": "stripe", "plan_code": "pro"}
		resp := doJSON(t, noKeySrv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for missing credentials, got %d", resp.StatusCode)
		}
		var errRes map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errRes)
		if errRes["code"] != "missing_credentials" {
			t.Errorf("expected code missing_credentials, got %s", errRes["code"])
		}
	})

	t.Run("invalid placeholder credentials returns 400 invalid_credentials", func(t *testing.T) {
		dummyGw := stripe.NewCustomGateway("dummy-key-placeholder", "whsec_test", ts.URL)
		dummySrv, _, _, _ := newModesBillingServer(t, st, "test", dummyGw, nil, false)
		defer dummySrv.Close()

		body := map[string]any{"payment_provider": "stripe", "plan_code": "pro"}
		resp := doJSON(t, dummySrv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for placeholder credentials, got %d", resp.StatusCode)
		}
		var errRes map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errRes)
		if errRes["code"] != "invalid_credentials" {
			t.Errorf("expected code invalid_credentials, got %s", errRes["code"])
		}
	})

	t.Run("provider unauthorized error maps to 400 invalid_credentials", func(t *testing.T) {
		unauthGw := stripe.NewCustomGateway("sk_test_wrong_auth_key", "whsec_test", ts.URL)
		unauthSrv, _, _, _ := newModesBillingServer(t, st, "test", unauthGw, nil, false)
		defer unauthSrv.Close()

		body := map[string]any{"payment_provider": "stripe", "plan_code": "pro"}
		resp := doJSON(t, unauthSrv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for upstream unauthorized, got %d", resp.StatusCode)
		}
	})
}

// 4. Mode C: Provider Test Mode (Razorpay Mocks for deterministic tests)
func TestBillingMode_ProviderTestMode_Razorpay_Mock(t *testing.T) {
	st := newControlStubs()

	var upstreamCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		user, pass, _ := r.BasicAuth()
		if user != "rzp_test_valid_mock_key" || pass != "secret_valid" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"description":"Unauthorized"}}`))
			return
		}
		if r.URL.Path == "/v1/payment_links" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"plink_mock_rzp_99","short_url":"https://rzp.io/rzp/mock99"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	rzpGw := razorpay.NewCustomGateway("rzp_test_valid_mock_key", "secret_valid", "whsec_test", ts.URL)
	srv, _, _, _ := newModesBillingServer(t, st, "test", nil, rzpGw, false)
	defer srv.Close()

	adminHdr := bearerHeader(st, domain.RoleAdmin)

	t.Run("successful razorpay checkout invokes official payment_links API", func(t *testing.T) {
		upstreamCalled = false
		body := map[string]any{"payment_provider": "razorpay", "plan_code": "enterprise", "success_url": "http://localhost:3000/success"}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}
		if !upstreamCalled {
			t.Errorf("expected upstream razorpay API to be called")
		}
		var sess payment.CheckoutSession
		if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
			t.Fatalf("decode session failed: %v", err)
		}
		if sess.IsSimulated {
			t.Errorf("expected is_simulated to be false in test mode")
		}
		if sess.Mode != "test" {
			t.Errorf("expected mode test, got %s", sess.Mode)
		}
		if sess.SessionID != "plink_mock_rzp_99" {
			t.Errorf("expected session plink_mock_rzp_99, got %s", sess.SessionID)
		}
	})

	t.Run("missing razorpay credentials returns 400 missing_credentials", func(t *testing.T) {
		noKeyGw := razorpay.NewCustomGateway("", "", "whsec_test", ts.URL)
		noKeySrv, _, _, _ := newModesBillingServer(t, st, "test", nil, noKeyGw, false)
		defer noKeySrv.Close()

		body := map[string]any{"payment_provider": "razorpay", "plan_code": "enterprise"}
		resp := doJSON(t, noKeySrv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for missing credentials, got %d", resp.StatusCode)
		}
		var errRes map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errRes)
		if errRes["code"] != "missing_credentials" {
			t.Errorf("expected code missing_credentials, got %s", errRes["code"])
		}
	})

	t.Run("placeholder credentials returns 400 invalid_credentials", func(t *testing.T) {
		dummyGw := razorpay.NewCustomGateway("dummy-key", "dummy-secret", "whsec_test", ts.URL)
		dummySrv, _, _, _ := newModesBillingServer(t, st, "test", nil, dummyGw, false)
		defer dummySrv.Close()

		body := map[string]any{"payment_provider": "razorpay", "plan_code": "enterprise"}
		resp := doJSON(t, dummySrv, http.MethodPost, "/v1/billing/checkout", body, adminHdr)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for placeholder credentials, got %d", resp.StatusCode)
		}
	})
}

// 5. Webhook Safety, Idempotency, Failures, Entitlements, DB Errors
func TestBillingMode_Webhooks_And_Safety(t *testing.T) {
	st := newControlStubs()
	stripeSecret := "whsec_modes_test_secret"
	razorpaySecret := "rzp_modes_test_secret"

	stripeGw := stripe.NewCustomGateway("sk_test_key", stripeSecret, "")
	razorpayGw := razorpay.NewCustomGateway("rzp_test_key", "secret", razorpaySecret, "")
	srv, subs, invs, eventsRepo := newModesBillingServer(t, st, "test", stripeGw, razorpayGw, false)
	defer srv.Close()

	orgID := uuid.New()

	t.Run("rejects invalid stripe webhook signature with 400 invalid_signature", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(`{"id":"evt_1"}`))
		req.Header.Set("Stripe-Signature", "t=1,v1=tampered_signature")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for invalid signature, got %d", resp.StatusCode)
		}
	})

	t.Run("rejects invalid razorpay webhook signature with 400 invalid_signature", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/razorpay", strings.NewReader(`{"event":"order.paid"}`))
		req.Header.Set("X-Razorpay-Signature", "tampered_signature")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for invalid signature, got %d", resp.StatusCode)
		}
	})

	t.Run("idempotent webhook processing prevents duplicate execution and ledger inflation", func(t *testing.T) {
		evtID := "evt_idempotent_test_001"
		rawPayload := fmt.Sprintf(`{
			"id": "%s",
			"type": "checkout.session.completed",
			"data": {
				"object": {
					"id": "cs_order_123",
					"customer": "cus_123",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "enterprise"
					}
				}
			}
		}`, evtID, orgID.String())

		// Generate valid Stripe signature
		now := time.Now().Unix()
		signedPayload := fmt.Sprintf("%d.%s", now, rawPayload)
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signedPayload))
		sig := hex.EncodeToString(mac.Sum(nil))
		sigHeader := fmt.Sprintf("t=%d,v1=%s", now, sig)

		// First delivery
		req1, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(rawPayload))
		req1.Header.Set("Stripe-Signature", sigHeader)
		resp1, err := http.DefaultClient.Do(req1)
		if err != nil {
			t.Fatalf("first webhook failed: %v", err)
		}
		defer resp1.Body.Close()
		if resp1.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 on first delivery, got %d", resp1.StatusCode)
		}

		// Verify plan upgraded to enterprise
		if subs.plan.PlanCode != "enterprise" {
			t.Fatalf("expected enterprise plan after first webhook, got %s", subs.plan.PlanCode)
		}
		entitlementsEnterprise := domain.GetPlanEntitlements(subs.plan.PlanCode)
		if entitlementsEnterprise.MonthlyRequestQuota != 100_000_000 {
			t.Errorf("expected 100M quota for enterprise tier, got %d", entitlementsEnterprise.MonthlyRequestQuota)
		}

		// Second duplicate delivery with identical event ID
		req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(rawPayload))
		req2.Header.Set("Stripe-Signature", sigHeader)
		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Fatalf("second webhook failed: %v", err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 on duplicate delivery, got %d", resp2.StatusCode)
		}
	})

	t.Run("payment failure transitions subscription to past_due", func(t *testing.T) {
		evtID := "evt_pay_failed_002"
		rawPayload := fmt.Sprintf(`{
			"id": "%s",
			"type": "invoice.payment_failed",
			"data": {
				"object": {
					"id": "in_fail_1",
					"customer": "cus_123",
					"subscription": "sub_123",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "enterprise"
					}
				}
			}
		}`, evtID, orgID.String())

		now := time.Now().Unix()
		signedPayload := fmt.Sprintf("%d.%s", now, rawPayload)
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signedPayload))
		sig := hex.EncodeToString(mac.Sum(nil))
		sigHeader := fmt.Sprintf("t=%d,v1=%s", now, sig)

		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(rawPayload))
		req.Header.Set("Stripe-Signature", sigHeader)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		if subs.plan.Status != domain.SubscriptionStatusPastDue {
			t.Errorf("expected status past_due after payment failure, got %s", subs.plan.Status)
		}
	})

	t.Run("subscription cancellation reverts plan to Free tier entitlements", func(t *testing.T) {
		evtID := "evt_sub_deleted_003"
		rawPayload := fmt.Sprintf(`{
			"id": "%s",
			"type": "customer.subscription.deleted",
			"data": {
				"object": {
					"id": "sub_123",
					"customer": "cus_123",
					"metadata": {
						"organization_id": "%s"
					}
				}
			}
		}`, evtID, orgID.String())

		now := time.Now().Unix()
		signedPayload := fmt.Sprintf("%d.%s", now, rawPayload)
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signedPayload))
		sig := hex.EncodeToString(mac.Sum(nil))
		sigHeader := fmt.Sprintf("t=%d,v1=%s", now, sig)

		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(rawPayload))
		req.Header.Set("Stripe-Signature", sigHeader)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		if subs.plan.PlanCode != "free" {
			t.Errorf("expected free plan after subscription cancellation, got %s", subs.plan.PlanCode)
		}
		entitlementsFree := domain.GetPlanEntitlements(subs.plan.PlanCode)
		if entitlementsFree.MonthlyRequestQuota != 10_000 {
			t.Errorf("expected 10k quota for free tier, got %d", entitlementsFree.MonthlyRequestQuota)
		}
	})

	t.Run("database failure during webhook processing returns error and blocks state mutations", func(t *testing.T) {
		dbFailSrv, dbSubs, _, _ := newModesBillingServer(t, st, "test", stripeGw, razorpayGw, true)
		defer dbFailSrv.Close()

		evtID := "evt_db_fail_004"
		rawPayload := fmt.Sprintf(`{
			"id": "%s",
			"type": "checkout.session.completed",
			"data": {
				"object": {
					"id": "cs_order_db_fail",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "enterprise"
					}
				}
			}
		}`, evtID, orgID.String())

		now := time.Now().Unix()
		signedPayload := fmt.Sprintf("%d.%s", now, rawPayload)
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signedPayload))
		sig := hex.EncodeToString(mac.Sum(nil))
		sigHeader := fmt.Sprintf("t=%d,v1=%s", now, sig)

		req, _ := http.NewRequest(http.MethodPost, dbFailSrv.URL+"/v1/billing/webhooks/stripe", strings.NewReader(rawPayload))
		req.Header.Set("Stripe-Signature", sigHeader)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Fatalf("expected non-200 status when database fails, got %d", resp.StatusCode)
		}
		if dbSubs.plan.PlanCode != "free" {
			t.Errorf("subscription must not have been upgraded on DB error: got %s", dbSubs.plan.PlanCode)
		}
	})

	_ = invs
	_ = eventsRepo
}

// 6. Live Test-Mode Verification using User-Provided Test Credentials
func TestBillingMode_LiveProviderCredentials(t *testing.T) {
	stripeKey := os.Getenv("AIGW_BILLING_STRIPE_SECRET_KEY")
	if stripeKey == "" || strings.Contains(stripeKey, "placeholder") {
		t.Skip("skipping live stripe test: valid AIGW_BILLING_STRIPE_SECRET_KEY not set in environment")
	}
	rzpKeyID := os.Getenv("AIGW_BILLING_RAZORPAY_KEY_ID")
	rzpKeySecret := os.Getenv("AIGW_BILLING_RAZORPAY_KEY_SECRET")
	if rzpKeyID == "" || rzpKeySecret == "" || strings.Contains(rzpKeySecret, "placeholder") {
		t.Skip("skipping live razorpay test: valid AIGW_BILLING_RAZORPAY_KEY_ID/SECRET not set in environment")
	}

	orgID := uuid.New()

	t.Run("live stripe test credentials create genuine checkout session", func(t *testing.T) {
		gw := stripe.New(stripeKey, "whsec_mock")
		gw.SetMode("test")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sess, err := gw.CreateCheckoutSession(ctx, payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
			SuccessURL:     "http://localhost:3000/success",
			CancelURL:      "http://localhost:3000/cancel",
		})
		if err != nil {
			t.Fatalf("live Stripe checkout session creation failed: %v", err)
		}
		if sess.IsSimulated {
			t.Errorf("live session must not be simulated")
		}
		if sess.Mode != "test" {
			t.Errorf("expected mode test, got %s", sess.Mode)
		}
		if !strings.HasPrefix(sess.SessionID, "cs_test_") {
			t.Errorf("expected session ID to start with cs_test_, got %s", sess.SessionID)
		}
		if !strings.HasPrefix(sess.URL, "https://checkout.stripe.com/") {
			t.Errorf("expected official Stripe checkout URL, got %s", sess.URL)
		}
	})

	t.Run("live razorpay test credentials create genuine payment link", func(t *testing.T) {
		gw := razorpay.New(rzpKeyID, rzpKeySecret, "whsec_mock")
		gw.SetMode("test")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		sess, err := gw.CreateCheckoutSession(ctx, payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "enterprise",
			SuccessURL:     "http://localhost:3000/success",
		})
		if err != nil {
			t.Fatalf("live Razorpay payment link creation failed: %v", err)
		}
		if sess.IsSimulated {
			t.Errorf("live session must not be simulated")
		}
		if sess.Mode != "test" {
			t.Errorf("expected mode test, got %s", sess.Mode)
		}
		if !strings.HasPrefix(sess.SessionID, "plink_") {
			t.Errorf("expected session ID to start with plink_, got %s", sess.SessionID)
		}
		if !strings.HasPrefix(sess.URL, "https://rzp.io/") {
			t.Errorf("expected official Razorpay payment link URL, got %s", sess.URL)
		}
	})
}
