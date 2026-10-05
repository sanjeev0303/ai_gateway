package controlapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/domain"
)

// fakeSecurityEventRepo returns canned events and records the tenant, filter,
// and pagination it was handed, so tests can assert what the handler parsed and
// scoped.
type fakeSecurityEventRepo struct {
	events []domain.SecurityEvent

	gotList   bool
	gotOrgID  uuid.UUID
	gotFilter domain.SecurityEventFilter
	gotLimit  int32
	gotOffset int32
}

func (f *fakeSecurityEventRepo) Append(context.Context, uuid.UUID, domain.NewSecurityEvent) (domain.SecurityEvent, error) {
	return domain.SecurityEvent{}, nil
}

func (f *fakeSecurityEventRepo) List(_ context.Context, orgID uuid.UUID, filter domain.SecurityEventFilter, limit, offset int32) ([]domain.SecurityEvent, error) {
	f.gotList, f.gotOrgID, f.gotFilter, f.gotLimit, f.gotOffset = true, orgID, filter, limit, offset
	return f.events, nil
}

func getSecurityEvents(t *testing.T, st *controlStubs, repo *fakeSecurityEventRepo, role, path string) (*http.Response, string) {
	t.Helper()
	srv := buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		SecurityEvents:  securityevent.NewService(securityevent.ServiceDeps{Events: repo}),
		DevLoginEnabled: true,
	})
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range bearerHeader(st, role) {
		req.Header.Set(k, v)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(raw)
}

func TestListSecurityEventsParsesFilters(t *testing.T) {
	st := newControlStubs()
	repo := &fakeSecurityEventRepo{}

	getSecurityEvents(t, st, repo, "owner", "/v1/security-events"+
		"?category=provider_incident&severity=critical"+
		"&from=2026-09-01T00:00:00Z&to=2026-09-30T00:00:00Z&limit=10&offset=5")

	if repo.gotFilter.Category == nil || *repo.gotFilter.Category != domain.SecurityEventProviderIncident {
		t.Error("category not parsed")
	}
	if repo.gotFilter.Severity == nil || *repo.gotFilter.Severity != domain.SecurityEventSeverityCritical {
		t.Error("severity not parsed")
	}
	if repo.gotFilter.From.IsZero() || repo.gotFilter.To.IsZero() {
		t.Error("from/to not parsed")
	}
	if repo.gotLimit != 10 || repo.gotOffset != 5 {
		t.Errorf("pagination = (%d, %d), want (10, 5)", repo.gotLimit, repo.gotOffset)
	}
}

func TestListSecurityEventsDefaultsPagination(t *testing.T) {
	st := newControlStubs()
	repo := &fakeSecurityEventRepo{}
	getSecurityEvents(t, st, repo, "owner", "/v1/security-events")
	if repo.gotLimit != 50 || repo.gotOffset != 0 {
		t.Errorf("default pagination = (%d, %d), want (50, 0)", repo.gotLimit, repo.gotOffset)
	}
}

// TestListSecurityEventsScopesToCallerOrg proves the feed is read through the
// caller's tenant, never a client-supplied organization id.
func TestListSecurityEventsScopesToCallerOrg(t *testing.T) {
	st := newControlStubs()
	repo := &fakeSecurityEventRepo{}
	getSecurityEvents(t, st, repo, "owner", "/v1/security-events")
	if !repo.gotList {
		t.Fatal("repository List was not called")
	}
	if repo.gotOrgID != testOrgID {
		t.Errorf("org = %s, want %s", repo.gotOrgID, testOrgID)
	}
}

// TestListSecurityEventsRejectsUnknownFilters proves a typo'd filter is reported
// as a bad request instead of looking like a query that matched nothing.
func TestListSecurityEventsRejectsUnknownFilters(t *testing.T) {
	for _, tc := range []struct{ name, qs string }{
		{"limit zero", "limit=0"},
		{"limit negative", "limit=-1"},
		{"limit too large", "limit=201"},
		{"offset negative", "offset=-1"},
		{"bad limit", "limit=abc"},
		{"bad category", "category=made_up"},
		{"bad severity", "severity=catastrophic"},
		{"bad from", "from=yesterday"},
		{"bad to", "to=yesterday"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newControlStubs()
			resp, _ := getSecurityEvents(t, st, &fakeSecurityEventRepo{}, "owner", "/v1/security-events?"+tc.qs)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

// TestSecurityEventsRequirePermission proves the feed is gated: a member and a
// billing admin lack security_events.read and are refused before the repository
// is consulted.
func TestSecurityEventsRequirePermission(t *testing.T) {
	for _, tc := range []struct {
		role string
		want int
	}{
		{"member", http.StatusForbidden},
		{"billing_admin", http.StatusForbidden},
		{"auditor", http.StatusOK},
		{"owner", http.StatusOK},
	} {
		t.Run(tc.role, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeSecurityEventRepo{}
			resp, _ := getSecurityEvents(t, st, repo, tc.role, "/v1/security-events")
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusForbidden && repo.gotList {
				t.Error("repository was consulted despite the permission denial")
			}
		})
	}
}

func TestListSecurityEventsReturnsItemsEnvelope(t *testing.T) {
	st := newControlStubs()
	repo := &fakeSecurityEventRepo{events: []domain.SecurityEvent{
		{
			ID:        uuid.New(),
			Category:  domain.SecurityEventProviderIncident,
			Severity:  domain.SecurityEventSeverityCritical,
			Title:     "Provider anthropic is now unhealthy",
			Detail:    []byte(`{"provider_kind":"anthropic"}`),
			Source:    "provider_health",
			CreatedAt: testNow,
		},
		{
			ID:        uuid.New(),
			Category:  domain.SecurityEventPipelineFailure,
			Severity:  domain.SecurityEventSeverityWarning,
			Title:     "Usage pipeline failed",
			Detail:    []byte(`{"stage":"rollup"}`),
			Source:    "usage_rollup",
			CreatedAt: testNow,
		},
	}}
	_, body := getSecurityEvents(t, st, repo, "owner", "/v1/security-events")
	for _, want := range []string{
		`"items"`,
		`"category":"provider_incident"`,
		`"severity":"critical"`,
		"Provider anthropic is now unhealthy",
		`"source":"usage_rollup"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want it to contain %s", body, want)
		}
	}
}

// TestSecurityEventDTOEmitsDetailObject proves detail reaches the client as a
// JSON object rather than base64.
func TestSecurityEventDTOEmitsDetailObject(t *testing.T) {
	st := newControlStubs()
	repo := &fakeSecurityEventRepo{events: []domain.SecurityEvent{{
		ID:     uuid.New(),
		Detail: []byte(`{"provider_kind":"anthropic","from":"healthy","to":"unhealthy"}`),
	}}}
	_, body := getSecurityEvents(t, st, repo, "owner", "/v1/security-events")
	if !strings.Contains(body, `"detail":{"provider_kind":"anthropic","from":"healthy","to":"unhealthy"}`) {
		t.Errorf("body = %s, want detail as a JSON object", body)
	}
}

// TestSecurityEventDTODetailFallsBackToObject proves one unparseable column
// cannot fail the whole feed.
func TestSecurityEventDTODetailFallsBackToObject(t *testing.T) {
	st := newControlStubs()
	repo := &fakeSecurityEventRepo{events: []domain.SecurityEvent{
		{ID: uuid.New(), Detail: []byte("not json")},
		{ID: uuid.New(), Detail: nil},
	}}
	resp, body := getSecurityEvents(t, st, repo, "owner", "/v1/security-events")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if strings.Count(body, `"detail":{}`) != 2 {
		t.Errorf("body = %s, want both details as empty objects", body)
	}
}

// TestListSecurityEventsEmptyIsItemsNotNull keeps the payload shape stable for
// clients: no events still yields an items array.
func TestListSecurityEventsEmptyIsItemsNotNull(t *testing.T) {
	st := newControlStubs()
	_, body := getSecurityEvents(t, st, &fakeSecurityEventRepo{}, "owner", "/v1/security-events")
	if !strings.Contains(body, `"items":[]`) {
		t.Errorf("body = %s, want an empty items array", body)
	}
}
