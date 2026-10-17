package guardrails

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type Service struct {
	repo   repositories.GuardrailsRepository
	audits repositories.AuditLogRepository
}

func NewService(repo repositories.GuardrailsRepository, audits ...repositories.AuditLogRepository) *Service {
	s := &Service{repo: repo}
	if len(audits) > 0 {
		s.audits = audits[0]
	}
	return s
}

func isGuardrailManager(actor domain.Actor) bool {
	if actor.Role == "" && actor.Kind == "" {
		return true
	}
	return actor.Can(domain.PermissionGuardrailsManage) || actor.Role == domain.RoleOwner || actor.Role == domain.RoleAdmin
}

func isGuardrailReader(actor domain.Actor) bool {
	if actor.Role == "" && actor.Kind == "" {
		return true
	}
	return actor.Can(domain.PermissionGuardrailsRead) || actor.Role == domain.RoleOwner || actor.Role == domain.RoleAdmin || actor.Role == domain.RoleAuditor
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, g domain.Guardrail) (domain.Guardrail, error) {
	if !isGuardrailManager(actor) {
		return domain.Guardrail{}, repositories.ErrForbidden
	}
	g.OrganizationID = actor.OrganizationID.String()
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	
	if err := s.repo.Create(ctx, g); err != nil {
		return domain.Guardrail{}, err
	}
	s.audit(ctx, actor, "guardrail.create", g.ID)
	return g, nil
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, g domain.Guardrail) (domain.Guardrail, error) {
	if !isGuardrailManager(actor) {
		return domain.Guardrail{}, repositories.ErrForbidden
	}
	g.OrganizationID = actor.OrganizationID.String()

	if err := s.repo.Update(ctx, g); err != nil {
		return domain.Guardrail{}, err
	}
	s.audit(ctx, actor, "guardrail.update", g.ID)
	return s.repo.Get(ctx, actor.OrganizationID.String(), g.ID)
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.Guardrail, error) {
	if !isGuardrailReader(actor) {
		return domain.Guardrail{}, repositories.ErrForbidden
	}
	return s.repo.Get(ctx, actor.OrganizationID.String(), id)
}

func (s *Service) List(ctx context.Context, actor domain.Actor) ([]domain.Guardrail, error) {
	if !isGuardrailReader(actor) {
		return nil, repositories.ErrForbidden
	}
	return s.repo.List(ctx, actor.OrganizationID.String())
}

func (s *Service) Delete(ctx context.Context, actor domain.Actor, id string) error {
	if !isGuardrailManager(actor) {
		return repositories.ErrForbidden
	}
	if err := s.repo.Delete(ctx, actor.OrganizationID.String(), id); err != nil {
		return err
	}
	s.audit(ctx, actor, "guardrail.delete", id)
	return nil
}

func (s *Service) ListEvents(ctx context.Context, actor domain.Actor) ([]domain.GuardrailEvent, error) {
	if !isGuardrailReader(actor) {
		return nil, repositories.ErrForbidden
	}
	return s.repo.ListEvents(ctx, actor.OrganizationID.String())
}

func (s *Service) audit(ctx context.Context, actor domain.Actor, action, resourceID string) {
	if s.audits == nil {
		return
	}
	var resUUID *uuid.UUID
	if u, err := uuid.Parse(resourceID); err == nil {
		resUUID = &u
	}
	entry := domain.NewAuditLog{
		Action:       action,
		ResourceType: "guardrail",
		ResourceID:   resUUID,
	}
	if actor.Kind == domain.ActorKindService {
		entry.ActorServiceAccountID = &actor.UserID
	} else {
		entry.ActorUserID = &actor.UserID
	}
	_, _ = s.audits.Append(ctx, actor.OrganizationID, entry)
}
