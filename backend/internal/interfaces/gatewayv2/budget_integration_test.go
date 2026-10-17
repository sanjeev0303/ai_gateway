// Integration test for the complete budget flow:
// 1. Budget created via control API
// 2. Backend persists budget and creates policy budget rules
// 3. Gateway loads budget from policy snapshot
// 4. Request accepted below limit
// 5. Request rejected above limit
package gatewayv2

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/budget"
	"ai-governance-saas/backend/internal/domain"
	repos "ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
	ports "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/testutil"
)

func TestIntegration_BudgetFlow_ControlAPIToGatewayEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start embedded Postgres
	tp, err := testutil.StartPostgres(ctx, true)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}
	defer tp.Stop()

	store := tp.Store()

	// Create test organization and actor
	orgRepo := repos.NewOrganizationRepository(store)
	org, err := orgRepo.Create(ctx, domain.NewOrganization{
		Name: "Test Org",
		Slug: "test-org-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	actor := domain.Actor{
		UserID:         uuid.New(),
		OrganizationID: org.ID,
		Kind:           domain.ActorKindUser,
	}

	// Setup repositories
	budgetRepo := repos.NewBudgetRepository(store)
	policyRepo := repos.NewPolicyRepository(store)
	ruleRepo := repos.NewPolicyRuleRepository(store)
	auditRepo := repos.NewAuditLogRepository(store)

	// Policy invalidator for snapshot invalidation
	var invalidatedOrg uuid.UUID
	invalidator := &testInvalidator{invalidatedOrg: &invalidatedOrg}

	// Create budget service
	budgetSvc := budget.NewService(budget.ServiceDeps{
		Budgets:     budgetRepo,
		Policies:    policyRepo,
		Rules:       ruleRepo,
		Audits:      auditRepo,
		Invalidator: invalidator,
	})

	// Test 1: Create budget via service
	t.Run("Create budget persists to database", func(t *testing.T) {
		newBudget := domain.NewBudget{
			Name:           "Test Monthly Budget",
			Description:    "Test budget for integration test",
			Scope:          domain.BudgetScopeOrganization,
			ScopeID:        uuid.Nil,
			LimitType:      domain.BudgetLimitTypeSpend,
			LimitAmount:    100.0,
			Window:         domain.BudgetWindowMonthly,
			AlertThreshold: 0.8,
			Enabled:        true,
		}

		created, err := budgetSvc.Create(ctx, actor, newBudget)
		if err != nil {
			t.Fatalf("create budget: %v", err)
		}

		t.Logf("created budget: %v", created)

		if created.ID == uuid.Nil {
			t.Fatal("expected budget ID to be set")
		}
		if created.LimitAmount != 100.0 {
			t.Fatalf("expected limit 100, got %v", created.LimitAmount)
		}
		if !created.Enabled {
			t.Fatal("expected budget to be enabled")
		}

		// Check if policy was created
		policies, err := policyRepo.List(ctx, org.ID)
		if err != nil {
			t.Fatalf("list policies after create: %v", err)
		}
		t.Logf("policies after create: %v", policies)

		for _, p := range policies {
			if p.Name == "Budget Enforcement" {
				rules, err := ruleRepo.List(ctx, org.ID, p.ID)
				if err != nil {
					t.Fatalf("list rules after create: %v", err)
				}
				t.Logf("rules after create: %v", rules)
				break
			}
		}
	})

	// Test persistence between subtests
	t.Run("Data persists between subtests", func(t *testing.T) {
		policies, err := policyRepo.List(ctx, org.ID)
		if err != nil {
			t.Fatalf("list policies in subtest 2: %v", err)
		}
		t.Logf("policies in subtest 2: %v", policies)
		if len(policies) == 0 {
			t.Fatal("expected policies to persist between subtests")
		}
	})

	// Test 2: Budget syncs to policy rules
	t.Run("Budget sync creates policy budget rule", func(t *testing.T) {
		// Find the budget policy
		policies, err := policyRepo.List(ctx, org.ID)
		if err != nil {
			t.Fatalf("list policies: %v", err)
		}
		t.Logf("policies: %v", policies)

		var budgetPolicy *domain.Policy
		for i := range policies {
			if policies[i].Name == "Budget Enforcement" {
				budgetPolicy = &policies[i]
				break
			}
		}

		if budgetPolicy == nil {
			t.Fatal("expected Budget Enforcement policy to be created")
		}
		t.Logf("budgetPolicy: %v", budgetPolicy)

		// Check rules on the budget policy
		rules, err := ruleRepo.List(ctx, org.ID, budgetPolicy.ID)
		if err != nil {
			t.Fatalf("list rules: %v", err)
		}
		t.Logf("rules: %v", rules)

		found := false
		for _, rule := range rules {
			if rule.RuleType == domain.RuleTypeBudget {
				var cond map[string]any
				if err := json.Unmarshal(rule.Conditions, &cond); err != nil {
					continue
				}
				t.Logf("rule conditions: %v", cond)
				if cond["type"] == "budget" {
					found = true
					break
				}
			}
		}

		if !found {
			t.Fatal("expected budget rule to be created on Budget Enforcement policy")
		}
	})

	// Test 3: Policy resolver loads budget from snapshot
	t.Run("Policy resolver extracts budget config from snapshot", func(t *testing.T) {
		policyResolver := NewPolicyResolver(policyRepo, ruleRepo)

		snapshot, err := policyResolver.GetSnapshot(ctx, org.ID)
		if err != nil {
			t.Fatalf("get snapshot: %v", err)
		}
		t.Logf("snapshot.Budgets: %v", snapshot.Budgets)
		for k, v := range snapshot.Budgets {
			t.Logf("  key=%s limit=%v scope=%s", k, v.Limit, v.Scope)
		}

		if len(snapshot.Budgets) == 0 {
			t.Fatal("expected budgets in snapshot")
		}

		var foundBudgetConfig BudgetConfig
		for _, config := range snapshot.Budgets {
			if config.Scope == "organization" {
				foundBudgetConfig = config
				break
			}
		}

		if foundBudgetConfig.Limit != 100.0 {
			t.Fatalf("expected budget limit 100, got %v", foundBudgetConfig.Limit)
		}
		if foundBudgetConfig.Scope != "organization" {
			t.Fatalf("expected scope organization, got %v", foundBudgetConfig.Scope)
		}
	})

	// Test 4: Gateway budget guard uses snapshot config
	t.Run("Gateway budget guard accepts request under limit", func(t *testing.T) {
		policyResolver := NewPolicyResolver(policyRepo, ruleRepo)
		budgetGuard := NewBudgetGuardWithResolver(policyResolver)

		// Build initial snapshot to populate budgets
		snapshot, err := policyResolver.GetSnapshot(ctx, org.ID)
		if err != nil {
			t.Fatalf("get snapshot: %v", err)
		}
		t.Logf("snapshot.Budgets keys: %v", snapshot.Budgets)
		for k, v := range snapshot.Budgets {
			t.Logf("  key=%s limit=%v scope=%s", k, v.Limit, v.Scope)
		}
		budgetGuard.SyncFromSnapshot(snapshot)

		// Check budget with key matching organization scope
		result := budgetGuard.Check(ctx, "org:"+org.ID.String(), 0, 0, 50.0)
		if !result.Allowed {
			t.Fatal("expected request under limit to be allowed")
		}
		if result.Limit != 100.0 {
			t.Fatalf("expected limit 100, got %v", result.Limit)
		}
	})

	// Test 5: Gateway budget guard rejects request over limit
	t.Run("Gateway budget guard rejects request over limit", func(t *testing.T) {
		policyResolver := NewPolicyResolver(policyRepo, ruleRepo)
		budgetGuard := NewBudgetGuardWithResolver(policyResolver)

		snapshot, err := policyResolver.GetSnapshot(ctx, org.ID)
		if err != nil {
			t.Fatalf("get snapshot: %v", err)
		}
		budgetGuard.SyncFromSnapshot(snapshot)

		// First, record some usage to simulate prior requests
		if err := budgetGuard.RecordUsage(ctx, "org:"+org.ID.String(), 50.0); err != nil {
			t.Fatalf("record usage: %v", err)
		}

		// Now check with a request that would exceed the limit (50 + 60 = 110 > 100)
		result := budgetGuard.Check(ctx, "org:"+org.ID.String(), 0, 0, 60.0)
		if result.Allowed {
			t.Fatal("expected request over limit to be denied")
		}
	})

	// Test 6: Budget update syncs to policy rules
	t.Run("Budget update syncs to policy rules", func(t *testing.T) {
		budgets, err := budgetRepo.List(ctx, org.ID, ports.BudgetFilter{})
		if err != nil {
			t.Fatalf("list budgets: %v", err)
		}
		if len(budgets) != 1 {
			t.Fatalf("expected 1 budget, got %d", len(budgets))
		}

		updated, err := budgetSvc.Update(ctx, actor, budgets[0].ID, domain.UpdateBudget{
			LimitAmount: float64Ptr(200.0),
		})
		if err != nil {
			t.Fatalf("update budget: %v", err)
		}

		if updated.LimitAmount != 200.0 {
			t.Fatalf("expected updated limit 200, got %v", updated.LimitAmount)
		}

		// Verify policy snapshot reflects the change
		policyResolver := NewPolicyResolver(policyRepo, ruleRepo)
		invalidator.Invalidate(org.ID) // Trigger snapshot rebuild

		snapshot, err := policyResolver.GetSnapshot(ctx, org.ID)
		if err != nil {
			t.Fatalf("get snapshot after update: %v", err)
		}

		var foundBudgetConfig BudgetConfig
		for _, config := range snapshot.Budgets {
			if config.Scope == "organization" {
				foundBudgetConfig = config
				break
			}
		}

		if foundBudgetConfig.Limit != 200.0 {
			t.Fatalf("expected updated budget limit 200 in snapshot, got %v", foundBudgetConfig.Limit)
		}
	})

	// Test 7: Budget delete removes policy rule
	t.Run("Budget delete removes policy rule", func(t *testing.T) {
		budgets, err := budgetRepo.List(ctx, org.ID, ports.BudgetFilter{})
		if err != nil {
			t.Fatalf("list budgets: %v", err)
		}
		if len(budgets) != 1 {
			t.Fatalf("expected 1 budget, got %d", len(budgets))
		}

		err = budgetSvc.Delete(ctx, actor, budgets[0].ID)
		if err != nil {
			t.Fatalf("delete budget: %v", err)
		}

		// Verify budget is gone
		budgets, err = budgetRepo.List(ctx, org.ID, ports.BudgetFilter{})
		if err != nil {
			t.Fatalf("list budgets after delete: %v", err)
		}
		if len(budgets) != 0 {
			t.Fatalf("expected 0 budgets after delete, got %d", len(budgets))
		}

		// Verify policy rule is cleaned up
		invalidator.Invalidate(org.ID)
		policyResolver := NewPolicyResolver(policyRepo, ruleRepo)
		snapshot, err := policyResolver.GetSnapshot(ctx, org.ID)
		if err != nil {
			t.Fatalf("get snapshot after delete: %v", err)
		}

		orgBudgetCount := 0
		for _, config := range snapshot.Budgets {
			if config.Scope == "organization" {
				orgBudgetCount++
			}
		}
		if orgBudgetCount != 0 {
			t.Fatalf("expected 0 organization budgets in snapshot after delete, got %d", orgBudgetCount)
		}
	})
}

// Test full gateway pipeline with budget enforcement
func TestIntegration_GatewayPipeline_BudgetEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	tp, err := testutil.StartPostgres(ctx, true)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}
	defer tp.Stop()

	store := tp.Store()

	// Create test organization
	orgRepo := repos.NewOrganizationRepository(store)
	org, err := orgRepo.Create(ctx, domain.NewOrganization{
		Name: "Test Org",
		Slug: "test-org-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	actor := domain.Actor{
		UserID:         uuid.New(),
		OrganizationID: org.ID,
		Kind:           domain.ActorKindUser,
	}

	// Setup repositories
	budgetRepo := repos.NewBudgetRepository(store)
	policyRepo := repos.NewPolicyRepository(store)
	ruleRepo := repos.NewPolicyRuleRepository(store)
	auditRepo := repos.NewAuditLogRepository(store)
	credRepo := repos.NewGatewayCredentialRepository(store)
	accountRepo := repos.NewProviderAccountRepository(store)
	providerCredRepo := repos.NewProviderCredentialRepository(store)

	var invalidatedOrg uuid.UUID
	invalidator := &testInvalidator{invalidatedOrg: &invalidatedOrg}

	budgetSvc := budget.NewService(budget.ServiceDeps{
		Budgets:     budgetRepo,
		Policies:    policyRepo,
		Rules:       ruleRepo,
		Audits:      auditRepo,
		Invalidator: invalidator,
	})

	// Create a budget
	_, err = budgetSvc.Create(ctx, actor, domain.NewBudget{
		Name:           "Gateway Budget",
		Description:    "Budget for gateway enforcement test",
		Scope:          domain.BudgetScopeOrganization,
		ScopeID:        uuid.Nil,
		LimitType:      domain.BudgetLimitTypeSpend,
		LimitAmount:    0.1,
		Window:         domain.BudgetWindowHourly,
		AlertThreshold: 0.8,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create budget: %v", err)
	}

	// Create gateway credential
	hasher := &stubHasher{}
	cred, err := credRepo.Create(ctx, org.ID, domain.NewGatewayCredential{
		Name:        "test-cred",
		KeyPrefix:   "aigw_test",
		KeyHash:     "hash:aigw_test.secret",
		SubjectType: domain.SubjectTypeUser,
		SubjectID:   actor.UserID,
		Scopes:      []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
		ExpiresAt:   nil,
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}

	// Create mock upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Send mock usage back so budget is decremented
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "msg_123",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]string{{"type": "text", "text": "Hello"}},
			"model":   "claude-3-opus-20240229",
			"usage": map[string]interface{}{
				"input_tokens":  1200,
				"output_tokens": 450,
			},
		})
	}))
	defer upstream.Close()

	// Create provider account
	account, err := accountRepo.Create(ctx, org.ID, domain.NewProviderAccount{
		ProviderKind: "anthropic",
		Name:         "test-account",
		BaseURL:      upstream.URL,
	})
	if err != nil {
		t.Fatalf("create provider account: %v", err)
	}

	// Create provider credential
	_, err = providerCredRepo.Create(ctx, org.ID, domain.NewProviderCredential{
		ProviderAccountID: account.ID,
		Name:              "test-provider-cred",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "vault:ref",
		Fingerprint:       "fp",
		Tail:              "tail",
	})
	if err != nil {
		t.Fatalf("create provider credential: %v", err)
	}

	// Build gateway
	gateway, err := NewGateway(GatewayDeps{
		Logger:              slog.Default(),
		Hasher:              hasher,
		GatewayCredRepo:     credRepo,
		ProviderAccounts:    accountRepo,
		ProviderCredentials: providerCredRepo,
		Secrets:             &stubSecretStore{values: map[string]string{"vault:ref": "test-secret"}},
		PolicyRepo:          policyRepo,
		RuleRepo:            ruleRepo,
	})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}

	if err := gateway.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer gateway.Shutdown(0)

	// Wait for policy snapshot to be built
	time.Sleep(100 * time.Millisecond)

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req.Header.Set("x-api-key", cred.KeyPrefix+".secret")
	req.Header.Set("Content-Type", "application/json")

	// First request should be allowed (under budget)
	rec := httptest.NewRecorder()
	gateway.ServeHTTP(rec, req)
	if rec.Code == http.StatusPaymentRequired {
		t.Fatal("first request should be allowed under budget")
	}
	t.Logf("First request status: %d", rec.Code)

	// Make multiple requests to exceed budget
	for i := 0; i < 10; i++ {
		rec = httptest.NewRecorder()
		gateway.ServeHTTP(rec, req)
		if rec.Code == http.StatusPaymentRequired {
			t.Logf("Request %d rejected with 402 (budget exceeded)", i+1)
			break
		}
		if i == 9 {
			t.Fatal("expected budget to be exceeded after multiple requests")
		}
	}
}

// testInvalidator implements BudgetInvalidator/PolicyInvalidator for testing
type testInvalidator struct {
	invalidatedOrg *uuid.UUID
}

func (i *testInvalidator) Invalidate(orgID uuid.UUID) {
	*i.invalidatedOrg = orgID
}

func float64Ptr(f float64) *float64 {
	return &f
}
