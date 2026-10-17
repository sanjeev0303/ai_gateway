package budget

import (
	"context"
	"encoding/json"
	"errors"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("budget not found")
	ErrInvalid  = errors.New("invalid budget")
)

type BudgetInvalidator interface {
	Invalidate(orgID uuid.UUID)
}

type ServiceDeps struct {
	Budgets     repositories.BudgetRepository
	Policies    repositories.PolicyRepository
	Rules       repositories.PolicyRuleRepository
	Audits      repositories.AuditLogRepository
	Invalidator BudgetInvalidator
}

type Service struct {
	budgets     repositories.BudgetRepository
	policies    repositories.PolicyRepository
	rules       repositories.PolicyRuleRepository
	audits      repositories.AuditLogRepository
	invalidator BudgetInvalidator
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		budgets:     deps.Budgets,
		policies:    deps.Policies,
		rules:       deps.Rules,
		audits:      deps.Audits,
		invalidator: deps.Invalidator,
	}
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, input domain.NewBudget) (domain.Budget, error) {
	if err := s.validateNewBudget(input); err != nil {
		return domain.Budget{}, err
	}

	budget, err := s.budgets.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.Budget{}, err
	}

	// Create corresponding policy budget rules for gateway enforcement
	if err := s.syncBudgetToPolicyRules(ctx, actor, budget); err != nil {
		return domain.Budget{}, err
	}

	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "budget.create", "budget", &budget.ID, nil, nil)
	return budget, nil
}

func (s *Service) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Budget, error) {
	return s.budgets.Get(ctx, orgID, id)
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, filter repositories.BudgetFilter) ([]domain.Budget, error) {
	return s.budgets.List(ctx, orgID, filter)
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, id uuid.UUID, input domain.UpdateBudget) (domain.Budget, error) {
	_, err := s.budgets.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return domain.Budget{}, err
	}

	if err := s.validateUpdateBudget(input); err != nil {
		return domain.Budget{}, err
	}

	budget, err := s.budgets.Update(ctx, actor.OrganizationID, id, input)
	if err != nil {
		return domain.Budget{}, err
	}

	// Sync changes to policy rules
	if err := s.syncBudgetToPolicyRules(ctx, actor, budget); err != nil {
		return domain.Budget{}, err
	}

	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "budget.update", "budget", &budget.ID, nil, nil)
	return budget, nil
}

func (s *Service) Delete(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	_, err := s.budgets.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return err
	}

	// Delete associated policy budget rules
	if err := s.deleteBudgetPolicyRules(ctx, actor.OrganizationID, id); err != nil {
		return err
	}

	if err := s.budgets.Delete(ctx, actor.OrganizationID, id); err != nil {
		return err
	}

	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "budget.delete", "budget", &id, nil, nil)
	return nil
}

func (s *Service) validateNewBudget(input domain.NewBudget) error {
	if input.Name == "" {
		return ErrInvalid
	}
	if input.LimitAmount <= 0 {
		return ErrInvalid
	}
	if input.AlertThreshold < 0 || input.AlertThreshold > 1 {
		return ErrInvalid
	}
	return nil
}

func (s *Service) validateUpdateBudget(input domain.UpdateBudget) error {
	if input.LimitAmount != nil && *input.LimitAmount <= 0 {
		return ErrInvalid
	}
	if input.AlertThreshold != nil && (*input.AlertThreshold < 0 || *input.AlertThreshold > 1) {
		return ErrInvalid
	}
	return nil
}

// syncBudgetToPolicyRules creates or updates policy budget rules that correspond to this budget.
// This is how budgets are enforced on the gateway hot path.
func (s *Service) syncBudgetToPolicyRules(ctx context.Context, actor domain.Actor, budget domain.Budget) error {
	// Find or create a policy for budgets
	pol, err := s.findOrCreateBudgetPolicy(ctx, actor.OrganizationID)
	if err != nil {
		return err
	}

	// Build the budget rule
	// Use scope-based key for gateway budget guard matching (format: scope:scope_id)
	scopeKey := string(budget.Scope)
	if budget.Scope != domain.BudgetScopeOrganization {
		scopeKey = string(budget.Scope) + ":" + budget.ScopeID.String()
	} else {
		scopeKey = "org:" + actor.OrganizationID.String()
	}
	conditions := map[string]any{
		"type":       "budget",
		"key":        scopeKey,
		"scope":      budget.Scope,
		"scope_id":   budget.ScopeID.String(),
		"limit_type": budget.LimitType,
	}
	parameters := map[string]any{
		"limit":          budget.LimitAmount,
		"window_seconds": windowToSeconds(budget.Window),
		"alert_at":       budget.LimitAmount * budget.AlertThreshold,
		"scope":          budget.Scope,
	}

	condBytes, _ := json.Marshal(conditions)
	paramBytes, _ := json.Marshal(parameters)

	// Check if rule already exists for this budget
	rules, err := s.rules.List(ctx, actor.OrganizationID, pol.ID)
	if err != nil {
		return err
	}

	var existingRule *domain.PolicyRule
	for _, r := range rules {
		var cond map[string]any
		if err := json.Unmarshal(r.Conditions, &cond); err == nil {
			if cond["key"] == budget.ID.String() {
				existingRule = &r
				break
			}
		}
	}

	if existingRule != nil {
		// Update existing rule
		_, err = s.rules.Update(ctx, actor.OrganizationID, pol.ID, existingRule.ID, domain.UpdatePolicyRule{
			Conditions: condBytes,
			Parameters: paramBytes,
			Enabled:    &budget.Enabled,
		})
	} else {
		// Create new rule
		_, err = s.rules.Create(ctx, actor.OrganizationID, domain.NewPolicyRule{
			PolicyID:    pol.ID,
			RuleType:    domain.RuleTypeBudget,
			Description: budget.Name,
			Conditions:  condBytes,
			Parameters:  paramBytes,
			Enabled:     budget.Enabled,
		})
	}

	return err
}

func (s *Service) deleteBudgetPolicyRules(ctx context.Context, orgID uuid.UUID, budgetID uuid.UUID) error {
	// First, get the budget to build the scope-based key
	budget, err := s.budgets.Get(ctx, orgID, budgetID)
	if err != nil {
		return err
	}

	// Build scope-based key (same as in syncBudgetToPolicyRules)
	scopeKey := string(budget.Scope)
	if budget.Scope != domain.BudgetScopeOrganization {
		scopeKey = string(budget.Scope) + ":" + budget.ScopeID.String()
	} else {
		scopeKey = "org:" + orgID.String()
	}

	pol, err := s.findOrCreateBudgetPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	rules, err := s.rules.List(ctx, orgID, pol.ID)
	if err != nil {
		return err
	}

	for _, r := range rules {
		var cond map[string]any
		if err := json.Unmarshal(r.Conditions, &cond); err == nil {
			if cond["key"] == scopeKey {
				if err := s.rules.Delete(ctx, orgID, pol.ID, r.ID); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Service) findOrCreateBudgetPolicy(ctx context.Context, orgID uuid.UUID) (domain.Policy, error) {
	policies, err := s.policies.List(ctx, orgID)
	if err != nil {
		return domain.Policy{}, err
	}

	for _, p := range policies {
		if p.Name == "Budget Enforcement" {
			return p, nil
		}
	}

	// Create budget policy
	return s.policies.Create(ctx, orgID, domain.NewPolicy{
		Name:        "Budget Enforcement",
		Description: "Auto-generated policy for budget enforcement rules",
		Enforcement: domain.EnforcementActive,
		Priority:    100, // High priority for budgets
	})
}

func windowToSeconds(w domain.BudgetWindow) int64 {
	switch w {
	case domain.BudgetWindowHourly:
		return 3600
	case domain.BudgetWindowDaily:
		return 86400
	case domain.BudgetWindowWeekly:
		return 604800
	case domain.BudgetWindowMonthly:
		return 2592000 // 30 days
	default:
		return 86400
	}
}

func (s *Service) audit(ctx context.Context, actor domain.Actor, action, resourceType string, resourceID *uuid.UUID, before, after []byte) {
	entry := domain.NewAuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
	}
	if actor.Kind == domain.ActorKindService {
		entry.ActorServiceAccountID = &actor.UserID
	} else {
		entry.ActorUserID = &actor.UserID
	}
	if _, err := s.audits.Append(ctx, actor.OrganizationID, entry); err != nil {
		// Log but don't fail the operation
	}
}
