// Package billing serves the organization's local billing state: the plan
// catalog, its current subscription, period usage cost, and locally computed
// invoices. Supports Stripe and Razorpay dual-gateway integration, verified
// checkout tracking, and idempotent webhook reconciliation.
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/payment"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ServiceDeps are the repositories and payment gateways a billing service requires.
type ServiceDeps struct {
	Mode            string
	Plans           repositories.PlanRepository
	Subscriptions   repositories.SubscriptionRepository
	Invoices        repositories.InvoiceRepository
	Checkouts       repositories.CheckoutRepository
	Rollups         repositories.UsageRollupRepository
	BillingEvents   repositories.BillingEventRepository
	Audits          repositories.AuditLogRepository
	PaymentGateways map[string]payment.PaymentGateway
}

// Service owns reads and mutations of billing state and checkout sessions.
type Service struct {
	mode            string
	plans           repositories.PlanRepository
	subscriptions   repositories.SubscriptionRepository
	invoices        repositories.InvoiceRepository
	checkouts       repositories.CheckoutRepository
	rollups         repositories.UsageRollupRepository
	billingEvents   repositories.BillingEventRepository
	audits          repositories.AuditLogRepository
	paymentGateways map[string]payment.PaymentGateway
}

// NewService returns a billing service.
func NewService(deps ServiceDeps) *Service {
	gateways := deps.PaymentGateways
	if gateways == nil {
		gateways = make(map[string]payment.PaymentGateway)
	}
	mode := deps.Mode
	if mode == "" {
		mode = "test"
	}
	for _, gw := range gateways {
		if gw != nil {
			gw.SetMode(mode)
		}
	}
	return &Service{
		mode:            mode,
		plans:           deps.Plans,
		subscriptions:   deps.Subscriptions,
		invoices:        deps.Invoices,
		checkouts:       deps.Checkouts,
		rollups:         deps.Rollups,
		billingEvents:   deps.BillingEvents,
		audits:          deps.Audits,
		paymentGateways: gateways,
	}
}

// Mode returns the active billing mode.
func (s *Service) Mode() string {
	if s.mode == "" {
		return "test"
	}
	return s.mode
}

// SetMode updates the active billing mode and propagates to payment gateways.
func (s *Service) SetMode(m string) {
	s.mode = m
	for _, gw := range s.paymentGateways {
		if gw != nil {
			gw.SetMode(m)
		}
	}
}

// RequirePermission authorizes actor for perm, returning
// repositories.ErrForbidden when the actor's role does not grant it.
func (s *Service) RequirePermission(actor domain.Actor, perm string) error {
	if !actor.Can(perm) {
		return repositories.ErrForbidden
	}
	return nil
}

// ListPlans returns the global plan catalog. Plans are not tenant data, but the
// read is still authorized so that only roles granted billing.read see them.
func (s *Service) ListPlans(ctx context.Context, actor domain.Actor) ([]domain.Plan, error) {
	if err := s.RequirePermission(actor, domain.PermissionBillingRead); err != nil {
		return nil, err
	}
	return s.plans.List(ctx)
}

// GetSubscription returns the organization's active or trialing subscription
// joined to its plan, or nil when it has none.
func (s *Service) GetSubscription(ctx context.Context, actor domain.Actor) (*domain.OrganizationPlan, error) {
	if err := s.RequirePermission(actor, domain.PermissionBillingRead); err != nil {
		return nil, err
	}
	plan, err := s.subscriptions.ActiveWithPlan(ctx, actor.OrganizationID)
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	}
	return &plan, nil
}

// GetUsage returns the organization's usage cost for the requested window,
// which callers bound with the same query parameters as the usage endpoints.
func (s *Service) GetUsage(ctx context.Context, actor domain.Actor, filter domain.UsageFilter) (domain.UsageSummary, error) {
	if err := s.RequirePermission(actor, domain.PermissionBillingRead); err != nil {
		return domain.UsageSummary{}, err
	}
	return s.rollups.GetSummary(ctx, actor.OrganizationID, filter)
}

// ListInvoices returns the organization's invoices newest-issued first.
func (s *Service) ListInvoices(ctx context.Context, actor domain.Actor) ([]domain.Invoice, error) {
	if err := s.RequirePermission(actor, domain.PermissionBillingRead); err != nil {
		return nil, err
	}
	return s.invoices.List(ctx, actor.OrganizationID)
}

// UpdateSubscription changes the organization's plan to target planCode.
func (s *Service) UpdateSubscription(ctx context.Context, actor domain.Actor, planCode string) (*domain.OrganizationPlan, error) {
	if s.Mode() == "disabled" {
		return nil, payment.ErrBillingDisabled
	}
	if !actor.Can(domain.PermissionBillingManage) && actor.Role != domain.RoleAdmin {
		return nil, repositories.ErrForbidden
	}
	targetPlan, err := s.findPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	if err := s.subscriptions.SetPlan(ctx, actor.OrganizationID, targetPlan.ID); err != nil {
		return nil, err
	}
	return s.GetSubscription(ctx, actor)
}

// CancelSubscription marks the organization's active subscription to cancel at period end.
func (s *Service) CancelSubscription(ctx context.Context, actor domain.Actor) (*domain.OrganizationPlan, error) {
	if s.Mode() == "disabled" {
		return nil, payment.ErrBillingDisabled
	}
	if !actor.Can(domain.PermissionBillingManage) && actor.Role != domain.RoleAdmin {
		return nil, repositories.ErrForbidden
	}
	if err := s.subscriptions.Cancel(ctx, actor.OrganizationID); err != nil {
		return nil, err
	}
	return s.GetSubscription(ctx, actor)
}

// CreateCheckoutSession initiates a payment session with the chosen provider (Stripe or Razorpay).
// Persists an authoritative server-side checkout record with expected price and currency.
func (s *Service) CreateCheckoutSession(ctx context.Context, actor domain.Actor, providerName, planCode, customerEmail, successURL, cancelURL string) (*payment.CheckoutSession, error) {
	if s.Mode() == "disabled" {
		return nil, payment.ErrBillingDisabled
	}
	if !actor.Can(domain.PermissionBillingManage) && actor.Role != domain.RoleAdmin {
		return nil, repositories.ErrForbidden
	}
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	if providerName == "" {
		providerName = "stripe"
	}
	gw := s.paymentGateways[providerName]
	if gw == nil {
		return nil, payment.ErrUnsupportedProvider
	}

	// Verify plan exists
	targetPlan, err := s.findPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}

	// Determine authoritative pricing and currency from plan catalog
	expectedMicros := targetPlan.PriceMicros
	expectedCurrency := targetPlan.Currency
	if expectedMicros <= 0 {
		expectedMicros, expectedCurrency = domain.GetPlanPrice(planCode)
	}
	if expectedCurrency == "" {
		expectedCurrency = "usd"
	}

	// Retrieve provider customer ID if already linked
	var customerID string
	if activeSub, err := s.subscriptions.ActiveWithPlan(ctx, actor.OrganizationID); err == nil && activeSub.ProviderCustomerID != nil {
		customerID = *activeSub.ProviderCustomerID
	}

	// If no customer ID exists, ensure one is provisioned upstream
	if customerID == "" && customerEmail != "" {
		if cid, err := gw.CreateOrGetCustomer(ctx, actor.OrganizationID, customerEmail, actor.DisplayName); err == nil && cid != "" {
			customerID = cid
		}
	}

	session, err := gw.CreateCheckoutSession(ctx, payment.CheckoutParams{
		OrganizationID: actor.OrganizationID,
		CustomerEmail:  customerEmail,
		CustomerID:     customerID,
		PlanCode:       planCode,
		SuccessURL:     successURL,
		CancelURL:      cancelURL,
		Currency:       expectedCurrency,
		AmountMicros:   expectedMicros,
	})
	if err != nil {
		return nil, err
	}

	// Persist trusted server-side checkout record
	if s.checkouts != nil && session != nil && session.SessionID != "" {
		orderID := session.OrderID
		if orderID == "" {
			orderID = session.SessionID
		}
		_, err = s.checkouts.Create(ctx, actor.OrganizationID, domain.NewCheckout{
			CheckoutSessionID: session.SessionID,
			Provider:          providerName,
			PlanCode:          planCode,
			PlanID:            &targetPlan.ID,
			AmountMicros:      expectedMicros,
			Currency:          expectedCurrency,
			CustomerEmail:     customerEmail,
			CustomerID:        customerID,
			OrderID:           orderID,
			ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
		})
		if err != nil {
			return nil, fmt.Errorf("billing: failed to persist checkout record: %w", err)
		}
	}

	// Record audit log for checkout creation without logging sensitive payment tokens
	if s.audits != nil {
		var actorUserID, actorServiceAccountID *uuid.UUID
		if actor.Kind == domain.ActorKindService {
			actorServiceAccountID = &actor.UserID
		} else {
			actorUserID = &actor.UserID
		}
		_, _ = s.audits.Append(ctx, actor.OrganizationID, domain.NewAuditLog{
			ActorUserID:           actorUserID,
			ActorServiceAccountID: actorServiceAccountID,
			Action:                "billing.checkout_created",
			ResourceType:          "subscription",
			After:                 fmt.Appendf(nil, `{"provider":"%s","plan":"%s","amount_micros":%d,"currency":"%s"}`, providerName, planCode, expectedMicros, expectedCurrency),
		})
	}

	return session, nil
}

// VerifyPaymentCallback verifies an upstream client payment callback and reconciles billing state.
// Derives expected amounts and plan version strictly from server-side records.
func (s *Service) VerifyPaymentCallback(ctx context.Context, actor domain.Actor, providerName, orderID, paymentID, signature, planCode string) (*domain.OrganizationPlan, error) {
	if s.Mode() == "disabled" {
		return nil, payment.ErrBillingDisabled
	}
	if s.Mode() == "simulation" {
		return nil, payment.ErrSimulationOnly
	}
	if !actor.Can(domain.PermissionBillingManage) && actor.Role != domain.RoleAdmin {
		return nil, repositories.ErrForbidden
	}
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(paymentID) == "" || strings.TrimSpace(signature) == "" {
		return nil, payment.ErrInvalidSignature
	}
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	gw := s.paymentGateways[providerName]
	if gw == nil {
		return nil, payment.ErrUnsupportedProvider
	}

	// 1. Cryptographic signature verification
	if err := gw.VerifyPaymentCallback(ctx, orderID, paymentID, signature); err != nil {
		return nil, err
	}

	// 2. Lookup trusted checkout record
	var checkout *domain.Checkout
	if s.checkouts != nil {
		if c, err := s.checkouts.GetByOrderID(ctx, actor.OrganizationID, orderID); err == nil && c.CheckoutSessionID != "" {
			checkout = &c
		}
	}

	var expectedMicros int64
	var expectedCurrency string

	if checkout != nil {
		if checkout.Status == domain.CheckoutStatusCompleted {
			// Idempotent retry: checkout already completed and reconciled
			return s.GetSubscription(ctx, actor)
		}
		if checkout.Status == domain.CheckoutStatusExpired || (!checkout.ExpiresAt.IsZero() && time.Now().UTC().After(checkout.ExpiresAt)) {
			_ = s.checkouts.UpdateStatus(ctx, actor.OrganizationID, checkout.CheckoutSessionID, domain.CheckoutStatusExpired, paymentID)
			return nil, payment.ErrCheckoutExpired
		}
		if checkout.Status == domain.CheckoutStatusCancelled {
			return nil, payment.ErrCheckoutCancelled
		}
		if planCode != "" && planCode != checkout.PlanCode {
			return nil, payment.ErrPlanMismatch
		}
		expectedMicros = checkout.AmountMicros
		expectedCurrency = checkout.Currency
		planCode = checkout.PlanCode
	} else {
		// Server-side plan lookup: determine authoritative plan and amount
		targetPlan, err := s.findPlan(ctx, planCode)
		if err != nil {
			return nil, err
		}
		if targetPlan.PriceMicros > 0 {
			expectedMicros = targetPlan.PriceMicros
			expectedCurrency = targetPlan.Currency
		} else {
			expectedMicros, expectedCurrency = domain.GetPlanPrice(planCode)
		}
	}

	targetPlan, err := s.findPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}

	// 3. State Machine Transitions & Entitlement Application (with zero ignored errors)
	// 3a. Upgrade subscription plan
	if err := s.subscriptions.SetPlan(ctx, actor.OrganizationID, targetPlan.ID); err != nil {
		return nil, fmt.Errorf("billing: update subscription plan: %w", err)
	}

	// 3b. Update payment details
	subID := orderID
	if err := s.subscriptions.UpdatePaymentDetails(ctx, actor.OrganizationID, providerName, subID, "", domain.SubscriptionStatusActive); err != nil {
		return nil, fmt.Errorf("billing: update payment details: %w", err)
	}

	// 3c. Record settled invoice with authoritative amount and currency
	if err := s.invoices.RecordPayment(ctx, actor.OrganizationID, providerName, paymentID, expectedMicros); err != nil {
		return nil, fmt.Errorf("billing: record invoice payment: %w", err)
	}

	// 3d. Mark checkout completed
	if checkout != nil && s.checkouts != nil {
		if err := s.checkouts.UpdateStatus(ctx, actor.OrganizationID, checkout.CheckoutSessionID, domain.CheckoutStatusCompleted, paymentID); err != nil {
			return nil, fmt.Errorf("billing: update checkout status: %w", err)
		}
	}

	// 3e. Record auditable billing event in ledger
	if s.billingEvents != nil {
		eventID := fmt.Sprintf("callback_%s_%s", providerName, paymentID)
		_, err := s.billingEvents.Record(ctx, actor.OrganizationID, domain.NewBillingEvent{
			BillingEventID: eventID,
			EventType:      "payment.callback_verified",
			AmountMicros:   &expectedMicros,
			Currency:       expectedCurrency,
			Payload:        fmt.Appendf(nil, `{"order_id":"%s","payment_id":"%s","plan_code":"%s"}`, orderID, paymentID, planCode),
		})
		if err != nil && !errors.Is(err, repositories.ErrConflict) {
			return nil, fmt.Errorf("billing: record billing event: %w", err)
		}
	}

	// 3f. Record audit log
	if s.audits != nil {
		var actorUserID, actorServiceAccountID *uuid.UUID
		if actor.Kind == domain.ActorKindService {
			actorServiceAccountID = &actor.UserID
		} else {
			actorUserID = &actor.UserID
		}
		_, err := s.audits.Append(ctx, actor.OrganizationID, domain.NewAuditLog{
			ActorUserID:           actorUserID,
			ActorServiceAccountID: actorServiceAccountID,
			Action:                "billing.payment_verified",
			ResourceType:          "subscription",
			After:                 fmt.Appendf(nil, `{"provider":"%s","plan":"%s","payment_id":"%s","amount_micros":%d}`, providerName, planCode, paymentID, expectedMicros),
		})
		if err != nil {
			return nil, fmt.Errorf("billing: record audit log: %w", err)
		}
	}

	return s.GetSubscription(ctx, actor)
}

// ProcessWebhook validates and handles an incoming payment processor webhook event.
// Enforces duplicate event protection, amount verification, and strict failure handling.
func (s *Service) ProcessWebhook(ctx context.Context, providerName string, payload []byte, signature string) (*payment.WebhookEvent, error) {
	if s.Mode() == "disabled" {
		return nil, payment.ErrBillingDisabled
	}
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	gw := s.paymentGateways[providerName]
	if gw == nil {
		return nil, payment.ErrUnsupportedProvider
	}

	evt, err := gw.VerifyAndParseWebhook(ctx, payload, signature)
	if err != nil {
		return nil, err
	}

	if evt.OrganizationID == uuid.Nil {
		// Event cannot be attributed to an organization
		return evt, nil
	}

	// 1. Idempotency check & billing ledger reservation: deduplicate event processing
	if s.billingEvents != nil && evt.EventID != "" {
		existing, err := s.billingEvents.Get(ctx, evt.OrganizationID, evt.EventID)
		if err == nil && existing.BillingEventID != "" {
			// Already successfully ingested and processed
			return evt, nil
		}
		// Record event in billing ledger
		_, err = s.billingEvents.Record(ctx, evt.OrganizationID, domain.NewBillingEvent{
			BillingEventID: evt.EventID,
			EventType:      evt.EventType,
			AmountMicros:   &evt.AmountMicros,
			Currency:       evt.Currency,
			Payload:        payload,
		})
		if errors.Is(err, repositories.ErrConflict) {
			// Duplicate event arriving concurrently
			return evt, nil
		}
		if err != nil {
			return nil, fmt.Errorf("billing event persistence failed: %w", err)
		}
	}

	// 2. Checkout validation & Amount verification
	var checkout *domain.Checkout
	if s.checkouts != nil {
		lookupID := evt.SubscriptionID
		if lookupID == "" {
			lookupID = evt.PaymentID
		}
		if lookupID != "" {
			if c, err := s.checkouts.GetByOrderID(ctx, evt.OrganizationID, lookupID); err == nil && c.CheckoutSessionID != "" {
				checkout = &c
			}
		}
	}

	if checkout != nil && (evt.EventType == "payment.succeeded" || evt.EventType == "checkout.completed") {
		if evt.AmountMicros > 0 && checkout.AmountMicros > 0 && evt.AmountMicros != checkout.AmountMicros {
			if s.billingEvents != nil {
				_, _ = s.billingEvents.Record(ctx, evt.OrganizationID, domain.NewBillingEvent{
					BillingEventID: evt.EventID,
					EventType:      evt.EventType + ".amount_mismatch",
					AmountMicros:   &evt.AmountMicros,
					Currency:       evt.Currency,
					Payload:        payload,
				})
			}
			return nil, payment.ErrAmountMismatch
		}
		if evt.Currency != "" && checkout.Currency != "" && !strings.EqualFold(evt.Currency, checkout.Currency) {
			if s.billingEvents != nil {
				_, _ = s.billingEvents.Record(ctx, evt.OrganizationID, domain.NewBillingEvent{
					BillingEventID: evt.EventID,
					EventType:      evt.EventType + ".currency_mismatch",
					AmountMicros:   &evt.AmountMicros,
					Currency:       evt.Currency,
					Payload:        payload,
				})
			}
			return nil, payment.ErrCurrencyMismatch
		}
	}

	// 3. State Machine Transitions & Entitlement Application (with strict error propagation)
	switch evt.EventType {
	case "checkout.completed":
		if evt.PlanCode != "" {
			if err := s.applyPlan(ctx, evt.OrganizationID, evt.PlanCode); err != nil {
				return nil, fmt.Errorf("billing: apply plan: %w", err)
			}
		}
		if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusActive); err != nil {
			return nil, fmt.Errorf("billing: update payment details: %w", err)
		}
		if checkout != nil && s.checkouts != nil {
			_ = s.checkouts.UpdateStatus(ctx, evt.OrganizationID, checkout.CheckoutSessionID, domain.CheckoutStatusCompleted, evt.PaymentID)
		}
		if evt.AmountMicros > 0 {
			paymentRef := evt.PaymentID
			if paymentRef == "" {
				paymentRef = evt.SubscriptionID
			}
			if err := s.invoices.RecordPayment(ctx, evt.OrganizationID, providerName, paymentRef, evt.AmountMicros); err != nil {
				return nil, fmt.Errorf("billing: record invoice: %w", err)
			}
		}

	case "subscription.updated":
		switch evt.Status {
		case domain.SubscriptionStatusActive, domain.SubscriptionStatusTrialing:
			if evt.PlanCode != "" {
				if err := s.applyPlan(ctx, evt.OrganizationID, evt.PlanCode); err != nil {
					return nil, fmt.Errorf("billing: apply plan: %w", err)
				}
			}
			if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, evt.Status); err != nil {
				return nil, fmt.Errorf("billing: update payment details: %w", err)
			}
		case domain.SubscriptionStatusPastDue:
			if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusPastDue); err != nil {
				return nil, fmt.Errorf("billing: update payment details past due: %w", err)
			}
		case domain.SubscriptionStatusCanceled, domain.SubscriptionStatusUnpaid:
			// Out-of-order defense: downgrade to Free tier on cancellation or non-payment
			if err := s.applyPlan(ctx, evt.OrganizationID, "free"); err != nil {
				return nil, fmt.Errorf("billing: downgrade to free: %w", err)
			}
			if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, "local", "", "", domain.SubscriptionStatusActive); err != nil {
				return nil, fmt.Errorf("billing: reset payment details: %w", err)
			}
		default:
			if evt.Status != "" {
				if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, evt.Status); err != nil {
					return nil, fmt.Errorf("billing: update payment details status: %w", err)
				}
			}
		}

	case "subscription.deleted":
		// Terminated upstream; revert organization to Free plan entitlements
		if err := s.applyPlan(ctx, evt.OrganizationID, "free"); err != nil {
			return nil, fmt.Errorf("billing: downgrade to free: %w", err)
		}
		if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, "local", "", "", domain.SubscriptionStatusActive); err != nil {
			return nil, fmt.Errorf("billing: reset payment details: %w", err)
		}

	case "payment.succeeded":
		if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusActive); err != nil {
			return nil, fmt.Errorf("billing: update payment details: %w", err)
		}
		if evt.PlanCode != "" {
			if err := s.applyPlan(ctx, evt.OrganizationID, evt.PlanCode); err != nil {
				return nil, fmt.Errorf("billing: apply plan: %w", err)
			}
		}
		paymentRef := evt.PaymentID
		if paymentRef == "" {
			paymentRef = evt.SubscriptionID
		}
		if err := s.invoices.RecordPayment(ctx, evt.OrganizationID, providerName, paymentRef, evt.AmountMicros); err != nil {
			return nil, fmt.Errorf("billing: record invoice payment: %w", err)
		}

	case "payment.failed":
		if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusPastDue); err != nil {
			return nil, fmt.Errorf("billing: update payment details past due: %w", err)
		}

	case "payment.refunded":
		if err := s.applyPlan(ctx, evt.OrganizationID, "free"); err != nil {
			return nil, fmt.Errorf("billing: downgrade to free: %w", err)
		}
		if err := s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, "local", "", "", domain.SubscriptionStatusActive); err != nil {
			return nil, fmt.Errorf("billing: reset payment details: %w", err)
		}
	}

	// 4. Record Audit Log for the verified event
	if s.audits != nil {
		_, err = s.audits.Append(ctx, evt.OrganizationID, domain.NewAuditLog{
			Action:       "billing.webhook_processed",
			ResourceType: "subscription",
			After:        fmt.Appendf(nil, `{"provider":"%s","event":"%s","status":"%s","plan":"%s"}`, providerName, evt.EventType, evt.Status, evt.PlanCode),
		})
		if err != nil {
			return nil, fmt.Errorf("audit log persistence failed: %w", err)
		}
	}

	return evt, nil
}

func (s *Service) findPlan(ctx context.Context, planCode string) (*domain.Plan, error) {
	plans, err := s.plans.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	for _, p := range plans {
		if p.Code == planCode {
			return &p, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (s *Service) applyPlan(ctx context.Context, orgID uuid.UUID, planCode string) error {
	plans, err := s.plans.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}
	for _, p := range plans {
		if p.Code == planCode {
			if err := s.subscriptions.SetPlan(ctx, orgID, p.ID); err != nil {
				return fmt.Errorf("failed to set plan: %w", err)
			}
			return nil
		}
	}
	return repositories.ErrNotFound
}
