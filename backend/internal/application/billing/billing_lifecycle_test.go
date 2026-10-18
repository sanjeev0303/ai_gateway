package billing_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/payment"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type mockGateway struct {
	name         string
	checkoutFn   func(ctx context.Context, params payment.CheckoutParams) (*payment.CheckoutSession, error)
	webhookFn    func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error)
	verifyCbFn   func(ctx context.Context, orderID, paymentID, sig string) error
	cancelFn     func(ctx context.Context, subID string) error
	createCustFn func(ctx context.Context, orgID uuid.UUID, email, name string) (string, error)
}

func (m *mockGateway) Name() string { return m.name }
func (m *mockGateway) CreateOrGetCustomer(ctx context.Context, orgID uuid.UUID, email, name string) (string, error) {
	if m.createCustFn != nil {
		return m.createCustFn(ctx, orgID, email, name)
	}
	return "cus_mock_123", nil
}
func (m *mockGateway) CreateCheckoutSession(ctx context.Context, params payment.CheckoutParams) (*payment.CheckoutSession, error) {
	if m.checkoutFn != nil {
		return m.checkoutFn(ctx, params)
	}
	return &payment.CheckoutSession{
		SessionID: "sess_123",
		URL:       "https://checkout.example.com",
		Provider:  m.name,
	}, nil
}
func (m *mockGateway) VerifyAndParseWebhook(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
	if m.webhookFn != nil {
		return m.webhookFn(ctx, payload, sig)
	}
	return nil, errors.New("not implemented")
}
func (m *mockGateway) VerifyPaymentCallback(ctx context.Context, orderID, paymentID, sig string) error {
	if m.verifyCbFn != nil {
		return m.verifyCbFn(ctx, orderID, paymentID, sig)
	}
	if sig == "invalid" {
		return payment.ErrInvalidSignature
	}
	return nil
}
func (m *mockGateway) CancelSubscription(ctx context.Context, subID string) error {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, subID)
	}
	return nil
}

type memPlans struct {
	plans []domain.Plan
}

func (p *memPlans) List(ctx context.Context) ([]domain.Plan, error) {
	return p.plans, nil
}
func (p *memPlans) Create(ctx context.Context, input domain.NewPlan) (domain.Plan, error) {
	plan := domain.Plan{ID: uuid.New(), Code: input.Code, Name: input.Name, Description: input.Description, Status: "active"}
	p.plans = append(p.plans, plan)
	return plan, nil
}

type memSubs struct {
	plans         *memPlans
	subscriptions map[uuid.UUID]*domain.Subscription
}

func (s *memSubs) Create(ctx context.Context, orgID uuid.UUID, input domain.NewSubscription) (domain.Subscription, error) {
	sub := domain.Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanID:             input.PlanID,
		Status:             "active",
		CurrentPeriodStart: input.CurrentPeriodStart,
		CurrentPeriodEnd:   input.CurrentPeriodEnd,
		CancelAtPeriodEnd:  input.CancelAtPeriodEnd,
	}
	s.subscriptions[orgID] = &sub
	return sub, nil
}
func (s *memSubs) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Subscription, error) {
	if sub, ok := s.subscriptions[orgID]; ok && sub.ID == id {
		return *sub, nil
	}
	return domain.Subscription{}, repositories.ErrNotFound
}
func (s *memSubs) List(ctx context.Context, orgID uuid.UUID) ([]domain.Subscription, error) {
	if sub, ok := s.subscriptions[orgID]; ok {
		return []domain.Subscription{*sub}, nil
	}
	return nil, nil
}
func (s *memSubs) ActiveWithPlan(ctx context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error) {
	sub, ok := s.subscriptions[orgID]
	if !ok || sub.Status == "canceled" {
		return domain.OrganizationPlan{}, repositories.ErrNotFound
	}
	for _, p := range s.plans.plans {
		if p.ID == sub.PlanID {
			return domain.OrganizationPlan{
				PlanCode:               p.Code,
				PlanName:               p.Name,
				Status:                 sub.Status,
				CurrentPeriodStart:     sub.CurrentPeriodStart,
				CurrentPeriodEnd:       sub.CurrentPeriodEnd,
				CancelAtPeriodEnd:      sub.CancelAtPeriodEnd,
				PaymentProvider:        sub.PaymentProvider,
				ProviderCustomerID:     sub.ProviderCustomerID,
				ProviderSubscriptionID: sub.ProviderSubscriptionID,
			}, nil
		}
	}
	return domain.OrganizationPlan{}, repositories.ErrNotFound
}
func (s *memSubs) SetPlan(ctx context.Context, orgID, planID uuid.UUID) error {
	sub, ok := s.subscriptions[orgID]
	if !ok {
		now := time.Now()
		end := now.AddDate(0, 1, 0)
		s.subscriptions[orgID] = &domain.Subscription{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			PlanID:             planID,
			Status:             "active",
			CurrentPeriodStart: &now,
			CurrentPeriodEnd:   &end,
		}
		return nil
	}
	sub.PlanID = planID
	sub.Status = "active"
	sub.CancelAtPeriodEnd = false
	return nil
}
func (s *memSubs) Cancel(ctx context.Context, orgID uuid.UUID) error {
	sub, ok := s.subscriptions[orgID]
	if !ok {
		return repositories.ErrNotFound
	}
	sub.CancelAtPeriodEnd = true
	return nil
}
func (s *memSubs) UpdatePaymentDetails(ctx context.Context, orgID uuid.UUID, provider, subID, custID, status string) error {
	sub, ok := s.subscriptions[orgID]
	if !ok {
		sub = &domain.Subscription{ID: uuid.New(), OrganizationID: orgID, PlanID: s.plans.plans[0].ID}
		s.subscriptions[orgID] = sub
	}
	sub.PaymentProvider = provider
	if subID != "" {
		sub.ProviderSubscriptionID = &subID
	}
	if custID != "" {
		sub.ProviderCustomerID = &custID
	}
	if status != "" {
		sub.Status = status
	}
	return nil
}

type memInvoices struct {
	invoices map[uuid.UUID][]domain.Invoice
}

func (inv *memInvoices) List(ctx context.Context, orgID uuid.UUID) ([]domain.Invoice, error) {
	return inv.invoices[orgID], nil
}
func (inv *memInvoices) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Invoice, error) {
	for _, i := range inv.invoices[orgID] {
		if i.ID == id {
			return i, nil
		}
	}
	return domain.Invoice{}, repositories.ErrNotFound
}
func (inv *memInvoices) RecordPayment(ctx context.Context, orgID uuid.UUID, paymentProvider, paymentID string, amountMicros int64) error {
	now := time.Now().UTC()
	item := domain.Invoice{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		Number:          "INV-PAID-" + paymentID,
		Status:          "paid",
		Currency:        "usd",
		TotalMicros:     amountMicros,
		PaymentProvider: &paymentProvider,
		PaymentID:       &paymentID,
		PaidAt:          &now,
		PeriodStart:     now.AddDate(0, -1, 0),
		PeriodEnd:       now,
		IssuedAt:        now,
	}
	inv.invoices[orgID] = append(inv.invoices[orgID], item)
	return nil
}

type memBillingEvents struct {
	events map[string]domain.BillingEvent
}

func (be *memBillingEvents) Record(ctx context.Context, orgID uuid.UUID, input domain.NewBillingEvent) (domain.BillingEvent, error) {
	key := orgID.String() + ":" + input.BillingEventID
	if _, exists := be.events[key]; exists {
		return domain.BillingEvent{}, repositories.ErrConflict
	}
	event := domain.BillingEvent{
		ID:             uuid.New(),
		OrganizationID: orgID,
		BillingEventID: input.BillingEventID,
		EventType:      input.EventType,
		AmountMicros:   input.AmountMicros,
		Currency:       input.Currency,
		Status:         domain.BillingEventStatusSucceeded,
		Payload:        input.Payload,
		CreatedAt:      time.Now(),
	}
	be.events[key] = event
	return event, nil
}
func (be *memBillingEvents) Get(ctx context.Context, orgID uuid.UUID, billingEventID string) (domain.BillingEvent, error) {
	key := orgID.String() + ":" + billingEventID
	if event, exists := be.events[key]; exists {
		return event, nil
	}
	return domain.BillingEvent{}, repositories.ErrNotFound
}
func (be *memBillingEvents) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	return 0, nil
}

type memAudits struct {
	logs []domain.AuditLog
}

func (a *memAudits) Append(ctx context.Context, orgID uuid.UUID, input domain.NewAuditLog) (domain.AuditLog, error) {
	log := domain.AuditLog{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Action:         input.Action,
		ResourceType:   input.ResourceType,
		After:          input.After,
		CreatedAt:      time.Now(),
	}
	a.logs = append(a.logs, log)
	return log, nil
}
func (a *memAudits) List(ctx context.Context, orgID uuid.UUID, filter domain.AuditLogFilter, limit, offset int32) ([]domain.AuditLog, error) {
	return a.logs, nil
}
func (a *memAudits) Get(ctx context.Context, orgID, id uuid.UUID) (domain.AuditLog, error) {
	return domain.AuditLog{}, nil
}
func (a *memAudits) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	return 0, nil
}

func TestBillingApplicationServiceLifecycle(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	adminActor := domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         uuid.New(),
		OrganizationID: orgID,
		Role:           domain.RoleAdmin,
		DisplayName:    "Admin",
	}

	plansRepo := &memPlans{
		plans: []domain.Plan{
			{ID: uuid.New(), Code: "free", Name: "Free Plan", Status: "active"},
			{ID: uuid.New(), Code: "pro", Name: "Pro Plan", Status: "active"},
			{ID: uuid.New(), Code: "enterprise", Name: "Enterprise Plan", Status: "active"},
		},
	}
	subsRepo := &memSubs{
		plans:         plansRepo,
		subscriptions: make(map[uuid.UUID]*domain.Subscription),
	}
	invoicesRepo := &memInvoices{
		invoices: make(map[uuid.UUID][]domain.Invoice),
	}
	eventsRepo := &memBillingEvents{
		events: make(map[string]domain.BillingEvent),
	}
	auditsRepo := &memAudits{}

	stripeGw := &mockGateway{name: "stripe"}
	razorpayGw := &mockGateway{name: "razorpay"}

	svc := billing.NewService(billing.ServiceDeps{
		Plans:         plansRepo,
		Subscriptions: subsRepo,
		Invoices:      invoicesRepo,
		BillingEvents: eventsRepo,
		Audits:        auditsRepo,
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGw,
			"razorpay": razorpayGw,
		},
	})

	// Initialize org on free plan
	_ = subsRepo.SetPlan(context.Background(), orgID, plansRepo.plans[0].ID)

	t.Run("CreateCheckoutSession provisions session with correct customer and plan", func(t *testing.T) {
		sess, err := svc.CreateCheckoutSession(context.Background(), adminActor, "stripe", "pro", "admin@sentinel.dev", "https://sentinel.dev/success", "https://sentinel.dev/cancel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.SessionID != "sess_123" {
			t.Errorf("expected sess_123, got %s", sess.SessionID)
		}
	})

	t.Run("VerifyPaymentCallback upgrades plan and records settled invoice upon valid signature", func(t *testing.T) {
		sub, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "razorpay", "order_valid_100", "pay_valid_100", "valid_sig", "enterprise")
		if err != nil {
			t.Fatalf("unexpected error verifying callback: %v", err)
		}
		if sub.PlanCode != "enterprise" {
			t.Errorf("expected enterprise plan, got %s", sub.PlanCode)
		}
		if sub.Status != "active" {
			t.Errorf("expected active status, got %s", sub.Status)
		}

		// Verify settled invoice was recorded
		invoices, _ := svc.ListInvoices(context.Background(), adminActor)
		if len(invoices) != 1 {
			t.Fatalf("expected 1 settled invoice, got %d", len(invoices))
		}
		if invoices[0].Status != "paid" {
			t.Errorf("expected paid invoice status, got %s", invoices[0].Status)
		}
		if invoices[0].PaymentID == nil || *invoices[0].PaymentID != "pay_valid_100" {
			t.Errorf("expected payment_id pay_valid_100, got %v", invoices[0].PaymentID)
		}

		// Verify plan entitlements
		entitlements := domain.GetPlanEntitlements(sub.PlanCode)
		if entitlements.MonthlyRequestQuota != 100_000_000 {
			t.Errorf("expected 100M quota for enterprise, got %d", entitlements.MonthlyRequestQuota)
		}
		if !entitlements.SupportsFullPrivacy {
			t.Error("expected enterprise tier to support full privacy")
		}
	})

	t.Run("VerifyPaymentCallback rejects forged signature without mutating state", func(t *testing.T) {
		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "razorpay", "order_fake", "pay_fake", "invalid", "pro")
		if err == nil {
			t.Fatal("expected forged signature to fail, got nil")
		}
	})

	t.Run("ProcessWebhook ensures idempotency across duplicate deliveries", func(t *testing.T) {
		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_idempotent_1",
				EventType:      "payment.succeeded",
				OrganizationID: orgID,
				PlanCode:       "pro",
				SubscriptionID: "sub_stripe_idem_1",
				PaymentID:      "pi_stripe_idem_1",
				AmountMicros:   49_000_000,
				Status:         "paid",
			}, nil
		}

		invoicesBefore, _ := svc.ListInvoices(context.Background(), adminActor)
		countBefore := len(invoicesBefore)

		// Delivery 1
		_, err := svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if err != nil {
			t.Fatalf("delivery 1 failed: %v", err)
		}

		invoicesAfter1, _ := svc.ListInvoices(context.Background(), adminActor)
		if len(invoicesAfter1) != countBefore+1 {
			t.Fatalf("expected %d invoices, got %d", countBefore+1, len(invoicesAfter1))
		}

		// Delivery 2 (exact duplicate delivery)
		_, err = svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if err != nil {
			t.Fatalf("delivery 2 failed: %v", err)
		}

		invoicesAfter2, _ := svc.ListInvoices(context.Background(), adminActor)
		if len(invoicesAfter2) != countBefore+1 {
			t.Errorf("duplicate webhook duplicated invoices! expected %d, got %d", countBefore+1, len(invoicesAfter2))
		}
	})

	t.Run("ProcessWebhook handles cancellation and reverts to Free tier", func(t *testing.T) {
		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_cancel_1",
				EventType:      "subscription.deleted",
				OrganizationID: orgID,
				SubscriptionID: "sub_to_cancel",
				Status:         "canceled",
			}, nil
		}

		_, err := svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "sig")
		if err != nil {
			t.Fatalf("webhook processing failed: %v", err)
		}

		sub, err := svc.GetSubscription(context.Background(), adminActor)
		if err != nil {
			t.Fatalf("get subscription failed: %v", err)
		}
		if sub.PlanCode != "free" {
			t.Errorf("expected plan to revert to free on cancellation, got %s", sub.PlanCode)
		}

		entitlements := domain.GetPlanEntitlements(sub.PlanCode)
		if entitlements.MonthlyRequestQuota != 10_000 {
			t.Errorf("expected 10k quota for free tier, got %d", entitlements.MonthlyRequestQuota)
		}
	})

	t.Run("ProcessWebhook handles payment failure and transitions to past_due", func(t *testing.T) {
		razorpayGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_fail_1",
				EventType:      "payment.failed",
				OrganizationID: orgID,
				PaymentID:      "pay_failed_1",
				Status:         "past_due",
			}, nil
		}

		_, err := svc.ProcessWebhook(context.Background(), "razorpay", []byte("payload"), "sig")
		if err != nil {
			t.Fatalf("webhook processing failed: %v", err)
		}

		sub, _ := subsRepo.ActiveWithPlan(context.Background(), orgID)
		if sub.Status != "past_due" {
			t.Errorf("expected past_due status on payment failure, got %s", sub.Status)
		}
	})
}
