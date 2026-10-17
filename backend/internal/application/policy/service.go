package policy

import (
	"context"
	"errors"

	"log/slog"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("policy or rule not found")
)

type PolicyInvalidator interface {
	Invalidate(orgID uuid.UUID)
}

type Service struct {
	policyRepo  repositories.PolicyRepository
	ruleRepo    repositories.PolicyRuleRepository
	audits      repositories.AuditLogRepository
	invalidator PolicyInvalidator
	log         *slog.Logger
}

func NewService(
	policyRepo repositories.PolicyRepository,
	ruleRepo repositories.PolicyRuleRepository,
	audits repositories.AuditLogRepository,
	invalidator PolicyInvalidator,
	log *slog.Logger,
) *Service {
	return &Service{
		policyRepo:  policyRepo,
		ruleRepo:    ruleRepo,
		audits:      audits,
		invalidator: invalidator,
		log:         log,
	}
}

func isPolicyManager(actor domain.Actor) bool {
	if actor.Role == "" && actor.Kind == "" {
		return true
	}
	return actor.Role == domain.RoleOwner || actor.Role == domain.RoleAdmin
}

func (s *Service) CreatePolicy(ctx context.Context, actor domain.Actor, input domain.NewPolicy) (domain.Policy, error) {
	if !isPolicyManager(actor) {
		return domain.Policy{}, repositories.ErrForbidden
	}
	pol, err := s.policyRepo.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.Policy{}, err
	}
	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "policy.create", "policy", &pol.ID, nil, nil)
	return pol, nil
}

func (s *Service) GetPolicy(ctx context.Context, orgID, id uuid.UUID) (domain.Policy, error) {
	return s.policyRepo.Get(ctx, orgID, id)
}

func (s *Service) ListPolicies(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	return s.policyRepo.List(ctx, orgID)
}

func (s *Service) UpdatePolicy(ctx context.Context, actor domain.Actor, id uuid.UUID, input domain.UpdatePolicy) (domain.Policy, error) {
	if !isPolicyManager(actor) {
		return domain.Policy{}, repositories.ErrForbidden
	}
	pol, err := s.policyRepo.Update(ctx, actor.OrganizationID, id, input)
	if err != nil {
		return domain.Policy{}, err
	}
	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "policy.update", "policy", &pol.ID, nil, nil)
	return pol, nil
}

func (s *Service) DeletePolicy(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	if !isPolicyManager(actor) {
		return repositories.ErrForbidden
	}
	_, err := s.policyRepo.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return err
	}
	err = s.policyRepo.Delete(ctx, actor.OrganizationID, id)
	if err != nil {
		return err
	}
	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "policy.delete", "policy", &id, nil, nil)
	return nil
}

// Rules

func (s *Service) CreateRule(ctx context.Context, actor domain.Actor, input domain.NewPolicyRule) (domain.PolicyRule, error) {
	if !isPolicyManager(actor) {
		return domain.PolicyRule{}, repositories.ErrForbidden
	}
	rule, err := s.ruleRepo.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.PolicyRule{}, err
	}
	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "policy_rule.create", "policy_rule", &rule.ID, nil, nil)
	return rule, nil
}

func (s *Service) GetRule(ctx context.Context, orgID, policyID, id uuid.UUID) (domain.PolicyRule, error) {
	return s.ruleRepo.Get(ctx, orgID, policyID, id)
}

func (s *Service) ListRules(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	return s.ruleRepo.List(ctx, orgID, policyID)
}

func (s *Service) UpdateRule(ctx context.Context, actor domain.Actor, policyID, id uuid.UUID, input domain.UpdatePolicyRule) (domain.PolicyRule, error) {
	if !isPolicyManager(actor) {
		return domain.PolicyRule{}, repositories.ErrForbidden
	}
	rule, err := s.ruleRepo.Update(ctx, actor.OrganizationID, policyID, id, input)
	if err != nil {
		return domain.PolicyRule{}, err
	}
	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "policy_rule.update", "policy_rule", &rule.ID, nil, nil)
	return rule, nil
}

func (s *Service) DeleteRule(ctx context.Context, actor domain.Actor, policyID, id uuid.UUID) error {
	if !isPolicyManager(actor) {
		return repositories.ErrForbidden
	}
	_, err := s.ruleRepo.Get(ctx, actor.OrganizationID, policyID, id)
	if err != nil {
		return err
	}
	err = s.ruleRepo.Delete(ctx, actor.OrganizationID, policyID, id)
	if err != nil {
		return err
	}
	s.invalidator.Invalidate(actor.OrganizationID)
	s.audit(ctx, actor, "policy_rule.delete", "policy_rule", &id, nil, nil)
	return nil
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
		s.log.WarnContext(ctx, "policy: audit append failed",
			"action", action, "resource_type", resourceType, "error", err)
	}
}
