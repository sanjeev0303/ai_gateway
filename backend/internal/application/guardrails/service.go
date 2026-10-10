package guardrails

import (
	"context"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type Service struct {
	repo repositories.GuardrailsRepository
}

func NewService(repo repositories.GuardrailsRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, g domain.Guardrail) (domain.Guardrail, error) {
	// Usually authorization happens in HTTP handlers or middleware in this repo
	g.OrganizationID = actor.OrganizationID.String()
	
	if err := s.repo.Create(ctx, g); err != nil {
		return domain.Guardrail{}, err
	}
	return g, nil
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, g domain.Guardrail) (domain.Guardrail, error) {
	g.OrganizationID = actor.OrganizationID.String()

	if err := s.repo.Update(ctx, g); err != nil {
		return domain.Guardrail{}, err
	}
	return s.repo.Get(ctx, actor.OrganizationID.String(), g.ID)
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id string) (domain.Guardrail, error) {
	return s.repo.Get(ctx, actor.OrganizationID.String(), id)
}

func (s *Service) List(ctx context.Context, actor domain.Actor) ([]domain.Guardrail, error) {
	return s.repo.List(ctx, actor.OrganizationID.String())
}

func (s *Service) Delete(ctx context.Context, actor domain.Actor, id string) error {
	return s.repo.Delete(ctx, actor.OrganizationID.String(), id)
}

func (s *Service) ListEvents(ctx context.Context, actor domain.Actor) ([]domain.GuardrailEvent, error) {
	return s.repo.ListEvents(ctx, actor.OrganizationID.String())
}
