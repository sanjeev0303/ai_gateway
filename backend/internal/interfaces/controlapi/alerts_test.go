package controlapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/alert"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// fakeAlertRepo returns canned rules and records the tenant, id, and update
// input it was handed, so tests can assert both scoping and what the handler
// validated before reaching persistence.
type fakeAlertRepo struct {
	alerts []domain.Alert
	alert  domain.Alert
	update domain.AlertUpdate

	// err is returned by Get, Update, and Delete, standing in for a rule that
	// is missing or belongs to another tenant.
	err error

	gotOrgID uuid.UUID
	gotID    uuid.UUID
	gotList  bool
	gotPut   bool
	gotDel   bool
}

func (f *fakeAlertRepo) Create(context.Context, uuid.UUID, domain.NewAlert) (domain.Alert, error) {
	return domain.Alert{}, nil
}

func (f *fakeAlertRepo) List(_ context.Context, orgID uuid.UUID) ([]domain.Alert, error) {
	f.gotList, f.gotOrgID = true, orgID
	return f.alerts, nil
}

func (f *fakeAlertRepo) Get(_ context.Context, orgID, id uuid.UUID) (domain.Alert, error) {
	f.gotOrgID, f.gotID = orgID, id
	if f.err != nil {
		return domain.Alert{}, f.err
	}
	return f.alert, nil
}

func (f *fakeAlertRepo) Update(_ context.Context, orgID, id uuid.UUID, input domain.AlertUpdate) (domain.Alert, error) {
	f.gotOrgID, f.gotID, f.gotPut, f.update = orgID, id, true, input
	if f.err != nil {
		return domain.Alert{}, f.err
	}
	return f.alert, nil
}

func (f *fakeAlertRepo) Delete(_ context.Context, orgID, id uuid.UUID) error {
	f.gotOrgID, f.gotID, f.gotDel = orgID, id, true
	return f.err
}

func newAlertServer(t *testing.T, st *controlStubs, repo *fakeAlertRepo) *httptest.Server {
	t.Helper()
	return buildServer(t, Deps{
		Identity:        identity.NewService(st.deps()),
		Alerts:          alert.NewService(alert.ServiceDeps{Alerts: repo}),
		DevLoginEnabled: true,
	})
}

// callAlert performs an authenticated request as role and returns the response
// with its body already drained.
func callAlert(t *testing.T, st *controlStubs, repo *fakeAlertRepo, method, role, path string, body any) (*http.Response, string) {
	t.Helper()
	srv := newAlertServer(t, st, repo)
	return doAlert(t, srv, st, method, role, path, body)
}

func doAlert(t *testing.T, srv *httptest.Server, st *controlStubs, method, role, path string, body any) (*http.Response, string) {
	t.Helper()
	resp := doJSON(t, srv, method, path, body, bearerHeader(st, role))
	return resp, string(readBody(t, resp))
}

// testAlert is a stored spend-limit rule; tests vary it per case.
func testAlert() domain.Alert {
	return domain.Alert{
		ID:             uuid.MustParse("00000000-0000-0000-0000-00000000000b"),
		OrganizationID: testOrgID,
		AlertType:      domain.AlertTypeSpendLimit,
		Name:           "Monthly spend cap",
		Severity:       domain.AlertSeverityCritical,
		Configuration:  []byte(`{"limit_usd":500}`),
		Status:         domain.AlertStatusActive,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
}

// validUpdateBody is a well-formed full replacement of a rule's mutable fields.
func validUpdateBody() map[string]any {
	return map[string]any{
		"name":          "Weekly spend cap",
		"severity":      domain.AlertSeverityWarning,
		"configuration": map[string]any{"limit_usd": 250},
		"status":        domain.AlertStatusActive,
	}
}

// TestListAlertsScopesToCallerOrg proves rules are read through the caller's
// tenant, never a client-supplied organization id.
func TestListAlertsScopesToCallerOrg(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{alerts: []domain.Alert{testAlert()}}
	_, body := callAlert(t, st, repo, http.MethodGet, domain.RoleOwner, "/v1/alerts", nil)

	if !repo.gotList {
		t.Fatal("repository List was not called")
	}
	if repo.gotOrgID != testOrgID {
		t.Errorf("org = %s, want %s", repo.gotOrgID, testOrgID)
	}
	if !strings.Contains(body, "Monthly spend cap") {
		t.Errorf("body = %s, want it to contain the stored rule", body)
	}
}

// TestListAlertsRequiresRead proves listing is gated on alerts.read: a member
// and a billing admin lack it and are refused before the repository is reached.
func TestListAlertsRequiresRead(t *testing.T) {
	for _, tc := range []struct {
		role string
		want int
	}{
		{domain.RoleOwner, http.StatusOK},
		{domain.RoleAdmin, http.StatusOK},
		{domain.RoleAuditor, http.StatusOK},
		{domain.RoleMember, http.StatusForbidden},
		{domain.RoleBillingAdmin, http.StatusForbidden},
	} {
		t.Run(tc.role, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeAlertRepo{alerts: []domain.Alert{testAlert()}}
			resp, _ := callAlert(t, st, repo, http.MethodGet, tc.role, "/v1/alerts", nil)
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusForbidden && repo.gotList {
				t.Error("repository was consulted despite the permission denial")
			}
		})
	}
}

// TestListAlertsEmptyIsItemsNotNull keeps the payload shape stable: no rules
// still yields an items array rather than null.
func TestListAlertsEmptyIsItemsNotNull(t *testing.T) {
	st := newControlStubs()
	_, body := callAlert(t, st, &fakeAlertRepo{}, http.MethodGet, domain.RoleOwner, "/v1/alerts", nil)
	if !strings.Contains(body, `"items":[]`) {
		t.Errorf("body = %s, want an empty items array", body)
	}
}

func TestGetAlertInvalidPathUUID(t *testing.T) {
	st := newControlStubs()
	resp, _ := callAlert(t, st, &fakeAlertRepo{}, http.MethodGet, domain.RoleOwner, "/v1/alerts/not-a-uuid", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestGetAlertMissingIs404 proves a rule the caller cannot see is reported as
// not found rather than leaking its existence.
func TestGetAlertMissingIs404(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{err: repositories.ErrNotFound}
	resp, _ := callAlert(t, st, repo, http.MethodGet, domain.RoleOwner, "/v1/alerts/"+uuid.New().String(), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetAlertScopesToCallerOrg(t *testing.T) {
	st := newControlStubs()
	alert := testAlert()
	repo := &fakeAlertRepo{alert: alert}
	callAlert(t, st, repo, http.MethodGet, domain.RoleOwner, "/v1/alerts/"+alert.ID.String(), nil)

	if repo.gotOrgID != testOrgID {
		t.Errorf("org = %s, want %s", repo.gotOrgID, testOrgID)
	}
	if repo.gotID != alert.ID {
		t.Errorf("id = %s, want %s", repo.gotID, alert.ID)
	}
}

// TestGetAlertEmitsConfigurationAsObject proves the rule's configuration reaches
// the client as a JSON object rather than base64 or a string blob.
func TestGetAlertEmitsConfigurationAsObject(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{alert: testAlert()}
	_, body := callAlert(t, st, repo, http.MethodGet, domain.RoleOwner, "/v1/alerts/"+testAlert().ID.String(), nil)

	if !strings.Contains(body, `"configuration":{"limit_usd":500}`) {
		t.Errorf("body = %s, want the configuration as a JSON object", body)
	}
	if !strings.Contains(body, `"alert_type":"spend_limit"`) {
		t.Errorf("body = %s, want the alert type", body)
	}
}

// TestGetAlertLastTriggeredIsNull proves the API never invents a firing time:
// nothing evaluates rules, so the column stays null rather than being faked.
func TestGetAlertLastTriggeredIsNull(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{alert: testAlert()}
	_, body := callAlert(t, st, repo, http.MethodGet, domain.RoleOwner, "/v1/alerts/"+testAlert().ID.String(), nil)

	if !strings.Contains(body, `"last_triggered_at":null`) {
		t.Errorf("body = %s, want a null last_triggered_at", body)
	}
}

// TestUpdateAlertValidationRejects proves an unusable rule is refused with 400
// before persistence, so no invalid configuration is ever stored.
func TestUpdateAlertValidationRejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"empty name", mutateBody(validUpdateBody(), map[string]any{"name": ""})},
		{"whitespace name", mutateBody(validUpdateBody(), map[string]any{"name": "   "})},
		{"unknown severity", mutateBody(validUpdateBody(), map[string]any{"severity": "urgent"})},
		{"unknown status", mutateBody(validUpdateBody(), map[string]any{"status": "paused"})},
		{"configuration is an array", mutateBody(validUpdateBody(), map[string]any{"configuration": []any{1, 2}})},
		{"configuration is null", mutateBody(validUpdateBody(), map[string]any{"configuration": nil})},
		{"configuration is a scalar", mutateBody(validUpdateBody(), map[string]any{"configuration": 500})},
		{"missing name", withoutKey(validUpdateBody(), "name")},
		{"missing severity", withoutKey(validUpdateBody(), "severity")},
		{"missing status", withoutKey(validUpdateBody(), "status")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeAlertRepo{alert: testAlert()}
			resp, _ := callAlert(t, st, repo, http.MethodPut, domain.RoleOwner,
				"/v1/alerts/"+testAlert().ID.String(), tc.body)

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
			if repo.gotPut {
				t.Error("repository was updated despite the invalid payload")
			}
		})
	}
}

// TestUpdateAlertAcceptsEmptyConfiguration proves configuration is optional: a
// rule with no thresholds yet is still a legal stored rule.
func TestUpdateAlertAcceptsEmptyConfiguration(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{alert: testAlert()}
	body := mutateBody(validUpdateBody(), map[string]any{"configuration": map[string]any{}})
	resp, _ := callAlert(t, st, repo, http.MethodPut, domain.RoleOwner,
		"/v1/alerts/"+testAlert().ID.String(), body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// TestUpdateAlertRequiresManage proves editing is gated on alerts.manage, which
// an auditor deliberately lacks: read access is not write access.
func TestUpdateAlertRequiresManage(t *testing.T) {
	for _, tc := range []struct {
		role string
		want int
	}{
		{domain.RoleOwner, http.StatusOK},
		{domain.RoleAdmin, http.StatusOK},
		{domain.RoleAuditor, http.StatusForbidden},
		{domain.RoleMember, http.StatusForbidden},
		{domain.RoleBillingAdmin, http.StatusForbidden},
	} {
		t.Run(tc.role, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeAlertRepo{alert: testAlert()}
			resp, _ := callAlert(t, st, repo, http.MethodPut, tc.role,
				"/v1/alerts/"+testAlert().ID.String(), validUpdateBody())

			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusForbidden && repo.gotPut {
				t.Error("repository was updated despite the permission denial")
			}
		})
	}
}

// TestUpdateAlertTrimsNameAndForwardsInput proves the service normalizes the
// name and forwards every mutable field to persistence under the caller's tenant.
func TestUpdateAlertTrimsNameAndForwardsInput(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{alert: testAlert()}
	body := mutateBody(validUpdateBody(), map[string]any{"name": "  Weekly spend cap  "})
	callAlert(t, st, repo, http.MethodPut, domain.RoleOwner,
		"/v1/alerts/"+testAlert().ID.String(), body)

	if !repo.gotPut {
		t.Fatal("repository Update was not called")
	}
	if repo.update.Name != "Weekly spend cap" {
		t.Errorf("name = %q, want the trimmed value", repo.update.Name)
	}
	if repo.update.Severity != domain.AlertSeverityWarning || repo.update.Status != domain.AlertStatusActive {
		t.Errorf("update = %+v, want the submitted severity and status", repo.update)
	}
	if !strings.Contains(string(repo.update.Configuration), "250") {
		t.Errorf("configuration = %s, want the submitted thresholds", repo.update.Configuration)
	}
	if repo.gotOrgID != testOrgID {
		t.Errorf("org = %s, want %s", repo.gotOrgID, testOrgID)
	}
}

// TestUpdateAlertRejectsAlertType proves alert_type is immutable: the decoder
// rejects the field outright, so a client cannot even attempt to retype a rule
// whose evaluator would be selected by that type.
func TestUpdateAlertRejectsAlertType(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{alert: testAlert()}
	body := mutateBody(validUpdateBody(), map[string]any{"alert_type": domain.AlertTypeAnomaly})
	resp, _ := callAlert(t, st, repo, http.MethodPut, domain.RoleOwner,
		"/v1/alerts/"+testAlert().ID.String(), body)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if repo.gotPut {
		t.Error("repository was updated despite the immutable alert_type field")
	}
}

func TestUpdateAlertMissingIs404(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{err: repositories.ErrNotFound}
	resp, _ := callAlert(t, st, repo, http.MethodPut, domain.RoleOwner,
		"/v1/alerts/"+uuid.New().String(), validUpdateBody())
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestDeleteAlertInvalidPathUUID(t *testing.T) {
	st := newControlStubs()
	resp, _ := callAlert(t, st, &fakeAlertRepo{}, http.MethodDelete, domain.RoleOwner, "/v1/alerts/not-a-uuid", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestDeleteAlertsReturnsNoContent proves a successful delete answers 204 with
// an empty body rather than a JSON envelope.
func TestDeleteAlertReturnsNoContent(t *testing.T) {
	st := newControlStubs()
	alert := testAlert()
	repo := &fakeAlertRepo{}
	resp, body := callAlert(t, st, repo, http.MethodDelete, domain.RoleOwner, "/v1/alerts/"+alert.ID.String(), nil)

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
	if body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	if !repo.gotDel {
		t.Fatal("repository Delete was not called")
	}
	if repo.gotOrgID != testOrgID || repo.gotID != alert.ID {
		t.Errorf("delete scoped to (%s, %s), want (%s, %s)", repo.gotOrgID, repo.gotID, testOrgID, alert.ID)
	}
}

func TestDeleteAlertMissingIs404(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{err: repositories.ErrNotFound}
	resp, _ := callAlert(t, st, repo, http.MethodDelete, domain.RoleOwner, "/v1/alerts/"+uuid.New().String(), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestDeleteAlertRequiresManage proves removal is gated on alerts.manage.
func TestDeleteAlertRequiresManage(t *testing.T) {
	for _, tc := range []struct {
		role string
		want int
	}{
		{domain.RoleOwner, http.StatusNoContent},
		{domain.RoleAdmin, http.StatusNoContent},
		{domain.RoleAuditor, http.StatusForbidden},
		{domain.RoleMember, http.StatusForbidden},
		{domain.RoleBillingAdmin, http.StatusForbidden},
	} {
		t.Run(tc.role, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeAlertRepo{}
			resp, _ := callAlert(t, st, repo, http.MethodDelete, tc.role, "/v1/alerts/"+testAlert().ID.String(), nil)

			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusForbidden && repo.gotDel {
				t.Error("repository was deleted despite the permission denial")
			}
		})
	}
}

// TestTestAlertReportsNoDelivery is the honesty contract for the test endpoint:
// it reports that nothing was evaluated and that no notification was sent, and
// it must not claim a firing verdict that no evaluator could produce.
func TestTestAlertReportsNoDelivery(t *testing.T) {
	st := newControlStubs()
	alert := testAlert()
	repo := &fakeAlertRepo{alert: alert}
	resp, body := callAlert(t, st, repo, http.MethodPost, domain.RoleOwner, "/v1/alerts/"+alert.ID.String()+"/test", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	for _, want := range []string{
		`"notification_sent":false`,
		`"status_is_muted":false`,
		`"alert_id":"` + alert.ID.String() + `"`,
		"no evaluator is wired",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want it to contain %s", body, want)
		}
	}
	for _, unwanted := range []string{"would_fire", "fired", "delivered", "triggered_at"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("body = %s, want it to avoid claiming %s", body, unwanted)
		}
	}
}

// TestTestAlertMutedReportsMuteState proves a muted rule is described as muted
// instead of pretending it was evaluated.
func TestTestAlertMutedReportsMuteState(t *testing.T) {
	st := newControlStubs()
	alert := testAlert()
	alert.Status = domain.AlertStatusMuted
	repo := &fakeAlertRepo{alert: alert}
	_, body := callAlert(t, st, repo, http.MethodPost, domain.RoleOwner, "/v1/alerts/"+alert.ID.String()+"/test", nil)

	if !strings.Contains(body, `"status_is_muted":true`) || !strings.Contains(body, "muted") {
		t.Errorf("body = %s, want the rule reported as muted", body)
	}
}

// TestTestAlertRejectsUnreadableStoredConfig proves a rule whose stored
// configuration is not an object is refused rather than validated as a success.
func TestTestAlertRejectsUnreadableStoredConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
	}{
		{"array", `[1,2]`},
		{"scalar", `500`},
		{"not json", `not json`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newControlStubs()
			alert := testAlert()
			alert.Configuration = []byte(tc.config)
			repo := &fakeAlertRepo{alert: alert}
			resp, _ := callAlert(t, st, repo, http.MethodPost, domain.RoleOwner,
				"/v1/alerts/"+alert.ID.String()+"/test", nil)

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestTestAlertMissingIs404(t *testing.T) {
	st := newControlStubs()
	repo := &fakeAlertRepo{err: repositories.ErrNotFound}
	resp, _ := callAlert(t, st, repo, http.MethodPost, domain.RoleOwner,
		"/v1/alerts/"+uuid.New().String()+"/test", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestTestAlertRequiresManage proves testing a rule is gated on alerts.manage.
func TestTestAlertRequiresManage(t *testing.T) {
	for _, tc := range []struct {
		role string
		want int
	}{
		{domain.RoleOwner, http.StatusOK},
		{domain.RoleAdmin, http.StatusOK},
		{domain.RoleAuditor, http.StatusForbidden},
		{domain.RoleMember, http.StatusForbidden},
		{domain.RoleBillingAdmin, http.StatusForbidden},
	} {
		t.Run(tc.role, func(t *testing.T) {
			st := newControlStubs()
			repo := &fakeAlertRepo{alert: testAlert()}
			resp, _ := callAlert(t, st, repo, http.MethodPost, tc.role,
				"/v1/alerts/"+testAlert().ID.String()+"/test", nil)

			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusForbidden && repo.gotID != uuid.Nil {
				t.Error("rule was read despite the permission denial")
			}
		})
	}
}

// mutateBody returns a copy of body with keys replaced, so each case starts
// from a valid payload and only overrides what it is testing.
func mutateBody(body map[string]any, overrides map[string]any) map[string]any {
	out := make(map[string]any, len(body))
	for k, v := range body {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// withoutKey returns a copy of body missing one key, proving a required field is
// genuinely required rather than defaulted.
func withoutKey(body map[string]any, key string) map[string]any {
	out := mutateBody(body, nil)
	delete(out, key)
	return out
}
