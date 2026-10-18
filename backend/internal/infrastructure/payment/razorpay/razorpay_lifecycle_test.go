package razorpay_test

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

	"ai-governance-saas/backend/internal/infrastructure/payment/razorpay"
	"ai-governance-saas/backend/internal/ports/payment"
)

func TestRazorpayLifecycle(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	keyID := "rzp_test_key_123"
	keySecret := "rzp_test_secret_456"
	webhookSecret := "whsec_test_rzp_lifecycle"

	t.Run("CreateOrGetCustomer provisions customer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/v1/customers" {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["email"] != "dev@acme.corp" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": "cust_rzp_mock_999",
				})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		gw := razorpay.NewCustomGateway(keyID, keySecret, webhookSecret, server.URL)
		custID, err := gw.CreateOrGetCustomer(context.Background(), orgID, "dev@acme.corp", "Acme Developer")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if custID != "cust_rzp_mock_999" {
			t.Errorf("expected cust_rzp_mock_999, got %s", custID)
		}
	})

	t.Run("VerifyPaymentCallback validates authentic signature and rejects forgeries", func(t *testing.T) {
		gw := razorpay.New(keyID, keySecret, webhookSecret)

		orderID := "order_test_98765"
		paymentID := "pay_test_54321"

		// 1. Generate authentic signature
		payload := orderID + "|" + paymentID
		mac := hmac.New(sha256.New, []byte(keySecret))
		mac.Write([]byte(payload))
		authenticSig := hex.EncodeToString(mac.Sum(nil))

		// Authentic signature must pass
		if err := gw.VerifyPaymentCallback(context.Background(), orderID, paymentID, authenticSig); err != nil {
			t.Errorf("expected valid signature to verify, got: %v", err)
		}

		// 2. Forged signature must be rejected
		forgedSig := authenticSig[:len(authenticSig)-4] + "dead"
		if err := gw.VerifyPaymentCallback(context.Background(), orderID, paymentID, forgedSig); err == nil {
			t.Error("expected forged signature to fail, got nil")
		}

		// 3. Subscription format (payment_id|subscription_id)
		subID := "sub_test_112233"
		subPayload := paymentID + "|" + subID
		macSub := hmac.New(sha256.New, []byte(keySecret))
		macSub.Write([]byte(subPayload))
		authenticSubSig := hex.EncodeToString(macSub.Sum(nil))

		if err := gw.VerifyPaymentCallback(context.Background(), subID, paymentID, authenticSubSig); err != nil {
			t.Errorf("expected subscription signature to verify, got: %v", err)
		}
	})

	t.Run("VerifyAndParseWebhook validates signatures and extracts entities", func(t *testing.T) {
		gw := razorpay.New(keyID, keySecret, webhookSecret)

		payload := fmt.Appendf(nil, `{
			"entity": "event",
			"account_id": "acc_123",
			"event": "payment.captured",
			"payload": {
				"payment": {
					"entity": {
						"id": "pay_captured_888",
						"amount": 490000,
						"currency": "INR",
						"notes": {
							"organization_id": "%s",
							"plan_code": "pro"
						}
					}
				}
			}
		}`, orgID)

		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write(payload)
		validSig := hex.EncodeToString(mac.Sum(nil))

		// 1. Valid signature passes
		evt, err := gw.VerifyAndParseWebhook(context.Background(), payload, validSig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if evt.EventType != "payment.succeeded" {
			t.Errorf("expected payment.succeeded, got %s", evt.EventType)
		}
		if evt.PaymentID != "pay_captured_888" {
			t.Errorf("expected pay_captured_888, got %s", evt.PaymentID)
		}
		if evt.EventID != "payment.captured_pay_captured_888" {
			t.Errorf("expected deterministic event ID, got %s", evt.EventID)
		}

		// 2. Tampered signature fails
		if _, err := gw.VerifyAndParseWebhook(context.Background(), payload, "bad_sig"); err == nil {
			t.Error("expected error for bad signature, got nil")
		}
	})

	t.Run("CancelSubscription sends cancel request", func(t *testing.T) {
		cancelled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/v1/subscriptions/sub_rzp_cancel/cancel" {
				cancelled = true
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()

		gw := razorpay.NewCustomGateway(keyID, keySecret, webhookSecret, server.URL)
		if err := gw.CancelSubscription(context.Background(), "sub_rzp_cancel"); err != nil {
			t.Fatalf("unexpected cancel error: %v", err)
		}
		if !cancelled {
			t.Error("expected razorpay cancel request to be sent")
		}
	})

	t.Run("CreateCheckoutSession handles provider timeouts", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusGatewayTimeout)
		}))
		defer server.Close()

		gw := razorpay.NewCustomGateway(keyID, keySecret, webhookSecret, server.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := gw.CreateCheckoutSession(ctx, payment.CheckoutParams{
			OrganizationID: orgID,
			PlanCode:       "pro",
			SuccessURL:     "http://example.com/success",
		})
		if err == nil {
			t.Fatal("expected timeout error, got nil")
		}
	})
}
