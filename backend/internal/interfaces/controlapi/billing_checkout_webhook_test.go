package controlapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/payment/razorpay"
	"ai-governance-saas/backend/internal/infrastructure/payment/stripe"
	"ai-governance-saas/backend/internal/ports/payment"
)

func newPaymentBillingServer(t *testing.T, st *controlStubs, stripeSecret, razorpaySecret string) (*httptest.Server, *fakeBillingSubs, *fakeBillingInvoices) {
	t.Helper()

	subs := &fakeBillingSubs{plan: domain.OrganizationPlan{PlanCode: "free", Status: "active"}}
	inv := &fakeBillingInvoices{}
	plans := &fakeBillingPlans{
		plans: []domain.Plan{
			{ID: testAcctID, Code: "pro", Name: "Pro Plan", Status: "active"},
			{ID: testTeamID, Code: "enterprise", Name: "Enterprise", Status: "active"},
		},
	}

	stripeGw := stripe.New("dummy-stripe-key", stripeSecret)
	razorpayGw := razorpay.New("dummy-rzp-key", "dummy-rzp-secret", razorpaySecret)

	billingSvc := billing.NewService(billing.ServiceDeps{
		Plans:         plans,
		Subscriptions: subs,
		Invoices:      inv,
		Rollups:       &fakeBillingRollups{},
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGw,
			"razorpay": razorpayGw,
		},
	})

	srv := buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		Billing:         billingSvc,
		DevLoginEnabled: true,
	})
	return srv, subs, inv
}

func TestBillingCheckoutEndpoint(t *testing.T) {
	st := newControlStubs()
	srv, _, _ := newPaymentBillingServer(t, st, "whsec_stripe_test", "whsec_rzp_test")
	defer srv.Close()

	t.Run("checkout with stripe", func(t *testing.T) {
		body := map[string]interface{}{
			"payment_provider": "stripe",
			"plan_code":        "pro",
			"success_url":      "http://localhost:3000/success",
		}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, bearerHeader(st, domain.RoleAdmin))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		var sess payment.CheckoutSession
		if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
			t.Fatalf("decode session failed: %v", err)
		}
		if sess.Provider != "stripe" {
			t.Errorf("expected stripe provider, got %s", sess.Provider)
		}
		if sess.URL == "" {
			t.Error("expected non-empty checkout URL")
		}
	})

	t.Run("checkout with razorpay", func(t *testing.T) {
		body := map[string]interface{}{
			"payment_provider": "razorpay",
			"plan_code":        "enterprise",
			"success_url":      "http://localhost:3000/success",
		}
		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/checkout", body, bearerHeader(st, domain.RoleAdmin))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		var sess payment.CheckoutSession
		if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
			t.Fatalf("decode session failed: %v", err)
		}
		if sess.Provider != "razorpay" {
			t.Errorf("expected razorpay provider, got %s", sess.Provider)
		}
		if sess.URL == "" {
			t.Error("expected non-empty checkout URL")
		}
	})
}

func TestBillingWebhookEndpoints(t *testing.T) {
	t.Parallel()

	st := newControlStubs()
	stripeSecret := "whsec_stripe_unit_test_secret"
	razorpaySecret := "whsec_rzp_unit_test_secret"

	srv, _, _ := newPaymentBillingServer(t, st, stripeSecret, razorpaySecret)
	defer srv.Close()

	orgID := uuid.New()

	t.Run("stripe webhook success", func(t *testing.T) {
		payload := []byte(fmt.Sprintf(`{
			"id": "evt_stripe_test_100",
			"type": "checkout.session.completed",
			"data": {
				"object": {
					"id": "cs_stripe_100",
					"customer": "cus_stripe_100",
					"subscription": "sub_stripe_100",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "pro"
					}
				}
			}
		}`, orgID.String()))

		timestamp := time.Now().Unix()
		signed := fmt.Sprintf("%d.%s", timestamp, string(payload))
		mac := hmac.New(sha256.New, []byte(stripeSecret))
		mac.Write([]byte(signed))
		sig := hex.EncodeToString(mac.Sum(nil))
		header := fmt.Sprintf("t=%d,v1=%s", timestamp, sig)

		req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}
		req.Header.Set("Stripe-Signature", header)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 from webhook, got %d", resp.StatusCode)
		}
	})

	t.Run("stripe webhook rejects invalid signature", func(t *testing.T) {
		payload := []byte(`{"id":"evt_bad","type":"checkout.session.completed"}`)
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/stripe", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}
		req.Header.Set("Stripe-Signature", "t=12345,v1=bad_signature_value")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for bad stripe signature, got %d", resp.StatusCode)
		}
	})

	t.Run("razorpay webhook rejects invalid signature", func(t *testing.T) {
		payload := []byte(`{"event":"payment.captured"}`)
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/billing/webhooks/razorpay", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}
		req.Header.Set("X-Razorpay-Signature", "forged_razorpay_signature")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("webhook request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for bad razorpay signature, got %d", resp.StatusCode)
		}
	})
}

func TestBillingVerifyPaymentEndpoint(t *testing.T) {
	st := newControlStubs()
	razorpayKeySecret := "rzp_secret_for_cb_test"

	subs := &fakeBillingSubs{plan: domain.OrganizationPlan{PlanCode: "free", Status: "active"}}
	inv := &fakeBillingInvoices{}
	plans := &fakeBillingPlans{
		plans: []domain.Plan{
			{ID: testAcctID, Code: "pro", Name: "Pro Plan", Status: "active"},
			{ID: testTeamID, Code: "enterprise", Name: "Enterprise", Status: "active"},
		},
	}

	razorpayGw := razorpay.New("dummy-rzp-key", razorpayKeySecret, "dummy-wh-secret")
	stripeGw := stripe.New("dummy-stripe-key", "dummy-wh-secret")

	billingSvc := billing.NewService(billing.ServiceDeps{
		Plans:         plans,
		Subscriptions: subs,
		Invoices:      inv,
		Rollups:       &fakeBillingRollups{},
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGw,
			"razorpay": razorpayGw,
		},
	})

	srv := buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		Billing:         billingSvc,
		DevLoginEnabled: true,
	})
	defer srv.Close()

	t.Run("verifies authentic razorpay callback and upgrades subscription", func(t *testing.T) {
		orderID := "order_test_rzp_99"
		paymentID := "pay_test_rzp_99"

		payload := orderID + "|" + paymentID
		mac := hmac.New(sha256.New, []byte(razorpayKeySecret))
		mac.Write([]byte(payload))
		sig := hex.EncodeToString(mac.Sum(nil))

		body := map[string]interface{}{
			"payment_provider": "razorpay",
			"order_id":         orderID,
			"payment_id":       paymentID,
			"signature":        sig,
			"plan_code":        "enterprise",
		}

		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/verify-payment", body, bearerHeader(st, domain.RoleAdmin))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
		}

		var sub subscriptionDTO
		if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}

		if sub.PlanCode != "enterprise" {
			t.Errorf("expected enterprise plan, got %s", sub.PlanCode)
		}
		if sub.PaymentProvider != "razorpay" {
			t.Errorf("expected razorpay payment provider, got %s", sub.PaymentProvider)
		}

		// Verify paid invoice was generated and returned via GET /v1/billing/invoices
		invResp := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, bearerHeader(st, domain.RoleAdmin))
		defer invResp.Body.Close()

		if invResp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 from invoices, got %d", invResp.StatusCode)
		}

		var invList struct {
			Items []invoiceDTO `json:"items"`
		}
		if err := json.NewDecoder(invResp.Body).Decode(&invList); err != nil {
			t.Fatalf("decode invoices failed: %v", err)
		}

		if len(invList.Items) != 1 {
			t.Fatalf("expected 1 paid invoice, got %d", len(invList.Items))
		}
		if invList.Items[0].Status != "paid" {
			t.Errorf("expected status paid, got %s", invList.Items[0].Status)
		}
		if invList.Items[0].PaymentProvider == nil || *invList.Items[0].PaymentProvider != "razorpay" {
			t.Errorf("expected payment_provider razorpay, got %v", invList.Items[0].PaymentProvider)
		}
		if invList.Items[0].PaymentID == nil || *invList.Items[0].PaymentID != paymentID {
			t.Errorf("expected payment_id %s, got %v", paymentID, invList.Items[0].PaymentID)
		}
	})

	t.Run("rejects forged signature with HTTP 400", func(t *testing.T) {
		body := map[string]interface{}{
			"payment_provider": "razorpay",
			"order_id":         "order_fake",
			"payment_id":       "pay_fake",
			"signature":        "fake_tampered_signature",
			"plan_code":        "pro",
		}

		resp := doJSON(t, srv, http.MethodPost, "/v1/billing/verify-payment", body, bearerHeader(st, domain.RoleAdmin))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for forged signature, got %d", resp.StatusCode)
		}
	})
}

