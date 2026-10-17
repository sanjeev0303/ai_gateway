// Package billing serves the organization's local billing state: the plan
// catalog, its current subscription, period usage cost, and locally computed
// invoices. No payment processor is integrated, so nothing this package returns
// represents a charge, a processor transaction, or a settled payment.
package billing

import (
	"context"
	"errors"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ServiceDeps are the repositories a billing service requires.
type ServiceDeps struct {
	Plans         repositories.PlanRepository
	Subscriptions repositories.SubscriptionRepository
	Invoices      repositories.InvoiceRepository
	Rollups       repositories.UsageRollupRepository
}

// Service owns reads of local billing state.
type Service struct {
	plans         repositories.PlanRepository
	subscriptions repositories.SubscriptionRepository
	invoices      repositories.InvoiceRepository
	rollups       repositories.UsageRollupRepository
}

// NewService returns a billing service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		plans:         deps.Plans,
		subscriptions: deps.Subscriptions,
		invoices:      deps.Invoices,
		rollups:       deps.Rollups,
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

// ListInvoices returns the organization's invoices newest-issued first. The
// result is empty until invoices are issued; nothing in this codebase issues
// them yet, because there is no settlement step.
func (s *Service) ListInvoices(ctx context.Context, actor domain.Actor) ([]domain.Invoice, error) {
	if err := s.RequirePermission(actor, domain.PermissionBillingRead); err != nil {
		return nil, err
	}
	return s.invoices.List(ctx, actor.OrganizationID)
}

// UpdateSubscription changes the organization's plan to target planCode.
func (s *Service) UpdateSubscription(ctx context.Context, actor domain.Actor, planCode string) (*domain.OrganizationPlan, error) {
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
	if !actor.Can(domain.PermissionBillingManage) && actor.Role != domain.RoleAdmin {
		return nil, repositories.ErrForbidden
	}
	if err := s.subscriptions.Cancel(ctx, actor.OrganizationID); err != nil {
		return nil, err
	}
	return s.GetSubscription(ctx, actor)
}
