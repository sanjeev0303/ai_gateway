// Package stripe implements the Stripe payment gateway adapter for Sentinel.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/ports/payment"
)

const (
	defaultStripeAPIURL = "https://api.stripe.com"
	defaultTolerance   = 5 * time.Minute
)

// Gateway implements payment.PaymentGateway for Stripe.
type Gateway struct {
	secretKey     string
	webhookSecret string
	apiURL        string
	httpClient    *http.Client
}

// New creates a new Stripe payment gateway.
func New(secretKey, webhookSecret string) *Gateway {
	return &Gateway{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		apiURL:        defaultStripeAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewCustomGateway allows specifying a custom base URL (e.g. for mock server tests).
func NewCustomGateway(secretKey, webhookSecret, apiURL string) *Gateway {
	if apiURL == "" {
		apiURL = defaultStripeAPIURL
	}
	return &Gateway{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		apiURL:        apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider identifier.
func (g *Gateway) Name() string {
	return "stripe"
}

// CreateOrGetCustomer provisions or retrieves the upstream Stripe customer entity.
func (g *Gateway) CreateOrGetCustomer(ctx context.Context, orgID uuid.UUID, email, name string) (string, error) {
	if orgID == uuid.Nil {
		return "", errors.New("stripe: organization ID is required")
	}
	if g.secretKey == "" || strings.HasPrefix(g.secretKey, "dummy-") || strings.HasPrefix(g.secretKey, "sk_test_placeholder") {
		return "cus_test_" + orgID.String()[:12], nil
	}

	// 1. Check if customer already exists by email
	if email != "" {
		reqURL := fmt.Sprintf("%s/v1/customers?email=%s&limit=1", strings.TrimRight(g.apiURL, "/"), url.QueryEscape(email))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+g.secretKey)
			resp, err := g.httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var listRes struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&listRes); err == nil && len(listRes.Data) > 0 {
						return listRes.Data[0].ID, nil
					}
				}
			}
		}
	}

	// 2. Create customer
	data := url.Values{}
	if email != "" {
		data.Set("email", email)
	}
	if name != "" {
		data.Set("name", name)
	}
	data.Set("metadata[organization_id]", orgID.String())

	reqURL := strings.TrimRight(g.apiURL, "/") + "/v1/customers"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("stripe create customer failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode stripe customer response: %w", err)
	}
	return res.ID, nil
}

// CreateCheckoutSession generates a Stripe Checkout Session URL.
func (g *Gateway) CreateCheckoutSession(ctx context.Context, params payment.CheckoutParams) (*payment.CheckoutSession, error) {
	if params.PlanCode == "" {
		return nil, payment.ErrPlanNotFound
	}
	if params.OrganizationID == uuid.Nil {
		return nil, errors.New("stripe: organization ID is required")
	}

	// If no secret key is set or using a test dummy, return a simulated deterministic checkout session
	if g.secretKey == "" || strings.HasPrefix(g.secretKey, "dummy-") || strings.HasPrefix(g.secretKey, "sk_test_placeholder") {
		sessionID := "cs_test_" + uuid.NewString()[:18]
		checkoutURL := fmt.Sprintf("https://checkout.stripe.com/c/pay/%s?plan=%s&org=%s", sessionID, params.PlanCode, params.OrganizationID.String())
		if params.SuccessURL != "" {
			checkoutURL += "&success_url=" + url.QueryEscape(params.SuccessURL)
		}
		return &payment.CheckoutSession{
			SessionID:  sessionID,
			URL:        checkoutURL,
			Provider:   "stripe",
			CustomerID: params.CustomerID,
		}, nil
	}

	data := url.Values{}
	data.Set("mode", "subscription")
	data.Set("success_url", params.SuccessURL)
	data.Set("cancel_url", params.CancelURL)
	data.Set("client_reference_id", params.OrganizationID.String())
	if params.CustomerID != "" {
		data.Set("customer", params.CustomerID)
	} else if params.CustomerEmail != "" {
		data.Set("customer_email", params.CustomerEmail)
	}
	data.Set("metadata[organization_id]", params.OrganizationID.String())
	data.Set("metadata[plan_code]", params.PlanCode)
	data.Set("line_items[0][price_data][currency]", "usd")
	data.Set("line_items[0][price_data][product_data][name]", fmt.Sprintf("Sentinel %s Plan", strings.ToUpper(params.PlanCode)))
	data.Set("line_items[0][price_data][recurring][interval]", "month")
	data.Set("line_items[0][price_data][unit_amount]", "4900") // $49.00
	data.Set("line_items[0][quantity]", "1")

	reqURL := strings.TrimRight(g.apiURL, "/") + "/v1/checkout/sessions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stripe checkout session failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Customer string `json:"customer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode stripe checkout response: %w", err)
	}

	return &payment.CheckoutSession{
		SessionID:  res.ID,
		URL:        res.URL,
		Provider:   "stripe",
		CustomerID: res.Customer,
	}, nil
}

// VerifyAndParseWebhook validates the Stripe signature and converts the event into a standard WebhookEvent.
func (g *Gateway) VerifyAndParseWebhook(ctx context.Context, payload []byte, sigHeader string) (*payment.WebhookEvent, error) {
	if g.webhookSecret == "" || sigHeader == "" {
		return nil, payment.ErrInvalidSignature
	}
	if err := verifySignature(payload, sigHeader, g.webhookSecret); err != nil {
		return nil, err
	}

	var root struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object map[string]any `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("unmarshal stripe payload: %w", err)
	}

	obj := root.Data.Object
	evt := &payment.WebhookEvent{
		EventID: root.ID,
		Payload: payload,
	}

	// Extract metadata or client_reference_id
	meta, _ := obj["metadata"].(map[string]any)
	if meta != nil {
		if orgStr, ok := meta["organization_id"].(string); ok {
			evt.OrganizationID, _ = uuid.Parse(orgStr)
		}
		if plan, ok := meta["plan_code"].(string); ok {
			evt.PlanCode = plan
		}
	}
	if evt.OrganizationID == uuid.Nil {
		if ref, ok := obj["client_reference_id"].(string); ok && ref != "" {
			evt.OrganizationID, _ = uuid.Parse(ref)
		}
	}

	switch root.Type {
	case "checkout.session.completed":
		evt.EventType = "checkout.completed"
		evt.Status = "active"
		if subID, ok := obj["subscription"].(string); ok {
			evt.SubscriptionID = subID
		}
		if custID, ok := obj["customer"].(string); ok {
			evt.CustomerID = custID
		}

	case "customer.subscription.created", "customer.subscription.updated":
		evt.EventType = "subscription.updated"
		if st, ok := obj["status"].(string); ok {
			evt.Status = st
		}
		if subID, ok := obj["id"].(string); ok {
			evt.SubscriptionID = subID
		}
		if custID, ok := obj["customer"].(string); ok {
			evt.CustomerID = custID
		}

	case "customer.subscription.deleted":
		evt.EventType = "subscription.deleted"
		evt.Status = "canceled"
		if subID, ok := obj["id"].(string); ok {
			evt.SubscriptionID = subID
		}
		if custID, ok := obj["customer"].(string); ok {
			evt.CustomerID = custID
		}

	case "invoice.payment_succeeded":
		evt.EventType = "payment.succeeded"
		evt.Status = "paid"
		if id, ok := obj["id"].(string); ok {
			evt.PaymentID = id
		}
		if subID, ok := obj["subscription"].(string); ok {
			evt.SubscriptionID = subID
		}
		if custID, ok := obj["customer"].(string); ok {
			evt.CustomerID = custID
		}
		if amt, ok := obj["amount_paid"].(float64); ok {
			evt.AmountMicros = int64(amt) * 10000 // cents to micro-dollars ($1 = 100 cents = 1,000,000 micros)
		}
		if curr, ok := obj["currency"].(string); ok {
			evt.Currency = curr
		}

	case "invoice.payment_failed":
		evt.EventType = "payment.failed"
		evt.Status = "past_due"
		if id, ok := obj["id"].(string); ok {
			evt.PaymentID = id
		}
		if subID, ok := obj["subscription"].(string); ok {
			evt.SubscriptionID = subID
		}
		if custID, ok := obj["customer"].(string); ok {
			evt.CustomerID = custID
		}

	case "charge.refunded":
		evt.EventType = "payment.refunded"
		evt.Status = "canceled"
		if custID, ok := obj["customer"].(string); ok {
			evt.CustomerID = custID
		}

	default:
		evt.EventType = root.Type
		if st, ok := obj["status"].(string); ok {
			evt.Status = st
		}
	}

	return evt, nil
}

// VerifyPaymentCallback verifies the checkout session status or payment intent if provided.
func (g *Gateway) VerifyPaymentCallback(ctx context.Context, orderID, paymentID, signature string) error {
	sessionID := orderID
	if sessionID == "" {
		sessionID = paymentID
	}
	if sessionID == "" {
		return errors.New("stripe: session_id or payment_id required for callback verification")
	}
	if g.secretKey == "" || strings.HasPrefix(g.secretKey, "dummy-") || strings.HasPrefix(g.secretKey, "sk_test_placeholder") {
		return nil
	}

	reqURL := fmt.Sprintf("%s/v1/checkout/sessions/%s", strings.TrimRight(g.apiURL, "/"), sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return payment.ErrInvalidSignature
	}

	var res struct {
		PaymentStatus string `json:"payment_status"`
		Status        string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return payment.ErrInvalidSignature
	}
	if res.PaymentStatus != "paid" && res.Status != "complete" {
		return errors.New("stripe: checkout session is not paid")
	}
	return nil
}

// CancelSubscription cancels an active Stripe subscription.
func (g *Gateway) CancelSubscription(ctx context.Context, subID string) error {
	if subID == "" {
		return errors.New("stripe: subscription ID required")
	}
	if g.secretKey == "" || strings.HasPrefix(g.secretKey, "dummy-") {
		return nil
	}

	reqURL := fmt.Sprintf("%s/v1/subscriptions/%s", strings.TrimRight(g.apiURL, "/"), subID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.secretKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stripe cancel subscription failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}


// verifySignature checks Stripe webhook signature format: t=<timestamp>,v1=<sig>.
func verifySignature(payload []byte, header, secret string) error {
	pairs := strings.Split(header, ",")
	var timestamp string
	var signatures []string

	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "t":
			timestamp = parts[1]
		case "v1":
			signatures = append(signatures, parts[1])
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return payment.ErrInvalidSignature
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return payment.ErrInvalidSignature
	}

	// Verify timestamp freshness
	eventTime := time.Unix(ts, 0)
	if time.Since(eventTime).Abs() > defaultTolerance {
		return fmt.Errorf("%w: timestamp out of tolerance", payment.ErrInvalidSignature)
	}

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expectedSig)) {
			return nil
		}
	}

	return payment.ErrInvalidSignature
}
