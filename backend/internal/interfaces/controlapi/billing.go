package controlapi

import (
	"errors"
	"io"
	"net/http"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports/payment"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// billingUsageWindow is the default range for GET /v1/billing/usage when the
// caller sends no from/to.
const billingUsageWindow = 30 * 24 * time.Hour

func registerBillingRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/billing/plans", a.requireAuth(a.listBillingPlans))
	mux.Handle("GET /v1/billing/subscription", a.requireAuth(a.getBillingSubscription))
	mux.Handle("PUT /v1/billing/subscription", a.requireAuth(a.updateBillingSubscription))
	mux.Handle("DELETE /v1/billing/subscription", a.requireAuth(a.cancelBillingSubscription))
	mux.Handle("POST /v1/billing/checkout", a.requireAuth(a.createBillingCheckout))
	mux.Handle("POST /v1/billing/verify-payment", a.requireAuth(a.verifyBillingPayment))
	mux.Handle("POST /v1/billing/webhooks/stripe", http.HandlerFunc(a.handleStripeWebhook))
	mux.Handle("POST /v1/billing/webhooks/razorpay", http.HandlerFunc(a.handleRazorpayWebhook))
	mux.Handle("GET /v1/billing/usage", a.requireAuth(a.getBillingUsage))
	mux.Handle("GET /v1/billing/invoices", a.requireAuth(a.listBillingInvoices))
}

// planDTO is the wire shape of one catalog plan.
type planDTO struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// subscriptionDTO is the wire shape of the organization's current plan and billing cycle.
type subscriptionDTO struct {
	PlanCode               string  `json:"plan_code"`
	PlanName               string  `json:"plan_name"`
	Status                 string  `json:"status"`
	CurrentPeriodStart     *string `json:"current_period_start"`
	CurrentPeriodEnd       *string `json:"current_period_end"`
	CancelAtPeriodEnd      bool    `json:"cancel_at_period_end"`
	PaymentProvider        string  `json:"payment_provider,omitempty"`
	ProviderCustomerID     *string `json:"provider_customer_id,omitempty"`
	ProviderSubscriptionID *string `json:"provider_subscription_id,omitempty"`
	BillingMode            string  `json:"billing_mode,omitempty"`
	BillingEnabled         bool    `json:"billing_enabled"`
}

// invoiceLineDTO is the wire shape of one invoice itemisation row.
type invoiceLineDTO struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	Quantity     int64  `json:"quantity"`
	UnitMicros   int64  `json:"unit_micros"`
	AmountMicros int64  `json:"amount_micros"`
}

// invoiceDTO is the wire shape of one locally computed or settled invoice.
type invoiceDTO struct {
	ID              string           `json:"id"`
	SubscriptionID  *string          `json:"subscription_id"`
	Number          string           `json:"number"`
	Status          string           `json:"status"`
	Currency        string           `json:"currency"`
	TotalMicros     int64            `json:"total_micros"`
	PeriodStart     string           `json:"period_start"`
	PeriodEnd       string           `json:"period_end"`
	IssuedAt        string           `json:"issued_at"`
	PaymentProvider *string          `json:"payment_provider,omitempty"`
	PaymentID       *string          `json:"payment_id,omitempty"`
	PaidAt          *string          `json:"paid_at,omitempty"`
	Lines           []invoiceLineDTO `json:"lines"`
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
		PlanCode:               p.PlanCode,
		PlanName:               p.PlanName,
		Status:                 p.Status,
		CurrentPeriodStart:     start,
		CurrentPeriodEnd:       end,
		CancelAtPeriodEnd:      p.CancelAtPeriodEnd,
		PaymentProvider:        p.PaymentProvider,
		ProviderCustomerID:     p.ProviderCustomerID,
		ProviderSubscriptionID: p.ProviderSubscriptionID,
	}
}

func toInvoiceDTO(inv domain.Invoice) invoiceDTO {
	var subID *string
	if inv.SubscriptionID != nil {
		v := inv.SubscriptionID.String()
		subID = &v
	}
	var paidAt *string
	if inv.PaidAt != nil {
		v := ts(*inv.PaidAt)
		paidAt = &v
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
		ID:              inv.ID.String(),
		SubscriptionID:  subID,
		Number:          inv.Number,
		Status:          inv.Status,
		Currency:        inv.Currency,
		TotalMicros:     inv.TotalMicros,
		PeriodStart:     ts(inv.PeriodStart),
		PeriodEnd:       ts(inv.PeriodEnd),
		IssuedAt:        ts(inv.IssuedAt),
		PaymentProvider: inv.PaymentProvider,
		PaymentID:       inv.PaymentID,
		PaidAt:          paidAt,
		Lines:           lines,
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
	dto := toSubscriptionDTO(*subscription)
	dto.BillingMode = a.billing.Mode()
	dto.BillingEnabled = a.billing.Mode() != "disabled"
	httpx.WriteJSON(w, http.StatusOK, dto)
}

type updateSubscriptionRequest struct {
	PlanCode string `json:"plan_code"`
}

func (a *api) updateBillingSubscription(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req updateSubscriptionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	if req.PlanCode == "" {
		writeServiceError(w, r, repositories.ErrInvalid)
		return
	}
	subscription, err := a.billing.UpdateSubscription(r.Context(), actor, req.PlanCode)
	if err != nil {
		if errors.Is(err, payment.ErrBillingDisabled) {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Billing is disabled",
				"code":  "billing_disabled",
			})
			return
		}
		writeServiceError(w, r, err)
		return
	}
	dto := toSubscriptionDTO(*subscription)
	dto.BillingMode = a.billing.Mode()
	dto.BillingEnabled = a.billing.Mode() != "disabled"
	httpx.WriteJSON(w, http.StatusOK, dto)
}

func (a *api) cancelBillingSubscription(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	subscription, err := a.billing.CancelSubscription(r.Context(), actor)
	if err != nil {
		if errors.Is(err, payment.ErrBillingDisabled) {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Billing is disabled",
				"code":  "billing_disabled",
			})
			return
		}
		writeServiceError(w, r, err)
		return
	}
	dto := toSubscriptionDTO(*subscription)
	dto.BillingMode = a.billing.Mode()
	dto.BillingEnabled = a.billing.Mode() != "disabled"
	httpx.WriteJSON(w, http.StatusOK, dto)
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

type checkoutRequest struct {
	Provider        string `json:"provider"`
	PaymentProvider string `json:"payment_provider"`
	PlanCode        string `json:"plan_code"`
	CustomerEmail   string `json:"customer_email"`
	SuccessURL      string `json:"success_url"`
	CancelURL       string `json:"cancel_url"`
}

func (a *api) createBillingCheckout(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req checkoutRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	if req.PlanCode == "" {
		writeServiceError(w, r, repositories.ErrInvalid)
		return
	}
	providerName := req.PaymentProvider
	if providerName == "" {
		providerName = req.Provider
	}
	sess, err := a.billing.CreateCheckoutSession(r.Context(), actor, providerName, req.PlanCode, req.CustomerEmail, req.SuccessURL, req.CancelURL)
	if err != nil {
		if errors.Is(err, payment.ErrBillingDisabled) {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Billing is disabled",
				"code":  "billing_disabled",
			})
			return
		}
		if errors.Is(err, payment.ErrMissingCredentials) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Payment provider credentials are missing",
				"code":  "missing_credentials",
			})
			return
		}
		if errors.Is(err, payment.ErrInvalidCredentials) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Payment provider credentials are invalid",
				"code":  "invalid_credentials",
			})
			return
		}
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sess)
}

type verifyPaymentRequest struct {
	Provider        string `json:"provider"`
	PaymentProvider string `json:"payment_provider"`
	OrderID         string `json:"order_id"`
	PaymentID       string `json:"payment_id"`
	Signature       string `json:"signature"`
	PlanCode        string `json:"plan_code"`
}

func (a *api) verifyBillingPayment(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req verifyPaymentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	providerName := req.PaymentProvider
	if providerName == "" {
		providerName = req.Provider
	}
	if providerName == "" || req.Signature == "" {
		writeServiceError(w, r, repositories.ErrInvalid)
		return
	}
	sub, err := a.billing.VerifyPaymentCallback(r.Context(), actor, providerName, req.OrderID, req.PaymentID, req.Signature, req.PlanCode)
	if err != nil {
		if errors.Is(err, payment.ErrBillingDisabled) {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Billing is disabled",
				"code":  "billing_disabled",
			})
			return
		}
		if errors.Is(err, payment.ErrSimulationOnly) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Cannot grant paid entitlements in simulation mode",
				"code":  "simulation_mode",
			})
			return
		}
		if errors.Is(err, payment.ErrInvalidSignature) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid payment signature",
				"code":  "invalid_signature",
			})
			return
		}
		if errors.Is(err, payment.ErrMissingCredentials) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Payment provider credentials are missing",
				"code":  "missing_credentials",
			})
			return
		}
		if errors.Is(err, payment.ErrInvalidCredentials) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Payment provider credentials are invalid",
				"code":  "invalid_credentials",
			})
			return
		}
		writeServiceError(w, r, err)
		return
	}
	if sub == nil {
		writeServiceError(w, r, repositories.ErrNotFound)
		return
	}
	dto := toSubscriptionDTO(*sub)
	dto.BillingMode = a.billing.Mode()
	dto.BillingEnabled = a.billing.Mode() != "disabled"
	httpx.WriteJSON(w, http.StatusOK, dto)
}

func (a *api) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("Stripe-Signature")
	_, err = a.billing.ProcessWebhook(r.Context(), "stripe", body, sig)
	if err != nil {
		if errors.Is(err, payment.ErrBillingDisabled) {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Billing is disabled",
				"code":  "billing_disabled",
			})
			return
		}
		if errors.Is(err, payment.ErrInvalidSignature) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid webhook signature",
				"code":  "invalid_signature",
			})
			return
		}
		if errors.Is(err, payment.ErrMissingCredentials) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Missing webhook credentials",
				"code":  "missing_credentials",
			})
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"received": true})
}

func (a *api) handleRazorpayWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	sig := r.Header.Get("X-Razorpay-Signature")
	_, err = a.billing.ProcessWebhook(r.Context(), "razorpay", body, sig)
	if err != nil {
		if errors.Is(err, payment.ErrBillingDisabled) {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Billing is disabled",
				"code":  "billing_disabled",
			})
			return
		}
		if errors.Is(err, payment.ErrInvalidSignature) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid webhook signature",
				"code":  "invalid_signature",
			})
			return
		}
		if errors.Is(err, payment.ErrMissingCredentials) {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Missing webhook credentials",
				"code":  "missing_credentials",
			})
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"received": true})
}
