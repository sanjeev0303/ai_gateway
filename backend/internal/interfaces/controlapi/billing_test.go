package controlapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// Billing fakes. These duplicate the ones in the billing package because that
// file is an internal test; only the surfaces the handlers touch are exercised.
type fakeBillingPlans struct{ plans []domain.Plan }

func (f *fakeBillingPlans) Create(context.Context, domain.NewPlan) (domain.Plan, error) {
	return domain.Plan{}, errors.New("unused")
}

func (f *fakeBillingPlans) List(context.Context) ([]domain.Plan, error) { return f.plans, nil }

type fakeBillingSubs struct {
	plan domain.OrganizationPlan
	err  error
}

func (f *fakeBillingSubs) Create(context.Context, uuid.UUID, domain.NewSubscription) (domain.Subscription, error) {
	return domain.Subscription{}, errors.New("unused")
}

func (f *fakeBillingSubs) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Subscription, error) {
	return domain.Subscription{}, errors.New("unused")
}

func (f *fakeBillingSubs) List(context.Context, uuid.UUID) ([]domain.Subscription, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingSubs) ActiveWithPlan(context.Context, uuid.UUID) (domain.OrganizationPlan, error) {
	return f.plan, f.err
}

func (f *fakeBillingSubs) SetPlan(context.Context, uuid.UUID, uuid.UUID) error {
	return f.err
}

func (f *fakeBillingSubs) Cancel(context.Context, uuid.UUID) error {
	return f.err
}

type fakeBillingInvoices struct{ invoices []domain.Invoice }

func (f *fakeBillingInvoices) List(context.Context, uuid.UUID) ([]domain.Invoice, error) {
	return f.invoices, nil
}

func (f *fakeBillingInvoices) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Invoice, error) {
	return domain.Invoice{}, errors.New("unused")
}

type fakeBillingRollups struct {
	summary domain.UsageSummary
	filter  domain.UsageFilter
}

func (f *fakeBillingRollups) Recompute(context.Context, uuid.UUID, time.Time, time.Time) error {
	return errors.New("unused")
}

func (f *fakeBillingRollups) GetSummary(_ context.Context, _ uuid.UUID, filter domain.UsageFilter) (domain.UsageSummary, error) {
	f.filter = filter
	return f.summary, nil
}

func (f *fakeBillingRollups) GetTimeSeries(context.Context, uuid.UUID, time.Time, time.Time, *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetByProject(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByProject, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetByModel(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByModel, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetByProvider(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByProvider, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetByUser(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByUser, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetByServiceAccount(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetByTeam(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByTeam, error) {
	return nil, errors.New("unused")
}

func (f *fakeBillingRollups) GetLatency(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageLatency, error) {
	return domain.UsageLatency{}, errors.New("unused")
}

func (f *fakeBillingRollups) GetLatencyByProviderModel(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageLatency, error) {
	return nil, errors.New("unused")
}

// testBillingInvoice is a locally computed invoice carrying itemisation lines.
// It deliberately has no payment reference: no processor is integrated.
var testBillingInvoice = domain.Invoice{
	ID: testAcctID, Number: "INV-2026-09", Status: "open", Currency: "USD",
	TotalMicros: 5000,
	PeriodStart: testNow.Add(-30 * 24 * time.Hour), PeriodEnd: testNow,
	IssuedAt: testNow, CreatedAt: testNow, UpdatedAt: testNow,
	Lines: []domain.InvoiceLine{{
		ID: testTeamID, InvoiceID: testAcctID, Description: "Starter base",
		Quantity: 1, UnitMicros: 5000, AmountMicros: 5000, CreatedAt: testNow,
	}},
}

func newBillingServer(t *testing.T, st *controlStubs, plans *fakeBillingPlans, subs *fakeBillingSubs, inv *fakeBillingInvoices, roll *fakeBillingRollups) *httptest.Server {
	t.Helper()
	return buildServer(t, Deps{
		Identity: identity.NewService(st.deps()),
		Billing: billing.NewService(billing.ServiceDeps{
			Plans: plans, Subscriptions: subs, Invoices: inv, Rollups: roll,
		}),
		DevLoginEnabled: true,
	})
}

func TestBillingRoutesRequireAuth(t *testing.T) {
	st := newControlStubs()
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, &fakeBillingRollups{})

	for _, path := range []string{
		"/v1/billing/plans", "/v1/billing/subscription",
		"/v1/billing/usage", "/v1/billing/invoices",
	} {
		resp := doJSON(t, srv, http.MethodGet, path, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: got %d, want 401", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestBillingListPlans(t *testing.T) {
	st := newControlStubs()
	plans := &fakeBillingPlans{plans: []domain.Plan{{
		ID: testModelID, Code: "starter", Name: "Starter", Description: "small",
		Status: "active", CreatedAt: testNow, UpdatedAt: testNow,
	}}}
	srv := newBillingServer(t, st, plans, &fakeBillingSubs{}, &fakeBillingInvoices{}, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/plans", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			ID     string `json:"id"`
			Code   string `json:"code"`
			Status string `json:"status"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) != 1 || body.Items[0].Code != "starter" {
		t.Fatalf("got %+v, want the starter plan", body.Items)
	}
	if body.Items[0].Status != "active" {
		t.Fatalf("got status %q, want active", body.Items[0].Status)
	}
}

func TestBillingPlansForbiddenWithoutBillingRead(t *testing.T) {
	st := newControlStubs()
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, &fakeBillingRollups{})

	// RoleMember can read usage but holds no billing.read.
	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/plans", nil, bearerHeader(st, domain.RoleMember))
	status, code := decodeError(t, resp)
	if status != http.StatusForbidden || code != "permission_denied" {
		t.Fatalf("got %d/%s, want 403/permission_denied", status, code)
	}
}

func TestBillingSubscription(t *testing.T) {
	st := newControlStubs()
	start, end := testNow.Add(-30*24*time.Hour), testNow
	subs := &fakeBillingSubs{plan: domain.OrganizationPlan{
		PlanCode: "starter", PlanName: "Starter", Status: "active",
		CurrentPeriodStart: &start, CurrentPeriodEnd: &end,
	}}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, subs, &fakeBillingInvoices{}, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/subscription", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		PlanCode           string  `json:"plan_code"`
		Status             string  `json:"status"`
		CurrentPeriodStart *string `json:"current_period_start"`
		CurrentPeriodEnd   *string `json:"current_period_end"`
	}
	decodeBody(t, resp, &body)
	if body.PlanCode != "starter" || body.Status != "active" {
		t.Fatalf("got %+v, want the active starter subscription", body)
	}
	if body.CurrentPeriodStart == nil || body.CurrentPeriodEnd == nil {
		t.Fatalf("got %+v, want both period bounds set", body)
	}
}

// An organization without a subscription must surface as a 404 rather than an
// empty object, so the UI can distinguish "no plan" from a malformed payload.
func TestBillingSubscriptionNotFound(t *testing.T) {
	st := newControlStubs()
	subs := &fakeBillingSubs{err: repositories.ErrNotFound}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, subs, &fakeBillingInvoices{}, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/subscription", nil, bearerHeader(st, domain.RoleOwner))
	status, code := decodeError(t, resp)
	if status != http.StatusNotFound || code != "not_found" {
		t.Fatalf("got %d/%s, want 404/not_found", status, code)
	}
}

func TestBillingUsage(t *testing.T) {
	st := newControlStubs()
	roll := &fakeBillingRollups{summary: domain.UsageSummary{
		Requests: 12, InputTokens: 100, OutputTokens: 50, TotalTokens: 150,
		TotalCostMicros: 4200,
	}}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, roll)

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/usage", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	// UsageSummary has no JSON tags, so the body is PascalCase, matching what
	// the dashboard already reads from /v1/usage/summary.
	raw := string(readBody(t, resp))
	if !strings.Contains(raw, `"Requests":12`) {
		t.Fatalf("got %s, want PascalCase Requests", raw)
	}
	if !strings.Contains(raw, `"TotalCostMicros":4200`) {
		t.Fatalf("got %s, want TotalCostMicros", raw)
	}
}

func TestBillingUsageDefaultsToThirtyDayWindow(t *testing.T) {
	st := newControlStubs()
	roll := &fakeBillingRollups{}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, roll)

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/usage", nil, bearerHeader(st, domain.RoleOwner))
	resp.Body.Close()
	if roll.filter.From.IsZero() || roll.filter.To.IsZero() {
		t.Fatalf("got window %v..%v, want both bounds defaulted", roll.filter.From, roll.filter.To)
	}
	// parseUsageFilter reads the clock separately for each bound, so the span is
	// the default window plus the gap between those two reads. Allow that skew
	// rather than demanding exact equality.
	window := roll.filter.To.Sub(roll.filter.From)
	if window < billingUsageWindow || window > billingUsageWindow+time.Second {
		t.Fatalf("window %v, want %v", window, billingUsageWindow)
	}
}

func TestBillingUsageRejectsBadFilter(t *testing.T) {
	st := newControlStubs()
	roll := &fakeBillingRollups{}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, roll)

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/usage?status=nope", nil, bearerHeader(st, domain.RoleOwner))
	status, code := decodeError(t, resp)
	if status != http.StatusBadRequest || code != "invalid_argument" {
		t.Fatalf("got %d/%s, want 400/invalid_argument", status, code)
	}
}

func TestBillingUsageForbiddenWithoutBillingRead(t *testing.T) {
	st := newControlStubs()
	roll := &fakeBillingRollups{}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, roll)

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/usage", nil, bearerHeader(st, domain.RoleMember))
	status, code := decodeError(t, resp)
	if status != http.StatusForbidden || code != "permission_denied" {
		t.Fatalf("got %d/%s, want 403/permission_denied", status, code)
	}
}

func TestBillingListInvoicesWithLines(t *testing.T) {
	st := newControlStubs()
	inv := &fakeBillingInvoices{invoices: []domain.Invoice{testBillingInvoice}}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, inv, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			ID          string `json:"id"`
			Number      string `json:"number"`
			Status      string `json:"status"`
			Currency    string `json:"currency"`
			TotalMicros int64  `json:"total_micros"`
			Lines       []struct {
				Description  string `json:"description"`
				Quantity     int64  `json:"quantity"`
				AmountMicros int64  `json:"amount_micros"`
			} `json:"lines"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) != 1 {
		t.Fatalf("got %d invoices, want 1", len(body.Items))
	}
	it := body.Items[0]
	if it.Number != "INV-2026-09" || it.Status != "open" || it.Currency != "USD" {
		t.Fatalf("got %+v, want the open INV-2026-09 invoice", it)
	}
	if it.TotalMicros != 5000 {
		t.Fatalf("got total %d, want 5000", it.TotalMicros)
	}
	if len(it.Lines) != 1 || it.Lines[0].AmountMicros != 5000 {
		t.Fatalf("got lines %+v, want one 5000 line", it.Lines)
	}
}

// Nothing issues invoices yet, so the empty case must stay a 200 with an empty
// array rather than a 404.
func TestBillingListInvoicesEmpty(t *testing.T) {
	st := newControlStubs()
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	raw := string(readBody(t, resp))
	if !strings.Contains(raw, `"items":[]`) {
		t.Fatalf("got %s, want an empty items array", raw)
	}
}

func TestBillingListInvoicesForbiddenWithoutBillingRead(t *testing.T) {
	st := newControlStubs()
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, &fakeBillingInvoices{}, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, bearerHeader(st, domain.RoleMember))
	status, code := decodeError(t, resp)
	if status != http.StatusForbidden || code != "permission_denied" {
		t.Fatalf("got %d/%s, want 403/permission_denied", status, code)
	}
}

// The invoice payload is local billing state only. No field may imply that money
// moved: no payment method, charge, transaction, or PDF link.
func TestBillingInvoiceCarriesNoPaymentReferences(t *testing.T) {
	st := newControlStubs()
	inv := &fakeBillingInvoices{invoices: []domain.Invoice{testBillingInvoice}}
	srv := newBillingServer(t, st, &fakeBillingPlans{}, &fakeBillingSubs{}, inv, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/invoices", nil, bearerHeader(st, domain.RoleOwner))
	var fields map[string]json.RawMessage
	decodeBody(t, resp, &fields)

	var items []map[string]json.RawMessage
	requireNoErr(t, json.Unmarshal(fields["items"], &items))
	if len(items) != 1 {
		t.Fatalf("got %d invoices, want 1", len(items))
	}
	for _, banned := range []string{
		"payment_method", "payment_method_id", "card", "charge", "charge_id",
		"transaction_id", "paid_at", "pdf_url", "download_url", "stripe_id",
		"hosted_invoice_url",
	} {
		if _, present := items[0][banned]; present {
			t.Fatalf("invoice payload exposed %q", banned)
		}
	}
	// A settled invoice is likewise not representable: only open/void are stored.
	if got := string(items[0]["status"]); got != `"open"` {
		t.Fatalf("got status %s, want open", got)
	}
}

// The catalog is documented as carrying no price, so a consumer cannot infer a
// chargeable amount that the backend never computed.
func TestBillingPlanCarriesNoPrice(t *testing.T) {
	st := newControlStubs()
	plans := &fakeBillingPlans{plans: []domain.Plan{{
		ID: testModelID, Code: "starter", Name: "Starter",
		Status: "active", CreatedAt: testNow, UpdatedAt: testNow,
	}}}
	srv := newBillingServer(t, st, plans, &fakeBillingSubs{}, &fakeBillingInvoices{}, &fakeBillingRollups{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/billing/plans", nil, bearerHeader(st, domain.RoleOwner))
	var fields map[string]json.RawMessage
	decodeBody(t, resp, &fields)

	var items []map[string]json.RawMessage
	requireNoErr(t, json.Unmarshal(fields["items"], &items))
	for _, banned := range []string{"price", "price_micros", "amount", "amount_micros", "currency"} {
		if _, present := items[0][banned]; present {
			t.Fatalf("plan payload exposed %q", banned)
		}
	}
}
