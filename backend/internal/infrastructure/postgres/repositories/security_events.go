package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type securityEventRepo struct {
	store *postgres.Store
}

// NewSecurityEventRepository returns a SecurityEventRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewSecurityEventRepository(store *postgres.Store) ports.SecurityEventRepository {
	return &securityEventRepo{store: store}
}

func (r *securityEventRepo) Append(ctx context.Context, orgID uuid.UUID, input domain.NewSecurityEvent) (domain.SecurityEvent, error) {
	var event domain.SecurityEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.AppendSecurityEvent(ctx, db.AppendSecurityEventParams{
			OrganizationID: orgID,
			Category:       string(input.Category),
			Severity:       string(input.Severity),
			Title:          input.Title,
			Column5:        jsonObject(input.Detail),
			Source:         input.Source,
		})
		if err != nil {
			return err
		}
		event = toSecurityEvent(row)
		return nil
	})
	return event, err
}

func (r *securityEventRepo) List(ctx context.Context, orgID uuid.UUID, filter domain.SecurityEventFilter, limit, offset int32) ([]domain.SecurityEvent, error) {
	var events []domain.SecurityEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListSecurityEvents(ctx, db.ListSecurityEventsParams{
			OrganizationID: orgID,
			Category:       categoryOrNil(filter.Category),
			Severity:       severityOrNil(filter.Severity),
			FromTs:         timeOrNil(filter.From),
			ToTs:           timeOrNil(filter.To),
			LimitCount:     limit,
			OffsetCount:    offset,
		})
		if err != nil {
			return err
		}
		events = make([]domain.SecurityEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, toSecurityEvent(row))
		}
		return nil
	})
	return events, err
}

func toSecurityEvent(row db.SecurityEvent) domain.SecurityEvent {
	return domain.SecurityEvent{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Category:       domain.SecurityEventCategory(row.Category),
		Severity:       domain.SecurityEventSeverity(row.Severity),
		Title:          row.Title,
		Detail:         row.Detail,
		Source:         row.Source,
		CreatedAt:      row.CreatedAt,
	}
}

func categoryOrNil(c *domain.SecurityEventCategory) *string {
	if c == nil {
		return nil
	}
	s := string(*c)
	return &s
}

func severityOrNil(s *domain.SecurityEventSeverity) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
