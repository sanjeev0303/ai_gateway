package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// ProjectMemberRepository manages project_members rows, joining the user
// profile on List. The join table is FORCE RLS; every operation runs inside
// the tenant context.
type ProjectMemberRepository interface {
	Add(ctx context.Context, orgID, projectID, userID uuid.UUID, role string) (domain.ProjectMember, error)
	Get(ctx context.Context, orgID, projectID, userID uuid.UUID) (domain.ProjectMember, error)
	Remove(ctx context.Context, orgID, projectID, userID uuid.UUID) error
	List(ctx context.Context, orgID, projectID uuid.UUID) ([]domain.ProjectMemberView, error)
}
