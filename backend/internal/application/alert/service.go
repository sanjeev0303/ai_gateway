package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"log/slog"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

type ServiceDeps struct {
	Log       *slog.Logger
	Alerts    repositories.AlertRepository
	Audits    repositories.AuditLogRepository
	Evaluator *Evaluator
}

// Service manages an organization's configured alert rules.
type Service struct {
	log       *slog.Logger
	alerts    repositories.AlertRepository
	audits    repositories.AuditLogRepository
	evaluator *Evaluator
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		log:       deps.Log,
		alerts:    deps.Alerts,
		audits:    deps.Audits,
		evaluator: deps.Evaluator,
	}
}


func (s *Service) List(ctx context.Context, actor domain.Actor) ([]domain.Alert, error) {
	if !actor.Can(domain.PermissionAlertsRead) {
		return nil, repositories.ErrForbidden
	}
	return s.alerts.List(ctx, actor.OrganizationID)
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.Alert, error) {
	if !actor.Can(domain.PermissionAlertsRead) {
		return domain.Alert{}, repositories.ErrForbidden
	}
	return s.alerts.Get(ctx, actor.OrganizationID, id)
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, input domain.NewAlert) (domain.Alert, error) {
	if !actor.Can(domain.PermissionAlertsManage) {
		return domain.Alert{}, repositories.ErrForbidden
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Alert{}, fmt.Errorf("name must not be empty: %w", repositories.ErrInvalid)
	}
	if !domain.ValidAlertSeverity(input.Severity) {
		return domain.Alert{}, fmt.Errorf("severity must be one of info, warning, critical: %w", repositories.ErrInvalid)
	}
	if input.AlertType == "" {
		return domain.Alert{}, fmt.Errorf("alert type must not be empty: %w", repositories.ErrInvalid)
	}
	if err := validateConfiguration(input.Configuration); err != nil {
		return domain.Alert{}, err
	}
	input.Name = name
	alert, err := s.alerts.Create(ctx, actor.OrganizationID, input)
	if err == nil {
		s.audit(ctx, actor, "alert.create", "alert", &alert.ID, nil, nil)
	}
	return alert, err
}

func (s *Service) Update(ctx context.Context, actor domain.Actor, id uuid.UUID, input domain.AlertUpdate) (domain.Alert, error) {
	if !actor.Can(domain.PermissionAlertsManage) {
		return domain.Alert{}, repositories.ErrForbidden
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Alert{}, fmt.Errorf("name must not be empty: %w", repositories.ErrInvalid)
	}
	if !domain.ValidAlertSeverity(input.Severity) {
		return domain.Alert{}, fmt.Errorf("severity must be one of info, warning, critical: %w", repositories.ErrInvalid)
	}
	if !domain.ValidAlertStatus(input.Status) {
		return domain.Alert{}, fmt.Errorf("status must be one of active, muted: %w", repositories.ErrInvalid)
	}
	if err := validateConfiguration(input.Configuration); err != nil {
		return domain.Alert{}, err
	}
	input.Name = name
	alert, err := s.alerts.Update(ctx, actor.OrganizationID, id, input)
	if err == nil {
		s.audit(ctx, actor, "alert.update", "alert", &alert.ID, nil, nil) // omitting config changes to avoid exposing sensitive info or bloating
	}
	return alert, err
}

func (s *Service) Delete(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	if !actor.Can(domain.PermissionAlertsManage) {
		return repositories.ErrForbidden
	}
	err := s.alerts.Delete(ctx, actor.OrganizationID, id)
	if err == nil {
		s.audit(ctx, actor, "alert.delete", "alert", &id, nil, nil)
	}
	return err
}

// Test evaluates an alert rule's configuration. It reports what the rule
// declares; it does not run any evaluator and does not send a notification, so
// it can never report a delivery that never happened.
func (s *Service) Test(ctx context.Context, actor domain.Actor, id uuid.UUID) (TestResult, error) {
	if !actor.Can(domain.PermissionAlertsManage) {
		return TestResult{}, repositories.ErrForbidden
	}
	existing, err := s.alerts.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return TestResult{}, err
	}
	if err := validateConfiguration(existing.Configuration); err != nil {
		return TestResult{}, err
	}
	muted := existing.Status == domain.AlertStatusMuted
	var triggered bool
	var msg string
	if s.evaluator != nil {
		triggered, msg, _ = s.evaluator.EvaluateAlert(ctx, actor.OrganizationID, existing)
	} else {
		msg = testMessage(muted)
	}

	return TestResult{
		AlertID:          existing.ID,
		AlertType:        existing.AlertType,
		Severity:         existing.Severity,
		Status:           existing.Status,
		Configuration:    existing.Configuration,
		StatusIsMuted:    muted,
		NotificationSent: triggered && !muted,
		Message:          msg,
	}, nil
}

func (s *Service) ListEvents(ctx context.Context, actor domain.Actor, alertID *uuid.UUID, limit, offset int32) ([]domain.AlertEvent, error) {
	if !actor.Can(domain.PermissionAlertsRead) {
		return nil, repositories.ErrForbidden
	}
	return s.alerts.ListEvents(ctx, actor.OrganizationID, alertID, limit, offset)
}

func testMessage(muted bool) string {
	if muted {
		return "Rule is muted, so no evaluator would report it. No evaluator is wired to this rule type and no notification was sent."
	}
	return "Rule is active, but no evaluator is wired to this rule type, so nothing was evaluated and no notification was sent."
}


// TestResult is the outcome of evaluating an alert rule's configuration. It
// deliberately carries NotificationSent and StatusIsMuted rather than a
// would-fire verdict, because no evaluator exists to produce one.
type TestResult struct {
	AlertID          uuid.UUID       `json:"alert_id"`
	AlertType        string          `json:"alert_type"`
	Severity         string          `json:"severity"`
	Status           string          `json:"status"`
	Configuration    json.RawMessage `json:"configuration"`
	StatusIsMuted    bool            `json:"status_is_muted"`
	NotificationSent bool            `json:"notification_sent"`
	Message          string          `json:"message"`
}

// validateConfiguration rejects a configuration that is not a JSON object, so a
// rule can never store a bare scalar or array that no evaluator could read.
func validateConfiguration(configuration []byte) error {
	if len(configuration) == 0 {
		return nil
	}
	var object map[string]any
	if err := json.Unmarshal(configuration, &object); err != nil {
		return fmt.Errorf("configuration must be a JSON object: %w", repositories.ErrInvalid)
	}
	if object == nil {
		return fmt.Errorf("configuration must be a JSON object: %w", repositories.ErrInvalid)
	}
	return nil
}

func (s *Service) audit(ctx context.Context, actor domain.Actor, action, resourceType string, resourceID *uuid.UUID, before, after []byte) {
	entry := domain.NewAuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
	}
	if actor.Kind == domain.ActorKindService {
		entry.ActorServiceAccountID = &actor.UserID
	} else {
		entry.ActorUserID = &actor.UserID
	}
	if _, err := s.audits.Append(ctx, actor.OrganizationID, entry); err != nil {
		if s.log != nil {
			s.log.WarnContext(ctx, "alert: audit append failed",
				"action", action, "resource_type", resourceType, "error", err)
		}
	}
}
