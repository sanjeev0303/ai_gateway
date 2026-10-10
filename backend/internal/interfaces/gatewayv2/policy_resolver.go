// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

// PolicyResolverImpl provides immutable policy snapshots from an in-memory cache.
type PolicyResolverImpl struct {
	policyRepo repositories.PolicyRepository
	ruleRepo   repositories.PolicyRuleRepository
	cache      *PolicyCache
	clock      func() time.Time
}

// PolicyCache caches compiled policy snapshots per organization.
type PolicyCache struct {
	mu        sync.RWMutex
	snapshots map[uuid.UUID]*CachedSnapshot
	maxSize   int
	ttl       time.Duration
	clock     func() time.Time
}

type CachedSnapshot struct {
	Snapshot  PolicySnapshot
	ExpiresAt time.Time
	Version   int64
}

// NewPolicyResolver creates a new policy resolver.
func NewPolicyResolver(
	policyRepo repositories.PolicyRepository,
	ruleRepo repositories.PolicyRuleRepository,
) *PolicyResolverImpl {
	return &PolicyResolverImpl{
		policyRepo: policyRepo,
		ruleRepo:   ruleRepo,
		cache:      NewPolicyCache(1000, 5*time.Minute),
		clock:      time.Now,
	}
}

// NewPolicyCache creates a new policy cache.
func NewPolicyCache(maxSize int, ttl time.Duration) *PolicyCache {
	return &PolicyCache{
		snapshots: make(map[uuid.UUID]*CachedSnapshot, maxSize),
		maxSize:   maxSize,
		ttl:       ttl,
		clock:     time.Now,
	}
}

// GetSnapshot returns the immutable policy snapshot for an organization.
func (p *PolicyResolverImpl) GetSnapshot(ctx context.Context, orgID uuid.UUID) (PolicySnapshot, error) {
	// Try cache first
	if cached := p.cache.Get(orgID); cached != nil {
		return *cached, nil
	}

	// Build snapshot from database
	snapshot, err := p.buildSnapshot(ctx, orgID)
	if err != nil {
		return PolicySnapshot{}, err
	}

	// Cache the snapshot
	p.cache.Set(orgID, snapshot)

	return snapshot, nil
}

// buildSnapshot compiles a policy snapshot from the database.
func (p *PolicyResolverImpl) buildSnapshot(ctx context.Context, orgID uuid.UUID) (PolicySnapshot, error) {
	policies, err := p.policyRepo.List(ctx, orgID)
	if err != nil {
		return PolicySnapshot{}, err
	}

	var allRules []PolicyRule
	rateLimits := make(map[string]RateLimitConfig)
	budgets := make(map[string]BudgetConfig)

	for _, policy := range policies {
		if policy.Enforcement != "active" {
			continue
		}

		rules, err := p.ruleRepo.List(ctx, orgID, policy.ID)
		if err != nil {
			return PolicySnapshot{}, err
		}

		for _, rule := range rules {
			if !rule.Enabled {
				continue
			}

			// Parse JSON conditions and parameters
			var conditions map[string]any
			if len(rule.Conditions) > 0 {
				if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
					conditions = make(map[string]any)
				}
			}

			var parameters map[string]any
			if len(rule.Parameters) > 0 {
				if err := json.Unmarshal(rule.Parameters, &parameters); err != nil {
					parameters = make(map[string]any)
				}
			}

			allRules = append(allRules, PolicyRule{
				ID:         rule.ID.String(),
				Type:       rule.RuleType,
				Conditions: conditions,
				Actions:    parameters,
				Enabled:    rule.Enabled,
			})

			// Extract rate limit and budget configs from rule
			p.extractConfigs(conditions, parameters, &rateLimits, &budgets, rule.RuleType)
		}
	}

	// Sort rules by priority (higher first)
	// ... sort implementation

	return PolicySnapshot{
		OrganizationID: orgID,
		Version:        time.Now().UnixNano(),
		Rules:          allRules,
		RateLimits:     rateLimits,
		Budgets:        budgets,
		EnforcedAt:     time.Now(),
	}, nil
}

// extractConfigs extracts rate limit and budget configs from a rule.
func (p *PolicyResolverImpl) extractConfigs(conditions, parameters map[string]any, rateLimits *map[string]RateLimitConfig, budgets *map[string]BudgetConfig, ruleType string) {
	switch ruleType {
	case "rate_limit":
		if key, ok := conditions["key"].(string); ok {
			(*rateLimits)[key] = RateLimitConfig{
				Key:        key,
				Limit:      int(toFloat64(parameters["limit"])),
				Window:     time.Duration(toFloat64(parameters["window_seconds"])) * time.Second,
				Scope:      toString(parameters["scope"]),
				BurstAllow: int(toFloat64(parameters["burst"])),
			}
		}
	case "budget":
		if key, ok := conditions["key"].(string); ok {
			(*budgets)[key] = BudgetConfig{
				Key:     key,
				Limit:   toFloat64(parameters["limit"]),
				Window:  time.Duration(toFloat64(parameters["window_seconds"])) * time.Second,
				Scope:   toString(parameters["scope"]),
				AlertAt: toFloat64(parameters["alert_at"]),
			}
		}
	}
}

// toFloat64 safely converts various numeric types to float64
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// toString safely converts various types to string
func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// Invalidate removes a policy snapshot from the cache.
func (p *PolicyResolverImpl) Invalidate(orgID uuid.UUID) {
	p.cache.Invalidate(orgID)
}

// PolicyCache methods with pointer receivers
func (c *PolicyCache) Get(orgID uuid.UUID) *PolicySnapshot {
	c.mu.RLock()
	entry, ok := c.snapshots[orgID]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	if c.clock().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.snapshots, orgID)
		c.mu.Unlock()
		return nil
	}

	return &entry.Snapshot
}

func (c *PolicyCache) Set(orgID uuid.UUID, snapshot PolicySnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.snapshots) >= c.maxSize {
		for k := range c.snapshots {
			delete(c.snapshots, k)
			break
		}
	}

	now := c.clock()
	c.snapshots[orgID] = &CachedSnapshot{
		Snapshot:  snapshot,
		ExpiresAt: now.Add(c.ttl),
		Version:   now.UnixNano(),
	}
}

func (c *PolicyCache) Invalidate(orgID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.snapshots, orgID)
}
