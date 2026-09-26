package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// InvitationRepository manages invitations within an organization.
// Create/Get/List/MarkAccepted/Revoke are tenant-scoped; ResolveByTokenHash
// reads across tenants via the SECURITY DEFINER app.resolve_invitation_by_token
// (the accept flow only knows the token, never the organization).
type InvitationRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewInvitation) (domain.Invitation, error)
	ResolveByTokenHash(ctx context.Context, tokenHash string) (domain.Invitation, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Invitation, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Invitation, error)
	MarkAccepted(ctx context.Context, orgID, id uuid.UUID, acceptedBy uuid.UUID) (domain.Invitation, error)
	Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.Invitation, error)
}
