package repositories

import (
	"context"

	"ai-governance-saas/backend/internal/domain"
)

type GuardrailsRepository interface {
	Create(ctx context.Context, guardrail domain.Guardrail) error
	Get(ctx context.Context, id, organizationID string) (domain.Guardrail, error)
	List(ctx context.Context, organizationID string) ([]domain.Guardrail, error)
	ListForEvaluation(ctx context.Context, organizationID string, targetType domain.GuardrailTargetType, targetID string) ([]domain.Guardrail, error)
	Update(ctx context.Context, guardrail domain.Guardrail) error
	Delete(ctx context.Context, id, organizationID string) error

	CreateEvent(ctx context.Context, event domain.GuardrailEvent) error
	ListEvents(ctx context.Context, organizationID string) ([]domain.GuardrailEvent, error)
}
