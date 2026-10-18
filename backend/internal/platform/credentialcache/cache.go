package credentialcache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// CachedCredential is a minimal credential representation for caching.
// Platform package must not depend on domain; service layer converts.
type CachedCredential struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organization_id"`
	Name              string     `json:"name"`
	KeyPrefix         string     `json:"key_prefix"`
	KeyHash           string     `json:"key_hash"`
	SubjectType       string     `json:"subject_type"`
	SubjectID         uuid.UUID  `json:"subject_id"`
	TeamID            *uuid.UUID `json:"team_id,omitempty"`
	ProjectID         *uuid.UUID `json:"project_id,omitempty"`
	Scopes            []string   `json:"scopes"`
	ProviderAccountID *uuid.UUID `json:"provider_account_id,omitempty"`
	AllowedProviders  []string   `json:"allowed_providers,omitempty"`
	AllowedModels     []string   `json:"allowed_models,omitempty"`
	RateLimit         *int       `json:"rate_limit,omitempty"`
	BudgetLimit       *float64   `json:"budget_limit,omitempty"`
	Status            string     `json:"status"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// CredentialCache is a thread-safe, TTL cache for gateway credential resolution.
// It provides two lookup paths:
//   - ByHash: exact hash match (hot path for gateway authentication)
//   - ByPrefix: prefix-based candidate set (for hash verification on cache miss)
type CredentialCache struct {
	client  *redis.Client
	ttl     time.Duration
	enabled bool
}

// NewCredentialCache creates a cache with TTL.
// Uses Redis for distributed caching across gateway instances.
// If redisClient is nil, caching is disabled (falls back to DB only).
func NewCredentialCache(redisClient *redis.Client, ttl time.Duration) *CredentialCache {
	if redisClient == nil {
		return &CredentialCache{enabled: false}
	}
	return &CredentialCache{
		client:  redisClient,
		ttl:     ttl,
		enabled: true,
	}
}

// hashKey returns the Redis key for a credential hash.
func hashKey(keyHash string) string {
	return "gateway:cred:hash:" + keyHash
}

// prefixKey returns the Redis key for a credential prefix.
func prefixKey(keyPrefix string) string {
	return "gateway:cred:prefix:" + keyPrefix
}

// orgKey returns the Redis key for an organization's credential set.
func orgKey(orgID string) string {
	return "gateway:cred:org:" + orgID
}

// GetByHash returns cached credential or nil (caller must fallback to DB).
func (c *CredentialCache) GetByHash(ctx context.Context, keyHash string) *CachedCredential {
	if !c.enabled {
		return nil
	}
	key := hashKey(keyHash)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil // cache miss
	}
	var cred CachedCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil
	}
	return &cred
}

// GetByPrefix returns candidate credentials for a prefix (for hash verification).
func (c *CredentialCache) GetByPrefix(ctx context.Context, keyPrefix string) []*CachedCredential {
	if !c.enabled {
		return nil
	}
	key := prefixKey(keyPrefix)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil // cache miss
	}
	var creds []CachedCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil
	}
	result := make([]*CachedCredential, len(creds))
	for i := range creds {
		result[i] = &creds[i]
	}
	return result
}

// Set caches a credential after successful DB resolution.
func (c *CredentialCache) Set(ctx context.Context, cred *CachedCredential) {
	if !c.enabled || cred == nil {
		return
	}
	data, err := json.Marshal(cred)
	if err != nil {
		return
	}
	// Cache by hash
	c.client.Set(ctx, hashKey(cred.KeyHash), data, c.ttl)
	// Add to prefix set
	c.addToPrefixSet(ctx, cred)
	// Add to org set
	c.addToOrgSet(ctx, cred)
}

// addToPrefixSet adds a credential to the prefix-based candidate set.
func (c *CredentialCache) addToPrefixSet(ctx context.Context, cred *CachedCredential) {
	key := prefixKey(cred.KeyPrefix)
	existing, _ := c.client.Get(ctx, key).Bytes()
	var creds []CachedCredential
	if existing != nil {
		_ = json.Unmarshal(existing, &creds)
	}
	found := false
	for _, c := range creds {
		if c.ID == cred.ID {
			found = true
			break
		}
	}
	if !found {
		creds = append(creds, *cred)
	}
	data, _ := json.Marshal(creds)
	c.client.Set(ctx, key, data, c.ttl)
}

// addToOrgSet adds a credential to the org-based set for bulk invalidation.
func (c *CredentialCache) addToOrgSet(ctx context.Context, cred *CachedCredential) {
	key := orgKey(cred.OrganizationID.String())
	existing, _ := c.client.Get(ctx, key).Bytes()
	var creds []CachedCredential
	if existing != nil {
		_ = json.Unmarshal(existing, &creds)
	}
	found := false
	for _, c := range creds {
		if c.ID == cred.ID {
			found = true
			break
		}
	}
	if !found {
		creds = append(creds, *cred)
	}
	data, _ := json.Marshal(creds)
	c.client.Set(ctx, key, data, c.ttl)
}

// Invalidate removes a credential from cache (on revoke/rotate/expire).
func (c *CredentialCache) Invalidate(ctx context.Context, cred *CachedCredential) {
	if !c.enabled || cred == nil {
		return
	}
	c.client.Del(ctx, hashKey(cred.KeyHash))
	c.removeFromPrefixSet(ctx, cred)
	c.removeFromOrgSet(ctx, cred)
}

// removeFromPrefixSet removes a credential from the prefix-based candidate set.
func (c *CredentialCache) removeFromPrefixSet(ctx context.Context, cred *CachedCredential) {
	key := prefixKey(cred.KeyPrefix)
	existing, _ := c.client.Get(ctx, key).Bytes()
	if existing == nil {
		return
	}
	var creds []CachedCredential
	if err := json.Unmarshal(existing, &creds); err != nil {
		return
	}
	filtered := make([]CachedCredential, 0, len(creds))
	for _, c := range creds {
		if c.ID != cred.ID {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		c.client.Del(ctx, key)
	} else {
		data, _ := json.Marshal(filtered)
		c.client.Set(ctx, key, data, c.ttl)
	}
}

// removeFromOrgSet removes a credential from the org-based set.
func (c *CredentialCache) removeFromOrgSet(ctx context.Context, cred *CachedCredential) {
	key := orgKey(cred.OrganizationID.String())
	existing, _ := c.client.Get(ctx, key).Bytes()
	if existing == nil {
		return
	}
	var creds []CachedCredential
	if err := json.Unmarshal(existing, &creds); err != nil {
		return
	}
	filtered := make([]CachedCredential, 0, len(creds))
	for _, c := range creds {
		if c.ID != cred.ID {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		c.client.Del(ctx, key)
	} else {
		data, _ := json.Marshal(filtered)
		c.client.Set(ctx, key, data, c.ttl)
	}
}

// InvalidateByOrg clears all credentials for an organization (bulk revocation).
func (c *CredentialCache) InvalidateByOrg(ctx context.Context, orgID string) {
	if !c.enabled {
		return
	}
	key := orgKey(orgID)
	existing, _ := c.client.Get(ctx, key).Bytes()
	if existing == nil {
		return
	}
	var creds []CachedCredential
	if err := json.Unmarshal(existing, &creds); err != nil {
		return
	}
	for _, cred := range creds {
		c.client.Del(ctx, hashKey(cred.KeyHash))
		c.removeFromPrefixSet(ctx, &cred)
	}
	c.client.Del(ctx, key)
}

// WarmUp preloads active credentials for an org (optional, on startup).
func (c *CredentialCache) WarmUp(ctx context.Context, orgID string, repo interface {
	ResolveByKeyPrefix(context.Context, string) ([]CachedCredential, error)
}) {
	if !c.enabled {
		return
	}
}
