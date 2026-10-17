package alert_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/application/alert"
	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

type mockAlertRepo struct {
	repositories.AlertRepository
	alerts        map[uuid.UUID]domain.Alert
	events        []domain.AlertEvent
	lastStatus    string
	triggeredID   uuid.UUID
}

func newMockAlertRepo() *mockAlertRepo {
	return &mockAlertRepo{
		alerts: make(map[uuid.UUID]domain.Alert),
	}
}

func (m *mockAlertRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Alert, error) {
	var list []domain.Alert
	for _, a := range m.alerts {
		if a.OrganizationID == orgID {
			list = append(list, a)
		}
	}
	return list, nil
}

func (m *mockAlertRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Alert, error) {
	if a, ok := m.alerts[id]; ok && a.OrganizationID == orgID {
		return a, nil
	}
	return domain.Alert{}, repositories.ErrNotFound
}

func (m *mockAlertRepo) UpdateTriggered(ctx context.Context, orgID, id uuid.UUID, status string) (domain.Alert, error) {
	a := m.alerts[id]
	a.Status = status
	now := time.Now().UTC()
	a.LastTriggeredAt = &now
	m.alerts[id] = a
	m.lastStatus = status
	m.triggeredID = id
	return a, nil
}

func (m *mockAlertRepo) CreateEvent(ctx context.Context, orgID uuid.UUID, input domain.NewAlertEvent) (domain.AlertEvent, error) {
	event := domain.AlertEvent{
		ID:             uuid.New(),
		OrganizationID: orgID,
		AlertID:        input.AlertID,
		AlertType:      input.AlertType,
		Severity:       input.Severity,
		Status:         input.Status,
		Message:        input.Message,
		Context:        input.Context,
		TriggeredAt:    input.TriggeredAt,
		ResolvedAt:     input.ResolvedAt,
		CreatedAt:      time.Now().UTC(),
	}
	m.events = append(m.events, event)
	return event, nil
}

func (m *mockAlertRepo) GetLatestEvent(ctx context.Context, orgID, alertID uuid.UUID) (*domain.AlertEvent, error) {
	for i := len(m.events) - 1; i >= 0; i-- {
		if m.events[i].AlertID == alertID && m.events[i].OrganizationID == orgID {
			return &m.events[i], nil
		}
	}
	return nil, nil
}

type mockNotifier struct {
	sent []alert.NotificationEvent
}

func (m *mockNotifier) Send(ctx context.Context, orgID uuid.UUID, event alert.NotificationEvent) error {
	m.sent = append(m.sent, event)
	return nil
}

type mockRollups struct {
	repositories.UsageRollupRepository
	summary domain.UsageSummary
}

func (m *mockRollups) GetSummary(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) (domain.UsageSummary, error) {
	return m.summary, nil
}

type mockHealthRepo struct {
	repositories.ProviderHealthRepository
	states map[string]domain.ProviderHealthState
}

func (m *mockHealthRepo) Get(ctx context.Context, kind string) (domain.ProviderHealthState, error) {
	if s, ok := m.states[kind]; ok {
		return s, nil
	}
	return domain.ProviderHealthState{ProviderKind: kind, Status: domain.ProviderHealthHealthy}, nil
}

func (m *mockHealthRepo) List(ctx context.Context) ([]domain.ProviderHealthState, error) {
	var list []domain.ProviderHealthState
	for _, s := range m.states {
		list = append(list, s)
	}
	return list, nil
}

type mockSecurityEvents struct {
	repositories.SecurityEventRepository
	events []domain.SecurityEvent
}

func (m *mockSecurityEvents) List(ctx context.Context, orgID uuid.UUID, filter domain.SecurityEventFilter, limit, offset int32) ([]domain.SecurityEvent, error) {
	return m.events, nil
}

type mockLlmRequests struct {
	repositories.LlmRequestRepository
	requests []domain.LlmRequest
}

func (m *mockLlmRequests) GetRequests(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter, limit, offset int32) ([]domain.LlmRequest, error) {
	return m.requests, nil
}

func TestAlertEvaluator_AllConditions(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	tests := []struct {
		name          string
		alertType     string
		config        map[string]any
		setup         func(*mockRollups, *mockHealthRepo, *mockSecurityEvents, *mockLlmRequests)
		wantTrigger   bool
	}{
		{
			name:      "spend threshold triggered",
			alertType: "spend_limit",
			config:    map[string]any{"threshold_micros": int64(100_000_000), "window": "daily"},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				r.summary = domain.UsageSummary{TotalCostMicros: 150_000_000, Requests: 50}
			},
			wantTrigger: true,
		},
		{
			name:      "spend threshold not triggered",
			alertType: "spend_limit",
			config:    map[string]any{"threshold_micros": int64(100_000_000), "window": "daily"},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				r.summary = domain.UsageSummary{TotalCostMicros: 50_000_000, Requests: 20}
			},
			wantTrigger: false,
		},
		{
			name:      "provider failure triggered",
			alertType: "provider_failure",
			config:    map[string]any{"provider": "anthropic", "consecutive_failures": 2},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				h.states = map[string]domain.ProviderHealthState{
					"anthropic": {ProviderKind: "anthropic", Status: "unhealthy", LastError: "connection refused"},
				}
			},
			wantTrigger: true,
		},
		{
			name:      "provider failure not triggered",
			alertType: "provider_failure",
			config:    map[string]any{"provider": "openai", "consecutive_failures": 3},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				h.states = map[string]domain.ProviderHealthState{
					"openai": {ProviderKind: "openai", Status: "healthy"},
				}
			},
			wantTrigger: false,
		},
		{
			name:      "policy violation triggered",
			alertType: "policy_violation",
			config:    map[string]any{"threshold": 2},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				s.events = []domain.SecurityEvent{
					{ID: uuid.New(), Category: domain.SecurityEventPolicyViolation},
					{ID: uuid.New(), Category: domain.SecurityEventPolicyViolation},
				}
			},
			wantTrigger: true,
		},
		{
			name:      "policy violation not triggered",
			alertType: "policy_violation",
			config:    map[string]any{"threshold": 5},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				s.events = []domain.SecurityEvent{
					{ID: uuid.New(), Category: domain.SecurityEventPolicyViolation},
				}
			},
			wantTrigger: false,
		},
		{
			name:      "rate limit spike triggered",
			alertType: "rate_limit_spike",
			config:    map[string]any{"threshold": 2},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				errCode := "429"
				l.requests = []domain.LlmRequest{
					{ID: uuid.New(), ErrorCode: &errCode},
					{ID: uuid.New(), ErrorCode: &errCode},
				}
			},
			wantTrigger: true,
		},
		{
			name:      "rate limit spike not triggered",
			alertType: "rate_limit_spike",
			config:    map[string]any{"threshold": 5},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				errCode := "429"
				l.requests = []domain.LlmRequest{
					{ID: uuid.New(), ErrorCode: &errCode},
				}
			},
			wantTrigger: false,
		},
		{
			name:      "credential failure triggered",
			alertType: "credential_failure",
			config:    map[string]any{"threshold": 2},
			setup: func(r *mockRollups, h *mockHealthRepo, s *mockSecurityEvents, l *mockLlmRequests) {
				errCode := "invalid_api_key"
				l.requests = []domain.LlmRequest{
					{ID: uuid.New(), ErrorCode: &errCode},
					{ID: uuid.New(), ErrorCode: &errCode},
				}
			},
			wantTrigger: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			alertRepo := newMockAlertRepo()
			notifier := &mockNotifier{}
			rollups := &mockRollups{}
			healthRepo := &mockHealthRepo{states: make(map[string]domain.ProviderHealthState)}
			securityRepo := &mockSecurityEvents{}
			requestsRepo := &mockLlmRequests{}

			if tc.setup != nil {
				tc.setup(rollups, healthRepo, securityRepo, requestsRepo)
			}

			eval := alert.NewEvaluator(alert.EvaluatorDeps{
				Alerts:         alertRepo,
				Notifier:       notifier,
				UsageRollups:   rollups,
				ProviderHealth: healthRepo,
				SecurityEvents: securityRepo,
				LlmRequests:    requestsRepo,
			})

			cfgBytes, _ := json.Marshal(tc.config)
			a := domain.Alert{
				ID:             uuid.New(),
				OrganizationID: orgID,
				AlertType:      tc.alertType,
				Name:           tc.name,
				Severity:       "warning",
				Configuration:  cfgBytes,
				Status:         domain.AlertStatusActive,
			}
			alertRepo.alerts[a.ID] = a

			triggered, msg, _ := eval.EvaluateAlert(ctx, orgID, a)
			if triggered != tc.wantTrigger {
				t.Fatalf("EvaluateAlert = %v (msg: %s), want %v", triggered, msg, tc.wantTrigger)
			}

			// Run EvaluateAll to test persistence, debouncing, and state update
			eval.EvaluateAll(ctx, []uuid.UUID{orgID})
			if tc.wantTrigger {
				if len(alertRepo.events) == 0 {
					t.Fatalf("expected trigger event persisted in alert history")
				}
				if len(notifier.sent) == 0 {
					t.Fatalf("expected notification dispatched")
				}

				// Debounce test: second call immediately within cooldown window should not double-send
				prevEvents := len(alertRepo.events)
				eval.EvaluateAll(ctx, []uuid.UUID{orgID})
				if len(alertRepo.events) != prevEvents {
					t.Fatalf("debounce failed: duplicate event created within cooldown window")
				}
			}
		})
	}
}

func TestAlertEvaluator_AutoRecoveryAndResolution(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	alertRepo := newMockAlertRepo()
	notifier := &mockNotifier{}
	rollups := &mockRollups{}
	healthRepo := &mockHealthRepo{states: make(map[string]domain.ProviderHealthState)}
	securityRepo := &mockSecurityEvents{}
	requestsRepo := &mockLlmRequests{}

	eval := alert.NewEvaluator(alert.EvaluatorDeps{
		Alerts:         alertRepo,
		Notifier:       notifier,
		UsageRollups:   rollups,
		ProviderHealth: healthRepo,
		SecurityEvents: securityRepo,
		LlmRequests:    requestsRepo,
	})

	cfgBytes, _ := json.Marshal(map[string]any{"threshold_micros": int64(100_000_000), "window": "daily"})
	a := domain.Alert{
		ID:             uuid.New(),
		OrganizationID: orgID,
		AlertType:      "spend_limit",
		Name:           "Daily Spend Cap",
		Severity:       "critical",
		Configuration:  cfgBytes,
		Status:         domain.AlertStatusActive,
	}
	alertRepo.alerts[a.ID] = a

	// Step 1: Trigger condition (spend = $150 > $100 threshold)
	rollups.summary = domain.UsageSummary{TotalCostMicros: 150_000_000, Requests: 100}
	eval.EvaluateAll(ctx, []uuid.UUID{orgID})

	if len(alertRepo.events) != 1 || alertRepo.events[0].Status != "triggered" {
		t.Fatalf("expected triggered event, got %+v", alertRepo.events)
	}

	// Step 2: Spend condition returns to normal (spend = $50 <= $100 threshold)
	rollups.summary = domain.UsageSummary{TotalCostMicros: 50_000_000, Requests: 30}
	eval.EvaluateAll(ctx, []uuid.UUID{orgID})

	// Must transition to resolved and append resolution event
	if len(alertRepo.events) != 2 {
		t.Fatalf("expected 2 total events (triggered + resolved), got %d", len(alertRepo.events))
	}
	resolvedEvent := alertRepo.events[1]
	if resolvedEvent.Status != "resolved" {
		t.Errorf("expected resolved event status, got %s", resolvedEvent.Status)
	}
	if resolvedEvent.ResolvedAt == nil {
		t.Errorf("expected non-nil ResolvedAt timestamp on resolution event")
	}
	if alertRepo.alerts[a.ID].Status != domain.AlertStatusResolved {
		t.Errorf("expected alert status resolved, got %s", alertRepo.alerts[a.ID].Status)
	}
}
