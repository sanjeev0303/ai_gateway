package controlapi

import (
	"net/http"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// billingUsageWindow is the default range for GET /v1/billing/usage when the
// caller sends no from/to.
const billingUsageWindow = 30 * 24 * time.Hour

func registerBillingRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/billing/plans", a.requireAuth(a.listBillingPlans))
	mux.Handle("GET /v1/billing/subscription", a.requireAuth(a.getBillingSubscription))
	mux.Handle("GET /v1/billing/usage", a.requireAuth(a.getBillingUsage))
	mux.Handle("GET /v1/billing/invoices", a.requireAuth(a.listBillingInvoices))
}

// planDTO is the wire shape of one catalog plan. It carries no price, because
// no price is stored and none is invented here.
type planDTO struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// subscriptionDTO is the wire shape of the organization's current plan and
// billing cycle.
type subscriptionDTO struct {
	PlanCode           string  `json:"plan_code"`
	PlanName           string  `json:"plan_name"`
	Status             string  `json:"status"`
	CurrentPeriodStart *string `json:"current_period_start"`
	CurrentPeriodEnd   *string `json:"current_period_end"`
	CancelAtPeriodEnd  bool    `json:"cancel_at_period_end"`
}

// invoiceLineDTO is the wire shape of one invoice itemisation row.
type invoiceLineDTO struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	Quantity     int64  `json:"quantity"`
	UnitMicros   int64  `json:"unit_micros"`
	AmountMicros int64  `json:"amount_micros"`
}

// invoiceDTO is the wire shape of one locally computed invoice. It has no
// payment reference, charge id or PDF URL: no payment processor is integrated,
// so no field here may suggest that money moved.
type invoiceDTO struct {
	ID             string           `json:"id"`
	SubscriptionID *string          `json:"subscription_id"`
	Number         string           `json:"number"`
	Status         string           `json:"status"`
	Currency       string           `json:"currency"`
	TotalMicros    int64            `json:"total_micros"`
	PeriodStart    string           `json:"period_start"`
	PeriodEnd      string           `json:"period_end"`
	IssuedAt       string           `json:"issued_at"`
	Lines          []invoiceLineDTO `json:"lines"`
}

func toPlanDTO(p domain.Plan) planDTO {
	return planDTO{
		ID:          p.ID.String(),
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		CreatedAt:   ts(p.CreatedAt),
		UpdatedAt:   ts(p.UpdatedAt),
	}
}

func toSubscriptionDTO(p domain.OrganizationPlan) subscriptionDTO {
	var start, end *string
	if p.CurrentPeriodStart != nil {
		v := ts(*p.CurrentPeriodStart)
		start = &v
	}
	if p.CurrentPeriodEnd != nil {
		v := ts(*p.CurrentPeriodEnd)
		end = &v
	}
	return subscriptionDTO{
		PlanCode:           p.PlanCode,
		PlanName:           p.PlanName,
		Status:             p.Status,
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
		CancelAtPeriodEnd:  p.CancelAtPeriodEnd,
	}
}

func toInvoiceDTO(inv domain.Invoice) invoiceDTO {
	var subID *string
	if inv.SubscriptionID != nil {
		v := inv.SubscriptionID.String()
		subID = &v
	}
	lines := make([]invoiceLineDTO, 0, len(inv.Lines))
	for _, l := range inv.Lines {
		lines = append(lines, invoiceLineDTO{
			ID:           l.ID.String(),
			Description:  l.Description,
			Quantity:     l.Quantity,
			UnitMicros:   l.UnitMicros,
			AmountMicros: l.AmountMicros,
		})
	}
	return invoiceDTO{
		ID:             inv.ID.String(),
		SubscriptionID: subID,
		Number:         inv.Number,
		Status:         inv.Status,
		Currency:       inv.Currency,
		TotalMicros:    inv.TotalMicros,
		PeriodStart:    ts(inv.PeriodStart),
		PeriodEnd:      ts(inv.PeriodEnd),
		IssuedAt:       ts(inv.IssuedAt),
		Lines:          lines,
	}
}

func (a *api) listBillingPlans(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	plans, err := a.billing.ListPlans(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]planDTO, 0, len(plans))
	for _, p := range plans {
		items = append(items, toPlanDTO(p))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *api) getBillingSubscription(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	subscription, err := a.billing.GetSubscription(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if subscription == nil {
		writeServiceError(w, r, repositories.ErrNotFound)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSubscriptionDTO(*subscription))
}

func (a *api) getBillingUsage(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	filter, err := parseUsageFilter(r, billingUsageWindow)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	summary, err := a.billing.GetUsage(r.Context(), actor, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (a *api) listBillingInvoices(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	invoices, err := a.billing.ListInvoices(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]invoiceDTO, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, toInvoiceDTO(inv))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
