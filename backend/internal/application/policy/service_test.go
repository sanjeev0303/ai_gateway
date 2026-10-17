package policy_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/policy"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// fakePolicyRepo implements repositories.PolicyRepository for unit testing.
type fakePolicyRepo struct {
	mu       sync.Mutex
	policies map[string]domain.Policy
}

func newFakePolicyRepo() *fakePolicyRepo {
	return &fakePolicyRepo{policies: make(map[string]domain.Policy)}
}

func (f *fakePolicyRepo) key(orgID, id uuid.UUID) string {
	return orgID.String() + ":" + id.String()
}

func (f *fakePolicyRepo) Create(_ context.Context, orgID uuid.UUID, in domain.NewPolicy) (domain.Policy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := domain.Policy{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           in.Name,
		Description:    in.Description,
		Enforcement:    in.Enforcement,
		Priority:       in.Priority,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	f.policies[f.key(orgID, p.ID)] = p
	return p, nil
}

func (f *fakePolicyRepo) Get(_ context.Context, orgID, id uuid.UUID) (domain.Policy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.policies[f.key(orgID, id)]
	if !ok {
		return domain.Policy{}, repositories.ErrNotFound
	}
	return p, nil
}

func (f *fakePolicyRepo) List(_ context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Policy
	for _, p := range f.policies {
		if p.OrganizationID == orgID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePolicyRepo) ListActive(_ context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Policy
	for _, p := range f.policies {
		if p.OrganizationID == orgID && p.Enforcement == domain.EnforcementActive {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePolicyRepo) Update(_ context.Context, orgID, id uuid.UUID, in domain.UpdatePolicy) (domain.Policy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.policies[f.key(orgID, id)]
	if !ok {
		return domain.Policy{}, repositories.ErrNotFound
	}
	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.Enforcement != nil {
		p.Enforcement = *in.Enforcement
	}
	if in.Priority != nil {
		p.Priority = *in.Priority
	}
	p.UpdatedAt = time.Now().UTC()
	f.policies[f.key(orgID, id)] = p
	return p, nil
}

func (f *fakePolicyRepo) Delete(_ context.Context, orgID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := f.key(orgID, id)
	if _, ok := f.policies[k]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.policies, k)
	return nil
}

// fakeRuleRepo implements repositories.PolicyRuleRepository for unit testing.
type fakeRuleRepo struct {
	mu    sync.Mutex
	rules map[string]domain.PolicyRule
}

func newFakeRuleRepo() *fakeRuleRepo {
	return &fakeRuleRepo{rules: make(map[string]domain.PolicyRule)}
}

func (f *fakeRuleRepo) key(orgID, policyID, id uuid.UUID) string {
	return orgID.String() + ":" + policyID.String() + ":" + id.String()
}

func (f *fakeRuleRepo) Create(_ context.Context, orgID uuid.UUID, in domain.NewPolicyRule) (domain.PolicyRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := domain.PolicyRule{
		ID:             uuid.New(),
		PolicyID:       in.PolicyID,
		OrganizationID: orgID,
		RuleType:       in.RuleType,
		Description:    in.Description,
		Conditions:     in.Conditions,
		Parameters:     in.Parameters,
		Enabled:        in.Enabled,
		CreatedAt:      time.Now().UTC(),
	}
	f.rules[f.key(orgID, in.PolicyID, r.ID)] = r
	return r, nil
}

func (f *fakeRuleRepo) Get(_ context.Context, orgID, policyID, id uuid.UUID) (domain.PolicyRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rules[f.key(orgID, policyID, id)]
	if !ok {
		return domain.PolicyRule{}, repositories.ErrNotFound
	}
	return r, nil
}

func (f *fakeRuleRepo) List(_ context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.PolicyRule
	for _, r := range f.rules {
		if r.OrganizationID == orgID && r.PolicyID == policyID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRuleRepo) Update(_ context.Context, orgID, policyID, id uuid.UUID, in domain.UpdatePolicyRule) (domain.PolicyRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rules[f.key(orgID, policyID, id)]
	if !ok {
		return domain.PolicyRule{}, repositories.ErrNotFound
	}
	if in.Description != nil {
		r.Description = *in.Description
	}
	if in.Conditions != nil {
		r.Conditions = in.Conditions
	}
	if in.Parameters != nil {
		r.Parameters = in.Parameters
	}
	if in.Enabled != nil {
		r.Enabled = *in.Enabled
	}
	f.rules[f.key(orgID, policyID, id)] = r
	return r, nil
}

func (f *fakeRuleRepo) Delete(_ context.Context, orgID, policyID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := f.key(orgID, policyID, id)
	if _, ok := f.rules[k]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.rules, k)
	return nil
}

// fakeAuditRepo records audit events for assertion.
type fakeAuditRepo struct {
	mu      sync.Mutex
	entries []domain.NewAuditLog
}

func (f *fakeAuditRepo) Append(_ context.Context, _ uuid.UUID, in domain.NewAuditLog) (domain.AuditLog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, in)
	return domain.AuditLog{
		ID:        uuid.New(),
		Action:    in.Action,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeAuditRepo) List(_ context.Context, _ uuid.UUID, _ domain.AuditLogFilter, _, _ int32) ([]domain.AuditLog, error) {
	return nil, nil
}

func (f *fakeAuditRepo) Get(_ context.Context, _, _ uuid.UUID) (domain.AuditLog, error) {
	return domain.AuditLog{}, nil
}

func (f *fakeAuditRepo) PurgeExpired(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	return 0, nil
}

// fakeInvalidator tracks invalidation calls.
type fakeInvalidator struct {
	mu       sync.Mutex
	orgCalls []uuid.UUID
}

func (f *fakeInvalidator) Invalidate(orgID uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orgCalls = append(f.orgCalls, orgID)
}

func (f *fakeInvalidator) countFor(orgID uuid.UUID) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := 0
	for _, id := range f.orgCalls {
		if id == orgID {
			c++
		}
	}
	return c
}

// TestPolicyService_Lifecycle verifies:
// - Policy creation, editing, enabling, disabling, and deletion.
// - RBAC enforcement (admin/owner allowed, viewer/member rejected).
// - Cache invalidation and audit logging on every mutation.
// - Multi-tenant isolation.
func TestPolicyService_Lifecycle(t *testing.T) {
	ctx := context.Background()
	orgA := uuid.New()
	orgB := uuid.New()
	adminActor := domain.Actor{OrganizationID: orgA, UserID: uuid.New(), Role: domain.RoleAdmin}
	memberActor := domain.Actor{OrganizationID: orgA, UserID: uuid.New(), Role: domain.RoleMember}
	orgBActor := domain.Actor{OrganizationID: orgB, UserID: uuid.New(), Role: domain.RoleAdmin}

	policyRepo := newFakePolicyRepo()
	ruleRepo := newFakeRuleRepo()
	auditRepo := &fakeAuditRepo{}
	invalidator := &fakeInvalidator{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc := policy.NewService(policyRepo, ruleRepo, auditRepo, invalidator, logger)

	// 1. RBAC: Member cannot create policy
	_, err := svc.CreatePolicy(ctx, memberActor, domain.NewPolicy{
		Name:        "unauthorized",
		Description: "should fail",
		Enforcement: domain.EnforcementActive,
		Priority:    1,
	})
	if err != repositories.ErrForbidden {
		t.Fatalf("expected ErrForbidden for member, got %v", err)
	}

	// 2. Admin creates policy
	pol, err := svc.CreatePolicy(ctx, adminActor, domain.NewPolicy{
		Name:        "Production Governance",
		Description: "Default controls for production org",
		Enforcement: domain.EnforcementActive,
		Priority:    10,
	})
	if err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}
	if pol.Name != "Production Governance" || pol.OrganizationID != orgA {
		t.Fatalf("unexpected policy created: %+v", pol)
	}
	if invalidator.countFor(orgA) != 1 {
		t.Fatalf("expected 1 cache invalidation for orgA, got %d", invalidator.countFor(orgA))
	}

	// 3. Admin updates policy (renaming & updating priority)
	newName := "Production Governance v2"
	newPriority := int32(20)
	updatedPol, err := svc.UpdatePolicy(ctx, adminActor, pol.ID, domain.UpdatePolicy{
		Name:     &newName,
		Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("failed to update policy: %v", err)
	}
	if updatedPol.Name != newName || updatedPol.Priority != 20 {
		t.Fatalf("unexpected updated policy: %+v", updatedPol)
	}
	if invalidator.countFor(orgA) != 2 {
		t.Fatalf("expected 2 cache invalidations, got %d", invalidator.countFor(orgA))
	}

	// 4. Disable policy
	disabledState := domain.EnforcementDisabled
	disabledPol, err := svc.UpdatePolicy(ctx, adminActor, pol.ID, domain.UpdatePolicy{
		Enforcement: &disabledState,
	})
	if err != nil {
		t.Fatalf("failed to disable policy: %v", err)
	}
	if disabledPol.Enforcement != domain.EnforcementDisabled {
		t.Fatalf("expected enforcement disabled, got %s", disabledPol.Enforcement)
	}

	// 5. Re-enable policy
	activeState := domain.EnforcementActive
	reEnabledPol, err := svc.UpdatePolicy(ctx, adminActor, pol.ID, domain.UpdatePolicy{
		Enforcement: &activeState,
	})
	if err != nil {
		t.Fatalf("failed to re-enable policy: %v", err)
	}
	if reEnabledPol.Enforcement != domain.EnforcementActive {
		t.Fatalf("expected enforcement active, got %s", reEnabledPol.Enforcement)
	}

	// 6. Tenant isolation: Org B cannot access Org A policy
	_, err = svc.GetPolicy(ctx, orgB, pol.ID)
	if err != repositories.ErrNotFound {
		t.Fatalf("expected ErrNotFound for Org B query, got %v", err)
	}
	_, err = svc.UpdatePolicy(ctx, orgBActor, pol.ID, domain.UpdatePolicy{Name: &newName})
	if err != repositories.ErrNotFound {
		t.Fatalf("expected ErrNotFound for Org B update, got %v", err)
	}

	// 7. Rule lifecycle: Create rule
	condJSON, _ := json.Marshal(map[string]any{"model": "gpt-4o"})
	paramJSON, _ := json.Marshal(map[string]any{"action": "allow"})
	rule, err := svc.CreateRule(ctx, adminActor, domain.NewPolicyRule{
		PolicyID:    pol.ID,
		RuleType:    domain.RuleTypeAllowlist,
		Description: "Allow gpt-4o",
		Conditions:  condJSON,
		Parameters:  paramJSON,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}
	if rule.OrganizationID != orgA || rule.PolicyID != pol.ID || !rule.Enabled {
		t.Fatalf("unexpected rule created: %+v", rule)
	}

	// 8. Disable rule
	disabledRuleBool := false
	updatedRule, err := svc.UpdateRule(ctx, adminActor, pol.ID, rule.ID, domain.UpdatePolicyRule{
		Enabled: &disabledRuleBool,
	})
	if err != nil {
		t.Fatalf("failed to update rule: %v", err)
	}
	if updatedRule.Enabled != false {
		t.Fatalf("expected rule to be disabled, got enabled")
	}

	// 9. Re-enable rule
	enabledRuleBool := true
	reEnabledRule, err := svc.UpdateRule(ctx, adminActor, pol.ID, rule.ID, domain.UpdatePolicyRule{
		Enabled: &enabledRuleBool,
	})
	if err != nil {
		t.Fatalf("failed to re-enable rule: %v", err)
	}
	if reEnabledRule.Enabled != true {
		t.Fatalf("expected rule to be enabled, got disabled")
	}

	// 10. Delete rule
	err = svc.DeleteRule(ctx, adminActor, pol.ID, rule.ID)
	if err != nil {
		t.Fatalf("failed to delete rule: %v", err)
	}
	_, err = svc.GetRule(ctx, orgA, pol.ID, rule.ID)
	if err != repositories.ErrNotFound {
		t.Fatalf("expected rule to be deleted, got %v", err)
	}

	// 11. Delete policy
	err = svc.DeletePolicy(ctx, adminActor, pol.ID)
	if err != nil {
		t.Fatalf("failed to delete policy: %v", err)
	}
	_, err = svc.GetPolicy(ctx, orgA, pol.ID)
	if err != repositories.ErrNotFound {
		t.Fatalf("expected policy to be deleted, got %v", err)
	}

	// 12. Verify audit events recorded
	expectedActions := map[string]bool{
		"policy.create":      false,
		"policy.update":      false,
		"policy.delete":      false,
		"policy_rule.create": false,
		"policy_rule.update": false,
		"policy_rule.delete": false,
	}
	for _, entry := range auditRepo.entries {
		if _, ok := expectedActions[entry.Action]; ok {
			expectedActions[entry.Action] = true
		}
	}
	for action, recorded := range expectedActions {
		if !recorded {
			t.Errorf("expected audit action %s to be recorded, but was not", action)
		}
	}
}
