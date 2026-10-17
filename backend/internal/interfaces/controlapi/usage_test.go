package controlapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type fakeUsageRollups struct {
	summary domain.UsageSummary
	byProj  []domain.UsageByProject
	byModel []domain.UsageByModel
	byProv  []domain.UsageByProvider
	byUser  []domain.UsageByUser
	bySA    []domain.UsageByServiceAccount
	byTeam  []domain.UsageByTeam
	latency []domain.UsageLatency
	points  []domain.UsageTimeSeriesPoint
}

func (f *fakeUsageRollups) Recompute(context.Context, uuid.UUID, time.Time, time.Time) error {
	return nil
}
func (f *fakeUsageRollups) GetSummary(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageSummary, error) {
	return f.summary, nil
}
func (f *fakeUsageRollups) GetTimeSeries(context.Context, uuid.UUID, time.Time, time.Time, *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	return f.points, nil
}
func (f *fakeUsageRollups) GetByProject(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByProject, error) {
	return f.byProj, nil
}
func (f *fakeUsageRollups) GetByModel(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByModel, error) {
	return f.byModel, nil
}
func (f *fakeUsageRollups) GetByProvider(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.UsageByProvider, error) {
	return f.byProv, nil
}
func (f *fakeUsageRollups) GetByUser(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByUser, error) {
	return f.byUser, nil
}
func (f *fakeUsageRollups) GetByServiceAccount(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	return f.bySA, nil
}
func (f *fakeUsageRollups) GetByTeam(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByTeam, error) {
	return f.byTeam, nil
}
func (f *fakeUsageRollups) GetLatency(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageLatency, error) {
	if len(f.latency) > 0 {
		return f.latency[0], nil
	}
	return domain.UsageLatency{}, nil
}
func (f *fakeUsageRollups) GetLatencyByProviderModel(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageLatency, error) {
	return f.latency, nil
}

type fakeLlmRequests struct {
	recorded []domain.LlmRequest
}

func (f *fakeLlmRequests) Record(_ context.Context, orgID uuid.UUID, in domain.NewLlmRequest) (domain.LlmRequest, error) {
	req := domain.LlmRequest{ID: uuid.New(), OrganizationID: orgID, RequestID: in.RequestID, TotalTokens: in.TotalTokens}
	f.recorded = append(f.recorded, req)
	return req, nil
}
func (f *fakeLlmRequests) GetByRequestID(context.Context, uuid.UUID, string) (domain.LlmRequest, error) {
	return domain.LlmRequest{}, nil
}
func (f *fakeLlmRequests) List(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.LlmRequest, error) {
	return f.recorded, nil
}
func (f *fakeLlmRequests) GetRecent(context.Context, uuid.UUID) ([]domain.LlmRequest, error) {
	return f.recorded, nil
}
func (f *fakeLlmRequests) GetRequests(_ context.Context, _ uuid.UUID, _ domain.UsageFilter, limit, offset int32) ([]domain.LlmRequest, error) {
	start := int(offset)
	if start > len(f.recorded) {
		return []domain.LlmRequest{}, nil
	}
	end := start + int(limit)
	if end > len(f.recorded) {
		end = len(f.recorded)
	}
	return f.recorded[start:end], nil
}
func (f *fakeLlmRequests) PurgeExpired(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}

type fakeUsageEvents struct{}

func (f *fakeUsageEvents) Record(context.Context, uuid.UUID, domain.NewUsageEvent) (domain.UsageEvent, error) {
	return domain.UsageEvent{}, nil
}
func (f *fakeUsageEvents) Get(context.Context, uuid.UUID, string, string) (domain.UsageEvent, error) {
	return domain.UsageEvent{}, nil
}
func (f *fakeUsageEvents) ListByRequest(context.Context, uuid.UUID, string) ([]domain.UsageEvent, error) {
	return nil, nil
}
func (f *fakeUsageEvents) PurgeExpired(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}

func newUsageServer(t *testing.T, st *controlStubs, rollups repositories.UsageRollupRepository, reqs repositories.LlmRequestRepository) *httptest.Server {
	t.Helper()
	return buildServer(t, Deps{
		Identity: identity.NewService(st.deps()),
		Usage: usage.NewService(usage.ServiceDeps{
			Orgs:        st.orgs,
			Rollups:     rollups,
			LlmRequests: reqs,
			UsageEvents: &fakeUsageEvents{},
		}),
		DevLoginEnabled: true,
	})
}

func TestUsageRoutes_RequireAuth(t *testing.T) {
	st := newControlStubs()
	srv := newUsageServer(t, st, &fakeUsageRollups{}, &fakeLlmRequests{})

	endpoints := []string{
		"/v1/usage/summary",
		"/v1/usage/timeseries",
		"/v1/usage/by-project",
		"/v1/usage/by-model",
		"/v1/usage/by-provider",
		"/v1/usage/by-user",
		"/v1/usage/by-service-account",
		"/v1/usage/by-team",
		"/v1/usage/latency",
		"/v1/usage/recent",
		"/v1/usage/requests",
	}

	for _, ep := range endpoints {
		resp := doJSON(t, srv, http.MethodGet, ep, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", ep, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestUsageSummary_Success(t *testing.T) {
	st := newControlStubs()
	rollups := &fakeUsageRollups{
		summary: domain.UsageSummary{
			Requests:        42,
			TotalTokens:     10500,
			TotalCostMicros: 250000,
			ActiveUsers:     3,
		},
	}
	srv := newUsageServer(t, st, rollups, &fakeLlmRequests{})

	resp := doJSON(t, srv, http.MethodGet, "/v1/usage/summary", nil, bearerHeader(st, domain.RoleMember))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var body domain.UsageSummary
	decodeBody(t, resp, &body)
	if body.Requests != 42 || body.TotalTokens != 10500 || body.TotalCostMicros != 250000 {
		t.Fatalf("unexpected summary body: %+v", body)
	}
}

func TestUsageSummary_DateBoundaryValidation(t *testing.T) {
	st := newControlStubs()
	srv := newUsageServer(t, st, &fakeUsageRollups{}, &fakeLlmRequests{})

	// from is after to
	path := "/v1/usage/summary?from=2026-10-17T12:00:00Z&to=2026-10-17T11:00:00Z"
	resp := doJSON(t, srv, http.MethodGet, path, nil, bearerHeader(st, domain.RoleMember))
	status, code := decodeError(t, resp)
	if status != http.StatusBadRequest || code != "invalid_argument" {
		t.Fatalf("got %d/%s, want 400/invalid_argument", status, code)
	}
}

func TestUsageRequests_Pagination(t *testing.T) {
	st := newControlStubs()
	var seeded []domain.LlmRequest
	for i := 1; i <= 25; i++ {
		seeded = append(seeded, domain.LlmRequest{
			ID:          uuid.New(),
			RequestID:   uuid.NewString(),
			TotalTokens: int32(100 * i),
		})
	}
	reqRepo := &fakeLlmRequests{recorded: seeded}
	srv := newUsageServer(t, st, &fakeUsageRollups{}, reqRepo)

	// Fetch limit 10, offset 5
	resp := doJSON(t, srv, http.MethodGet, "/v1/usage/requests?limit=10&offset=5", nil, bearerHeader(st, domain.RoleMember))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	var body struct {
		Items   []domain.LlmRequest `json:"items"`
		HasMore bool                `json:"has_more"`
	}
	decodeBody(t, resp, &body)

	if len(body.Items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(body.Items))
	}
	if !body.HasMore {
		t.Fatal("expected has_more=true when more records exist")
	}
}

func TestUsageEmptyStates_ReturnEmptyArrays(t *testing.T) {
	st := newControlStubs()
	// All slices nil or empty in fake
	rollups := &fakeUsageRollups{
		byProj:  []domain.UsageByProject{},
		byModel: []domain.UsageByModel{},
		byProv:  []domain.UsageByProvider{},
		byUser:  []domain.UsageByUser{},
		byTeam:  []domain.UsageByTeam{},
		bySA:    []domain.UsageByServiceAccount{},
	}
	srv := newUsageServer(t, st, rollups, &fakeLlmRequests{})

	endpoints := []string{
		"/v1/usage/by-project",
		"/v1/usage/by-model",
		"/v1/usage/by-provider",
		"/v1/usage/by-user",
		"/v1/usage/by-service-account",
		"/v1/usage/by-team",
	}

	for _, ep := range endpoints {
		resp := doJSON(t, srv, http.MethodGet, ep, nil, bearerHeader(st, domain.RoleMember))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: got %d, want 200", ep, resp.StatusCode)
		}
		var body struct {
			Items []any `json:"items"`
		}
		decodeBody(t, resp, &body)
		if body.Items == nil {
			t.Fatalf("%s: expected empty array [], got null", ep)
		}
		if len(body.Items) != 0 {
			t.Fatalf("%s: expected 0 items, got %d", ep, len(body.Items))
		}
	}
}
