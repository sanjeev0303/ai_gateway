// Package usage implements the application-layer service that owns the durable
// usage ledger and its rollups. The gateway derives usage from the upstream
// response and publishes it onto the event bus; every write happens here, so
// the data plane holds no analytics persistence responsibility. Rollups rebuild
// from llm_requests rather than from in-memory counters, so a worker restart or
// a missed run is repaired by simply running again.
package usage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ServiceDeps are the repositories a usage service requires.
type ServiceDeps struct {
	Orgs        repositories.OrganizationRepository
	Rollups     repositories.UsageRollupRepository
	LlmRequests repositories.LlmRequestRepository
	UsageEvents repositories.UsageEventRepository
	Log         *slog.Logger
}

// Service owns usage ledger writes and rollup rebuilds.
type Service struct {
	orgs        repositories.OrganizationRepository
	rollups     repositories.UsageRollupRepository
	llmRequests repositories.LlmRequestRepository
	usageEvents repositories.UsageEventRepository
	log         *slog.Logger
}

// NewService returns a usage service. Log is optional.
func NewService(deps ServiceDeps) *Service {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &Service{
		orgs:        deps.Orgs,
		rollups:     deps.Rollups,
		llmRequests: deps.LlmRequests,
		usageEvents: deps.UsageEvents,
		log:         deps.Log,
	}
}

// RecordRequest persists one completed provider request together with its
// token-usage event. Both writes are idempotent per
// (organization_id, request_id, event_type), so a redelivered message is
// treated as success rather than an error.
func (s *Service) RecordRequest(ctx context.Context, orgID uuid.UUID, req domain.NewLlmRequest, ev domain.NewUsageEvent) error {
	if _, err := s.llmRequests.Record(ctx, orgID, req); err != nil && !errors.Is(err, repositories.ErrConflict) {
		return fmt.Errorf("record llm request %s: %w", req.RequestID, err)
	}
	if _, err := s.usageEvents.Record(ctx, orgID, ev); err != nil && !errors.Is(err, repositories.ErrConflict) {
		return fmt.Errorf("record usage event %s: %w", ev.RequestID, err)
	}
	return nil
}

// RecomputeAll rebuilds [from, to) for every organization. Each organization
// is rebuilt in its own transaction, so one failure is reported without
// starving the others, and every reported error is returned joined.
func (s *Service) RecomputeAll(ctx context.Context, from, to time.Time) error {
	orgIDs, err := s.orgs.ListAllIDs(ctx)
	if err != nil {
		return fmt.Errorf("list organizations: %w", err)
	}

	var failures []error
	for _, orgID := range orgIDs {
		if err := s.rollups.Recompute(ctx, orgID, from, to); err != nil {
			s.log.Error("usage rollup failed", "organization_id", orgID, "error", err)
			failures = append(failures, fmt.Errorf("organization %s: %w", orgID, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("rebuild %d of %d organizations: %w", len(failures), len(orgIDs), errors.Join(failures...))
	}
	s.log.Info("usage rollup complete", "organizations", len(orgIDs), "from", from, "to", to)
	return nil
}

// RequirePermission authorizes actor for perm, returning
// repositories.ErrForbidden when the actor's role does not grant it.
func (s *Service) RequirePermission(actor domain.Actor, perm string) error {
	if !actor.Can(perm) {
		return repositories.ErrForbidden
	}
	return nil
}

// GetSummary returns the aggregated usage summary for the actor's organization.
func (s *Service) GetSummary(ctx context.Context, actor domain.Actor, from, to time.Time, projectID *uuid.UUID, model *string) (domain.UsageSummary, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return domain.UsageSummary{}, err
	}
	return s.rollups.GetSummary(ctx, actor.OrganizationID, from, to, projectID, model)
}

// GetTimeSeries returns the usage history over time.
func (s *Service) GetTimeSeries(ctx context.Context, actor domain.Actor, from, to time.Time, projectID *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetTimeSeries(ctx, actor.OrganizationID, from, to, projectID)
}

// GetByProject returns usage grouped by project.
func (s *Service) GetByProject(ctx context.Context, actor domain.Actor, from, to time.Time) ([]domain.UsageByProject, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByProject(ctx, actor.OrganizationID, from, to)
}

// GetByModel returns usage grouped by model.
func (s *Service) GetByModel(ctx context.Context, actor domain.Actor, from, to time.Time) ([]domain.UsageByModel, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByModel(ctx, actor.OrganizationID, from, to)
}

// GetByProvider returns usage grouped by provider.
func (s *Service) GetByProvider(ctx context.Context, actor domain.Actor, from, to time.Time) ([]domain.UsageByProvider, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByProvider(ctx, actor.OrganizationID, from, to)
}

// GetByUser returns usage grouped by user.
func (s *Service) GetByUser(ctx context.Context, actor domain.Actor, from, to time.Time) ([]domain.UsageByUser, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByUser(ctx, actor.OrganizationID, from, to)
}

// GetByTeam returns usage grouped by team.
func (s *Service) GetByTeam(ctx context.Context, actor domain.Actor, from, to time.Time) ([]domain.UsageByTeam, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByTeam(ctx, actor.OrganizationID, from, to)
}

// GetRecent returns the most recent 50 requests.
func (s *Service) GetRecent(ctx context.Context, actor domain.Actor) ([]domain.LlmRequest, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.llmRequests.GetRecent(ctx, actor.OrganizationID)
}

// GetRequests returns a paginated list of requests within a time window.
func (s *Service) GetRequests(ctx context.Context, actor domain.Actor, from, to time.Time, limit, offset int32) ([]domain.LlmRequest, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.llmRequests.GetRequests(ctx, actor.OrganizationID, from, to, limit, offset)
}
