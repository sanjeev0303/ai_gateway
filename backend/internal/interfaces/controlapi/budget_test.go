package controlapi

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

var (
	testBudgetID = uuid.MustParse("00000000-0000-0000-0000-000000000010")
)

var testBudget = domain.Budget{
	ID:              testBudgetID,
	OrganizationID:  testOrgID,
	Name:            "Monthly Spend Cap",
	Description:     "Monthly spend limit for the organization",
	Scope:           domain.BudgetScopeOrganization,
	ScopeID:         uuid.Nil,
	LimitType:       domain.BudgetLimitTypeSpend,
	LimitAmount:     1000.0,
	Window:          domain.BudgetWindowMonthly,
	AlertThreshold:  0.8,
	Enabled:         true,
	CreatedAt:       testNow,
	UpdatedAt:       testNow,
	CurrentSpend:    500.0,
	CurrentTokens:   0,
	CurrentRequests: 0,
	WindowStart:     testNow.Add(-15 * 24 * time.Hour),
	WindowEnd:       testNow.Add(15 * 24 * time.Hour),
}

func TestBudgetsRequireAuth(t *testing.T) {
	st := newControlStubs()
	srv := newBudgetServer(t, st)

	for _, path := range []string{
		"/v1/budgets",
		"/v1/budgets/" + testBudgetID.String(),
	} {
		resp := doJSON(t, srv, http.MethodGet, path, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: got %d, want 401", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestBudgetList(t *testing.T) {
	st := newControlStubs()
	st.budgets.list = func(_ context.Context, _ uuid.UUID, _ repositories.BudgetFilter) ([]domain.Budget, error) {
		return []domain.Budget{testBudget}, nil
	}
	srv := newBudgetServer(t, st)

	resp := doJSON(t, srv, http.MethodGet, "/v1/budgets", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Scope       string  `json:"scope"`
			LimitType   string  `json:"limit_type"`
			LimitAmount float64 `json:"limit_amount"`
			Enabled     bool    `json:"enabled"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) != 1 {
		t.Fatalf("got %d budgets, want 1", len(body.Items))
	}
	if body.Items[0].Name != "Monthly Spend Cap" {
		t.Fatalf("got name %q, want %q", body.Items[0].Name, "Monthly Spend Cap")
	}
	if body.Items[0].LimitType != "spend" {
		t.Fatalf("got limit_type %q, want spend", body.Items[0].LimitType)
	}
	if body.Items[0].LimitAmount != 1000.0 {
		t.Fatalf("got limit_amount %f, want 1000.0", body.Items[0].LimitAmount)
	}
	if !body.Items[0].Enabled {
		t.Fatalf("got enabled %v, want true", body.Items[0].Enabled)
	}
}

func TestBudgetListWithFilters(t *testing.T) {
	st := newControlStubs()
	var capturedFilter repositories.BudgetFilter
	st.budgets.list = func(_ context.Context, _ uuid.UUID, f repositories.BudgetFilter) ([]domain.Budget, error) {
		capturedFilter = f
		return []domain.Budget{}, nil
	}
	srv := newBudgetServer(t, st)

	resp := doJSON(t, srv, http.MethodGet, "/v1/budgets?scope=team&limit_type=tokens&enabled=true", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	scope := domain.BudgetScope("team")
	limitType := domain.BudgetLimitType("tokens")
	enabled := true
	if capturedFilter.Scope == nil || *capturedFilter.Scope != scope {
		t.Fatalf("got scope %v, want %v", capturedFilter.Scope, scope)
	}
	if capturedFilter.LimitType == nil || *capturedFilter.LimitType != limitType {
		t.Fatalf("got limit_type %v, want %v", capturedFilter.LimitType, limitType)
	}
	if capturedFilter.Enabled == nil || *capturedFilter.Enabled != enabled {
		t.Fatalf("got enabled %v, want %v", capturedFilter.Enabled, enabled)
	}
}

func TestBudgetCreate(t *testing.T) {
	st := newControlStubs()
	var createdBudget domain.Budget
	st.budgets.create = func(_ context.Context, _ uuid.UUID, in domain.NewBudget) (domain.Budget, error) {
		createdBudget = domain.Budget{
			ID:              testBudgetID,
			OrganizationID:  testOrgID,
			Name:            in.Name,
			Description:     in.Description,
			Scope:           in.Scope,
			ScopeID:         in.ScopeID,
			LimitType:       in.LimitType,
			LimitAmount:     in.LimitAmount,
			Window:          in.Window,
			AlertThreshold:  in.AlertThreshold,
			Enabled:         in.Enabled,
			CreatedAt:       testNow,
			UpdatedAt:       testNow,
		}
		return createdBudget, nil
	}
	srv := newBudgetServer(t, st)

	reqBody := map[string]any{
		"name":            "Team Token Budget",
		"description":     "Token budget for team",
		"scope":           "team",
		"scope_id":        testTeamID.String(),
		"limit_type":      "tokens",
		"limit_amount":    100000,
		"window":          "monthly",
		"alert_threshold": 0.75,
		"enabled":         true,
	}
	resp := doJSON(t, srv, http.MethodPost, "/v1/budgets", reqBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("got %d, want 201, body: %s", resp.StatusCode, string(bodyBytes))
	}
	var body struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Scope       string  `json:"scope"`
		LimitType   string  `json:"limit_type"`
		LimitAmount float64 `json:"limit_amount"`
		Enabled     bool    `json:"enabled"`
	}
	decodeBody(t, resp, &body)
	if body.Name != "Team Token Budget" {
		t.Fatalf("got name %q, want %q", body.Name, "Team Token Budget")
	}
	if body.Scope != "team" {
		t.Fatalf("got scope %q, want team", body.Scope)
	}
	if body.LimitType != "tokens" {
		t.Fatalf("got limit_type %q, want tokens", body.LimitType)
	}
	if body.LimitAmount != 100000 {
		t.Fatalf("got limit_amount %f, want 100000", body.LimitAmount)
	}
	if !body.Enabled {
		t.Fatalf("got enabled %v, want true", body.Enabled)
	}
}

func TestBudgetGet(t *testing.T) {
	st := newControlStubs()
	st.budgets.get = func(_ context.Context, _, _ uuid.UUID) (domain.Budget, error) {
		return testBudget, nil
	}
	srv := newBudgetServer(t, st)

	resp := doJSON(t, srv, http.MethodGet, "/v1/budgets/"+testBudgetID.String(), nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Scope       string  `json:"scope"`
		LimitType   string  `json:"limit_type"`
		LimitAmount float64 `json:"limit_amount"`
		Enabled     bool    `json:"enabled"`
	}
	decodeBody(t, resp, &body)
	if body.ID != testBudgetID.String() {
		t.Fatalf("got id %q, want %q", body.ID, testBudgetID.String())
	}
	if body.Name != "Monthly Spend Cap" {
		t.Fatalf("got name %q, want %q", body.Name, "Monthly Spend Cap")
	}
}

func TestBudgetUpdate(t *testing.T) {
	st := newControlStubs()
	st.budgets.get = func(_ context.Context, _, _ uuid.UUID) (domain.Budget, error) {
		return domain.Budget{
			ID:             testBudgetID,
			OrganizationID: testOrgID,
			Name:           "Test Budget",
			Scope:          "organization",
			ScopeID:        testOrgID,
		}, nil
	}
	var updatedBudget domain.Budget
	st.budgets.update = func(_ context.Context, _, _ uuid.UUID, in domain.UpdateBudget) (domain.Budget, error) {
		updatedBudget = domain.Budget{
			ID:             testBudgetID,
			OrganizationID: testOrgID,
			Name:           *in.Name,
			Description:    *in.Description,
			LimitAmount:    *in.LimitAmount,
			AlertThreshold: *in.AlertThreshold,
			Enabled:        *in.Enabled,
		}
		return updatedBudget, nil
	}
	srv := newBudgetServer(t, st)

	reqBody := map[string]any{
		"name":            "Updated Budget Name",
		"description":     "Updated description",
		"limit_amount":    2000.0,
		"alert_threshold": 0.9,
		"enabled":         false,
	}
	resp := doJSON(t, srv, http.MethodPatch, "/v1/budgets/"+testBudgetID.String(), reqBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Name        string  `json:"name"`
		LimitAmount float64 `json:"limit_amount"`
		Enabled     bool    `json:"enabled"`
	}
	decodeBody(t, resp, &body)
	if body.Name != "Updated Budget Name" {
		t.Fatalf("got name %q, want %q", body.Name, "Updated Budget Name")
	}
	if body.LimitAmount != 2000.0 {
		t.Fatalf("got limit_amount %f, want 2000.0", body.LimitAmount)
	}
	if body.Enabled {
		t.Fatalf("got enabled %v, want false", body.Enabled)
	}
}

func TestBudgetDelete(t *testing.T) {
	st := newControlStubs()
	st.budgets.get = func(_ context.Context, _, _ uuid.UUID) (domain.Budget, error) {
		return domain.Budget{
			ID:             testBudgetID,
			OrganizationID: testOrgID,
			Name:           "Test Budget",
			Scope:          "organization",
			ScopeID:        testOrgID,
		}, nil
	}
	deleted := false
	st.budgets.delete = func(_ context.Context, _, id uuid.UUID) error {
		deleted = true
		if id != testBudgetID {
			t.Fatalf("got id %q, want %q", id, testBudgetID)
		}
		return nil
	}
	srv := newBudgetServer(t, st)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/budgets/"+testBudgetID.String(), nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("got %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()
	if !deleted {
		t.Fatal("delete was not called")
	}
}

func TestBudgetForbiddenWithoutOwnerRole(t *testing.T) {
	st := newControlStubs()
	srv := newBudgetServer(t, st)

	// RoleMember cannot manage budgets
	resp := doJSON(t, srv, http.MethodPost, "/v1/budgets", map[string]any{
		"name":            "Test",
		"scope":           "organization",
		"scope_id":        "",
		"limit_type":      "spend",
		"limit_amount":    100,
		"window":          "monthly",
		"alert_threshold": 0.8,
		"enabled":         true,
	}, bearerHeader(st, domain.RoleMember))
	status, code := decodeError(t, resp)
	if status != http.StatusForbidden || code != "permission_denied" {
		t.Fatalf("got %d/%s, want 403/permission_denied", status, code)
	}
}