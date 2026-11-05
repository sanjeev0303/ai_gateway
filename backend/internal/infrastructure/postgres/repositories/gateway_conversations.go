package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type gatewayConversationStore struct {
	store *postgres.Store
}

// NewGatewayConversationStore returns the store gateway-v2 records turns into.
func NewGatewayConversationStore(store *postgres.Store) ports.GatewayConversationStore {
	return &gatewayConversationStore{store: store}
}

// RecordTurn upserts the conversation and appends the user and assistant
// messages in one tenant transaction. Only metadata is written: the title and
// the message text stay empty, and a CHECK constraint enforces that content is
// never stored. The gateway keeps the text in memory instead.
func (s *gatewayConversationStore) RecordTurn(ctx context.Context, orgID uuid.UUID, turn domain.GatewayConversationTurn) (uuid.UUID, uuid.UUID, error) {
	var ids [2]uuid.UUID
	var credentialID *uuid.UUID
	if turn.CredentialID != uuid.Nil {
		credentialID = &turn.CredentialID
	}
	err := s.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		conv, err := q.UpsertGatewayConversation(ctx, db.UpsertGatewayConversationParams{
			ID:             turn.ConversationID,
			OrganizationID: orgID,
			UserID:         turn.UserID,
			ProjectID:      turn.ProjectID,
			CredentialID:   credentialID,
			Model:          optionalString(turn.Model),
			Provider:       optionalString(turn.Provider),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			// The id belongs to another organization; never write into it.
			return nil
		}
		if err != nil {
			return err
		}

		messages := []struct {
			role, text string
			tokens     int32
		}{
			{"user", turn.UserText, turn.InputTokens},
			{"assistant", turn.AssistantText, turn.OutputTokens},
		}
		for i, m := range messages {
			if m.text == "" {
				continue
			}
			row, err := q.CreateConversationMessage(ctx, db.CreateConversationMessageParams{
				ConversationID: conv.ID,
				OrganizationID: orgID,
				UserID:         turn.UserID,
				Role:           m.role,
				Tokens:         m.tokens,
			})
			if err != nil {
				return err
			}
			ids[i] = row.ID
		}
		return nil
	})
	return ids[0], ids[1], err
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
