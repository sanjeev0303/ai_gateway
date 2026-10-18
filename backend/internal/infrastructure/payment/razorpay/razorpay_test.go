package razorpay

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

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/ports/payment"
)

func TestCreateCheckoutSession(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()

	t.Run("simulated fallback for dummy keys", func(t *testing.T) {
		gw := New("dummy-key", "dummy-secret", "dummy-webhook")
		params := payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "enterprise",
			SuccessURL:     "http://localhost:3000/app/billing",
		}

		sess, err := gw.CreateCheckoutSession(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.Provider != "razorpay" {
			t.Errorf("expected razorpay provider, got %s", sess.Provider)
		}
		if !strings.HasPrefix(sess.SessionID, "sub_link_") {
			t.Errorf("expected session ID to start with sub_link_, got %s", sess.SessionID)
		}
		if !strings.Contains(sess.URL, "rzp.io/i/") {
			t.Errorf("expected URL to contain rzp.io/i/, got %s", sess.URL)
		}
	})

	t.Run("mock upstream razorpay server", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/payment_links" && r.Method == http.MethodPost {
				username, password, ok := r.BasicAuth()
				if !ok || username != "rzp_live_test" || password != "secret_live_test" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"plink_12345","short_url":"https://rzp.io/i/plink_12345"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		gw := NewCustomGateway("rzp_live_test", "secret_live_test", "whsec_test", ts.URL)
		params := payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
			CustomerEmail:  "test@example.com",
			SuccessURL:     "http://localhost:3000/success",
		}

		sess, err := gw.CreateCheckoutSession(context.Background(), params)
		if err != nil {
			t.Fatalf("checkout session failed: %v", err)
		}
		if sess.SessionID != "plink_12345" {
			t.Errorf("unexpected session ID: %s", sess.SessionID)
		}
		if sess.URL != "https://rzp.io/i/plink_12345" {
			t.Errorf("unexpected checkout URL: %s", sess.URL)
		}
	})
}

func makeRazorpaySignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyAndParseWebhook(t *testing.T) {
	t.Parallel()

	secret := "rzp_webhook_secret_999"
	gw := New("dummy", "dummy", secret)
	orgID := uuid.New()

	t.Run("subscription.activated", func(t *testing.T) {
		payload := fmt.Appendf(nil, `{
			"event": "subscription.activated",
			"payload": {
				"subscription": {
					"entity": {
						"id": "sub_rzp_111",
						"customer_id": "cust_rzp_222",
						"status": "active",
						"notes": {
							"organization_id": "%s",
							"plan_code": "scale"
						}
					}
				}
			}
		}`, orgID.String())

		sig := makeRazorpaySignature(payload, secret)
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, sig)
		if err != nil {
			t.Fatalf("webhook verify failed: %v", err)
		}

		if evt.EventType != "subscription.updated" {
			t.Errorf("expected subscription.updated, got %s", evt.EventType)
		}
		if evt.OrganizationID != orgID {
			t.Errorf("expected org %s, got %s", orgID, evt.OrganizationID)
		}
		if evt.PlanCode != "scale" {
			t.Errorf("expected plan scale, got %s", evt.PlanCode)
		}
		if evt.SubscriptionID != "sub_rzp_111" {
			t.Errorf("expected sub_rzp_111, got %s", evt.SubscriptionID)
		}
		if evt.Status != "active" {
			t.Errorf("expected active, got %s", evt.Status)
		}
	})

	t.Run("payment.captured / charged", func(t *testing.T) {
		payload := fmt.Appendf(nil, `{
			"event": "payment.captured",
			"payload": {
				"payment": {
					"entity": {
						"id": "pay_rzp_333",
						"customer_id": "cust_rzp_222",
						"amount": 490000,
						"currency": "INR",
						"notes": {
							"organization_id": "%s",
							"plan_code": "pro"
						}
					}
				}
			}
		}`, orgID.String())

		sig := makeRazorpaySignature(payload, secret)
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, sig)
		if err != nil {
			t.Fatalf("webhook verify failed: %v", err)
		}

		if evt.EventType != "payment.succeeded" {
			t.Errorf("expected payment.succeeded, got %s", evt.EventType)
		}
		if evt.Status != "paid" {
			t.Errorf("expected paid, got %s", evt.Status)
		}
		if evt.AmountMicros != 4900000000 {
			t.Errorf("expected 4900000000 micros, got %d", evt.AmountMicros)
		}
		if evt.Currency != "inr" {
			t.Errorf("expected inr, got %s", evt.Currency)
		}
	})

	t.Run("subscription.cancelled", func(t *testing.T) {
		payload := fmt.Appendf(nil, `{
			"event": "subscription.cancelled",
			"payload": {
				"subscription": {
					"entity": {
						"id": "sub_rzp_111",
						"status": "cancelled",
						"notes": {
							"organization_id": "%s"
						}
					}
				}
			}
		}`, orgID.String())

		sig := makeRazorpaySignature(payload, secret)
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, sig)
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

	t.Run("tampered payload rejected", func(t *testing.T) {
		payload := []byte(`{"event":"test"}`)
		sig := makeRazorpaySignature(payload, secret)

		tampered := []byte(`{"event":"tampered"}`)
		_, err := gw.VerifyAndParseWebhook(context.Background(), tampered, sig)
		if err == nil {
			t.Fatal("expected error for tampered payload, got nil")
		}
		if !errors.Is(err, payment.ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature, got: %v", err)
		}
	})
}
