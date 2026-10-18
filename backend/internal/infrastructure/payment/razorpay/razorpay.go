// Package razorpay implements the Razorpay payment gateway adapter for Sentinel.
package razorpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/ports/payment"
)

const (
	defaultRazorpayAPIURL = "https://api.razorpay.com"
)

// Gateway implements payment.PaymentGateway for Razorpay.
type Gateway struct {
	keyID         string
	keySecret     string
	webhookSecret string
	apiURL        string
	httpClient    *http.Client
}

// New creates a new Razorpay payment gateway.
func New(keyID, keySecret, webhookSecret string) *Gateway {
	return &Gateway{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		apiURL:        defaultRazorpayAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewCustomGateway allows specifying a custom base URL (e.g. for mock server tests).
func NewCustomGateway(keyID, keySecret, webhookSecret, apiURL string) *Gateway {
	if apiURL == "" {
		apiURL = defaultRazorpayAPIURL
	}
	return &Gateway{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		apiURL:        apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider identifier.
func (g *Gateway) Name() string {
	return "razorpay"
}

// CreateOrGetCustomer provisions or retrieves the upstream Razorpay customer entity.
func (g *Gateway) CreateOrGetCustomer(ctx context.Context, orgID uuid.UUID, email, name string) (string, error) {
	if orgID == uuid.Nil {
		return "", errors.New("razorpay: organization ID is required")
	}
	if g.keyID == "" || strings.HasPrefix(g.keyID, "dummy-") || strings.HasPrefix(g.keyID, "rzp_test_placeholder") {
		return "cust_rzp_" + orgID.String()[:12], nil
	}

	bodyMap := map[string]any{
		"name":  name,
		"email": email,
		"notes": map[string]string{
			"organization_id": orgID.String(),
		},
	}
	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return "", err
	}

	reqURL := strings.TrimRight(g.apiURL, "/") + "/v1/customers"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(g.keyID, g.keySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("razorpay create customer failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode razorpay customer response: %w", err)
	}
	return res.ID, nil
}

// CreateCheckoutSession generates a Razorpay payment/subscription link.
func (g *Gateway) CreateCheckoutSession(ctx context.Context, params payment.CheckoutParams) (*payment.CheckoutSession, error) {
	if params.PlanCode == "" {
		return nil, payment.ErrPlanNotFound
	}
	if params.OrganizationID == uuid.Nil {
		return nil, errors.New("razorpay: organization ID is required")
	}

	// If no keys configured or test dummy, return a simulated deterministic checkout link
	if g.keyID == "" || strings.HasPrefix(g.keyID, "dummy-") || strings.HasPrefix(g.keyID, "rzp_test_placeholder") {
		sessionID := "sub_link_" + uuid.NewString()[:18]
		checkoutURL := fmt.Sprintf("https://rzp.io/i/%s?plan=%s&org=%s", sessionID, params.PlanCode, params.OrganizationID.String())
		return &payment.CheckoutSession{
			SessionID:  sessionID,
			URL:        checkoutURL,
			Provider:   "razorpay",
			CustomerID: params.CustomerID,
			OrderID:    "order_test_" + uuid.NewString()[:12],
			KeyID:      "rzp_test_dummy",
		}, nil
	}

	bodyMap := map[string]any{
		"amount":       490000, // 4900.00 INR (in paise)
		"currency":     "INR",
		"description":  fmt.Sprintf("Sentinel %s Plan Subscription", strings.ToUpper(params.PlanCode)),
		"callback_url": params.SuccessURL,
		"notes": map[string]string{
			"organization_id": params.OrganizationID.String(),
			"plan_code":       params.PlanCode,
		},
	}
	if params.CustomerID != "" {
		bodyMap["customer_id"] = params.CustomerID
	}
	if params.CustomerEmail != "" {
		bodyMap["customer"] = map[string]string{
			"email": params.CustomerEmail,
		}
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	reqURL := strings.TrimRight(g.apiURL, "/") + "/v1/payment_links"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(g.keyID, g.keySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("razorpay payment link creation failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		ID       string `json:"id"`
		ShortURL string `json:"short_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode razorpay link response: %w", err)
	}

	return &payment.CheckoutSession{
		SessionID:  res.ID,
		URL:        res.ShortURL,
		Provider:   "razorpay",
		CustomerID: params.CustomerID,
		OrderID:    res.ID,
		KeyID:      g.keyID,
	}, nil
}

// VerifyPaymentCallback cryptographically verifies the browser checkout signature (order_id|payment_id).
func (g *Gateway) VerifyPaymentCallback(ctx context.Context, orderID, paymentID, signature string) error {
	if signature == "" {
		return payment.ErrInvalidSignature
	}
	if paymentID == "" {
		return errors.New("razorpay: payment_id is required")
	}
	if g.keySecret == "" || strings.HasPrefix(g.keySecret, "dummy-") || strings.HasPrefix(g.keySecret, "rzp_test_placeholder") {
		return nil
	}

	// Subscriptions sign payment_id|subscription_id; standard orders sign order_id|payment_id
	var payload string
	if strings.HasPrefix(orderID, "sub_") {
		payload = paymentID + "|" + orderID
	} else if orderID != "" {
		payload = orderID + "|" + paymentID
	} else {
		payload = paymentID
	}

	mac := hmac.New(sha256.New, []byte(g.keySecret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		// Try alternate order if orderID present
		if orderID != "" {
			altPayload := paymentID + "|" + orderID
			macAlt := hmac.New(sha256.New, []byte(g.keySecret))
			macAlt.Write([]byte(altPayload))
			altExpected := hex.EncodeToString(macAlt.Sum(nil))
			if hmac.Equal([]byte(signature), []byte(altExpected)) {
				return nil
			}
		}
		return payment.ErrInvalidSignature
	}
	return nil
}

// VerifyAndParseWebhook validates the Razorpay webhook signature and maps the payload into a WebhookEvent.
func (g *Gateway) VerifyAndParseWebhook(ctx context.Context, payload []byte, signature string) (*payment.WebhookEvent, error) {
	if g.webhookSecret == "" || signature == "" {
		return nil, payment.ErrInvalidSignature
	}
	mac := hmac.New(sha256.New, []byte(g.webhookSecret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return nil, payment.ErrInvalidSignature
	}

	var root struct {
		Event     string `json:"event"`
		AccountID string `json:"account_id"`
		CreatedAt int64  `json:"created_at"`
		Payload   struct {
			Subscription struct {
				Entity map[string]any `json:"entity"`
			} `json:"subscription"`
			Payment struct {
				Entity map[string]any `json:"entity"`
			} `json:"payment"`
			Order struct {
				Entity map[string]any `json:"entity"`
			} `json:"order"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("unmarshal razorpay payload: %w", err)
	}

	// Read entity
	entity := root.Payload.Subscription.Entity
	if entity == nil || len(entity) == 0 {
		entity = root.Payload.Payment.Entity
	}
	if entity == nil || len(entity) == 0 {
		entity = root.Payload.Order.Entity
	}

	entityID := ""
	if entity != nil {
		if id, ok := entity["id"].(string); ok {
			entityID = id
		}
	}

	eventID := root.Event
	if entityID != "" {
		eventID = root.Event + "_" + entityID
	} else {
		eventID = fmt.Sprintf("%s_%s_%d", root.Event, root.AccountID, root.CreatedAt)
	}

	evt := &payment.WebhookEvent{
		EventID: eventID,
		Payload: payload,
	}

	if notes, ok := entity["notes"].(map[string]any); ok && notes != nil {
		if orgStr, ok := notes["organization_id"].(string); ok {
			evt.OrganizationID, _ = uuid.Parse(orgStr)
		}
		if plan, ok := notes["plan_code"].(string); ok {
			evt.PlanCode = plan
		}
	}

	switch root.Event {
	case "subscription.authenticated", "subscription.activated":
		evt.EventType = "subscription.updated"
		evt.Status = "active"
		if id, ok := entity["id"].(string); ok {
			evt.SubscriptionID = id
		}
		if custID, ok := entity["customer_id"].(string); ok {
			evt.CustomerID = custID
		}

	case "subscription.charged", "payment.captured":
		evt.EventType = "payment.succeeded"
		evt.Status = "paid"
		if id, ok := entity["id"].(string); ok {
			evt.PaymentID = id
		}
		if subID, ok := entity["subscription_id"].(string); ok {
			evt.SubscriptionID = subID
		} else if id, ok := entity["id"].(string); ok && strings.HasPrefix(id, "sub_") {
			evt.SubscriptionID = id
		}
		if custID, ok := entity["customer_id"].(string); ok {
			evt.CustomerID = custID
		}
		if amt, ok := entity["amount"].(float64); ok {
			// Razorpay amounts are in smallest currency units (e.g. paise: 100 paise = 1 INR = 10,000 micros)
			evt.AmountMicros = int64(amt) * 10000
		}
		if curr, ok := entity["currency"].(string); ok {
			evt.Currency = strings.ToLower(curr)
		}

	case "subscription.cancelled", "subscription.halted":
		evt.EventType = "subscription.deleted"
		evt.Status = "canceled"
		if id, ok := entity["id"].(string); ok {
			evt.SubscriptionID = id
		}
		if custID, ok := entity["customer_id"].(string); ok {
			evt.CustomerID = custID
		}

	case "payment.failed":
		evt.EventType = "payment.failed"
		evt.Status = "past_due"
		if id, ok := entity["id"].(string); ok {
			evt.PaymentID = id
		}
		if custID, ok := entity["customer_id"].(string); ok {
			evt.CustomerID = custID
		}

	default:
		evt.EventType = root.Event
		if st, ok := entity["status"].(string); ok {
			evt.Status = st
		}
	}

	return evt, nil
}

// CancelSubscription cancels an active Razorpay subscription.
func (g *Gateway) CancelSubscription(ctx context.Context, subID string) error {
	if subID == "" {
		return errors.New("razorpay: subscription ID required")
	}
	if g.keyID == "" || strings.HasPrefix(g.keyID, "dummy-") {
		return nil
	}

	reqURL := fmt.Sprintf("%s/v1/subscriptions/%s/cancel", strings.TrimRight(g.apiURL, "/"), subID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.keyID, g.keySecret)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("razorpay cancel subscription failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

