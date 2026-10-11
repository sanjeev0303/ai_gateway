package gatewayv2

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/invalidation"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

type stubPolicyRepoForInvalidation struct {
	mu       sync.Mutex
	policies []domain.Policy
}

func (s *stubPolicyRepoForInvalidation) List(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.Policy(nil), s.policies...), nil
}

func (s *stubPolicyRepoForInvalidation) Create(ctx context.Context, orgID uuid.UUID, p domain.NewPolicy) (domain.Policy, error) {
	return domain.Policy{}, nil
}

func (s *stubPolicyRepoForInvalidation) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Policy, error) {
	return domain.Policy{}, nil
}

func (s *stubPolicyRepoForInvalidation) Update(ctx context.Context, orgID, id uuid.UUID, p domain.UpdatePolicy) (domain.Policy, error) {
	return domain.Policy{}, nil
}

func (s *stubPolicyRepoForInvalidation) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return nil
}

func (s *stubPolicyRepoForInvalidation) SetPolicies(p []domain.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = p
}

type stubRuleRepoForInvalidation struct {
	mu    sync.Mutex
	rules []domain.PolicyRule
}

func (s *stubRuleRepoForInvalidation) List(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.PolicyRule(nil), s.rules...), nil
}

func (s *stubRuleRepoForInvalidation) Create(ctx context.Context, orgID uuid.UUID, r domain.NewPolicyRule) (domain.PolicyRule, error) {
	return domain.PolicyRule{}, nil
}

func (s *stubRuleRepoForInvalidation) Get(ctx context.Context, orgID, policyID, id uuid.UUID) (domain.PolicyRule, error) {
	return domain.PolicyRule{}, nil
}

func (s *stubRuleRepoForInvalidation) Update(ctx context.Context, orgID, policyID, id uuid.UUID, r domain.UpdatePolicyRule) (domain.PolicyRule, error) {
	return domain.PolicyRule{}, nil
}

func (s *stubRuleRepoForInvalidation) Delete(ctx context.Context, orgID, policyID, id uuid.UUID) error {
	return nil
}

func (s *stubRuleRepoForInvalidation) SetRules(r []domain.PolicyRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = r
}

func TestPolicyInvalidation_ImmediateSnapshotBust(t *testing.T) {
	orgID := uuid.New()
	polID := uuid.New()

	policyRepo := &stubPolicyRepoForInvalidation{
		policies: []domain.Policy{
			{
				ID:             polID,
				OrganizationID: orgID,
				Name:           "Default Policy",
				Enforcement:    "active",
			},
		},
	}
	ruleRepo := &stubRuleRepoForInvalidation{
		rules: []domain.PolicyRule{
			{
				ID:             uuid.New(),
				OrganizationID: orgID,
				PolicyID:       polID,
				RuleType:       domain.RuleTypeAllowlist,
				Enabled:        true,
				Conditions:     []byte(`{"models":["claude-3-5-sonnet"]}`),
			},
		},
	}

	resolver := NewPolicyResolver(policyRepo, ruleRepo)
	g := &Gateway{
		policyResolver: resolver,
		logger:         discardLogger(),
	}

	ctx := context.Background()

	// 1. Initial snapshot should compile and cache
	snap1, err := resolver.GetSnapshot(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap1.Rules) != 1 {
		t.Fatalf("expected 1 rule in snap1, got %d", len(snap1.Rules))
	}

	// 2. Mutate repository (add a second rule)
	ruleRepo.SetRules([]domain.PolicyRule{
		{
			ID:             uuid.New(),
			OrganizationID: orgID,
			PolicyID:       polID,
			RuleType:       domain.RuleTypeAllowlist,
			Enabled:        true,
			Conditions:     []byte(`{"models":["claude-3-5-sonnet"]}`),
		},
		{
			ID:             uuid.New(),
			OrganizationID: orgID,
			PolicyID:       polID,
			RuleType:       domain.RuleTypeRateLimit,
			Enabled:        true,
			Conditions:     []byte(`{"limit":100}`),
		},
	})

	// 3. Without invalidation, cache returns the stale snapshot (1 rule)
	cachedSnap, err := resolver.GetSnapshot(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cachedSnap.Rules) != 1 {
		t.Fatalf("expected cached snapshot to have 1 rule before invalidation, got %d", len(cachedSnap.Rules))
	}

	// 4. Invalidate policy snapshot via gateway
	g.InvalidatePolicy(orgID)

	// 5. Immediately, next GetSnapshot fetches fresh data with 2 rules
	snap2, err := resolver.GetSnapshot(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap2.Rules) != 2 {
		t.Fatalf("expected 2 rules in snap2 after invalidation, got %d", len(snap2.Rules))
	}
}

type stubGatewayCredRepoForInvalidation struct {
	mu   sync.Mutex
	cred *domain.GatewayCredential
}

func (s *stubGatewayCredRepoForInvalidation) ResolveByKeyHash(ctx context.Context, hash string) (domain.GatewayCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred == nil {
		return domain.GatewayCredential{}, repositories.ErrNotFound
	}
	return *s.cred, nil
}

func (s *stubGatewayCredRepoForInvalidation) ResolveByKeyPrefix(ctx context.Context, prefix string) ([]domain.GatewayCredential, error) {
	return nil, nil
}

func (s *stubGatewayCredRepoForInvalidation) List(ctx context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error) {
	return nil, nil
}

func (s *stubGatewayCredRepoForInvalidation) Create(ctx context.Context, orgID uuid.UUID, cred domain.NewGatewayCredential) (domain.GatewayCredential, error) {
	return domain.GatewayCredential{}, nil
}

func (s *stubGatewayCredRepoForInvalidation) Get(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred == nil || s.cred.ID != id {
		return domain.GatewayCredential{}, repositories.ErrNotFound
	}
	return *s.cred, nil
}

func (s *stubGatewayCredRepoForInvalidation) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred != nil {
		s.cred.Status = domain.GatewayCredentialStatusRevoked
		return *s.cred, nil
	}
	return domain.GatewayCredential{}, repositories.ErrNotFound
}

func (s *stubGatewayCredRepoForInvalidation) Rotate(ctx context.Context, orgID, id uuid.UUID, newCred domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
	return domain.GatewayCredential{}, "", nil
}

func (s *stubGatewayCredRepoForInvalidation) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status domain.GatewayCredentialStatus) (domain.GatewayCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cred != nil {
		s.cred.Status = status
		return *s.cred, nil
	}
	return domain.GatewayCredential{}, repositories.ErrNotFound
}

func (s *stubGatewayCredRepoForInvalidation) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestCredentialInvalidation_ImmediateRevocation(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	token := "aigw_prod_valid_token"
	keyHash := "hash:" + token

	initialCred := domain.GatewayCredential{
		ID:             credID,
		OrganizationID: orgID,
		Name:           "Production Gateway Key",
		KeyHash:        keyHash,
		KeyPrefix:      "aigw_prod_",
		SubjectType:    domain.SubjectTypeUser,
		SubjectID:      uuid.New(),
		Scopes:         []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
		Status:         domain.GatewayCredentialStatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	repo := &stubGatewayCredRepoForInvalidation{
		cred: &initialCred,
	}

	hasher := &stubHasher{}
	cache := NewInMemoryCredentialCache(100, 5*time.Minute)
	authImpl := NewAuthenticator(hasher, repo, cache)

	g := &Gateway{
		authenticator: authImpl,
		logger:        discardLogger(),
	}

	ctx := context.Background()

	// 1. Initial authentication should succeed and populate cache
	cred, err := authImpl.Authenticate(ctx, "aigw_prod_valid_token")
	if err != nil {
		t.Fatalf("expected successful auth, got: %v", err)
	}
	if cred.ID != credID {
		t.Fatalf("expected cred ID %s, got %s", credID, cred.ID)
	}

	// 2. Revoke credential in repository
	_, _ = repo.Revoke(ctx, orgID, credID)

	// 3. Without invalidation, auth still succeeds from in-memory cache
	cachedAuth, err := authImpl.Authenticate(ctx, "aigw_prod_valid_token")
	if err != nil {
		t.Fatalf("expected cached auth before invalidation to succeed, got: %v", err)
	}
	if cachedAuth.Status != "active" {
		t.Fatalf("expected active from cache, got %s", cachedAuth.Status)
	}

	// 4. Invalidate credential cache entry by ID (as triggered by Redis invalidation subscriber)
	g.InvalidateCredentialByID(ctx, credID.String())

	// 5. Next authentication misses cache, queries DB, and is immediately rejected
	_, err = authImpl.Authenticate(ctx, "aigw_prod_valid_token")
	if err == nil {
		t.Fatalf("expected authentication to fail after revocation and cache invalidation")
	}
	if err != ErrCredentialRevoked {
		t.Fatalf("expected ErrCredentialRevoked, got: %v", err)
	}
}

func TestRedisRateLimiter_ConcurrentSlidingWindowFallback(t *testing.T) {
	// Tests atomic concurrency behavior of the rate limiter fallback
	limiter := NewRedisRateLimiter(nil, discardLogger())
	ctx := context.Background()

	limit := 10
	window := 1 * time.Minute
	key := "test-tenant-rate-limit"

	var wg sync.WaitGroup
	numWorkers := 40
	var allowedCount int64
	var rejectedCount int64

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := limiter.Check(ctx, key, limit, window)
			if res.Allowed {
				atomic.AddInt64(&allowedCount, 1)
			} else {
				atomic.AddInt64(&rejectedCount, 1)
			}
		}()
	}

	wg.Wait()

	if allowedCount != int64(limit) {
		t.Fatalf("expected exactly %d allowed requests, got %d", limit, allowedCount)
	}
	if rejectedCount != int64(numWorkers-limit) {
		t.Fatalf("expected exactly %d rejected requests, got %d", numWorkers-limit, rejectedCount)
	}
}

func TestRedisInvalidator_LocalListener(t *testing.T) {
	inval := invalidation.NewRedisInvalidator(nil, discardLogger())

	orgID := uuid.New()
	var notifiedOrgID uuid.UUID
	var called bool

	inval.AddListener(func(id uuid.UUID) {
		called = true
		notifiedOrgID = id
	})

	inval.Invalidate(orgID)

	if !called {
		t.Fatalf("expected local listener to be called on Invalidate")
	}
	if notifiedOrgID != orgID {
		t.Fatalf("expected notified org %s, got %s", orgID, notifiedOrgID)
	}
}
