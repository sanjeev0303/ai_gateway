package gatewayv2

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/logging"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

type memoryGuardrailsTestRepo struct {
	items  map[string]domain.Guardrail
	events []domain.GuardrailEvent
	mu     sync.Mutex
}

func newMemoryGuardrailsTestRepo() *memoryGuardrailsTestRepo {
	return &memoryGuardrailsTestRepo{
		items: make(map[string]domain.Guardrail),
	}
}

func (m *memoryGuardrailsTestRepo) Create(_ context.Context, g domain.Guardrail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[g.ID] = g
	return nil
}

func (m *memoryGuardrailsTestRepo) Get(_ context.Context, orgID, id string) (domain.Guardrail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.items[id]
	if !ok || g.OrganizationID != orgID {
		return domain.Guardrail{}, repositories.ErrNotFound
	}
	return g, nil
}

func (m *memoryGuardrailsTestRepo) List(_ context.Context, orgID string) ([]domain.Guardrail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Guardrail
	for _, g := range m.items {
		if g.OrganizationID == orgID {
			list = append(list, g)
		}
	}
	return list, nil
}

func (m *memoryGuardrailsTestRepo) ListForEvaluation(_ context.Context, orgID string, target domain.GuardrailTargetType, targetID string) ([]domain.Guardrail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Guardrail
	for _, g := range m.items {
		if g.OrganizationID == orgID && g.TargetType == target {
			if targetID == "" || (g.TargetID != nil && *g.TargetID == targetID) {
				list = append(list, g)
			}
		}
	}
	return list, nil
}

func (m *memoryGuardrailsTestRepo) Update(_ context.Context, g domain.Guardrail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[g.ID] = g
	return nil
}

func (m *memoryGuardrailsTestRepo) Delete(_ context.Context, orgID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

func (m *memoryGuardrailsTestRepo) CreateEvent(_ context.Context, ev domain.GuardrailEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, ev)
	return nil
}

func (m *memoryGuardrailsTestRepo) ListEvents(_ context.Context, orgID string) ([]domain.GuardrailEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.GuardrailEvent
	for _, e := range m.events {
		if e.OrganizationID == orgID {
			list = append(list, e)
		}
	}
	return list, nil
}

// ----------------------------------------------------------------------------
// 1. Policies and Model Access Tests
// ----------------------------------------------------------------------------

func TestGovernance_PolicyPrecedenceAndOverrides(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	orgID := stubs.auth.cred.OrganizationID
	userID := uuid.New()
	stubs.auth.cred.SubjectType = "user"
	stubs.auth.cred.SubjectID = userID

	// Scenario: Org allows gpt-4o and claude-sonnet-4-5, but user-specific blocklist blocks gpt-4o.
	// User rule (rank 4) should override org rule (rank 1).
	stubs.policy.snapshot = PolicySnapshot{
		OrganizationID: orgID,
		Rules: []PolicyRule{
			{
				ID:       "rule-org-allow",
				Type:     domain.RuleTypeAllowlist,
				Scope:    "organization",
				ScopeID:  orgID.String(),
				Actions:  map[string]any{"models": []any{"claude-sonnet-4-5", "gpt-4o"}},
				Enabled:  true,
				Priority: 10,
			},
			{
				ID:       "rule-user-block",
				Type:     domain.RuleTypeBlocklist,
				Scope:    "user",
				ScopeID:  userID.String(),
				Actions:  map[string]any{"models": []any{"gpt-4o"}},
				Enabled:  true,
				Priority: 1,
			},
		},
	}

	// 1. Request for gpt-4o (blocked by user rule)
	stubs.inspect.hints.Model = "gpt-4o"
	req := pipelineRequest("aigw_valid_key", `{"model":"gpt-4o"}`)
	res := g.executePipeline(context.Background(), req, "req-pol-1")

	if res.Allowed {
		t.Fatal("expected gpt-4o to be blocked by user-level policy rule")
	}
	if res.StatusCode != http.StatusForbidden || res.ErrorType != "policy_denied" {
		t.Fatalf("expected 403 policy_denied, got %d / %s", res.StatusCode, res.ErrorType)
	}

	// 2. Request for claude-sonnet-4-5 (allowed by org rule, not blocked by user)
	stubs.inspect.hints.Model = "claude-sonnet-4-5"
	reqAllowed := pipelineRequest("aigw_valid_key", `{"model":"claude-sonnet-4-5"}`)
	resAllowed := g.executePipeline(context.Background(), reqAllowed, "req-pol-2")
	if !resAllowed.Allowed {
		t.Fatalf("expected claude-sonnet-4-5 to pass policy check, denied: %s", resAllowed.ErrorMessage)
	}

	// 3. Request for unsupported model (when policy does not restrict models, fails model_resolver with 400 Bad Request)
	stubs.policy.snapshot = PolicySnapshot{OrganizationID: orgID}
	stubs.inspect.hints.Model = "unsupported-random-llm-v1"
	reqUnsupported := pipelineRequest("aigw_valid_key", `{"model":"unsupported-random-llm-v1"}`)
	resUnsupported := g.executePipeline(context.Background(), reqUnsupported, "req-pol-3")
	if resUnsupported.Allowed {
		t.Fatal("expected unsupported model to be rejected")
	}
	if resUnsupported.StatusCode != http.StatusBadRequest || resUnsupported.ErrorType != "unsupported_model" {
		t.Fatalf("expected 400 unsupported_model, got %d / %s", resUnsupported.StatusCode, resUnsupported.ErrorType)
	}
}

func TestGovernance_PolicyInvalidationTimeBound(t *testing.T) {
	cache := NewPolicyCache(100, 5*time.Minute)
	orgID := uuid.New()

	initialSnapshot := PolicySnapshot{
		OrganizationID: orgID,
		Version:        1,
		Rules: []PolicyRule{
			{ID: "r-1", Type: domain.RuleTypeAllowlist, Actions: map[string]any{"models": []any{"gpt-4o"}}, Enabled: true},
		},
	}

	// Prime cache
	cache.Set(orgID, initialSnapshot)

	// Verify cache hit
	cached := cache.Get(orgID)
	if cached == nil || cached.Version != 1 {
		t.Fatal("expected cached version 1 snapshot")
	}

	// Invalidate immediately (< 1ms bound)
	start := time.Now()
	cache.Invalidate(orgID)
	duration := time.Since(start)

	if duration > 10*time.Millisecond {
		t.Fatalf("invalidation exceeded 10ms bound: %v", duration)
	}

	// Confirm cache eviction
	if evicted := cache.Get(orgID); evicted != nil {
		t.Fatalf("expected snapshot to be evicted after invalidation, found: %v", evicted)
	}
}

// ----------------------------------------------------------------------------
// 2. Budgets and Rate Limits Tests
// ----------------------------------------------------------------------------

func TestGovernance_BudgetEnforcementAndZeroAccrualOnRejection(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()
	budgetKey := "org:" + uuid.New().String()

	limit := 10.0 // $10 limit
	window := 1 * time.Hour

	// 1. Request immediately below limit: $5 cost against $10 limit (Allowed)
	res1 := bg.Check(ctx, budgetKey, limit, window, 5.0)
	if !res1.Allowed {
		t.Fatalf("expected $5 spend to be allowed under $10 limit")
	}
	_ = bg.RecordUsage(ctx, budgetKey, 5.0)

	// 2. Request exactly at remaining limit: $5 cost against $5 remaining (Allowed)
	res2 := bg.Check(ctx, budgetKey, limit, window, 5.0)
	if !res2.Allowed {
		t.Fatalf("expected $5 spend to fit remaining $5 budget")
	}
	_ = bg.RecordUsage(ctx, budgetKey, 5.0)

	// 3. Request above limit: $0.01 cost when spent == $10 (Rejected with 402)
	res3 := bg.Check(ctx, budgetKey, limit, window, 0.01)
	if res3.Allowed {
		t.Fatal("expected request above budget to be rejected")
	}

	// 4. Verify rejected requests DO NOT accrue usage
	// Attempt to record 0 or verify spent has not increased
	spent, currentLimit, _, ok := bg.GetBudget(budgetKey)
	if !ok || spent != 10.0 || currentLimit != 10.0 {
		t.Fatalf("expected spent to stay at exactly 10.0, got spent=%.2f, limit=%.2f", spent, currentLimit)
	}

	// 5. Test window reset
	mockNow := time.Now().Add(2 * time.Hour)
	bg.clock = func() time.Time { return mockNow }

	resAfterReset := bg.Check(ctx, budgetKey, limit, window, 5.0)
	if !resAfterReset.Allowed {
		t.Fatal("expected budget to reset and allow requests in fresh window")
	}
}

func TestGovernance_RateLimiterConcurrencyRaceDefense(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Close()

	ctx := context.Background()
	rateKey := "tenant:rate:" + uuid.New().String()
	limit := 50
	window := 1 * time.Second

	var allowedCount int64
	var deniedCount int64

	var wg sync.WaitGroup
	concurrentClients := 100

	for i := 0; i < concurrentClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := rl.Check(ctx, rateKey, limit, window)
			if res.Allowed {
				atomic.AddInt64(&allowedCount, 1)
			} else {
				atomic.AddInt64(&deniedCount, 1)
			}
		}()
	}

	wg.Wait()

	if allowedCount != int64(limit) {
		t.Fatalf("rate limiter race condition: allowed %d, want exactly %d", allowedCount, limit)
	}
	if deniedCount != int64(concurrentClients-limit) {
		t.Fatalf("rate limiter race condition: denied %d, want %d", deniedCount, concurrentClients-limit)
	}
}

// ----------------------------------------------------------------------------
// 3. Guardrails Pre-Forwarding Execution Tests
// ----------------------------------------------------------------------------

func TestGovernance_GuardrailsPreForwardingExecution(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	repo := newMemoryGuardrailsTestRepo()
	g.guardrails = NewGuardrailEvaluator(repo)

	orgID := stubs.auth.cred.OrganizationID

	// Create a blocking PII guardrail
	cfgBytes, _ := json.Marshal(domain.PIIConfiguration{
		Types: []string{"email"},
	})
	_ = repo.Create(context.Background(), domain.Guardrail{
		ID:             "gr-pii",
		OrganizationID: orgID.String(),
		Name:           "Email Guardrail",
		TargetType:     domain.GuardrailTargetGlobal,
		Detector:       domain.GuardrailDetectorPII,
		Action:         domain.GuardrailActionBlock,
		Configuration:  cfgBytes,
		Enabled:        true,
	})

	// 1. Request with PII (email in payload)
	piiPayload := `{"messages":[{"role":"user","content":"Contact me at leaked_user@sentinel.ai"}]}`
	reqPII := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(piiPayload))
	reqPII.Header.Set("Authorization", "Bearer aigw_valid_key")
	wPII := httptest.NewRecorder()

	g.ServeHTTP(wPII, reqPII)

	// Verify blocked with 403 Forbidden
	if wPII.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403 Forbidden on guardrail violation, got %d (body: %s)", wPII.Code, wPII.Body.String())
	}
	var errBody map[string]any
	_ = json.Unmarshal(wPII.Body.Bytes(), &errBody)
	errObj, _ := errBody["error"].(map[string]any)
	if errObj == nil || errObj["type"] != "guardrail_blocked" {
		t.Fatalf("expected error.type guardrail_blocked, got %v", errBody)
	}

	// Verify upstream was NEVER called
	if stubs.proxy.forwardCalled {
		t.Fatal("critical failure: stream proxy was called for blocked guardrail request!")
	}

	// Verify guardrail event was recorded asynchronously
	time.Sleep(50 * time.Millisecond)
	events, _ := repo.ListEvents(context.Background(), orgID.String())
	if len(events) == 0 {
		t.Fatal("expected guardrail event to be recorded in repository")
	}
	if events[0].MatchedCategory != "email" || events[0].Action != domain.GuardrailActionBlock {
		t.Fatalf("unexpected guardrail event details: %+v", events[0])
	}

	// 2. Clean request without PII
	cleanPayload := `{"messages":[{"role":"user","content":"Tell me about distributed consensus protocols"}]}`
	reqClean := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(cleanPayload))
	reqClean.Header.Set("Authorization", "Bearer aigw_valid_key")
	wClean := httptest.NewRecorder()

	g.ServeHTTP(wClean, reqClean)

	// Upstream forward MUST now be called
	if !stubs.proxy.forwardCalled {
		t.Fatal("expected clean request to be forwarded to provider")
	}
}

// ----------------------------------------------------------------------------
// 4. Audit & Security Events Tamper Resistance & Redaction Tests
// ----------------------------------------------------------------------------

func TestGovernance_SensitiveValueRedactionInLogs(t *testing.T) {
	// Verify sensitive keys are detected and redacted
	sensitiveKeys := []string{
		"authorization", "api_key", "secret", "password",
		"signing_key", "cookie", "set-cookie", "token",
	}

	for _, k := range sensitiveKeys {
		if !logging.IsSensitiveKey(k) {
			t.Errorf("expected %q to be recognized as sensitive key", k)
		}
	}

	safeKeys := []string{"request_id", "organization_id", "model", "latency_ms", "status"}
	for _, k := range safeKeys {
		if logging.IsSensitiveKey(k) {
			t.Errorf("expected %q to NOT be recognized as sensitive key", k)
		}
	}
}
