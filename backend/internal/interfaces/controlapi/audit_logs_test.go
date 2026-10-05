package controlapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/auditlog"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// fakeAuditRepo returns canned records and records the tenant and filter it was
// handed, so tests can assert what the handler parsed and scoped.
type fakeAuditRepo struct {
	records []domain.AuditLog
	record  domain.AuditLog
	getErr  error

	gotOrgID  uuid.UUID
	gotID     uuid.UUID
	gotList   bool
	gotFilter domain.AuditLogFilter
	gotLimit  int32
	gotOffset int32
}

func (f *fakeAuditRepo) Append(context.Context, uuid.UUID, domain.NewAuditLog) (domain.AuditLog, error) {
	return domain.AuditLog{}, nil
}

func (f *fakeAuditRepo) List(_ context.Context, orgID uuid.UUID, filter domain.AuditLogFilter, limit, offset int32) ([]domain.AuditLog, error) {
	f.gotList, f.gotOrgID, f.gotFilter, f.gotLimit, f.gotOffset = true, orgID, filter, limit, offset
	return f.records, nil
}

func (f *fakeAuditRepo) Get(_ context.Context, orgID, id uuid.UUID) (domain.AuditLog, error) {
	f.gotOrgID, f.gotID = orgID, id
	if f.getErr != nil {
		return domain.AuditLog{}, f.getErr
	}
	return f.record, nil
}

func newAuditServer(t *testing.T, st *controlStubs, repo *fakeAuditRepo) *httptest.Server {
	t.Helper()
	return buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		AuditLogs:       auditlog.NewService(auditlog.ServiceDeps{Audits: repo}),
		DevLoginEnabled: true,
	})
}

// getAudit performs an authenticated GET as role and returns the response with
// its body already drained.
func getAudit(t *testing.T, st *controlStubs, repo *fakeAuditRepo, role, path string) (*http.Response, string) {
	t.Helper()
	srv := newAuditServer(t, st, repo)
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

func TestListAuditLogsParsesFilters(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{}

	getAudit(t, st, repo, "owner", "/v1/audit-logs?actor_id="+testUserID.String()+
		"&actor_type=user&action=organization.update&resource_type=organization"+
		"&resource_id="+testProjID.String()+
		"&from=2026-09-01T00:00:00Z&to=2026-09-30T00:00:00Z&limit=10&offset=5")

	if repo.gotFilter.ActorID == nil || *repo.gotFilter.ActorID != testUserID {
		t.Error("actor_id not parsed")
	}
	if repo.gotFilter.ActorType == nil || *repo.gotFilter.ActorType != "user" {
		t.Error("actor_type not parsed")
	}
	if repo.gotFilter.Action == nil || *repo.gotFilter.Action != "organization.update" {
		t.Error("action not parsed")
	}
	if repo.gotFilter.ResourceType == nil || *repo.gotFilter.ResourceType != "organization" {
		t.Error("resource_type not parsed")
	}
	if repo.gotFilter.ResourceID == nil || *repo.gotFilter.ResourceID != testProjID {
		t.Error("resource_id not parsed")
	}
	if repo.gotFilter.From.IsZero() || repo.gotFilter.To.IsZero() {
		t.Error("from/to not parsed")
	}
	if repo.gotLimit != 10 || repo.gotOffset != 5 {
		t.Errorf("pagination = (%d, %d), want (10, 5)", repo.gotLimit, repo.gotOffset)
	}
}

func TestListAuditLogsDefaultsPagination(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{}
	getAudit(t, st, repo, "owner", "/v1/audit-logs")
	if repo.gotLimit != 50 || repo.gotOffset != 0 {
		t.Errorf("default pagination = (%d, %d), want (50, 0)", repo.gotLimit, repo.gotOffset)
	}
}

// TestListAuditLogsScopesToCallerOrg proves the ledger is read through the
// caller's tenant, never a client-supplied organization id.
func TestListAuditLogsScopesToCallerOrg(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{}
	getAudit(t, st, repo, "owner", "/v1/audit-logs")
	if !repo.gotList {
		t.Fatal("repository List was not called")
	}
	if repo.gotOrgID != testOrgID {
		t.Errorf("org = %s, want %s", repo.gotOrgID, testOrgID)
	}
}

func TestListAuditLogsBadQueryParams(t *testing.T) {
	for _, tc := range []struct{ name, qs string }{
		{"limit zero", "limit=0"},
		{"limit negative", "limit=-1"},
		{"limit too large", "limit=201"},
		{"offset negative", "offset=-1"},
		{"bad limit", "limit=abc"},
		{"bad actor_id", "actor_id=not-a-uuid"},
		{"bad resource_id", "resource_id=not-a-uuid"},
		{"bad actor_type", "actor_type=robot"},
		{"bad from", "from=yesterday"},
		{"bad to", "to=yesterday"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newControlStubs()
			resp, _ := getAudit(t, st, &fakeAuditRepo{}, "owner", "/v1/audit-logs?"+tc.qs)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

// TestAuditLogsRequirePermission proves the ledger is gated on audit.read: a
// member lacks it and is refused before the repository is consulted.
func TestAuditLogsRequirePermission(t *testing.T) {
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
			repo := &fakeAuditRepo{}
			resp, _ := getAudit(t, st, repo, tc.role, "/v1/audit-logs")
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusForbidden && repo.gotList {
				t.Error("repository was consulted despite the permission denial")
			}
		})
	}
}

func TestGetAuditLogInvalidPathUUID(t *testing.T) {
	st := newControlStubs()
	resp, _ := getAudit(t, st, &fakeAuditRepo{}, "owner", "/v1/audit-logs/not-a-uuid")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestGetAuditLogMissingIs404 proves a record the caller cannot see is reported
// as not found rather than leaking its existence.
func TestGetAuditLogMissingIs404(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{getErr: repositories.ErrNotFound}
	resp, _ := getAudit(t, st, repo, "owner", "/v1/audit-logs/"+uuid.New().String())
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetAuditLogScopesToCallerOrg(t *testing.T) {
	st := newControlStubs()
	id := uuid.New()
	repo := &fakeAuditRepo{record: domain.AuditLog{ID: id, Action: "organization.update", CreatedAt: testNow}}
	getAudit(t, st, repo, "owner", "/v1/audit-logs/"+id.String())
	if repo.gotOrgID != testOrgID {
		t.Errorf("org = %s, want %s", repo.gotOrgID, testOrgID)
	}
	if repo.gotID != id {
		t.Errorf("id = %s, want %s", repo.gotID, id)
	}
}

func TestAuditLogDTOActorType(t *testing.T) {
	saID := uuid.New()
	for _, tc := range []struct {
		name   string
		record domain.AuditLog
		want   string
	}{
		{
			name:   "user actor",
			record: domain.AuditLog{ActorUserID: &testUserID},
			want:   `"actor_type":"user"`,
		},
		{
			name:   "service account actor",
			record: domain.AuditLog{ActorServiceAccountID: &saID},
			want:   `"actor_type":"service_account"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeAuditRepo{record: tc.record}
			_, body := getAudit(t, st, repo, "owner", "/v1/audit-logs/"+uuid.New().String())
			if !strings.Contains(body, tc.want) {
				t.Errorf("body = %s, want it to contain %s", body, tc.want)
			}
		})
	}
}

// TestAuditLogDTOEmitsJSONObjects proves before/after reach the client as JSON
// objects rather than base64 or string blobs.
func TestAuditLogDTOEmitsJSONObjects(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{record: domain.AuditLog{
		Action: "organization.update",
		Before: []byte(`{"name":"Old"}`),
		After:  []byte(`{"name":"New"}`),
	}}
	_, body := getAudit(t, st, repo, "owner", "/v1/audit-logs/"+uuid.New().String())
	for _, want := range []string{`"before":{"name":"Old"}`, `"after":{"name":"New"}`} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want it to contain %s", body, want)
		}
	}
}

// TestAuditLogDTOFallsBackToObject proves one unparseable column cannot fail the
// whole response.
func TestAuditLogDTOFallsBackToObject(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{record: domain.AuditLog{
		Action: "organization.update",
		Before: []byte("not json"),
		After:  nil,
	}}
	resp, body := getAudit(t, st, repo, "owner", "/v1/audit-logs/"+uuid.New().String())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, `"before":{}`) || !strings.Contains(body, `"after":{}`) {
		t.Errorf("body = %s, want empty objects for unparseable snapshots", body)
	}
}

func TestListAuditLogsReturnsItemsEnvelope(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAuditRepo{records: []domain.AuditLog{
		{ID: uuid.New(), ActorUserID: &testUserID, Action: "organization.update", CreatedAt: testNow},
		{ID: uuid.New(), Action: "credential.rotate", CreatedAt: testNow},
	}}
	_, body := getAudit(t, st, repo, "owner", "/v1/audit-logs")
	for _, want := range []string{`"items"`, "organization.update", "credential.rotate"} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want it to contain %s", body, want)
		}
	}
}

// TestListAuditLogsEmptyIsItemsNotNull keeps the payload shape stable for
// clients: no records still yields an items array.
func TestListAuditLogsEmptyIsItemsNotNull(t *testing.T) {
	st := newControlStubs()
	_, body := getAudit(t, st, &fakeAuditRepo{}, "owner", "/v1/audit-logs")
	if !strings.Contains(body, `"items":[]`) {
		t.Errorf("body = %s, want an empty items array", body)
	}
}
