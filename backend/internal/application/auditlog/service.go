package auditlog

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

type ServiceDeps struct {
	Audits repositories.AuditLogRepository
	Log    *slog.Logger
}

// Service reads the append-only audit ledger for a single tenant.
type Service struct {
	audits repositories.AuditLogRepository
	log    *slog.Logger
}

func NewService(deps ServiceDeps) *Service {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &Service{audits: deps.Audits, log: deps.Log}
}

func (s *Service) List(ctx context.Context, actor domain.Actor, filter domain.AuditLogFilter, limit, offset int32) ([]domain.AuditLog, bool, error) {
	if err := require(actor); err != nil {
		return nil, false, err
	}
	logs, err := s.audits.List(ctx, actor.OrganizationID, filter, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := false
	if len(logs) > int(limit) {
		hasMore = true
		logs = logs[:limit]
	}
	return logs, hasMore, nil
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.AuditLog, error) {
	if err := require(actor); err != nil {
		return domain.AuditLog{}, err
	}
	return s.audits.Get(ctx, actor.OrganizationID, id)
}

func require(actor domain.Actor) error {
	if !actor.Can(domain.PermissionAuditRead) {
		return repositories.ErrForbidden
	}
	return nil
}
