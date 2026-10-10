package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type GuardrailsRepository struct {
	store *postgres.Store
}

func NewGuardrailsRepository(store *postgres.Store) *GuardrailsRepository {
	return &GuardrailsRepository{store: store}
}

func (r *GuardrailsRepository) Create(ctx context.Context, g domain.Guardrail) error {
	query := `
		INSERT INTO guardrails (id, organization_id, name, description, target_type, target_id, detector, action, configuration, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	var targetID uuid.NullUUID
	if g.TargetID != nil && *g.TargetID != "" {
		if parsed, err := uuid.Parse(*g.TargetID); err == nil {
			targetID.UUID = parsed
			targetID.Valid = true
		}
	}

	_, err := r.store.Pool().Exec(ctx, query,
		g.ID,
		g.OrganizationID,
		g.Name,
		g.Description,
		g.TargetType,
		targetID,
		g.Detector,
		g.Action,
		g.Configuration,
		g.Enabled,
		g.CreatedAt,
		g.UpdatedAt,
	)
	return err
}

func (r *GuardrailsRepository) Get(ctx context.Context, organizationID, id string) (domain.Guardrail, error) {
	query := `
		SELECT id, organization_id, name, description, target_type, target_id, detector, action, configuration, enabled, created_at, updated_at
		FROM guardrails
		WHERE id = $1 AND organization_id = $2
	`
	var g domain.Guardrail
	var targetID uuid.NullUUID
	var rawConfig []byte
	err := r.store.Pool().QueryRow(ctx, query, id, organizationID).Scan(
		&g.ID,
		&g.OrganizationID,
		&g.Name,
		&g.Description,
		&g.TargetType,
		&targetID,
		&g.Detector,
		&g.Action,
		&rawConfig,
		&g.Enabled,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Guardrail{}, repositories.ErrNotFound
		}
		return domain.Guardrail{}, err
	}
	if targetID.Valid {
		s := targetID.UUID.String()
		g.TargetID = &s
	}
	g.Configuration = json.RawMessage(rawConfig)
	return g, nil
}

func (r *GuardrailsRepository) List(ctx context.Context, organizationID string) ([]domain.Guardrail, error) {
	query := `
		SELECT id, organization_id, name, description, target_type, target_id, detector, action, configuration, enabled, created_at, updated_at
		FROM guardrails
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.store.Pool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guardrails []domain.Guardrail
	for rows.Next() {
		var g domain.Guardrail
		var targetID uuid.NullUUID
		var rawConfig []byte
		if err := rows.Scan(
			&g.ID,
			&g.OrganizationID,
			&g.Name,
			&g.Description,
			&g.TargetType,
			&targetID,
			&g.Detector,
			&g.Action,
			&rawConfig,
			&g.Enabled,
			&g.CreatedAt,
			&g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if targetID.Valid {
			s := targetID.UUID.String()
			g.TargetID = &s
		}
		g.Configuration = json.RawMessage(rawConfig)
		guardrails = append(guardrails, g)
	}
	return guardrails, nil
}

func (r *GuardrailsRepository) ListForEvaluation(ctx context.Context, organizationID string, targetType domain.GuardrailTargetType, targetID string) ([]domain.Guardrail, error) {
	query := `
		SELECT id, organization_id, name, description, target_type, target_id, detector, action, configuration, enabled, created_at, updated_at
		FROM guardrails
		WHERE organization_id = $1 AND enabled = true AND (
			target_type = 'global' OR
			(target_type = $2 AND target_id = $3)
		)
	`
	rows, err := r.store.Pool().Query(ctx, query, organizationID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guardrails []domain.Guardrail
	for rows.Next() {
		var g domain.Guardrail
		var tID uuid.NullUUID
		var rawConfig []byte
		if err := rows.Scan(
			&g.ID,
			&g.OrganizationID,
			&g.Name,
			&g.Description,
			&g.TargetType,
			&tID,
			&g.Detector,
			&g.Action,
			&rawConfig,
			&g.Enabled,
			&g.CreatedAt,
			&g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if tID.Valid {
			s := tID.UUID.String()
			g.TargetID = &s
		}
		g.Configuration = json.RawMessage(rawConfig)
		guardrails = append(guardrails, g)
	}
	return guardrails, nil
}

func (r *GuardrailsRepository) Update(ctx context.Context, g domain.Guardrail) error {
	query := `
		UPDATE guardrails
		SET name = $1, description = $2, action = $3, configuration = $4, enabled = $5, updated_at = $6
		WHERE id = $7 AND organization_id = $8
	`
	tag, err := r.store.Pool().Exec(ctx, query,
		g.Name,
		g.Description,
		g.Action,
		g.Configuration,
		g.Enabled,
		time.Now(),
		g.ID,
		g.OrganizationID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *GuardrailsRepository) Delete(ctx context.Context, organizationID, id string) error {
	query := `DELETE FROM guardrails WHERE id = $1 AND organization_id = $2`
	tag, err := r.store.Pool().Exec(ctx, query, id, organizationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return repositories.ErrNotFound
	}
	return nil
}

func (r *GuardrailsRepository) CreateEvent(ctx context.Context, e domain.GuardrailEvent) error {
	query := `
		INSERT INTO guardrail_events (id, organization_id, guardrail_id, request_id, direction, detector, action, matched_category, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.store.Pool().Exec(ctx, query,
		e.ID,
		e.OrganizationID,
		e.GuardrailID,
		e.RequestID,
		e.Direction,
		e.Detector,
		e.Action,
		e.MatchedCategory,
		e.Timestamp,
	)
	return err
}

func (r *GuardrailsRepository) ListEvents(ctx context.Context, organizationID string) ([]domain.GuardrailEvent, error) {
	query := `
		SELECT id, organization_id, guardrail_id, request_id, direction, detector, action, matched_category, timestamp
		FROM guardrail_events
		WHERE organization_id = $1
		ORDER BY timestamp DESC
		LIMIT 100
	`
	rows, err := r.store.Pool().Query(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.GuardrailEvent
	for rows.Next() {
		var e domain.GuardrailEvent
		if err := rows.Scan(
			&e.ID,
			&e.OrganizationID,
			&e.GuardrailID,
			&e.RequestID,
			&e.Direction,
			&e.Detector,
			&e.Action,
			&e.MatchedCategory,
			&e.Timestamp,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
