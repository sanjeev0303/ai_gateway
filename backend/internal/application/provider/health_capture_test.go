package provider

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// stubSecurityEvents captures the transitions a probe produces. Both recorder
// methods are stubbed so the type satisfies securityevent.Recorder.
type stubSecurityEvents struct {
	transitions   []recordedTransition
	failures      []recordedFailure
	transitionErr error
	failureErr    error
}

type recordedTransition struct {
	orgID uuid.UUID
	state domain.ProviderHealthState
	from  string
}

type recordedFailure struct {
	orgID uuid.UUID
	stage string
	cause error
}

func (s *stubSecurityEvents) RecordProviderTransition(_ context.Context, orgID uuid.UUID, state domain.ProviderHealthState, from string) error {
	s.transitions = append(s.transitions, recordedTransition{orgID: orgID, state: state, from: from})
	return s.transitionErr
}

func (s *stubSecurityEvents) RecordPipelineFailure(_ context.Context, orgID uuid.UUID, stage string, cause error) error {
	s.failures = append(s.failures, recordedFailure{orgID: orgID, stage: stage, cause: cause})
	return s.failureErr
}

var _ securityevent.Recorder = (*stubSecurityEvents)(nil)

// newHealthCaptureService wires a Service whose health probe outcome is
// controlled and whose prior state comes from prior, so a test can state
// exactly what the probe sees and what it is transitioning from.
func newHealthCaptureService(recorder securityevent.Recorder, prior *domain.ProviderHealthState, healthErr error) *Service {
	health := &stubProviderHealth{
		get: func(context.Context, string) (domain.ProviderHealthState, error) {
			if prior == nil {
				return domain.ProviderHealthState{}, repositories.ErrNotFound
			}
			return *prior, nil
		},
	}
	return NewService(ServiceDeps{
		Health:         health,
		Secrets:        &stubSecrets{values: map[string]string{}},
		SecurityEvents: recorder,
		Providers: map[string]providerclient.Provider{
			domain.ProviderKindAnthropic: &stubProvider{healthErr: healthErr},
		},
		Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return testNow },
	})
}

func TestCheckHealthRecordsOnlyRealTransitions(t *testing.T) {
	healthy := domain.ProviderHealthState{ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderHealthHealthy}

	t.Run("first observation is not a transition", func(t *testing.T) {
		recorder := &stubSecurityEvents{}
		svc := newHealthCaptureService(recorder, nil, errors.New("connection refused"))

		state, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
		requireNoErr(t, err)
		if state.Status != domain.ProviderHealthUnhealthy {
			t.Fatalf("status = %q, want unhealthy", state.Status)
		}
		if len(recorder.transitions) != 0 {
			t.Fatalf("first observation recorded %d transitions, want 0", len(recorder.transitions))
		}
	})

	t.Run("healthy to unhealthy is recorded", func(t *testing.T) {
		recorder := &stubSecurityEvents{}
		svc := newHealthCaptureService(recorder, &healthy, errors.New("connection refused"))

		_, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
		requireNoErr(t, err)
		if len(recorder.transitions) != 1 {
			t.Fatalf("recorded %d transitions, want 1", len(recorder.transitions))
		}
		got := recorder.transitions[0]
		if got.from != domain.ProviderHealthHealthy || got.state.Status != domain.ProviderHealthUnhealthy {
			t.Fatalf("transition = %s -> %s, want healthy -> unhealthy", got.from, got.state.Status)
		}
		if got.orgID != testOrgID {
			t.Fatalf("event attributed to %v, want the acting org %v", got.orgID, testOrgID)
		}
	})

	t.Run("recovery is recorded so an operator sees the return", func(t *testing.T) {
		unhealthy := domain.ProviderHealthState{ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderHealthUnhealthy}
		recorder := &stubSecurityEvents{}
		svc := newHealthCaptureService(recorder, &unhealthy, nil)

		_, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
		requireNoErr(t, err)
		if len(recorder.transitions) != 1 {
			t.Fatalf("recovery not recorded: got %d transitions", len(recorder.transitions))
		}
		got := recorder.transitions[0]
		if got.from != domain.ProviderHealthUnhealthy || got.state.Status != domain.ProviderHealthHealthy {
			t.Fatalf("transition = %s -> %s, want unhealthy -> healthy", got.from, got.state.Status)
		}
	})

	t.Run("no change records nothing", func(t *testing.T) {
		recorder := &stubSecurityEvents{}
		svc := newHealthCaptureService(recorder, &healthy, nil)

		_, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
		requireNoErr(t, err)
		if len(recorder.transitions) != 0 {
			t.Fatalf("healthy -> healthy recorded %d transitions, want 0", len(recorder.transitions))
		}
	})
}

func TestCheckHealthRecorderFailureDoesNotMaskTheProbe(t *testing.T) {
	healthy := domain.ProviderHealthState{ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderHealthHealthy}
	recorder := &stubSecurityEvents{transitionErr: errors.New("event ledger down")}
	svc := newHealthCaptureService(recorder, &healthy, errors.New("connection refused"))

	state, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
	requireNoErr(t, err)
	if state.Status != domain.ProviderHealthUnhealthy {
		t.Fatalf("status = %q, want the probe's own result unhealthy", state.Status)
	}
}

func TestCheckHealthUnreadablePriorStateIsNotATransition(t *testing.T) {
	recorder := &stubSecurityEvents{}
	health := &stubProviderHealth{
		get: func(context.Context, string) (domain.ProviderHealthState, error) {
			return domain.ProviderHealthState{}, errors.New("connection reset")
		},
	}
	svc := NewService(ServiceDeps{
		Health:         health,
		Secrets:        &stubSecrets{values: map[string]string{}},
		SecurityEvents: recorder,
		Providers: map[string]providerclient.Provider{
			domain.ProviderKindAnthropic: &stubProvider{},
		},
		Log: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return testNow },
	})

	// The probe still returns a usable result; without a trustworthy prior we
	// cannot claim a transition happened.
	_, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
	requireNoErr(t, err)
	if len(recorder.transitions) != 0 {
		t.Fatalf("invented a transition from an unreadable prior state: %+v", recorder.transitions)
	}
}

func TestCheckHealthWithoutRecorderStillWorks(t *testing.T) {
	healthy := domain.ProviderHealthState{ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderHealthHealthy}
	svc := newHealthCaptureService(nil, &healthy, errors.New("connection refused"))

	state, err := svc.CheckHealth(context.Background(), userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-test")
	requireNoErr(t, err)
	if state.Status != domain.ProviderHealthUnhealthy {
		t.Fatalf("status = %q, want unhealthy", state.Status)
	}
}
