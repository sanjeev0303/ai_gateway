// Contract tests for policy snapshot compilation: enforcement gating, rule
// filtering, rate-limit/budget config extraction and snapshot caching.
package gatewayv2

import (
	"context"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"

	"github.com/google/uuid"
)

func enforcedPolicy(id string) domain.Policy {
	return domain.Policy{
		ID:           mustUUIDs[0],
		OrganizationID: mustUUIDs[0],
		Name:         id,
		Enforcement:  "enforced",
		Priority:     10,
	}
}

func rateLimitRule() domain.PolicyRule {
	return domain.PolicyRule{
		ID:         mustUUIDs[1],
		PolicyID:   mustUUIDs[0],
		RuleType:   domain.RuleTypeRateLimit,
		Enabled:    true,
		Conditions: []byte(`{"type":"rate_limit","key":"org:infer"}`),
		Parameters: []byte(`{"limit":100,"window_seconds":60,"scope":"organization","burst":0}`),
	}
}

func budgetRule() domain.PolicyRule {
	return domain.PolicyRule{
		ID:         mustUUIDs[2],
		PolicyID:   mustUUIDs[0],
		RuleType:   domain.RuleTypeTokenBudget,
		Enabled:    true,
		Conditions: []byte(`{"type":"budget","key":"org:budget"}`),
		Parameters: []byte(`{"limit":1000,"window_seconds":3600,"scope":"organization","alert_at":0.8}`),
	}
}

func makePolicyResolver(t *testing.T, policies []domain.Policy, rulesByPolicy map[uuid.UUID][]domain.PolicyRule) *PolicyResolverImpl {
	t.Helper()
	var policyCalls, ruleCalls int
	tr := NewPolicyResolver(
		&stubPolicyRepo{listFn: func(context.Context, uuid.UUID) ([]domain.Policy, error) {
			policyCalls++
			return policies, nil
		}},
		&stubPolicyRuleRepo{listFn: func(_ context.Context, _ uuid.UUID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
			ruleCalls++
			return rulesByPolicy[policyID], nil
		}},
	)
	t.Cleanup(func() {
		t.Logf("policy list calls=%d rule list calls=%d", policyCalls, ruleCalls)
	})
	return tr
}

func TestPolicyResolver_HappyPath(t *testing.T) {
	policy := enforcedPolicy("infer")
	rules := map[uuid.UUID][]domain.PolicyRule{
		policy.ID: {rateLimitRule(), budgetRule()},
	}
	tr := makePolicyResolver(t, []domain.Policy{policy}, rules)

	snapshot, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	if snapshot.OrganizationID != mustUUIDs[0] {
		t.Fatalf("expected org id in snapshot, got %v", snapshot.OrganizationID)
	}
	if len(snapshot.Rules) != 2 {
		t.Fatalf("expected both enabled rules in snapshot, got %d", len(snapshot.Rules))
	}
	rl, ok := snapshot.RateLimits["org:infer"]
	if !ok {
		t.Fatal("expected extracted rate limit config for org:infer")
	}
	if rl.Limit != 100 || rl.Window != time.Minute || rl.Scope != "organization" || rl.BurstAllow != 0 {
		t.Fatalf("unexpected rate limit config: %+v", rl)
	}
	bud, ok := snapshot.Budgets["org:budget"]
	if !ok {
		t.Fatal("expected extracted budget config for org:budget")
	}
	if bud.Limit != 1000 || bud.Window != time.Hour || bud.AlertAt != 0.8 {
		t.Fatalf("unexpected budget config: %+v", bud)
	}
	if snapshot.Version <= 0 {
		t.Fatal("expected positive snapshot version")
	}
	if snapshot.EnforcedAt.IsZero() {
		t.Fatal("expected enforced-at timestamp")
	}
}

func TestPolicyResolver_NonEnforcedPolicyExcluded(t *testing.T) {
	enforced := enforcedPolicy("enforced")
	passive := enforcedPolicy("passive")
	passive.Enforcement = "permissive"
	passive.ID = mustUUIDs[1]
	rules := map[uuid.UUID][]domain.PolicyRule{
		enforced.ID: {rateLimitRule()},
		passive.ID:  {budgetRule()},
	}
	tr := makePolicyResolver(t, []domain.Policy{enforced, passive}, rules)

	snapshot, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	if len(snapshot.Rules) != 1 {
		t.Fatalf("expected only enforced policy rules, got %d", len(snapshot.Rules))
	}
	if _, ok := snapshot.Budgets["org:budget"]; ok {
		t.Fatal("expected budget rule of non-enforced policy to be excluded")
	}
}

func TestPolicyResolver_DisabledRuleExcluded(t *testing.T) {
	policy := enforcedPolicy("infer")
	disabled := rateLimitRule()
	disabled.Enabled = false
	rules := map[uuid.UUID][]domain.PolicyRule{
		policy.ID: {disabled},
	}
	tr := makePolicyResolver(t, []domain.Policy{policy}, rules)

	snapshot, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	if len(snapshot.Rules) != 0 {
		t.Fatalf("expected disabled rule to be excluded, got %d rules", len(snapshot.Rules))
	}
	if len(snapshot.RateLimits) != 0 {
		t.Fatalf("expected disabled rate limit rule to be excluded, got %d", len(snapshot.RateLimits))
	}
}

func TestPolicyResolver_EmptyOrg(t *testing.T) {
	tr := makePolicyResolver(t, nil, nil)

	snapshot, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	if len(snapshot.Rules) != 0 || len(snapshot.RateLimits) != 0 || len(snapshot.Budgets) != 0 {
		t.Fatalf("expected empty snapshot for org without policies: %+v", snapshot)
	}
}

func TestPolicyResolver_CachesAndInvalidates(t *testing.T) {
	policy := enforcedPolicy("infer")
	rulesByPolicy := map[uuid.UUID][]domain.PolicyRule{policy.ID: {rateLimitRule()}}

	policyCalls := 0
	tr := NewPolicyResolver(
		&stubPolicyRepo{listFn: func(context.Context, uuid.UUID) ([]domain.Policy, error) {
			policyCalls++
			return []domain.Policy{policy}, nil
		}},
		&stubPolicyRuleRepo{listFn: func(_ context.Context, _ uuid.UUID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
			return rulesByPolicy[policyID], nil
		}},
	)

	ctx := context.Background()
	orgID := mustUUIDs[0]

	first, err := tr.GetSnapshot(ctx, orgID)
	requireNoErr(t, err)
	second, err := tr.GetSnapshot(ctx, orgID)
	requireNoErr(t, err)

	if policyCalls != 1 {
		t.Fatalf("expected cache hit on second snapshot (1 policy call), got %d", policyCalls)
	}
	if first.Version != second.Version {
		t.Fatal("expected identical cached snapshot on second call")
	}

	tr.Invalidate(orgID)
	if _, err := tr.GetSnapshot(ctx, orgID); err != nil {
		t.Fatalf("expected rebuild after invalidate: %v", err)
	}
	if policyCalls != 2 {
		t.Fatalf("expected rebuild after invalidate (2 policy calls), got %d", policyCalls)
	}
}

func TestPolicyResolver_MalformedRuleJSONNoPanic(t *testing.T) {
	policy := enforcedPolicy("infer")
	rule := domain.PolicyRule{
		ID:         mustUUIDs[1],
		PolicyID:   mustUUIDs[0],
		RuleType:   domain.RuleTypeContentFilter,
		Enabled:    true,
		Conditions: []byte(`not-json`),
		Parameters: []byte(`not-json`),
	}
	rules := map[uuid.UUID][]domain.PolicyRule{policy.ID: {rule}}
	tr := makePolicyResolver(t, []domain.Policy{policy}, rules)

	snapshot, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	if len(snapshot.Rules) != 1 {
		t.Fatalf("expected rule preserved despite malformed JSON, got %d", len(snapshot.Rules))
	}
	if len(snapshot.Rules[0].Conditions) != 0 || len(snapshot.Rules[0].Actions) != 0 {
		t.Fatal("expected empty condition/action maps for malformed JSON")
	}
}

func TestPolicyResolver_RepoErrorPropagates(t *testing.T) {
	tr := NewPolicyResolver(
		&stubPolicyRepo{listFn: func(context.Context, uuid.UUID) ([]domain.Policy, error) {
			return nil, errStub
		}},
		&stubPolicyRuleRepo{listFn: func(context.Context, uuid.UUID, uuid.UUID) ([]domain.PolicyRule, error) {
			return nil, nil
		}},
	)

	_, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireErrIs(t, err, errStub)
}

func TestPolicyResolver_RuleRepoErrorPropagates(t *testing.T) {
	tr := NewPolicyResolver(
		&stubPolicyRepo{listFn: func(context.Context, uuid.UUID) ([]domain.Policy, error) {
			return []domain.Policy{enforcedPolicy("infer")}, nil
		}},
		&stubPolicyRuleRepo{listFn: func(context.Context, uuid.UUID, uuid.UUID) ([]domain.PolicyRule, error) {
			return nil, errStub
		}},
	)

	_, err := tr.GetSnapshot(context.Background(), mustUUIDs[0])
	requireErrIs(t, err, errStub)
}

func TestPolicyCache_TTLExpiry(t *testing.T) {
	now := time.Now()
	cache := NewPolicyCache(10, time.Minute)
	cache.clock = func() time.Time { return now }

	cache.Set(mustUUIDs[0], PolicySnapshot{OrganizationID: mustUUIDs[0]})

	if cached := cache.Get(mustUUIDs[0]); cached == nil {
		t.Fatal("expected cache hit before TTL")
	}

	now = now.Add(61 * time.Second)
	if cached := cache.Get(mustUUIDs[0]); cached != nil {
		t.Fatal("expected cache miss after TTL")
	}
}

func TestPolicyCache_EvictsWhenFull(t *testing.T) {
	cache := NewPolicyCache(1, time.Minute)
	first := mustUUIDs[0]
	second := mustUUIDs[1]

	cache.Set(first, PolicySnapshot{OrganizationID: first})
	cache.Set(second, PolicySnapshot{OrganizationID: second})

	if cached := cache.Get(second); cached == nil {
		t.Fatal("expected most recent snapshot to survive eviction")
	}
	if cached := cache.Get(first); cached != nil {
		t.Fatal("expected first snapshot to be evicted when the cache is full")
	}
}

func TestPolicyCache_Invalidate(t *testing.T) {
	cache := NewPolicyCache(10, time.Minute)
	cache.Set(mustUUIDs[0], PolicySnapshot{OrganizationID: mustUUIDs[0]})
	cache.Invalidate(mustUUIDs[0])

	if cached := cache.Get(mustUUIDs[0]); cached != nil {
		t.Fatal("expected nil after invalidate")
	}
}