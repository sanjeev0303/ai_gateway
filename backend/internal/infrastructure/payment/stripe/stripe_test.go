package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/ports/payment"
)

func TestCreateCheckoutSession(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()

	t.Run("explicit simulation mode", func(t *testing.T) {
		gw := New("dummy-key", "whsec_test")
		gw.SetMode("simulation")
		params := payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
			SuccessURL:     "http://localhost:3000/app/billing?session_id={CHECKOUT_SESSION_ID}",
			CancelURL:      "http://localhost:3000/app/billing",
		}

		sess, err := gw.CreateCheckoutSession(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.Provider != "stripe" {
			t.Errorf("expected provider stripe, got %s", sess.Provider)
		}
		if !sess.IsSimulated {
			t.Errorf("expected IsSimulated to be true")
		}
		if sess.Mode != "simulation" {
			t.Errorf("expected Mode simulation, got %s", sess.Mode)
		}
		if !strings.HasPrefix(sess.SessionID, "cs_sim_") {
			t.Errorf("expected session ID to start with cs_sim_, got %s", sess.SessionID)
		}
		if !strings.Contains(sess.URL, "checkout.stripe.com/simulate/") {
			t.Errorf("expected URL to contain simulation path, got %s", sess.URL)
		}
	})

	t.Run("test mode rejects dummy key with invalid credentials error", func(t *testing.T) {
		gw := New("dummy-key", "whsec_test")
		gw.SetMode("test")
		params := payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
		}
		_, err := gw.CreateCheckoutSession(context.Background(), params)
		if !errors.Is(err, payment.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("test mode rejects empty key with missing credentials error", func(t *testing.T) {
		gw := New("", "whsec_test")
		gw.SetMode("test")
		params := payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
		}
		_, err := gw.CreateCheckoutSession(context.Background(), params)
		if !errors.Is(err, payment.ErrMissingCredentials) {
			t.Fatalf("expected ErrMissingCredentials, got %v", err)
		}
	})

	t.Run("mock upstream stripe server", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/checkout/sessions" && r.Method == http.MethodPost {
				if r.Header.Get("Authorization") != "Bearer sk_live_actual_test_key" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"cs_live_99999","url":"https://checkout.stripe.com/pay/cs_live_99999"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		gw := NewCustomGateway("sk_live_actual_test_key", "whsec_test", ts.URL)
		params := payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
			CustomerEmail:  "test@example.com",
			SuccessURL:     "http://localhost:3000/success",
			CancelURL:      "http://localhost:3000/cancel",
		}

		sess, err := gw.CreateCheckoutSession(context.Background(), params)
		if err != nil {
			t.Fatalf("checkout session failed: %v", err)
		}
		if sess.SessionID != "cs_live_99999" {
			t.Errorf("unexpected session ID: %s", sess.SessionID)
		}
		if sess.URL != "https://checkout.stripe.com/pay/cs_live_99999" {
			t.Errorf("unexpected checkout URL: %s", sess.URL)
		}
	})
}

func makeStripeHeader(payload []byte, secret string, timestamp int64) string {
	signed := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, sig)
}

func TestVerifyAndParseWebhook(t *testing.T) {
	t.Parallel()

	secret := "whsec_mock_stripe_secret_123"
	gw := New("dummy", secret)
	orgID := uuid.New()

	t.Run("checkout.session.completed", func(t *testing.T) {
		payload := []byte(fmt.Sprintf(`{
			"id": "evt_checkout_1",
			"type": "checkout.session.completed",
			"data": {
				"object": {
					"id": "cs_test_123",
					"customer": "cus_stripe_111",
					"subscription": "sub_stripe_222",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "pro"
					}
				}
			}
		}`, orgID.String()))

		header := makeStripeHeader(payload, secret, time.Now().Unix())
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, header)
		if err != nil {
			t.Fatalf("webhook verify failed: %v", err)
		}

		if evt.EventType != "checkout.completed" {
			t.Errorf("expected checkout.completed, got %s", evt.EventType)
		}
		if evt.OrganizationID != orgID {
			t.Errorf("expected org %s, got %s", orgID, evt.OrganizationID)
		}
		if evt.PlanCode != "pro" {
			t.Errorf("expected plan pro, got %s", evt.PlanCode)
		}
		if evt.SubscriptionID != "sub_stripe_222" {
			t.Errorf("expected sub_stripe_222, got %s", evt.SubscriptionID)
		}
		if evt.CustomerID != "cus_stripe_111" {
			t.Errorf("expected cus_stripe_111, got %s", evt.CustomerID)
		}
		if evt.Status != "active" {
			t.Errorf("expected active, got %s", evt.Status)
		}
	})

	t.Run("customer.subscription.deleted", func(t *testing.T) {
		payload := []byte(fmt.Sprintf(`{
			"id": "evt_sub_del_1",
			"type": "customer.subscription.deleted",
			"data": {
				"object": {
					"id": "sub_stripe_222",
					"customer": "cus_stripe_111",
					"status": "canceled",
					"metadata": {
						"organization_id": "%s",
						"plan_code": "pro"
					}
				}
			}
		}`, orgID.String()))

		header := makeStripeHeader(payload, secret, time.Now().Unix())
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, header)
		if err != nil {
			t.Fatalf("webhook verify failed: %v", err)
		}

		if evt.EventType != "subscription.deleted" {
			t.Errorf("expected subscription.deleted, got %s", evt.EventType)
		}
		if evt.Status != "canceled" {
			t.Errorf("expected canceled, got %s", evt.Status)
		}
	})

	t.Run("invoice.payment_succeeded", func(t *testing.T) {
		payload := []byte(fmt.Sprintf(`{
			"id": "evt_inv_1",
			"type": "invoice.payment_succeeded",
			"data": {
				"object": {
					"id": "in_stripe_444",
					"customer": "cus_stripe_111",
					"subscription": "sub_stripe_222",
					"amount_paid": 4900,
					"currency": "usd",
					"metadata": {
						"organization_id": "%s"
					}
				}
			}
		}`, orgID.String()))

		header := makeStripeHeader(payload, secret, time.Now().Unix())
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, header)
		if err != nil {
			t.Fatalf("webhook verify failed: %v", err)
		}

		if evt.EventType != "payment.succeeded" {
			t.Errorf("expected payment.succeeded, got %s", evt.EventType)
		}
		if evt.Status != "paid" {
			t.Errorf("expected paid, got %s", evt.Status)
		}
		if evt.AmountMicros != 49000000 { // 4900 cents * 10000 = 49,000,000 micro-dollars = $49.00
			t.Errorf("expected 49000000 micros, got %d", evt.AmountMicros)
		}
	})

	t.Run("tampered payload rejected", func(t *testing.T) {
		payload := []byte(`{"id":"evt_1","type":"test"}`)
		header := makeStripeHeader(payload, secret, time.Now().Unix())

		tampered := []byte(`{"id":"evt_1","type":"tampered"}`)
		_, err := gw.VerifyAndParseWebhook(context.Background(), tampered, header)
		if err == nil {
			t.Fatal("expected error for tampered payload, got nil")
		}
		if !errors.Is(err, payment.ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature, got: %v", err)
		}
	})

	t.Run("expired timestamp rejected", func(t *testing.T) {
		payload := []byte(`{"id":"evt_1","type":"test"}`)
		expiredTime := time.Now().Add(-10 * time.Minute).Unix()
		header := makeStripeHeader(payload, secret, expiredTime)

		_, err := gw.VerifyAndParseWebhook(context.Background(), payload, header)
		if err == nil {
			t.Fatal("expected error for expired timestamp, got nil")
		}
		if !errors.Is(err, payment.ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature, got: %v", err)
		}
	})
}
