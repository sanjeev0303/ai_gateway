package securityevent

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// fakeEvents records what was appended so a test can assert on the wire value
// rather than on internal bookkeeping.
type fakeEvents struct {
	appended              []appendedEvent
	appendErr             error
	listed                []domain.SecurityEvent
	listCalls             int
	listOrgID             uuid.UUID
	listFilter            domain.SecurityEventFilter
	listLimit, listOffset int32
}

type appendedEvent struct {
	orgID uuid.UUID
	input domain.NewSecurityEvent
}

func (f *fakeEvents) Append(_ context.Context, orgID uuid.UUID, input domain.NewSecurityEvent) (domain.SecurityEvent, error) {
	if f.appendErr != nil {
		return domain.SecurityEvent{}, f.appendErr
	}
	f.appended = append(f.appended, appendedEvent{orgID: orgID, input: input})
	return domain.SecurityEvent{OrganizationID: orgID, Category: input.Category, Severity: input.Severity}, nil
}

func (f *fakeEvents) List(_ context.Context, orgID uuid.UUID, filter domain.SecurityEventFilter, limit, offset int32) ([]domain.SecurityEvent, error) {
	f.listCalls++
	f.listOrgID = orgID
	f.listFilter = filter
	f.listLimit, f.listOffset = limit, offset
	return f.listed, nil
}

func quietLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestRecordValidation(t *testing.T) {
	valid := domain.NewSecurityEvent{
		Category: domain.SecurityEventPipelineFailure,
		Severity: domain.SecurityEventSeverityWarning,
		Title:    "Usage pipeline failed",
		Source:   "usage_rollup",
	}

	cases := []struct {
		name  string
		orgID uuid.UUID
		input domain.NewSecurityEvent
	}{
		{"missing organization", uuid.Nil, valid},
		{"unknown category", uuid.New(), domain.NewSecurityEvent{
			Category: "made_up", Severity: valid.Severity, Title: valid.Title, Source: valid.Source,
		}},
		{"unknown severity", uuid.New(), domain.NewSecurityEvent{
			Category: valid.Category, Severity: "catastrophic", Title: valid.Title, Source: valid.Source,
		}},
		{"empty title", uuid.New(), domain.NewSecurityEvent{
			Category: valid.Category, Severity: valid.Severity, Source: valid.Source,
		}},
		{"empty source", uuid.New(), domain.NewSecurityEvent{
			Category: valid.Category, Severity: valid.Severity, Title: valid.Title,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := &fakeEvents{}
			svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})

			err := svc.Record(context.Background(), tc.orgID, tc.input)
			if !errors.Is(err, repositories.ErrInvalid) {
				t.Fatalf("got %v, want ErrInvalid", err)
			}
			if len(events.appended) != 0 {
				t.Fatalf("invalid input reached the repository: %+v", events.appended)
			}
		})
	}
}

func TestRecordAppends(t *testing.T) {
	orgID := uuid.New()
	events := &fakeEvents{}
	svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})

	input := domain.NewSecurityEvent{
		Category: domain.SecurityEventPipelineFailure,
		Severity: domain.SecurityEventSeverityWarning,
		Title:    "Usage pipeline failed",
		Source:   "usage_rollup",
	}
	if err := svc.Record(context.Background(), orgID, input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events.appended) != 1 {
		t.Fatalf("expected one append, got %d", len(events.appended))
	}
	got := events.appended[0]
	if got.orgID != orgID {
		t.Fatalf("tenant scope lost: got %v want %v", got.orgID, orgID)
	}
	if got.input.Title != input.Title || got.input.Source != input.Source {
		t.Fatalf("fields mangled: %+v", got.input)
	}
}

func TestRecordProviderTransition(t *testing.T) {
	cases := []struct {
		status         string
		wantSeverity   domain.SecurityEventSeverity
		wantTitleParts string
	}{
		{domain.ProviderHealthUnhealthy, domain.SecurityEventSeverityCritical, "is now unhealthy"},
		{domain.ProviderHealthDegraded, domain.SecurityEventSeverityWarning, "is now degraded"},
		{domain.ProviderHealthHealthy, domain.SecurityEventSeverityInfo, "recovered"},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			events := &fakeEvents{}
			svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})

			state := domain.ProviderHealthState{
				ProviderKind: "anthropic",
				Status:       tc.status,
				StatusReason: "probe timed out",
				LastError:    "dial tcp: i/o timeout",
			}
			if err := svc.RecordProviderTransition(context.Background(), uuid.New(), state, domain.ProviderHealthHealthy); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events.appended) != 1 {
				t.Fatalf("expected one append, got %d", len(events.appended))
			}
			got := events.appended[0].input

			if got.Category != domain.SecurityEventProviderIncident {
				t.Fatalf("category = %q, want provider_incident", got.Category)
			}
			if got.Severity != tc.wantSeverity {
				t.Fatalf("severity = %q, want %q", got.Severity, tc.wantSeverity)
			}
			if got.Source != "provider_health" {
				t.Fatalf("source = %q, want provider_health", got.Source)
			}
			if !contains(got.Title, tc.wantTitleParts) {
				t.Fatalf("title = %q, want it to contain %q", got.Title, tc.wantTitleParts)
			}

			var detail providerTransitionDetail
			if err := json.Unmarshal(got.Detail, &detail); err != nil {
				t.Fatalf("detail is not the recorded shape: %v (%s)", err, got.Detail)
			}
			if detail.ProviderKind != "anthropic" || detail.To != tc.status || detail.From != domain.ProviderHealthHealthy {
				t.Fatalf("transition detail wrong: %+v", detail)
			}
			if detail.StatusReason != "probe timed out" || detail.LastErrorText != "dial tcp: i/o timeout" {
				t.Fatalf("probe evidence dropped: %+v", detail)
			}
		})
	}
}

func TestRecordPipelineFailure(t *testing.T) {
	events := &fakeEvents{}
	svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})

	cause := errors.New("relation \"usage_rollups_hourly\" does not exist")
	if err := svc.RecordPipelineFailure(context.Background(), uuid.New(), "rollup 2026-01-01T00:00:00Z to 2026-01-01T01:00:00Z", cause); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := events.appended[0].input

	if got.Category != domain.SecurityEventPipelineFailure {
		t.Fatalf("category = %q, want pipeline_failure", got.Category)
	}
	if got.Severity != domain.SecurityEventSeverityWarning {
		t.Fatalf("severity = %q, want warning", got.Severity)
	}
	if got.Source != "usage_rollup" {
		t.Fatalf("source = %q, want usage_rollup", got.Source)
	}

	var detail pipelineFailureDetail
	if err := json.Unmarshal(got.Detail, &detail); err != nil {
		t.Fatalf("detail is not the recorded shape: %v (%s)", err, got.Detail)
	}
	if detail.Stage == "" {
		t.Fatal("stage dropped from detail; the failure window is the actionable part")
	}
	if detail.Message != cause.Error() {
		t.Fatalf("cause text = %q, want %q", detail.Message, cause.Error())
	}
}

func TestListRequiresPermissionAndScopesToTenant(t *testing.T) {
	t.Run("member is refused", func(t *testing.T) {
		events := &fakeEvents{}
		svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})
		actor := domain.Actor{Kind: domain.ActorKindUser, OrganizationID: uuid.New(), Role: domain.RoleMember}

		_, err := svc.List(context.Background(), actor, domain.SecurityEventFilter{}, 50, 0)
		if !errors.Is(err, repositories.ErrForbidden) {
			t.Fatalf("got %v, want ErrForbidden", err)
		}
		if events.listCalls != 0 {
			t.Fatal("a refused reader still reached the repository")
		}
	})

	for _, role := range []string{domain.RoleOwner, domain.RoleAdmin, domain.RoleAuditor} {
		t.Run(role+" may read", func(t *testing.T) {
			orgID := uuid.New()
			events := &fakeEvents{listed: []domain.SecurityEvent{{ID: uuid.New(), OrganizationID: orgID}}}
			svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})
			actor := domain.Actor{Kind: domain.ActorKindUser, OrganizationID: orgID, Role: role}

			got, err := svc.List(context.Background(), actor, domain.SecurityEventFilter{}, 25, 50)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 event, got %d", len(got))
			}
			if events.listOrgID != orgID {
				t.Fatalf("read scoped to %v, want the caller's org %v", events.listOrgID, orgID)
			}
			if events.listLimit != 25 || events.listOffset != 50 {
				t.Fatalf("pagination mangled: limit=%d offset=%d", events.listLimit, events.listOffset)
			}
		})
	}
}

func TestListForwardsFilters(t *testing.T) {
	orgID := uuid.New()
	events := &fakeEvents{}
	svc := NewService(ServiceDeps{Events: events, Log: quietLogger()})
	actor := domain.Actor{Kind: domain.ActorKindUser, OrganizationID: orgID, Role: domain.RoleAuditor}

	category := domain.SecurityEventProviderIncident
	severity := domain.SecurityEventSeverityCritical
	filter := domain.SecurityEventFilter{Category: &category, Severity: &severity}

	if _, err := svc.List(context.Background(), actor, filter, 10, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events.listFilter.Category == nil || *events.listFilter.Category != category {
		t.Fatal("category filter not forwarded")
	}
	if events.listFilter.Severity == nil || *events.listFilter.Severity != severity {
		t.Fatal("severity filter not forwarded")
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
