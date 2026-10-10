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

	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ServiceDeps are the repositories a usage service requires.
type ServiceDeps struct {
	Orgs           repositories.OrganizationRepository
	Rollups        repositories.UsageRollupRepository
	LlmRequests    repositories.LlmRequestRepository
	UsageEvents    repositories.UsageEventRepository
	SecurityEvents securityevent.Recorder
	Log            *slog.Logger
}

// Service owns usage ledger writes and rollup rebuilds.
type Service struct {
	orgs           repositories.OrganizationRepository
	rollups        repositories.UsageRollupRepository
	llmRequests    repositories.LlmRequestRepository
	usageEvents    repositories.UsageEventRepository
	securityEvents securityevent.Recorder
	log            *slog.Logger
}

// NewService returns a usage service. Log is optional.
func NewService(deps ServiceDeps) *Service {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &Service{
		orgs:           deps.Orgs,
		rollups:        deps.Rollups,
		llmRequests:    deps.LlmRequests,
		usageEvents:    deps.UsageEvents,
		securityEvents: deps.SecurityEvents,
		log:            deps.Log,
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
			s.recordRollupFailure(ctx, orgID, from, to, err)
			failures = append(failures, fmt.Errorf("organization %s: %w", orgID, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("rebuild %d of %d organizations: %w", len(failures), len(orgIDs), errors.Join(failures...))
	}
	s.log.Info("usage rollup complete", "organizations", len(orgIDs), "from", from, "to", to)
	return nil
}

// recordRollupFailure surfaces one organization's rollup failure on the
// security feed. It never returns an error and never changes RecomputeAll's
// result: the rollup already failed and is already reported, and the event
// write must not add a second failure or mask the first. Rollup failures also
// recur on every retry, so the window is folded into the event instead.
func (s *Service) recordRollupFailure(ctx context.Context, orgID uuid.UUID, from, to time.Time, cause error) {
	if s.securityEvents == nil {
		return
	}
	stage := fmt.Sprintf("rollup %s to %s", from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err := s.securityEvents.RecordPipelineFailure(ctx, orgID, stage, cause); err != nil {
		s.log.Error("recording pipeline failure event failed", "organization_id", orgID, "error", err)
	}
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
func (s *Service) GetSummary(ctx context.Context, actor domain.Actor, filter domain.UsageFilter) (domain.UsageSummary, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return domain.UsageSummary{}, err
	}
	return s.rollups.GetSummary(ctx, actor.OrganizationID, filter)
}

// GetTimeSeries returns the usage history over time.
func (s *Service) GetTimeSeries(ctx context.Context, actor domain.Actor, from, to time.Time, projectID *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetTimeSeries(ctx, actor.OrganizationID, from, to, projectID)
}

// GetByProject returns usage grouped by project.
func (s *Service) GetByProject(ctx context.Context, actor domain.Actor, filter domain.UsageFilter) ([]domain.UsageByProject, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByProject(ctx, actor.OrganizationID, filter)
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
func (s *Service) GetByUser(ctx context.Context, actor domain.Actor, filter domain.UsageFilter) ([]domain.UsageByUser, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByUser(ctx, actor.OrganizationID, filter)
}

// GetByServiceAccount returns usage grouped by service account.
func (s *Service) GetByServiceAccount(ctx context.Context, actor domain.Actor, filter domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByServiceAccount(ctx, actor.OrganizationID, filter)
}

// GetByTeam returns usage grouped by team.
func (s *Service) GetByTeam(ctx context.Context, actor domain.Actor, filter domain.UsageFilter) ([]domain.UsageByTeam, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.rollups.GetByTeam(ctx, actor.OrganizationID, filter)
}

// GetLatency returns percentile timings, either as one row for the whole
// filtered set or broken down per provider and model. Percentiles are always
// computed in Postgres, never from raw samples here.
func (s *Service) GetLatency(ctx context.Context, actor domain.Actor, filter domain.UsageFilter, groupBy bool) ([]domain.UsageLatency, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	if groupBy {
		return s.rollups.GetLatencyByProviderModel(ctx, actor.OrganizationID, filter)
	}
	overall, err := s.rollups.GetLatency(ctx, actor.OrganizationID, filter)
	if err != nil {
		return nil, err
	}
	return []domain.UsageLatency{overall}, nil
}

// GetRecent returns the most recent 50 requests.
func (s *Service) GetRecent(ctx context.Context, actor domain.Actor) ([]domain.LlmRequest, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, err
	}
	return s.llmRequests.GetRecent(ctx, actor.OrganizationID)
}

// GetRequests returns a paginated list of requests matching the filter.
func (s *Service) GetRequests(ctx context.Context, actor domain.Actor, filter domain.UsageFilter, limit, offset int32) ([]domain.LlmRequest, bool, error) {
	if err := s.RequirePermission(actor, domain.PermissionUsageRead); err != nil {
		return nil, false, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	reqs, err := s.llmRequests.GetRequests(ctx, actor.OrganizationID, filter, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := false
	if len(reqs) > int(limit) {
		hasMore = true
		reqs = reqs[:limit]
	}
	return reqs, hasMore, nil
}
