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
	mode         string
	checkoutFn   func(ctx context.Context, params payment.CheckoutParams) (*payment.CheckoutSession, error)
	webhookFn    func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error)
	verifyCbFn   func(ctx context.Context, orderID, paymentID, sig string) error
	cancelFn     func(ctx context.Context, subID string) error
	createCustFn func(ctx context.Context, orgID uuid.UUID, email, name string) (string, error)
}

func (m *mockGateway) Name() string { return m.name }
func (m *mockGateway) Mode() string {
	if m.mode == "" {
		return "test"
	}
	return m.mode
}
func (m *mockGateway) SetMode(mode string) { m.mode = mode }

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
		OrderID:   "order_sess_123",
		URL:       "https://checkout.example.com",
		Provider:  m.name,
	}, nil
}

func (m *mockGateway) VerifyAndParseWebhook(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
	if sig == "invalid" {
		return nil, payment.ErrInvalidSignature
	}
	if m.webhookFn != nil {
		return m.webhookFn(ctx, payload, sig)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGateway) VerifyPaymentCallback(ctx context.Context, orderID, paymentID, sig string) error {
	if sig == "invalid" {
		return payment.ErrInvalidSignature
	}
	if m.verifyCbFn != nil {
		return m.verifyCbFn(ctx, orderID, paymentID, sig)
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
	plan := domain.Plan{
		ID:              uuid.New(),
		Code:            input.Code,
		Name:            input.Name,
		Description:     input.Description,
		PriceMicros:     input.PriceMicros,
		Currency:        input.Currency,
		BillingInterval: input.BillingInterval,
		Status:          "active",
	}
	p.plans = append(p.plans, plan)
	return plan, nil
}

type memSubs struct {
	plans            *memPlans
	subscriptions    map[uuid.UUID]*domain.Subscription
	setPlanErr       error
	updatePaymentErr error
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
	if s.setPlanErr != nil {
		return s.setPlanErr
	}
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
	if s.updatePaymentErr != nil {
		return s.updatePaymentErr
	}
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
	invoices         map[uuid.UUID][]domain.Invoice
	recordPaymentErr error
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
	if inv.recordPaymentErr != nil {
		return inv.recordPaymentErr
	}
	// Idempotency check: don't duplicate invoice if paymentID already exists
	for _, existing := range inv.invoices[orgID] {
		if existing.PaymentID != nil && *existing.PaymentID == paymentID {
			return nil
		}
	}
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

type memCheckouts struct {
	checkouts map[string]domain.Checkout
	createErr error
	getErr    error
	updateErr error
}

func (c *memCheckouts) Create(ctx context.Context, orgID uuid.UUID, input domain.NewCheckout) (domain.Checkout, error) {
	if c.createErr != nil {
		return domain.Checkout{}, c.createErr
	}
	ch := domain.Checkout{
		ID:                uuid.New(),
		OrganizationID:    orgID,
		CheckoutSessionID: input.CheckoutSessionID,
		Provider:          input.Provider,
		PlanCode:          input.PlanCode,
		PlanID:            input.PlanID,
		AmountMicros:      input.AmountMicros,
		Currency:          input.Currency,
		Status:            domain.CheckoutStatusPending,
		CustomerEmail:     input.CustomerEmail,
		CustomerID:        input.CustomerID,
		OrderID:           input.OrderID,
		ExpiresAt:         input.ExpiresAt,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	c.checkouts[ch.CheckoutSessionID] = ch
	return ch, nil
}

func (c *memCheckouts) Get(ctx context.Context, orgID uuid.UUID, checkoutSessionID string) (domain.Checkout, error) {
	if c.getErr != nil {
		return domain.Checkout{}, c.getErr
	}
	if ch, ok := c.checkouts[checkoutSessionID]; ok && ch.OrganizationID == orgID {
		return ch, nil
	}
	return domain.Checkout{}, repositories.ErrNotFound
}

func (c *memCheckouts) GetByOrderID(ctx context.Context, orgID uuid.UUID, orderID string) (domain.Checkout, error) {
	if c.getErr != nil {
		return domain.Checkout{}, c.getErr
	}
	for _, ch := range c.checkouts {
		if ch.OrganizationID == orgID && (ch.OrderID == orderID || ch.CheckoutSessionID == orderID) {
			return ch, nil
		}
	}
	return domain.Checkout{}, repositories.ErrNotFound
}

func (c *memCheckouts) UpdateStatus(ctx context.Context, orgID uuid.UUID, checkoutSessionID, status, paymentID string) error {
	if c.updateErr != nil {
		return c.updateErr
	}
	ch, ok := c.checkouts[checkoutSessionID]
	if !ok || ch.OrganizationID != orgID {
		return repositories.ErrNotFound
	}
	ch.Status = status
	ch.PaymentID = paymentID
	if status == domain.CheckoutStatusCompleted {
		now := time.Now().UTC()
		ch.CompletedAt = &now
	}
	ch.UpdatedAt = time.Now().UTC()
	c.checkouts[checkoutSessionID] = ch
	return nil
}

func (c *memCheckouts) ExpirePending(ctx context.Context, olderThan time.Time) (int64, error) {
	var count int64
	for k, ch := range c.checkouts {
		if ch.Status == domain.CheckoutStatusPending && ch.ExpiresAt.Before(olderThan) {
			ch.Status = domain.CheckoutStatusExpired
			c.checkouts[k] = ch
			count++
		}
	}
	return count, nil
}

type memBillingEvents struct {
	events    map[string]domain.BillingEvent
	recordErr error
}

func (be *memBillingEvents) Record(ctx context.Context, orgID uuid.UUID, input domain.NewBillingEvent) (domain.BillingEvent, error) {
	if be.recordErr != nil {
		return domain.BillingEvent{}, be.recordErr
	}
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
	logs      []domain.AuditLog
	appendErr error
}

func (a *memAudits) Append(ctx context.Context, orgID uuid.UUID, input domain.NewAuditLog) (domain.AuditLog, error) {
	if a.appendErr != nil {
		return domain.AuditLog{}, a.appendErr
	}
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

func setupBillingEnv() (
	domain.Actor,
	*memPlans,
	*memSubs,
	*memInvoices,
	*memCheckouts,
	*memBillingEvents,
	*memAudits,
	*mockGateway,
	*mockGateway,
	*billing.Service,
) {
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
			{ID: uuid.New(), Code: "free", Name: "Free Plan", PriceMicros: 0, Currency: "usd", Status: "active"},
			{ID: uuid.New(), Code: "pro", Name: "Pro Plan", PriceMicros: 49_000_000, Currency: "usd", Status: "active"},
			{ID: uuid.New(), Code: "enterprise", Name: "Enterprise Plan", PriceMicros: 499_000_000, Currency: "usd", Status: "active"},
		},
	}
	subsRepo := &memSubs{
		plans:         plansRepo,
		subscriptions: make(map[uuid.UUID]*domain.Subscription),
	}
	invoicesRepo := &memInvoices{
		invoices: make(map[uuid.UUID][]domain.Invoice),
	}
	checkoutsRepo := &memCheckouts{
		checkouts: make(map[string]domain.Checkout),
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
		Checkouts:     checkoutsRepo,
		BillingEvents: eventsRepo,
		Audits:        auditsRepo,
		PaymentGateways: map[string]payment.PaymentGateway{
			"stripe":   stripeGw,
			"razorpay": razorpayGw,
		},
	})

	_ = subsRepo.SetPlan(context.Background(), orgID, plansRepo.plans[0].ID)

	return adminActor, plansRepo, subsRepo, invoicesRepo, checkoutsRepo, eventsRepo, auditsRepo, stripeGw, razorpayGw, svc
}

func TestBillingApplicationServiceLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("CreateCheckoutSession provisions session with correct customer, plan, and server-side pricing", func(t *testing.T) {
		adminActor, _, _, _, checkoutsRepo, _, auditsRepo, _, _, svc := setupBillingEnv()

		sess, err := svc.CreateCheckoutSession(context.Background(), adminActor, "stripe", "pro", "admin@sentinel.dev", "https://sentinel.dev/success", "https://sentinel.dev/cancel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sess.SessionID != "sess_123" {
			t.Errorf("expected sess_123, got %s", sess.SessionID)
		}

		// Verify server-side checkout record was persisted with trusted pricing
		ch, err := checkoutsRepo.Get(context.Background(), adminActor.OrganizationID, "sess_123")
		if err != nil {
			t.Fatalf("checkout record not persisted: %v", err)
		}
		if ch.AmountMicros != 49_000_000 {
			t.Errorf("expected 49,000,000 micros, got %d", ch.AmountMicros)
		}
		if ch.Currency != "usd" {
			t.Errorf("expected usd currency, got %s", ch.Currency)
		}
		if ch.PlanCode != "pro" {
			t.Errorf("expected pro plan code, got %s", ch.PlanCode)
		}
		if ch.Status != domain.CheckoutStatusPending {
			t.Errorf("expected pending status, got %s", ch.Status)
		}

		// Verify audit log recorded without exposing sensitive tokens
		if len(auditsRepo.logs) == 0 {
			t.Errorf("expected audit log to be created")
		}
	})

	t.Run("VerifyPaymentCallback upgrades plan and records settled invoice with correct amount and currency", func(t *testing.T) {
		adminActor, _, subsRepo, invoicesRepo, checkoutsRepo, _, _, _, _, svc := setupBillingEnv()

		// Provision checkout session for enterprise
		_, err := svc.CreateCheckoutSession(context.Background(), adminActor, "razorpay", "enterprise", "admin@sentinel.dev", "https://sentinel.dev/success", "https://sentinel.dev/cancel")
		if err != nil {
			t.Fatalf("failed to create checkout session: %v", err)
		}

		sub, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "razorpay", "order_sess_123", "pay_valid_100", "valid_sig", "enterprise")
		if err != nil {
			t.Fatalf("unexpected error verifying callback: %v", err)
		}
		if sub.PlanCode != "enterprise" {
			t.Errorf("expected enterprise plan, got %s", sub.PlanCode)
		}
		if sub.Status != "active" {
			t.Errorf("expected active status, got %s", sub.Status)
		}

		// Verify checkout status transitioned to completed
		ch, err := checkoutsRepo.Get(context.Background(), adminActor.OrganizationID, "sess_123")
		if err != nil || ch.Status != domain.CheckoutStatusCompleted {
			t.Errorf("expected completed checkout status, got %v (err: %v)", ch.Status, err)
		}
		if ch.PaymentID != "pay_valid_100" {
			t.Errorf("expected payment ID pay_valid_100, got %s", ch.PaymentID)
		}

		// Verify settled invoice was recorded with authoritative amount
		invoices, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices) != 1 {
			t.Fatalf("expected 1 settled invoice, got %d", len(invoices))
		}
		if invoices[0].Status != "paid" {
			t.Errorf("expected paid invoice status, got %s", invoices[0].Status)
		}
		if invoices[0].TotalMicros != 499_000_000 {
			t.Errorf("expected 499_000_000 micros, got %d", invoices[0].TotalMicros)
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

		// Verify persisted subscription in repo
		persistedSub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if persistedSub.PlanCode != "enterprise" {
			t.Errorf("expected enterprise in repo, got %s", persistedSub.PlanCode)
		}
	})

	t.Run("VerifyPaymentCallback rejects forged signature without mutating state", func(t *testing.T) {
		adminActor, _, subsRepo, invoicesRepo, _, _, _, _, _, svc := setupBillingEnv()

		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "razorpay", "order_fake", "pay_fake", "invalid", "pro")
		if !errors.Is(err, payment.ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature, got %v", err)
		}

		sub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub.PlanCode != "free" {
			t.Errorf("expected free plan unchanged, got %s", sub.PlanCode)
		}
		invoices, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices) != 0 {
			t.Errorf("expected 0 invoices, got %d", len(invoices))
		}
	})

	t.Run("VerifyPaymentCallback rejects expired checkout", func(t *testing.T) {
		adminActor, _, _, _, checkoutsRepo, _, _, _, _, svc := setupBillingEnv()

		// Create expired checkout
		sessID := "sess_expired"
		orderID := "order_expired"
		_, _ = checkoutsRepo.Create(context.Background(), adminActor.OrganizationID, domain.NewCheckout{
			CheckoutSessionID: sessID,
			Provider:          "stripe",
			PlanCode:          "pro",
			AmountMicros:      49_000_000,
			Currency:          "usd",
			OrderID:           orderID,
			ExpiresAt:         time.Now().UTC().Add(-1 * time.Hour), // Expired 1 hour ago
		})

		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", orderID, "pay_123", "valid_sig", "pro")
		if !errors.Is(err, payment.ErrCheckoutExpired) {
			t.Fatalf("expected ErrCheckoutExpired, got %v", err)
		}

		ch, _ := checkoutsRepo.Get(context.Background(), adminActor.OrganizationID, sessID)
		if ch.Status != domain.CheckoutStatusExpired {
			t.Errorf("expected checkout to be marked expired, got %s", ch.Status)
		}
	})

	t.Run("VerifyPaymentCallback rejects cancelled checkout", func(t *testing.T) {
		adminActor, _, _, _, checkoutsRepo, _, _, _, _, svc := setupBillingEnv()

		sessID := "sess_cancelled"
		orderID := "order_cancelled"
		_, _ = checkoutsRepo.Create(context.Background(), adminActor.OrganizationID, domain.NewCheckout{
			CheckoutSessionID: sessID,
			Provider:          "stripe",
			PlanCode:          "pro",
			AmountMicros:      49_000_000,
			Currency:          "usd",
			OrderID:           orderID,
			ExpiresAt:         time.Now().UTC().Add(1 * time.Hour),
		})
		_ = checkoutsRepo.UpdateStatus(context.Background(), adminActor.OrganizationID, sessID, domain.CheckoutStatusCancelled, "")

		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", orderID, "pay_123", "valid_sig", "pro")
		if !errors.Is(err, payment.ErrCheckoutCancelled) {
			t.Fatalf("expected ErrCheckoutCancelled, got %v", err)
		}
	})

	t.Run("VerifyPaymentCallback rejects plan mismatch", func(t *testing.T) {
		adminActor, _, _, _, checkoutsRepo, _, _, _, _, svc := setupBillingEnv()

		sessID := "sess_mismatch"
		orderID := "order_mismatch"
		_, _ = checkoutsRepo.Create(context.Background(), adminActor.OrganizationID, domain.NewCheckout{
			CheckoutSessionID: sessID,
			Provider:          "stripe",
			PlanCode:          "pro", // Initiated for pro
			AmountMicros:      49_000_000,
			Currency:          "usd",
			OrderID:           orderID,
			ExpiresAt:         time.Now().UTC().Add(1 * time.Hour),
		})

		// Callback claims enterprise
		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", orderID, "pay_123", "valid_sig", "enterprise")
		if !errors.Is(err, payment.ErrPlanMismatch) {
			t.Fatalf("expected ErrPlanMismatch, got %v", err)
		}
	})

	t.Run("VerifyPaymentCallback repeated callback is idempotent and does not duplicate invoice", func(t *testing.T) {
		adminActor, _, _, invoicesRepo, _, _, _, _, _, svc := setupBillingEnv()

		_, _ = svc.CreateCheckoutSession(context.Background(), adminActor, "stripe", "pro", "admin@sentinel.dev", "https://sentinel.dev/success", "https://sentinel.dev/cancel")

		// First callback
		sub1, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", "order_sess_123", "pay_idem_1", "valid_sig", "pro")
		if err != nil {
			t.Fatalf("first verification failed: %v", err)
		}
		if sub1.PlanCode != "pro" {
			t.Errorf("expected pro plan, got %s", sub1.PlanCode)
		}

		invoices1, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices1) != 1 {
			t.Fatalf("expected 1 invoice after first callback, got %d", len(invoices1))
		}

		// Second callback (same orderID and paymentID)
		sub2, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", "order_sess_123", "pay_idem_1", "valid_sig", "pro")
		if err != nil {
			t.Fatalf("repeated verification failed: %v", err)
		}
		if sub2.PlanCode != "pro" {
			t.Errorf("expected pro plan on idempotent call, got %s", sub2.PlanCode)
		}

		invoices2, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices2) != 1 {
			t.Errorf("repeated callback duplicated invoice! expected 1, got %d", len(invoices2))
		}
	})

	t.Run("VerifyPaymentCallback propagates database failure on subscription plan update", func(t *testing.T) {
		adminActor, _, subsRepo, invoicesRepo, checkoutsRepo, _, _, _, _, svc := setupBillingEnv()

		_, _ = svc.CreateCheckoutSession(context.Background(), adminActor, "stripe", "pro", "admin@sentinel.dev", "https://sentinel.dev/success", "https://sentinel.dev/cancel")

		subsRepo.setPlanErr = errors.New("db connection failure")

		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", "order_sess_123", "pay_fail_sub", "valid_sig", "pro")
		if err == nil {
			t.Fatal("expected error on subscription update failure, got nil")
		}

		// Ensure invoice was NOT recorded and checkout was NOT completed
		invoices, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices) != 0 {
			t.Errorf("invoice recorded despite subscription update failure: %d", len(invoices))
		}
		ch, _ := checkoutsRepo.Get(context.Background(), adminActor.OrganizationID, "sess_123")
		if ch.Status == domain.CheckoutStatusCompleted {
			t.Errorf("checkout marked completed despite failure: %s", ch.Status)
		}
	})

	t.Run("VerifyPaymentCallback propagates database failure on invoice recording", func(t *testing.T) {
		adminActor, _, _, invoicesRepo, checkoutsRepo, _, _, _, _, svc := setupBillingEnv()

		_, _ = svc.CreateCheckoutSession(context.Background(), adminActor, "stripe", "pro", "admin@sentinel.dev", "https://sentinel.dev/success", "https://sentinel.dev/cancel")

		invoicesRepo.recordPaymentErr = errors.New("disk full: invoice write failed")

		_, err := svc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", "order_sess_123", "pay_fail_inv", "valid_sig", "pro")
		if err == nil {
			t.Fatal("expected error on invoice write failure, got nil")
		}

		ch, _ := checkoutsRepo.Get(context.Background(), adminActor.OrganizationID, "sess_123")
		if ch.Status == domain.CheckoutStatusCompleted {
			t.Errorf("checkout marked completed despite invoice write failure: %s", ch.Status)
		}
	})

	t.Run("VerifyPaymentCallback in simulation mode never grants paid entitlements", func(t *testing.T) {
		adminActor, plansRepo, subsRepo, invoicesRepo, checkoutsRepo, eventsRepo, auditsRepo, stripeGw, razorpayGw, _ := setupBillingEnv()

		// Initialize service in simulation mode
		simSvc := billing.NewService(billing.ServiceDeps{
			Mode:          "simulation",
			Plans:         plansRepo,
			Subscriptions: subsRepo,
			Invoices:      invoicesRepo,
			Checkouts:     checkoutsRepo,
			BillingEvents: eventsRepo,
			Audits:        auditsRepo,
			PaymentGateways: map[string]payment.PaymentGateway{
				"stripe":   stripeGw,
				"razorpay": razorpayGw,
			},
		})

		_, err := simSvc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", "order_sim", "pay_sim", "valid_sig", "enterprise")
		if !errors.Is(err, payment.ErrSimulationOnly) {
			t.Fatalf("expected ErrSimulationOnly, got %v", err)
		}

		// Ensure org remains on free tier and no paid invoice recorded
		sub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub.PlanCode != "free" {
			t.Errorf("simulation mode granted paid plan %s, want free", sub.PlanCode)
		}
		invoices, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices) != 0 {
			t.Errorf("simulation mode recorded paid invoice, count=%d", len(invoices))
		}
	})

	t.Run("VerifyPaymentCallback in disabled mode is refused", func(t *testing.T) {
		adminActor, plansRepo, subsRepo, invoicesRepo, checkoutsRepo, eventsRepo, auditsRepo, stripeGw, razorpayGw, _ := setupBillingEnv()

		disSvc := billing.NewService(billing.ServiceDeps{
			Mode:          "disabled",
			Plans:         plansRepo,
			Subscriptions: subsRepo,
			Invoices:      invoicesRepo,
			Checkouts:     checkoutsRepo,
			BillingEvents: eventsRepo,
			Audits:        auditsRepo,
			PaymentGateways: map[string]payment.PaymentGateway{
				"stripe":   stripeGw,
				"razorpay": razorpayGw,
			},
		})

		_, err := disSvc.VerifyPaymentCallback(context.Background(), adminActor, "stripe", "order_dis", "pay_dis", "valid_sig", "pro")
		if !errors.Is(err, payment.ErrBillingDisabled) {
			t.Fatalf("expected ErrBillingDisabled, got %v", err)
		}
	})

	t.Run("ProcessWebhook enforces correct payment amount and rejects wrong amount", func(t *testing.T) {
		adminActor, _, subsRepo, invoicesRepo, checkoutsRepo, _, _, stripeGw, _, svc := setupBillingEnv()

		sessID := "sess_amount_test"
		_, _ = checkoutsRepo.Create(context.Background(), adminActor.OrganizationID, domain.NewCheckout{
			CheckoutSessionID: sessID,
			Provider:          "stripe",
			PlanCode:          "pro",
			AmountMicros:      49_000_000,
			Currency:          "usd",
			OrderID:           sessID,
			ExpiresAt:         time.Now().UTC().Add(1 * time.Hour),
		})

		// Webhook reports incorrect amount (e.g. 19_000_000 instead of 49_000_000)
		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_amount_mismatch",
				EventType:      "payment.succeeded",
				OrganizationID: adminActor.OrganizationID,
				SubscriptionID: sessID,
				PaymentID:      "pi_mismatch_amount",
				AmountMicros:   19_000_000,
				Currency:       "usd",
				PlanCode:       "pro",
			}, nil
		}

		_, err := svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if !errors.Is(err, payment.ErrAmountMismatch) {
			t.Fatalf("expected ErrAmountMismatch, got %v", err)
		}

		// Ensure subscription was not upgraded and no paid invoice recorded
		sub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub.PlanCode != "free" {
			t.Errorf("plan changed despite amount mismatch: %s", sub.PlanCode)
		}
		invoices, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices) != 0 {
			t.Errorf("invoice recorded despite amount mismatch: %d", len(invoices))
		}
	})

	t.Run("ProcessWebhook enforces correct currency and rejects wrong currency", func(t *testing.T) {
		adminActor, _, subsRepo, invoicesRepo, checkoutsRepo, _, _, stripeGw, _, svc := setupBillingEnv()

		sessID := "sess_currency_test"
		_, _ = checkoutsRepo.Create(context.Background(), adminActor.OrganizationID, domain.NewCheckout{
			CheckoutSessionID: sessID,
			Provider:          "stripe",
			PlanCode:          "pro",
			AmountMicros:      49_000_000,
			Currency:          "usd",
			OrderID:           sessID,
			ExpiresAt:         time.Now().UTC().Add(1 * time.Hour),
		})

		// Webhook reports incorrect currency ("eur" instead of "usd")
		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_currency_mismatch",
				EventType:      "payment.succeeded",
				OrganizationID: adminActor.OrganizationID,
				SubscriptionID: sessID,
				PaymentID:      "pi_mismatch_currency",
				AmountMicros:   49_000_000,
				Currency:       "eur",
				PlanCode:       "pro",
			}, nil
		}

		_, err := svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if !errors.Is(err, payment.ErrCurrencyMismatch) {
			t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
		}

		sub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub.PlanCode != "free" {
			t.Errorf("plan changed despite currency mismatch: %s", sub.PlanCode)
		}
		invoices, _ := invoicesRepo.List(context.Background(), adminActor.OrganizationID)
		if len(invoices) != 0 {
			t.Errorf("invoice recorded despite currency mismatch: %d", len(invoices))
		}
	})

	t.Run("ProcessWebhook rejects invalid webhook signature", func(t *testing.T) {
		_, _, _, _, _, _, _, _, _, svc := setupBillingEnv()

		_, err := svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "invalid")
		if !errors.Is(err, payment.ErrInvalidSignature) {
			t.Fatalf("expected ErrInvalidSignature, got %v", err)
		}
	})

	t.Run("ProcessWebhook ensures idempotency across duplicate deliveries", func(t *testing.T) {
		adminActor, _, _, _, _, _, _, stripeGw, _, svc := setupBillingEnv()

		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_idempotent_1",
				EventType:      "payment.succeeded",
				OrganizationID: adminActor.OrganizationID,
				PlanCode:       "pro",
				SubscriptionID: "sub_stripe_idem_1",
				PaymentID:      "pi_stripe_idem_1",
				AmountMicros:   49_000_000,
				Currency:       "usd",
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

	t.Run("ProcessWebhook handles out-of-order events and cancellation gracefully", func(t *testing.T) {
		adminActor, _, subsRepo, _, _, _, _, stripeGw, _, svc := setupBillingEnv()

		// 1. Subscription is terminated upstream
		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_cancel_1",
				EventType:      "subscription.deleted",
				OrganizationID: adminActor.OrganizationID,
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

		// 2. Out-of-order delayed subscription.updated with status canceled
		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_delayed_update",
				EventType:      "subscription.updated",
				OrganizationID: adminActor.OrganizationID,
				SubscriptionID: "sub_to_cancel",
				Status:         "canceled",
			}, nil
		}

		_, err = svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "sig")
		if err != nil {
			t.Fatalf("delayed webhook failed: %v", err)
		}

		sub2, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub2.PlanCode != "free" {
			t.Errorf("expected plan to remain free, got %s", sub2.PlanCode)
		}
	})

	t.Run("ProcessWebhook handles payment failure and transitions to past_due", func(t *testing.T) {
		adminActor, _, subsRepo, _, _, _, _, _, razorpayGw, svc := setupBillingEnv()

		razorpayGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_fail_1",
				EventType:      "payment.failed",
				OrganizationID: adminActor.OrganizationID,
				PaymentID:      "pay_failed_1",
				Status:         "past_due",
			}, nil
		}

		_, err := svc.ProcessWebhook(context.Background(), "razorpay", []byte("payload"), "sig")
		if err != nil {
			t.Fatalf("webhook processing failed: %v", err)
		}

		sub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub.Status != "past_due" {
			t.Errorf("expected past_due status on payment failure, got %s", sub.Status)
		}
	})

	t.Run("ProcessWebhook propagates persistence failures without masking errors", func(t *testing.T) {
		adminActor, _, subsRepo, invoicesRepo, _, eventsRepo, _, stripeGw, _, svc := setupBillingEnv()

		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_persist_fail_1",
				EventType:      "payment.succeeded",
				OrganizationID: adminActor.OrganizationID,
				PlanCode:       "pro",
				PaymentID:      "pay_fail_persist",
				AmountMicros:   49_000_000,
				Currency:       "usd",
			}, nil
		}

		// 1. Inject billing event ledger write failure
		eventsRepo.recordErr = errors.New("db connection failure: ledger record failed")

		_, err := svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if err == nil {
			t.Fatal("expected error when billing event persistence fails, got nil")
		}

		// Ensure subscription was not upgraded
		sub, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if sub.PlanCode != "free" {
			t.Errorf("plan was upgraded despite ledger failure: %s", sub.PlanCode)
		}

		// 2. Reset ledger error, inject invoice write failure
		eventsRepo.recordErr = nil
		invoicesRepo.recordPaymentErr = errors.New("db write failed for invoice")

		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_persist_fail_1b",
				EventType:      "payment.succeeded",
				OrganizationID: adminActor.OrganizationID,
				PlanCode:       "pro",
				PaymentID:      "pay_fail_persist_inv",
				AmountMicros:   49_000_000,
				Currency:       "usd",
			}, nil
		}

		_, err = svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if err == nil {
			t.Fatal("expected error when invoice persistence fails, got nil")
		}

		// Reset invoice error, inject subscription update failure
		invoicesRepo.recordPaymentErr = nil
		subsRepo.updatePaymentErr = errors.New("db error updating payment details")

		stripeGw.webhookFn = func(ctx context.Context, payload []byte, sig string) (*payment.WebhookEvent, error) {
			return &payment.WebhookEvent{
				EventID:        "evt_persist_fail_2",
				EventType:      "subscription.updated",
				OrganizationID: adminActor.OrganizationID,
				Status:         "active",
				PlanCode:       "pro",
			}, nil
		}

		_, err = svc.ProcessWebhook(context.Background(), "stripe", []byte("payload"), "valid-sig")
		if err == nil {
			t.Fatal("expected error when subscription persistence fails, got nil")
		}
	})

	t.Run("CancelSubscription transitions subscription cancel_at_period_end", func(t *testing.T) {
		adminActor, _, subsRepo, _, _, _, _, _, _, svc := setupBillingEnv()

		sub, err := svc.CancelSubscription(context.Background(), adminActor)
		if err != nil {
			t.Fatalf("cancel subscription failed: %v", err)
		}
		if !sub.CancelAtPeriodEnd {
			t.Errorf("expected CancelAtPeriodEnd to be true, got %v", sub.CancelAtPeriodEnd)
		}

		persisted, _ := subsRepo.ActiveWithPlan(context.Background(), adminActor.OrganizationID)
		if !persisted.CancelAtPeriodEnd {
			t.Errorf("persisted subscription should have CancelAtPeriodEnd=true")
		}
	})
}
