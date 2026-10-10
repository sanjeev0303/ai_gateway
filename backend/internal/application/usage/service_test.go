package usage

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type fakeOrgs struct {
	ids     []uuid.UUID
	listErr error
}

func (f *fakeOrgs) Create(context.Context, domain.NewOrganization) (domain.Organization, error) {
	return domain.Organization{}, errors.New("unused")
}

func (f *fakeOrgs) GetByID(context.Context, uuid.UUID) (domain.Organization, error) {
	return domain.Organization{}, errors.New("unused")
}

func (f *fakeOrgs) GetBySlug(context.Context, string) (domain.Organization, error) {
	return domain.Organization{}, errors.New("unused")
}

func (f *fakeOrgs) Update(context.Context, uuid.UUID, domain.UpdateOrganization) (domain.Organization, error) {
	return domain.Organization{}, errors.New("unused")
}

func (f *fakeOrgs) ListAllIDs(context.Context) ([]uuid.UUID, error) {
	return f.ids, f.listErr
}

type fakeRollups struct {
	calls    []uuid.UUID
	from, to time.Time
	fail     map[uuid.UUID]error
}

func (f *fakeRollups) Recompute(_ context.Context, orgID uuid.UUID, from, to time.Time) error {
	f.calls = append(f.calls, orgID)
	f.from, f.to = from, to
	return f.fail[orgID]
}

func (f *fakeRollups) GetByProvider(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByProvider, error) {
	return nil, nil
}
func (f *fakeRollups) GetByUser(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByUser, error) {
	return nil, nil
}
func (f *fakeRollups) GetByServiceAccount(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	return nil, nil
}
func (f *fakeRollups) GetByTeam(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByTeam, error) {
	return nil, nil
}
func (f *fakeRollups) GetByModel(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByModel, error) {
	return nil, nil
}
func (f *fakeRollups) GetByProject(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByProject, error) {
	return nil, nil
}
func (f *fakeRollups) GetTimeSeries(context.Context, uuid.UUID, time.Time, time.Time, *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	return nil, nil
}
func (f *fakeRollups) GetSummary(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageSummary, error) {
	return domain.UsageSummary{}, nil
}
func (f *fakeRollups) GetLatency(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageLatency, error) {
	return domain.UsageLatency{}, nil
}
func (f *fakeRollups) GetLatencyByProviderModel(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageLatency, error) {
	return nil, nil
}

func TestRecomputeAll(t *testing.T) {
	orgA, orgB, orgC := uuid.New(), uuid.New(), uuid.New()
	from := time.Now().UTC().Add(-time.Hour)
	to := time.Now().UTC()
	quiet := slog.New(slog.DiscardHandler)

	t.Run("rebuilds every organization", func(t *testing.T) {
		rollups := &fakeRollups{}
		svc := NewService(ServiceDeps{Orgs: &fakeOrgs{ids: []uuid.UUID{orgA, orgB}}, Rollups: rollups, Log: quiet})
		if err := svc.RecomputeAll(context.Background(), from, to); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rollups.calls) != 2 {
			t.Fatalf("expected 2 rebuilds, got %d", len(rollups.calls))
		}
		if rollups.calls[0] != orgA || rollups.calls[1] != orgB {
			t.Fatalf("rebuilt %v, want [%v %v]", rollups.calls, orgA, orgB)
		}
		if !rollups.from.Equal(from) || !rollups.to.Equal(to) {
			t.Fatalf("window mangled: got [%v %v] want [%v %v]", rollups.from, rollups.to, from, to)
		}
	})

	t.Run("a failing organization does not starve the rest", func(t *testing.T) {
		boom := errors.New("boom")
		rollups := &fakeRollups{fail: map[uuid.UUID]error{orgB: boom}}
		svc := NewService(ServiceDeps{Orgs: &fakeOrgs{ids: []uuid.UUID{orgA, orgB, orgC}}, Rollups: rollups, Log: quiet})

		err := svc.RecomputeAll(context.Background(), from, to)
		if !errors.Is(err, boom) {
			t.Fatalf("error does not wrap the cause: %v", err)
		}
		if len(rollups.calls) != 3 {
			t.Fatalf("expected all 3 attempted, got %d", len(rollups.calls))
		}
		if rollups.calls[2] != orgC {
			t.Fatalf("did not continue past the failure, last was %v", rollups.calls[2])
		}
	})

	t.Run("enumeration failure rebuilds nothing", func(t *testing.T) {
		boom := errors.New("no db")
		rollups := &fakeRollups{}
		svc := NewService(ServiceDeps{Orgs: &fakeOrgs{listErr: boom}, Rollups: rollups, Log: quiet})

		if err := svc.RecomputeAll(context.Background(), from, to); !errors.Is(err, boom) {
			t.Fatalf("got %v, want the enumeration failure", err)
		}
		if len(rollups.calls) != 0 {
			t.Fatalf("rebuilt %d orgs without a tenant list", len(rollups.calls))
		}
	})
}

type fakeLlmRequests struct {
	err     error
	calls   int
	orgIDs  []uuid.UUID
	records []domain.NewLlmRequest
}

func (f *fakeLlmRequests) Record(_ context.Context, orgID uuid.UUID, input domain.NewLlmRequest) (domain.LlmRequest, error) {
	f.calls++
	f.orgIDs = append(f.orgIDs, orgID)
	if f.err != nil {
		return domain.LlmRequest{}, f.err
	}
	f.records = append(f.records, input)
	return domain.LlmRequest{RequestID: input.RequestID}, nil
}

func (f *fakeLlmRequests) GetByRequestID(context.Context, uuid.UUID, string) (domain.LlmRequest, error) {
	return domain.LlmRequest{}, errors.New("unused")
}

func (f *fakeLlmRequests) List(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.LlmRequest, error) {
	return nil, nil
}

func (f *fakeLlmRequests) GetRecent(context.Context, uuid.UUID) ([]domain.LlmRequest, error) {
	return nil, nil
}
func (f *fakeLlmRequests) GetRequests(context.Context, uuid.UUID, domain.UsageFilter, int32, int32) ([]domain.LlmRequest, error) {
	return nil, nil
}

type fakeUsageEvents struct {
	err    error
	calls  int
	orgIDs []uuid.UUID
	events []domain.NewUsageEvent
}

func (f *fakeUsageEvents) Record(_ context.Context, orgID uuid.UUID, input domain.NewUsageEvent) (domain.UsageEvent, error) {
	f.calls++
	f.orgIDs = append(f.orgIDs, orgID)
	if f.err != nil {
		return domain.UsageEvent{}, f.err
	}
	f.events = append(f.events, input)
	return domain.UsageEvent{RequestID: input.RequestID, EventType: input.EventType}, nil
}

func (f *fakeUsageEvents) Get(context.Context, uuid.UUID, string, string) (domain.UsageEvent, error) {
	return domain.UsageEvent{}, errors.New("unused")
}

func (f *fakeUsageEvents) ListByRequest(context.Context, uuid.UUID, string) ([]domain.UsageEvent, error) {
	return nil, nil
}

func TestRecordRequest(t *testing.T) {
	orgID := uuid.New()
	req := domain.NewLlmRequest{RequestID: "req-1"}
	ev := domain.NewUsageEvent{RequestID: "req-1", EventType: domain.UsageEventTypeTokenUsage}
	quiet := slog.New(slog.DiscardHandler)

	t.Run("writes both ledgers scoped to the tenant", func(t *testing.T) {
		llm := &fakeLlmRequests{}
		events := &fakeUsageEvents{}
		svc := NewService(ServiceDeps{LlmRequests: llm, UsageEvents: events, Log: quiet})

		if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if llm.calls != 1 || events.calls != 1 {
			t.Fatalf("want one write each, got %d/%d", llm.calls, events.calls)
		}
		if llm.orgIDs[0] != orgID || events.orgIDs[0] != orgID {
			t.Fatalf("tenant scope lost: %v / %v", llm.orgIDs, events.orgIDs)
		}
		if llm.records[0].RequestID != "req-1" || events.events[0].RequestID != "req-1" {
			t.Fatal("request id lost in transit")
		}
	})

	t.Run("redelivery is absorbed, not failed", func(t *testing.T) {
		llm := &fakeLlmRequests{err: repositories.ErrConflict}
		events := &fakeUsageEvents{err: repositories.ErrConflict}
		svc := NewService(ServiceDeps{LlmRequests: llm, UsageEvents: events, Log: quiet})

		if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
			t.Fatalf("duplicate delivery must be success, got %v", err)
		}
	})

	t.Run("a request conflict does not skip the usage event", func(t *testing.T) {
		llm := &fakeLlmRequests{err: repositories.ErrConflict}
		events := &fakeUsageEvents{}
		svc := NewService(ServiceDeps{LlmRequests: llm, UsageEvents: events, Log: quiet})

		if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if events.calls != 1 {
			t.Fatalf("event write skipped after request conflict, got %d", events.calls)
		}
	})

	t.Run("a real request failure surfaces and stops the event write", func(t *testing.T) {
		boom := errors.New("boom")
		llm := &fakeLlmRequests{err: boom}
		events := &fakeUsageEvents{}
		svc := NewService(ServiceDeps{LlmRequests: llm, UsageEvents: events, Log: quiet})

		if err := svc.RecordRequest(context.Background(), orgID, req, ev); !errors.Is(err, boom) {
			t.Fatalf("error does not wrap the cause: %v", err)
		}
		if events.calls != 0 {
			t.Fatalf("usage event written after the request failed, got %d", events.calls)
		}
	})

	t.Run("a real event failure surfaces", func(t *testing.T) {
		boom := errors.New("boom")
		llm := &fakeLlmRequests{}
		events := &fakeUsageEvents{err: boom}
		svc := NewService(ServiceDeps{LlmRequests: llm, UsageEvents: events, Log: quiet})

		if err := svc.RecordRequest(context.Background(), orgID, req, ev); !errors.Is(err, boom) {
			t.Fatalf("got %v, want the event failure", err)
		}
	})
}

type fakeSecurityEvents struct {
	failures   []recordedFailure
	failureErr error
}

type recordedFailure struct {
	orgID uuid.UUID
	stage string
	cause error
}

func (f *fakeSecurityEvents) RecordProviderTransition(context.Context, uuid.UUID, domain.ProviderHealthState, string) error {
	return nil
}

func (f *fakeSecurityEvents) RecordPipelineFailure(_ context.Context, orgID uuid.UUID, stage string, cause error) error {
	f.failures = append(f.failures, recordedFailure{orgID: orgID, stage: stage, cause: cause})
	return f.failureErr
}

func TestRecomputeAllCapturesPipelineFailures(t *testing.T) {
	orgA, orgB := uuid.New(), uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	quiet := slog.New(slog.DiscardHandler)
	boom := errors.New("relation does not exist")

	t.Run("a failing organization is captured with its window", func(t *testing.T) {
		events := &fakeSecurityEvents{}
		svc := NewService(ServiceDeps{
			Orgs:           &fakeOrgs{ids: []uuid.UUID{orgA, orgB}},
			Rollups:        &fakeRollups{fail: map[uuid.UUID]error{orgB: boom}},
			SecurityEvents: events,
			Log:            quiet,
		})

		if err := svc.RecomputeAll(context.Background(), from, to); !errors.Is(err, boom) {
			t.Fatalf("rollup failure stopped surfacing: %v", err)
		}
		if len(events.failures) != 1 {
			t.Fatalf("captured %d failures, want 1 (only the failing org)", len(events.failures))
		}
		got := events.failures[0]
		if got.orgID != orgB {
			t.Fatalf("failure attributed to %v, want the failing org %v", got.orgID, orgB)
		}
		if !errors.Is(got.cause, boom) {
			t.Fatalf("cause lost: %v", got.cause)
		}
		// The window is what an operator needs to reproduce the failure.
		wantStage := "rollup 2026-01-01T00:00:00Z to 2026-01-01T01:00:00Z"
		if got.stage != wantStage {
			t.Fatalf("stage = %q, want %q", got.stage, wantStage)
		}
	})

	t.Run("healthy organizations capture nothing", func(t *testing.T) {
		events := &fakeSecurityEvents{}
		svc := NewService(ServiceDeps{
			Orgs:           &fakeOrgs{ids: []uuid.UUID{orgA, orgB}},
			Rollups:        &fakeRollups{},
			SecurityEvents: events,
			Log:            quiet,
		})

		if err := svc.RecomputeAll(context.Background(), from, to); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(events.failures) != 0 {
			t.Fatalf("a clean run recorded %d failures", len(events.failures))
		}
	})

	t.Run("recorder failure does not change the returned error", func(t *testing.T) {
		events := &fakeSecurityEvents{failureErr: errors.New("event ledger down")}
		svc := NewService(ServiceDeps{
			Orgs:           &fakeOrgs{ids: []uuid.UUID{orgA}},
			Rollups:        &fakeRollups{fail: map[uuid.UUID]error{orgA: boom}},
			SecurityEvents: events,
			Log:            quiet,
		})

		err := svc.RecomputeAll(context.Background(), from, to)
		if !errors.Is(err, boom) {
			t.Fatalf("got %v, want the rollup failure, not the recorder failure", err)
		}
	})

	t.Run("enumeration failure captures nothing", func(t *testing.T) {
		events := &fakeSecurityEvents{}
		svc := NewService(ServiceDeps{
			Orgs:           &fakeOrgs{listErr: errors.New("no db")},
			Rollups:        &fakeRollups{},
			SecurityEvents: events,
			Log:            quiet,
		})

		if err := svc.RecomputeAll(context.Background(), from, to); err == nil {
			t.Fatal("expected the enumeration failure")
		}
		if len(events.failures) != 0 {
			t.Fatalf("recorded %d failures without knowing any tenant", len(events.failures))
		}
	})
}
