package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

type ServiceDeps struct {
	Alerts repositories.AlertRepository
}

// Service manages an organization's configured alert rules.
type Service struct {
	alerts repositories.AlertRepository
}

func NewService(deps ServiceDeps) *Service {
	return &Service{alerts: deps.Alerts}
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
	return s.alerts.Update(ctx, actor.OrganizationID, id, input)
}

func (s *Service) Delete(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	if !actor.Can(domain.PermissionAlertsManage) {
		return repositories.ErrForbidden
	}
	return s.alerts.Delete(ctx, actor.OrganizationID, id)
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
	return TestResult{
		AlertID:          existing.ID,
		AlertType:        existing.AlertType,
		Severity:         existing.Severity,
		Status:           existing.Status,
		Configuration:    existing.Configuration,
		StatusIsMuted:    muted,
		NotificationSent: false,
		Message:          testMessage(muted),
	}, nil
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
