// Package payment defines the payment gateway port (M-Billing) for Sentinel.
// Supports multi-provider checkout sessions, subscriptions, and webhooks (Stripe and Razorpay).
package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidSignature    = errors.New("payment: invalid webhook signature")
	ErrUnsupportedProvider = errors.New("payment: unsupported payment provider")
	ErrPlanNotFound        = errors.New("payment: plan not found")
	ErrCustomerNotFound    = errors.New("payment: customer not found")
)

// CheckoutParams defines the inputs required to initiate a checkout session.
type CheckoutParams struct {
	OrganizationID uuid.UUID
	CustomerEmail  string
	CustomerID     string
	PlanCode       string
	SuccessURL     string
	CancelURL      string
	Currency       string
	AmountMicros   int64
}

// CheckoutSession represents the generated session with a redirection URL.
type CheckoutSession struct {
	SessionID      string `json:"session_id"`
	URL            string `json:"url"`
	Provider       string `json:"provider"`
	CustomerID     string `json:"customer_id,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	OrderID        string `json:"order_id,omitempty"`
	KeyID          string `json:"key_id,omitempty"`
}

// WebhookEvent represents a standardized billing lifecycle event from an external processor.
type WebhookEvent struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"` // e.g. "checkout.completed", "subscription.updated", "subscription.deleted", "payment.succeeded", "payment.failed"
	OrganizationID uuid.UUID `json:"organization_id"`
	PlanCode       string `json:"plan_code"`
	SubscriptionID string `json:"subscription_id"`
	CustomerID     string `json:"customer_id"`
	PaymentID      string `json:"payment_id"`
	AmountMicros   int64  `json:"amount_micros"`
	Currency       string `json:"currency"`
	Status         string `json:"status"` // "active", "trialing", "canceled", "past_due", "paid", "incomplete"
	Payload        []byte `json:"payload"`
}

// PaymentGateway abstracts external payment providers (Stripe, Razorpay).
type PaymentGateway interface {
	// Name returns the provider identifier ("stripe", "razorpay").
	Name() string
	// CreateOrGetCustomer provisions or retrieves the upstream customer entity.
	CreateOrGetCustomer(ctx context.Context, orgID uuid.UUID, email, name string) (string, error)
	// CreateCheckoutSession generates a checkout URL or session for plan subscription.
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (*CheckoutSession, error)
	// VerifyAndParseWebhook validates the payload signature and standardizes the event.
	VerifyAndParseWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
	// VerifyPaymentCallback cryptographically verifies a frontend checkout completion callback.
	VerifyPaymentCallback(ctx context.Context, orderID, paymentID, signature string) error
	// CancelSubscription terminates or schedules cancellation of an upstream subscription.
	CancelSubscription(ctx context.Context, subID string) error
}

