package securityevent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

type ServiceDeps struct {
	Events repositories.SecurityEventRepository
	Log    *slog.Logger
}

// Service appends and reads the platform's observations about itself.
type Service struct {
	events repositories.SecurityEventRepository
	log    *slog.Logger
}

func NewService(deps ServiceDeps) *Service {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &Service{events: deps.Events, log: deps.Log}
}

// Recorder is the write-side dependency for components that detect a
// condition. It is deliberately narrower than Service: a component records the
// events it causes and never reads the feed back.
type Recorder interface {
	RecordProviderTransition(ctx context.Context, orgID uuid.UUID, state domain.ProviderHealthState, from string) error
	RecordPipelineFailure(ctx context.Context, orgID uuid.UUID, stage string, cause error) error
}

// Record appends one event. It takes no actor because only components that
// detect a condition call it: there is no caller-supplied identity whose
// permission to write could be checked, and the detecting component supplies
// the organization the event belongs to.
func (s *Service) Record(ctx context.Context, orgID uuid.UUID, input domain.NewSecurityEvent) error {
	if orgID == uuid.Nil {
		return fmt.Errorf("securityevent: record: %w: missing organization", repositories.ErrInvalid)
	}
	if !input.Category.Valid() {
		return fmt.Errorf("securityevent: record: %w: unknown category %q", repositories.ErrInvalid, input.Category)
	}
	if !input.Severity.Valid() {
		return fmt.Errorf("securityevent: record: %w: unknown severity %q", repositories.ErrInvalid, input.Severity)
	}
	if input.Title == "" {
		return fmt.Errorf("securityevent: record: %w: empty title", repositories.ErrInvalid)
	}
	if input.Source == "" {
		return fmt.Errorf("securityevent: record: %w: empty source", repositories.ErrInvalid)
	}
	_, err := s.events.Append(ctx, orgID, input)
	return err
}

type providerTransitionDetail struct {
	ProviderKind  string `json:"provider_kind"`
	From          string `json:"from"`
	To            string `json:"to"`
	StatusReason  string `json:"status_reason,omitempty"`
	LastErrorText string `json:"last_error,omitempty"`
}

// RecordProviderTransition appends a provider_incident event for one observed
// change in a provider's reachability. Recovery is recorded too: an operator
// needs to see that a provider came back, not only that it went down. The
// subject is the provider kind because that is the granularity at which
// reachability is observed; a provider account is not in scope here.
func (s *Service) RecordProviderTransition(ctx context.Context, orgID uuid.UUID, state domain.ProviderHealthState, from string) error {
	detail, err := json.Marshal(providerTransitionDetail{
		ProviderKind:  state.ProviderKind,
		From:          from,
		To:            state.Status,
		StatusReason:  state.StatusReason,
		LastErrorText: state.LastError,
	})
	if err != nil {
		return err
	}
	title := fmt.Sprintf("Provider %s is now %s", state.ProviderKind, state.Status)
	if state.Status == domain.ProviderHealthHealthy {
		title = fmt.Sprintf("Provider %s recovered", state.ProviderKind)
	}
	return s.Record(ctx, orgID, domain.NewSecurityEvent{
		Category: domain.SecurityEventProviderIncident,
		Severity: severityForProviderStatus(state.Status),
		Title:    title,
		Detail:   detail,
		Source:   "provider_health",
	})
}

type pipelineFailureDetail struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// RecordPipelineFailure appends a pipeline_failure event for one organization's
// rollup failing.
func (s *Service) RecordPipelineFailure(ctx context.Context, orgID uuid.UUID, stage string, cause error) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	detail, err := json.Marshal(pipelineFailureDetail{Stage: stage, Message: message})
	if err != nil {
		return err
	}
	return s.Record(ctx, orgID, domain.NewSecurityEvent{
		Category: domain.SecurityEventPipelineFailure,
		Severity: domain.SecurityEventSeverityWarning,
		Title:    fmt.Sprintf("Usage pipeline failed during %s", stage),
		Detail:   detail,
		Source:   "usage_rollup",
	})
}

// severityForProviderStatus maps an observed status onto how loudly the event
// should read. A recovery is informational; an unreachable provider is
// critical because traffic routed to it fails.
func severityForProviderStatus(status string) domain.SecurityEventSeverity {
	switch status {
	case domain.ProviderHealthUnhealthy:
		return domain.SecurityEventSeverityCritical
	case domain.ProviderHealthDegraded:
		return domain.SecurityEventSeverityWarning
	default:
		return domain.SecurityEventSeverityInfo
	}
}

// List returns the organization's recorded events, newest first.
func (s *Service) List(ctx context.Context, actor domain.Actor, filter domain.SecurityEventFilter, limit, offset int32) ([]domain.SecurityEvent, error) {
	if err := require(actor); err != nil {
		return nil, err
	}
	return s.events.List(ctx, actor.OrganizationID, filter, limit, offset)
}

func require(actor domain.Actor) error {
	if !actor.Can(domain.PermissionSecurityEventsRead) {
		return repositories.ErrForbidden
	}
	return nil
}
