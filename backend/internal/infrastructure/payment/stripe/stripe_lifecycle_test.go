package stripe_test

import (
	"context"
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

	"ai-governance-saas/backend/internal/infrastructure/payment/stripe"
	"ai-governance-saas/backend/internal/ports/payment"
)

func TestStripeLifecycle(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	webhookSecret := "whsec_test_stripe_lifecycle"

	t.Run("CreateOrGetCustomer provisions and retrieves customer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/v1/customers" {
				// Return existing customer
				resp := map[string]any{
					"data": []map[string]any{
						{"id": "cus_existing_123"},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			if r.Method == http.MethodPost && r.URL.Path == "/v1/customers" {
				resp := map[string]any{
					"id": "cus_created_456",
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		gw := stripe.NewCustomGateway("sk_test_mock_key", webhookSecret, server.URL)
		custID, err := gw.CreateOrGetCustomer(context.Background(), orgID, "finance@acme.corp", "Acme Admin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if custID != "cus_existing_123" {
			t.Errorf("expected cus_existing_123, got %s", custID)
		}
	})

	t.Run("CreateCheckoutSession handles timeouts and network failures", func(t *testing.T) {
		// Server that times out
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusGatewayTimeout)
		}))
		defer server.Close()

		gw := stripe.NewCustomGateway("sk_test_live_key", webhookSecret, server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := gw.CreateCheckoutSession(ctx, payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
			SuccessURL:     "http://example.com/success",
			CancelURL:      "http://example.com/cancel",
		})
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
	})

	t.Run("VerifyAndParseWebhook validates signatures and rejects tampering", func(t *testing.T) {
		gw := stripe.New("sk_test_key", webhookSecret)
		payload := fmt.Appendf(nil, `{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"metadata":{"organization_id":"%s"}}}}`, orgID)

		// 1. Invalid signature format
		_, err := gw.VerifyAndParseWebhook(context.Background(), payload, "invalid-sig")
		if err == nil {
			t.Error("expected error for invalid signature, got nil")
		}

		// 2. Tampered payload
		ts := time.Now().Unix()
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(fmt.Appendf(nil, "%d.%s", ts, string(payload)))
		validSig := hex.EncodeToString(mac.Sum(nil))
		sigHeader := fmt.Sprintf("t=%d,v1=%s", ts, validSig)

		tamperedPayload := []byte(string(payload) + "tampered")
		_, err = gw.VerifyAndParseWebhook(context.Background(), tamperedPayload, sigHeader)
		if err == nil {
			t.Error("expected error for tampered payload, got nil")
		}

		// 3. Valid signature succeeds
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, sigHeader)
		if err != nil {
			t.Fatalf("expected valid signature to pass, got: %v", err)
		}
		if evt.EventType != "checkout.completed" {
			t.Errorf("expected checkout.completed, got %s", evt.EventType)
		}
	})

	t.Run("VerifyPaymentCallback checks checkout session status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/checkout/sessions/cs_paid_123" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"payment_status": "paid",
					"status":         "complete",
				})
				return
			}
			if r.URL.Path == "/v1/checkout/sessions/cs_unpaid_456" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"payment_status": "unpaid",
					"status":         "open",
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		gw := stripe.NewCustomGateway("sk_test_mock", webhookSecret, server.URL)

		// Paid session succeeds
		if err := gw.VerifyPaymentCallback(context.Background(), "cs_paid_123", "", ""); err != nil {
			t.Errorf("expected paid session to verify, got %v", err)
		}

		// Unpaid session fails
		if err := gw.VerifyPaymentCallback(context.Background(), "cs_unpaid_456", "", ""); err == nil {
			t.Error("expected unpaid session to fail, got nil")
		}
	})

	t.Run("CancelSubscription sends DELETE request", func(t *testing.T) {
		deleted := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete && r.URL.Path == "/v1/subscriptions/sub_test_del" {
				deleted = true
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		gw := stripe.NewCustomGateway("sk_test_mock", webhookSecret, server.URL)
		if err := gw.CancelSubscription(context.Background(), "sub_test_del"); err != nil {
			t.Fatalf("unexpected cancel error: %v", err)
		}
		if !deleted {
			t.Error("expected subscription delete request to be sent")
		}
	})
}
