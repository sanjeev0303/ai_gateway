package controlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

var (
	testPolicyID = uuid.MustParse("00000000-0000-0000-0000-000000000020")
	testRuleID   = uuid.MustParse("00000000-0000-0000-0000-000000000021")
)

var testModelAccessEntry = domain.ModelAccessEntry{
	Provider: "anthropic",
	Model:    "claude-3-5-sonnet",
	Allowed:  true,
	Source:   "allowlist",
	PolicyID: testPolicyID,
	RuleID:   testRuleID,
	Priority: 50,
	Effective: true,
}

func TestModelAccessRequireAuth(t *testing.T) {
	st := newControlStubs()
	srv := newPolicyServer(t, st)

	for _, test := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/model-access"},
		{http.MethodPost, "/v1/model-access/preview"},
	} {
		resp := doJSON(t, srv, test.method, test.path, nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s: got %d, want 401", test.method, test.path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestModelAccessList(t *testing.T) {
	st := newControlStubs()
	st.policyRepo.list = func(_ context.Context, _ uuid.UUID) ([]domain.Policy, error) {
		return []domain.Policy{{
			ID:             testPolicyID,
			OrganizationID: testOrgID,
			Name:           "Model Access Control",
			Description:    "Auto-generated policy for model allowlist/blocklist rules",
			Enforcement:    domain.EnforcementActive,
			Priority:       50,
			CreatedAt:      testNow,
			UpdatedAt:      testNow,
		}}, nil
	}
	st.policyRuleRepo.list = func(_ context.Context, _, _ uuid.UUID) ([]domain.PolicyRule, error) {
		cond, _ := json.Marshal(map[string]any{"provider": "anthropic", "model": "claude-3-5-sonnet"})
		return []domain.PolicyRule{{
			ID:             testRuleID,
			PolicyID:       testPolicyID,
			OrganizationID: testOrgID,
			RuleType:       domain.RuleTypeAllowlist,
			Description:    "Allow anthropic/claude-3-5-sonnet",
			Conditions:     cond,
			Parameters:     []byte("{}"),
			Enabled:        true,
			CreatedAt:      testNow,
		}}, nil
	}
	srv := newPolicyServer(t, st)

	resp := doJSON(t, srv, http.MethodGet, "/v1/model-access", nil, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
			Allowed  bool   `json:"allowed"`
			Source   string `json:"source"`
			PolicyID string `json:"policy_id"`
			RuleID   string `json:"rule_id"`
			Priority int32  `json:"priority"`
		} `json:"items"`
	}
	decodeBody(t, resp, &body)
	if len(body.Items) != 1 {
		t.Fatalf("got %d entries, want 1", len(body.Items))
	}
	if body.Items[0].Provider != "anthropic" {
		t.Fatalf("got provider %q, want anthropic", body.Items[0].Provider)
	}
	if body.Items[0].Model != "claude-3-5-sonnet" {
		t.Fatalf("got model %q, want claude-3-5-sonnet", body.Items[0].Model)
	}
	if !body.Items[0].Allowed {
		t.Fatalf("got allowed %v, want true", body.Items[0].Allowed)
	}
	if body.Items[0].Source != "allowlist" {
		t.Fatalf("got source %q, want allowlist", body.Items[0].Source)
	}
}

func TestModelAccessUpdate(t *testing.T) {
	st := newControlStubs()
	st.policyRepo.list = func(_ context.Context, _ uuid.UUID) ([]domain.Policy, error) {
		return []domain.Policy{{
			ID:             testPolicyID,
			OrganizationID: testOrgID,
			Name:           "Model Access Control",
			Enforcement:    domain.EnforcementActive,
			Priority:       50,
		}}, nil
	}
	st.policyRuleRepo.list = func(_ context.Context, _, _ uuid.UUID) ([]domain.PolicyRule, error) {
		return []domain.PolicyRule{}, nil
	}
	st.policyRuleRepo.create = func(_ context.Context, _ uuid.UUID, in domain.NewPolicyRule) (domain.PolicyRule, error) {
		return domain.PolicyRule{
			ID:             testRuleID,
			PolicyID:       in.PolicyID,
			OrganizationID: testOrgID,
			RuleType:       in.RuleType,
			Description:    in.Description,
			Conditions:     in.Conditions,
			Parameters:     in.Parameters,
			Enabled:        in.Enabled,
			CreatedAt:      testNow,
		}, nil
	}
	srv := newPolicyServer(t, st)

	reqBody := map[string]any{
		"entries": []map[string]any{
			{
				"provider": "anthropic",
				"model":    "claude-3-opus",
				"action":   "allow",
			},
			{
				"provider": "openai",
				"model":    "gpt-4o",
				"action":   "block",
			},
		},
	}
	resp := doJSON(t, srv, http.MethodPut, "/v1/model-access", reqBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Status string `json:"status"`
	}
	decodeBody(t, resp, &body)
	if body.Status != "updated" {
		t.Fatalf("got status %q, want updated", body.Status)
	}
}

func TestModelAccessPreview(t *testing.T) {
	st := newControlStubs()
	st.policyRepo.list = func(_ context.Context, _ uuid.UUID) ([]domain.Policy, error) {
		return []domain.Policy{{
			ID:             testPolicyID,
			OrganizationID: testOrgID,
			Name:           "Model Access Control",
			Enforcement:    domain.EnforcementActive,
			Priority:       50,
		}}, nil
	}
	st.policyRuleRepo.list = func(_ context.Context, _, _ uuid.UUID) ([]domain.PolicyRule, error) {
		cond, _ := json.Marshal(map[string]any{"provider": "anthropic", "model": "claude-3-opus"})
		return []domain.PolicyRule{{
			ID:             testRuleID,
			PolicyID:       testPolicyID,
			OrganizationID: testOrgID,
			RuleType:       domain.RuleTypeBlocklist,
			Description:    "Block anthropic/claude-3-opus",
			Conditions:     cond,
			Parameters:     []byte("{}"),
			Enabled:        true,
			CreatedAt:      testNow,
		}}, nil
	}
	srv := newPolicyServer(t, st)

	reqBody := map[string]any{
		"provider": "anthropic",
		"model":    "claude-3-opus",
		"action":   "allow",
	}
	resp := doJSON(t, srv, http.MethodPost, "/v1/model-access/preview", reqBody, bearerHeader(st, domain.RoleOwner))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Provider          string `json:"provider"`
		Model             string `json:"model"`
		CurrentDecision   string `json:"current_decision"`
		PreviewDecision   string `json:"preview_decision"`
		WouldChange       bool   `json:"would_change"`
		BlockingRules     []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"blocking_rules"`
		AllowingRules     []struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"allowing_rules"`
	}
	decodeBody(t, resp, &body)
	if body.Provider != "anthropic" {
		t.Fatalf("got provider %q, want anthropic", body.Provider)
	}
	if body.Model != "claude-3-opus" {
		t.Fatalf("got model %q, want claude-3-opus", body.Model)
	}
	if body.CurrentDecision != "block" {
		t.Fatalf("got current_decision %q, want block", body.CurrentDecision)
	}
	if body.PreviewDecision != "allow" {
		t.Fatalf("got preview_decision %q, want allow", body.PreviewDecision)
	}
	if !body.WouldChange {
		t.Fatalf("got would_change %v, want true", body.WouldChange)
	}
	if len(body.BlockingRules) != 1 {
		t.Fatalf("got %d blocking rules, want 1", len(body.BlockingRules))
	}
	if body.BlockingRules[0].Provider != "anthropic" {
		t.Fatalf("got blocking provider %q, want anthropic", body.BlockingRules[0].Provider)
	}
}

func TestModelAccessForbiddenWithoutOwnerRole(t *testing.T) {
	st := newControlStubs()
	srv := newPolicyServer(t, st)

	resp := doJSON(t, srv, http.MethodPut, "/v1/model-access", map[string]any{
		"entries": []map[string]any{
			{
				"provider": "anthropic",
				"model":    "claude-3-opus",
				"action":   "allow",
			},
		},
	}, bearerHeader(st, domain.RoleMember))
	status, code := decodeError(t, resp)
	if status != http.StatusForbidden || code != "permission_denied" {
		t.Fatalf("got %d/%s, want 403/permission_denied", status, code)
	}
}