// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"
)

// TenantResolverImpl resolves upstream provider connections for organizations.
type TenantResolverImpl struct {
	providerAccounts    repositories.ProviderAccountRepository
	providerCredentials repositories.ProviderCredentialRepository
	secrets             secrets.Store
	cache               *UpstreamCache
	clock               func() time.Time
}

// UpstreamCache caches resolved upstream connections.
type UpstreamCache struct {
	mu      sync.RWMutex
	entries map[string]*CachedUpstream
	maxSize int
	ttl     time.Duration
	clock   func() time.Time
}

type CachedUpstream struct {
	Upstream  ResolvedUpstream
	ExpiresAt time.Time
	CreatedAt time.Time
}

func cacheKey(orgID uuid.UUID, providerKind string) string {
	return orgID.String() + ":" + providerKind
}

// NewTenantResolver creates a new tenant resolver.
func NewTenantResolver(
	accounts repositories.ProviderAccountRepository,
	credentials repositories.ProviderCredentialRepository,
	secrets secrets.Store,
) *TenantResolverImpl {
	return &TenantResolverImpl{
		providerAccounts:    accounts,
		providerCredentials: credentials,
		secrets:             secrets,
		cache:               NewUpstreamCache(1000, 5*time.Minute),
		clock:               time.Now,
	}
}

// NewUpstreamCache creates a new upstream cache.
func NewUpstreamCache(maxSize int, ttl time.Duration) *UpstreamCache {
	return &UpstreamCache{
		entries: make(map[string]*CachedUpstream, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
		clock:   time.Now,
	}
}

// ResolveUpstream resolves the upstream provider for an organization (defaults to Anthropic for backwards compatibility).
func (t *TenantResolverImpl) ResolveUpstream(ctx context.Context, orgID uuid.UUID) (ResolvedUpstream, error) {
	return t.ResolveUpstreamForProvider(ctx, orgID, "")
}

// ResolveUpstreamForProvider resolves the upstream provider account and credential for an organization and provider kind.
func (t *TenantResolverImpl) ResolveUpstreamForProvider(ctx context.Context, orgID uuid.UUID, providerKind string) (ResolvedUpstream, error) {
	// Try cache first
	if cached := t.cache.GetForProvider(orgID, providerKind); cached != nil {
		return *cached, nil
	}

	// Resolve from database
	upstream, err := t.resolveFromDB(ctx, orgID, providerKind)
	if err != nil {
		return ResolvedUpstream{}, err
	}

	// Cache the result
	t.cache.SetForProvider(orgID, providerKind, upstream)

	return upstream, nil
}

// resolveFromDB performs the actual database resolution for a given provider kind.
func (t *TenantResolverImpl) resolveFromDB(ctx context.Context, orgID uuid.UUID, providerKind string) (ResolvedUpstream, error) {
	accounts, err := t.providerAccounts.List(ctx, orgID)
	if err != nil {
		return ResolvedUpstream{}, err
	}

	var account *domain.ProviderAccount
	if providerKind != "" {
		for i := range accounts {
			if accounts[i].ProviderKind == providerKind &&
				accounts[i].Status == domain.ProviderStatusActive {
				account = &accounts[i]
				break
			}
		}
	} else {
		// Default: find active Anthropic provider account
		for i := range accounts {
			if accounts[i].ProviderKind == domain.ProviderKindAnthropic &&
				accounts[i].Status == domain.ProviderStatusActive {
				account = &accounts[i]
				break
			}
		}
	}
	if account == nil {
		return ResolvedUpstream{}, ErrNoUpstream
	}

	// Get active credentials for this account
	creds, err := t.providerCredentials.ListByAccount(ctx, orgID, account.ID)
	if err != nil {
		return ResolvedUpstream{}, err
	}
	if len(creds) == 0 || creds[0].Status != domain.ProviderCredentialStatusActive {
		return ResolvedUpstream{}, ErrNoUpstream
	}

	// Get the secret
	upstreamKey, err := t.secrets.Get(ctx, creds[0].SecretRef)
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			return ResolvedUpstream{}, ErrNoUpstream
		}
		return ResolvedUpstream{}, err
	}

	// Build base URL according to provider kind
	baseURL := account.BaseURL
	if baseURL == "" {
		switch account.ProviderKind {
		case domain.ProviderKindAnthropic:
			baseURL = "https://api.anthropic.com"
		case domain.ProviderKindOpenAI:
			baseURL = "https://api.openai.com/"
		case domain.ProviderKindGemini:
			baseURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
		default:
			baseURL = "https://api.anthropic.com/"
		}
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return ResolvedUpstream{
		ProviderAccount: ProviderAccount{
			ID:             account.ID,
			OrganizationID: account.OrganizationID,
			ProviderKind:   account.ProviderKind,
			Name:           account.Name,
			BaseURL:        baseURL,
			Status:         account.Status,
			CreatedAt:      account.CreatedAt,
			UpdatedAt:      account.UpdatedAt,
		},
		Credential: ProviderCredential{
			ID:                creds[0].ID,
			OrganizationID:    creds[0].OrganizationID,
			ProviderAccountID: creds[0].ProviderAccountID,
			SecretRef:         creds[0].SecretRef,
			Fingerprint:       creds[0].Fingerprint,
			Tail:              creds[0].Tail,
			Status:            creds[0].Status,
			CreatedAt:         creds[0].CreatedAt,
			UpdatedAt:         creds[0].UpdatedAt,
		},
		UpstreamKey: upstreamKey,
		BaseURL:     baseURL,
	}, nil
}

// UpstreamCache methods with pointer receivers
func (c *UpstreamCache) Get(orgID uuid.UUID) *ResolvedUpstream {
	return c.GetForProvider(orgID, "")
}

func (c *UpstreamCache) GetForProvider(orgID uuid.UUID, providerKind string) *ResolvedUpstream {
	key := cacheKey(orgID, providerKind)
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	if c.clock().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil
	}

	return &entry.Upstream
}

func (c *UpstreamCache) Set(orgID uuid.UUID, upstream ResolvedUpstream) {
	c.SetForProvider(orgID, "", upstream)
}

func (c *UpstreamCache) SetForProvider(orgID uuid.UUID, providerKind string, upstream ResolvedUpstream) {
	key := cacheKey(orgID, providerKind)
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	now := c.clock()
	c.entries[key] = &CachedUpstream{
		Upstream:  upstream,
		ExpiresAt: now.Add(c.ttl),
	}
}

// Invalidate removes an organization's cached upstream entries.
func (c *UpstreamCache) Invalidate(orgID uuid.UUID) {
	prefix := orgID.String() + ":"
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
}
