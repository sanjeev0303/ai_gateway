package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

var (
	testOrgID  = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	testPlanID = uuid.MustParse("00000000-0000-0000-0000-0000000000b2")
	testNow    = time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
)

// ownerActor holds billing.read; memberActor does not, so it stands in for a
// caller that must be refused.
func ownerActor() domain.Actor {
	return domain.Actor{
		Kind: domain.ActorKindUser, UserID: testOrgID, OrganizationID: testOrgID,
		Role: domain.RoleOwner, DisplayName: "Owner",
	}
}

func memberActor() domain.Actor {
	return domain.Actor{
		Kind: domain.ActorKindUser, UserID: testOrgID, OrganizationID: testOrgID,
		Role: domain.RoleMember, DisplayName: "Member",
	}
}

type fakePlans struct {
	plans []domain.Plan
	err   error
	calls int
}

func (f *fakePlans) Create(context.Context, domain.NewPlan) (domain.Plan, error) {
	return domain.Plan{}, errors.New("unused")
}

func (f *fakePlans) List(context.Context) ([]domain.Plan, error) {
	f.calls++
	return f.plans, f.err
}

type fakeSubscriptions struct {
	plan   domain.OrganizationPlan
	err    error
	gotOrg uuid.UUID
	calls  int
}

func (f *fakeSubscriptions) Create(context.Context, uuid.UUID, domain.NewSubscription) (domain.Subscription, error) {
	return domain.Subscription{}, errors.New("unused")
}

func (f *fakeSubscriptions) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Subscription, error) {
	return domain.Subscription{}, errors.New("unused")
}

func (f *fakeSubscriptions) List(context.Context, uuid.UUID) ([]domain.Subscription, error) {
	return nil, errors.New("unused")
}

func (f *fakeSubscriptions) ActiveWithPlan(_ context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error) {
	f.calls++
	f.gotOrg = orgID
	return f.plan, f.err
}

func (f *fakeSubscriptions) SetPlan(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return f.err
}

func (f *fakeSubscriptions) Cancel(_ context.Context, _ uuid.UUID) error {
	return f.err
}

func (f *fakeSubscriptions) UpdatePaymentDetails(_ context.Context, _ uuid.UUID, _ string, _ string, _ string, _ string) error {
	return f.err
}

type fakeInvoices struct {
	invoices []domain.Invoice
	err      error
	gotOrg   uuid.UUID
	calls    int
}

func (f *fakeInvoices) List(_ context.Context, orgID uuid.UUID) ([]domain.Invoice, error) {
	f.calls++
	f.gotOrg = orgID
	return f.invoices, f.err
}

func (f *fakeInvoices) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Invoice, error) {
	return domain.Invoice{}, errors.New("unused")
}

func (f *fakeInvoices) RecordPayment(_ context.Context, _ uuid.UUID, _ string, _ string, _ int64) error {
	return f.err
}

// fakeRollups records the tenant and filter it was handed so the usage tests can
// assert the organization came from the actor, not from a request parameter.
type fakeRollups struct {
	summary domain.UsageSummary
	err     error
	gotOrg  uuid.UUID
	gotFrom time.Time
	gotTo   time.Time
	calls   int
}

func (f *fakeRollups) Recompute(context.Context, uuid.UUID, time.Time, time.Time) error {
	return errors.New("unused")
}

func (f *fakeRollups) GetSummary(_ context.Context, orgID uuid.UUID, filter domain.UsageFilter) (domain.UsageSummary, error) {
	f.calls++
	f.gotOrg = orgID
	f.gotFrom, f.gotTo = filter.From, filter.To
	return f.summary, f.err
}

func (f *fakeRollups) GetTimeSeries(context.Context, uuid.UUID, time.Time, time.Time, *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	return nil, errors.New("unused")
}

func (f *fakeRollups) GetByProject(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByProject, error) {
	return nil, errors.New("unused")
}

func (f *fakeRollups) GetByModel(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByModel, error) {
	return nil, errors.New("unused")
}

func (f *fakeRollups) GetByProvider(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByProvider, error) {
	return nil, errors.New("unused")
}

func (f *fakeRollups) GetByUser(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByUser, error) {
	return nil, errors.New("unused")
}

func (f *fakeRollups) GetByTeam(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByTeam, error) {
	return nil, errors.New("unused")
}

func (f *fakeRollups) GetLatency(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageLatency, error) {
	return domain.UsageLatency{}, errors.New("unused")
}

func (f *fakeRollups) GetLatencyByProviderModel(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageLatency, error) {
	return nil, errors.New("unused")
}

func newService(plans *fakePlans, subs *fakeSubscriptions, inv *fakeInvoices, roll *fakeRollups) *Service {
	return NewService(ServiceDeps{
		Plans: plans, Subscriptions: subs, Invoices: inv, Rollups: roll,
	})
}

func TestListPlansReturnsCatalog(t *testing.T) {
	plans := &fakePlans{plans: []domain.Plan{{
		ID: testPlanID, Code: "starter", Name: "Starter", Status: "active",
		CreatedAt: testNow, UpdatedAt: testNow,
	}}}
	svc := newService(plans, &fakeSubscriptions{}, &fakeInvoices{}, &fakeRollups{})

	got, err := svc.ListPlans(context.Background(), ownerActor())
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(got) != 1 || got[0].Code != "starter" {
		t.Fatalf("got %+v, want the single starter plan", got)
	}
}

// Plans are global rather than tenant data, but the read is still authorized so
// only roles granted billing.read observe them.
func TestListPlansRequiresBillingRead(t *testing.T) {
	plans := &fakePlans{plans: []domain.Plan{{ID: testPlanID}}}
	svc := newService(plans, &fakeSubscriptions{}, &fakeInvoices{}, &fakeRollups{})

	_, err := svc.ListPlans(context.Background(), memberActor())
	if !errors.Is(err, repositories.ErrForbidden) {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
	if plans.calls != 0 {
		t.Fatalf("repository called %d times, want 0", plans.calls)
	}
}

func TestGetSubscriptionScopesToActorOrganization(t *testing.T) {
	start, end := testNow.Add(-24*time.Hour), testNow.Add(24*time.Hour)
	subs := &fakeSubscriptions{plan: domain.OrganizationPlan{
		PlanCode: "starter", PlanName: "Starter", Status: "active",
		CurrentPeriodStart: &start, CurrentPeriodEnd: &end,
	}}
	svc := newService(&fakePlans{}, subs, &fakeInvoices{}, &fakeRollups{})

	got, err := svc.GetSubscription(context.Background(), ownerActor())
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if got == nil || got.PlanCode != "starter" {
		t.Fatalf("got %+v, want the starter subscription", got)
	}
	if subs.gotOrg != testOrgID {
		t.Fatalf("scoped to %v, want %v", subs.gotOrg, testOrgID)
	}
}

// An organization with no subscription is a normal state, so it surfaces as a
// nil subscription the caller can render, not an error.
func TestGetSubscriptionReturnsNilWhenNone(t *testing.T) {
	subs := &fakeSubscriptions{err: repositories.ErrNotFound}
	svc := newService(&fakePlans{}, subs, &fakeInvoices{}, &fakeRollups{})

	got, err := svc.GetSubscription(context.Background(), ownerActor())
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if got != nil {
		t.Fatalf("got %+v, want nil", got)
	}
}

// Only ErrNotFound is absorbed; a real failure must still surface.
func TestGetSubscriptionPropagatesRepositoryError(t *testing.T) {
	boom := errors.New("db down")
	svc := newService(&fakePlans{}, &fakeSubscriptions{err: boom}, &fakeInvoices{}, &fakeRollups{})

	if _, err := svc.GetSubscription(context.Background(), ownerActor()); !errors.Is(err, boom) {
		t.Fatalf("got %v, want the repository error", err)
	}
}

func TestGetSubscriptionRequiresBillingRead(t *testing.T) {
	subs := &fakeSubscriptions{}
	svc := newService(&fakePlans{}, subs, &fakeInvoices{}, &fakeRollups{})

	if _, err := svc.GetSubscription(context.Background(), memberActor()); !errors.Is(err, repositories.ErrForbidden) {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
	if subs.calls != 0 {
		t.Fatalf("repository called %d times, want 0", subs.calls)
	}
}

func TestGetUsagePassesActorOrganizationAndWindow(t *testing.T) {
	roll := &fakeRollups{summary: domain.UsageSummary{Requests: 7}}
	svc := newService(&fakePlans{}, &fakeSubscriptions{}, &fakeInvoices{}, roll)
	from, to := testNow.Add(-7*24*time.Hour), testNow

	got, err := svc.GetUsage(context.Background(), ownerActor(), domain.UsageFilter{From: from, To: to})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if got.Requests != 7 {
		t.Fatalf("got %+v, want 7 requests", got)
	}
	if roll.gotOrg != testOrgID {
		t.Fatalf("scoped to %v, want %v", roll.gotOrg, testOrgID)
	}
	if !roll.gotFrom.Equal(from) || !roll.gotTo.Equal(to) {
		t.Fatalf("window %v..%v, want %v..%v", roll.gotFrom, roll.gotTo, from, to)
	}
}

func TestGetUsageRequiresBillingRead(t *testing.T) {
	roll := &fakeRollups{}
	svc := newService(&fakePlans{}, &fakeSubscriptions{}, &fakeInvoices{}, roll)

	if _, err := svc.GetUsage(context.Background(), memberActor(), domain.UsageFilter{}); !errors.Is(err, repositories.ErrForbidden) {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
	if roll.calls != 0 {
		t.Fatalf("repository called %d times, want 0", roll.calls)
	}
}

func TestListInvoicesReturnsLines(t *testing.T) {
	inv := &fakeInvoices{invoices: []domain.Invoice{{
		ID: testPlanID, Number: "INV-1", Status: "open", Currency: "USD",
		TotalMicros: 5000, PeriodStart: testNow, PeriodEnd: testNow.Add(24 * time.Hour),
		IssuedAt: testNow,
		Lines: []domain.InvoiceLine{{
			ID: testPlanID, Description: "Starter base", Quantity: 1,
			UnitMicros: 5000, AmountMicros: 5000,
		}},
	}}}
	svc := newService(&fakePlans{}, &fakeSubscriptions{}, inv, &fakeRollups{})

	got, err := svc.ListInvoices(context.Background(), ownerActor())
	if err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if len(got) != 1 || len(got[0].Lines) != 1 {
		t.Fatalf("got %+v, want one invoice carrying one line", got)
	}
	if inv.gotOrg != testOrgID {
		t.Fatalf("scoped to %v, want %v", inv.gotOrg, testOrgID)
	}
}

// Nothing issues invoices yet, so the empty result must stay a successful empty
// list rather than an error.
func TestListInvoicesEmptyIsNotAnError(t *testing.T) {
	inv := &fakeInvoices{}
	svc := newService(&fakePlans{}, &fakeSubscriptions{}, inv, &fakeRollups{})

	got, err := svc.ListInvoices(context.Background(), ownerActor())
	if err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func TestListInvoicesRequiresBillingRead(t *testing.T) {
	inv := &fakeInvoices{}
	svc := newService(&fakePlans{}, &fakeSubscriptions{}, inv, &fakeRollups{})

	if _, err := svc.ListInvoices(context.Background(), memberActor()); !errors.Is(err, repositories.ErrForbidden) {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
	if inv.calls != 0 {
		t.Fatalf("repository called %d times, want 0", inv.calls)
	}
}
func (f *fakeRollups) GetByServiceAccount(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	return nil, nil
}
