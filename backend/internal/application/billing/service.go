// Package billing serves the organization's local billing state: the plan
// catalog, its current subscription, period usage cost, and locally computed
// invoices. No payment processor is integrated, so nothing this package returns
// represents a charge, a processor transaction, or a settled payment.
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	plans, err := s.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	var targetPlan *domain.Plan
	for _, p := range plans {
		if p.Code == planCode {
			targetPlan = &p
			break
		}
	}
	if targetPlan == nil {
		return nil, repositories.ErrNotFound
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
	plans, err := s.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	var targetPlan *domain.Plan
	for _, p := range plans {
		if p.Code == planCode {
			targetPlan = &p
			break
		}
	}
	if targetPlan == nil {
		return nil, repositories.ErrNotFound
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

	return gw.CreateCheckoutSession(ctx, payment.CheckoutParams{
		OrganizationID: actor.OrganizationID,
		CustomerEmail:  customerEmail,
		CustomerID:     customerID,
		PlanCode:       planCode,
		SuccessURL:     successURL,
		CancelURL:      cancelURL,
	})
}

// VerifyPaymentCallback verifies an upstream client payment callback and reconciles billing state.
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
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	gw := s.paymentGateways[providerName]
	if gw == nil {
		return nil, payment.ErrUnsupportedProvider
	}

	if err := gw.VerifyPaymentCallback(ctx, orderID, paymentID, signature); err != nil {
		return nil, err
	}

	// Verify plan code exists
	plans, err := s.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	var targetPlan *domain.Plan
	for _, p := range plans {
		if p.Code == planCode {
			targetPlan = &p
			break
		}
	}
	if targetPlan == nil {
		return nil, repositories.ErrNotFound
	}

	// 1. Upgrade subscription plan
	if err := s.subscriptions.SetPlan(ctx, actor.OrganizationID, targetPlan.ID); err != nil {
		return nil, err
	}

	// 2. Update payment details
	subID := orderID
	_ = s.subscriptions.UpdatePaymentDetails(ctx, actor.OrganizationID, providerName, subID, "", domain.SubscriptionStatusActive)

	// 3. Record settled invoice
	amountMicros := int64(49_000_000) // Default $49.00
	if planCode == "enterprise" {
		amountMicros = int64(499_000_000) // $499.00
	}
	_ = s.invoices.RecordPayment(ctx, actor.OrganizationID, providerName, paymentID, amountMicros)

	// 4. Record auditable billing event
	if s.billingEvents != nil {
		eventID := fmt.Sprintf("callback_%s_%s", providerName, paymentID)
		_, _ = s.billingEvents.Record(ctx, actor.OrganizationID, domain.NewBillingEvent{
			BillingEventID: eventID,
			EventType:      "payment.callback_verified",
			AmountMicros:   &amountMicros,
			Currency:       "usd",
			Payload:        fmt.Appendf(nil, `{"order_id":"%s","payment_id":"%s","plan_code":"%s"}`, orderID, paymentID, planCode),
		})
	}

	// 5. Record audit log
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
			Action:                "billing.payment_verified",
			ResourceType:          "subscription",
			After:                 fmt.Appendf(nil, `{"provider":"%s","plan":"%s","payment_id":"%s"}`, providerName, planCode, paymentID),
		})
	}

	return s.GetSubscription(ctx, actor)
}

// ProcessWebhook validates and handles an incoming payment processor webhook event.
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

	// 1. Idempotency check: deduplicate event processing
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

	// 2. State Machine Transitions & Entitlement Application
	switch evt.EventType {
	case "checkout.completed":
		if evt.PlanCode != "" {
			s.applyPlan(ctx, evt.OrganizationID, evt.PlanCode)
		}
		_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusActive)

	case "subscription.updated":
		switch evt.Status {
		case domain.SubscriptionStatusActive, domain.SubscriptionStatusTrialing:
			if evt.PlanCode != "" {
				s.applyPlan(ctx, evt.OrganizationID, evt.PlanCode)
			}
			_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, evt.Status)
		case domain.SubscriptionStatusPastDue:
			_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusPastDue)
		case domain.SubscriptionStatusCanceled, domain.SubscriptionStatusUnpaid:
			// Downgrade to Free tier on cancellation or non-payment
			s.applyPlan(ctx, evt.OrganizationID, "free")
			_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, "local", "", "", domain.SubscriptionStatusActive)
		default:
			if evt.Status != "" {
				_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, evt.Status)
			}
		}

	case "subscription.deleted":
		// Subscription terminated upstream; revert organization to Free plan entitlements
		s.applyPlan(ctx, evt.OrganizationID, "free")
		_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, "local", "", "", domain.SubscriptionStatusActive)

	case "payment.succeeded":
		_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusActive)
		if evt.PlanCode != "" {
			s.applyPlan(ctx, evt.OrganizationID, evt.PlanCode)
		}
		paymentRef := evt.PaymentID
		if paymentRef == "" {
			paymentRef = evt.SubscriptionID
		}
		_ = s.invoices.RecordPayment(ctx, evt.OrganizationID, providerName, paymentRef, evt.AmountMicros)

	case "payment.failed":
		_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, providerName, evt.SubscriptionID, evt.CustomerID, domain.SubscriptionStatusPastDue)

	case "payment.refunded":
		s.applyPlan(ctx, evt.OrganizationID, "free")
		_ = s.subscriptions.UpdatePaymentDetails(ctx, evt.OrganizationID, "local", "", "", domain.SubscriptionStatusActive)
	}

	// 3. Record Audit Log for the verified event
	if s.audits != nil {
		_, _ = s.audits.Append(ctx, evt.OrganizationID, domain.NewAuditLog{
			Action:       "billing.webhook_processed",
			ResourceType: "subscription",
			After:        fmt.Appendf(nil, `{"provider":"%s","event":"%s","status":"%s","plan":"%s"}`, providerName, evt.EventType, evt.Status, evt.PlanCode),
		})
	}

	return evt, nil
}

func (s *Service) applyPlan(ctx context.Context, orgID uuid.UUID, planCode string) {
	plans, err := s.plans.List(ctx)
	if err != nil {
		return
	}
	for _, p := range plans {
		if p.Code == planCode {
			_ = s.subscriptions.SetPlan(ctx, orgID, p.ID)
			break
		}
	}
}
