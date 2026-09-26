package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type invitationRepo struct {
	store *postgres.Store
}

// NewInvitationRepository returns an InvitationRepository backed by the store.
// Create/Get/List/MarkAccepted/Revoke run in the org tenant context;
// ResolveByTokenHash runs without a tenant binding via the SECURITY DEFINER
// app.resolve_invitation_by_token.
func NewInvitationRepository(store *postgres.Store) ports.InvitationRepository {
	return &invitationRepo{store: store}
}

func (r *invitationRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewInvitation) (domain.Invitation, error) {
	var invitation domain.Invitation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateInvitation(ctx, db.CreateInvitationParams{
			OrganizationID: orgID,
			Email:          input.Email,
			Role:           input.Role,
			TokenHash:      input.TokenHash,
			ExpiresAt:      input.Expires,
			InvitedBy:      input.InvitedBy,
		})
		if err != nil {
			return err
		}
		invitation = toInvitation(row)
		return nil
	})
	return invitation, mapUniqueViolation(err)
}

func (r *invitationRepo) ResolveByTokenHash(ctx context.Context, tokenHash string) (domain.Invitation, error) {
	var invitation domain.Invitation
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.ResolveInvitationByToken(ctx, tokenHash)
		if err != nil {
			return err
		}
		invitation = toInvitation(row)
		return nil
	})
	return invitation, mapNotFound(err)
}

func (r *invitationRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Invitation, error) {
	var invitation domain.Invitation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetInvitation(ctx, db.GetInvitationParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		invitation = toInvitation(row)
		return nil
	})
	return invitation, mapNotFound(err)
}

func (r *invitationRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Invitation, error) {
	var invitations []domain.Invitation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListInvitations(ctx, orgID)
		if err != nil {
			return err
		}
		invitations = make([]domain.Invitation, 0, len(rows))
		for _, row := range rows {
			invitations = append(invitations, toInvitation(row))
		}
		return nil
	})
	return invitations, err
}

func (r *invitationRepo) MarkAccepted(ctx context.Context, orgID, id uuid.UUID, acceptedBy uuid.UUID) (domain.Invitation, error) {
	var invitation domain.Invitation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.MarkInvitationAccepted(ctx, db.MarkInvitationAcceptedParams{
			OrganizationID: orgID,
			AcceptedBy:     &acceptedBy,
			ID:             id,
		})
		if err != nil {
			return err
		}
		invitation = toInvitation(row)
		return nil
	})
	return invitation, mapNotFound(err)
}

func (r *invitationRepo) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.Invitation, error) {
	var invitation domain.Invitation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RevokeInvitation(ctx, db.RevokeInvitationParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		invitation = toInvitation(row)
		return nil
	})
	return invitation, mapNotFound(err)
}

func toInvitation(i db.Invitation) domain.Invitation {
	return domain.Invitation{
		ID:             i.ID,
		OrganizationID: i.OrganizationID,
		Email:          i.Email,
		Role:           i.Role,
		TokenHash:      i.TokenHash,
		Status:         i.Status,
		ExpiresAt:      i.ExpiresAt,
		InvitedBy:      i.InvitedBy,
		AcceptedBy:     i.AcceptedBy,
		AcceptedAt:     i.AcceptedAt,
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
	}
}
