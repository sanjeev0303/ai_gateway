// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"errors"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

// AuthenticatorImpl implements credential authentication using the identity service.
type AuthenticatorImpl struct {
	hasher         ports.TokenHasher
	credentialRepo repositories.GatewayCredentialRepository
	cache          CredentialCache
	clock          func() time.Time
}

var (
	ErrInvalidCredential = errors.New("invalid credential")
	ErrCredentialExpired = errors.New("credential expired")
	ErrCredentialRevoked = errors.New("credential revoked")
	ErrWrongScope        = errors.New("credential scope not allowed for gateway")
)

// NewAuthenticator creates a new authenticator.
func NewAuthenticator(
	hasher ports.TokenHasher,
	credentialRepo repositories.GatewayCredentialRepository,
	cache CredentialCache,
) *AuthenticatorImpl {
	return &AuthenticatorImpl{
		hasher:         hasher,
		credentialRepo: credentialRepo,
		cache:          cache,
		clock:          time.Now,
	}
}

// Authenticate validates a raw API key and returns the gateway credential.
func (a *AuthenticatorImpl) Authenticate(ctx context.Context, rawKey string) (GatewayCredential, error) {
	if rawKey == "" {
		return GatewayCredential{}, ErrInvalidCredential
	}

	// Hash the key for lookup
	hash, err := a.hasher.Hash(ctx, rawKey)
	if err != nil {
		return GatewayCredential{}, err
	}

	// Try cache first (fast path)
	if a.cache != nil {
		if cached := a.cache.Get(ctx, hash); cached != nil {
			if a.validateCached(cached) {
				return a.fromCached(cached), nil
			}
			// Cache entry invalid, evict it and fall through to DB
			a.cache.Invalidate(ctx, hash)
		}
	}

	// Fallback to database
	cred, err := a.credentialRepo.ResolveByKeyHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return GatewayCredential{}, ErrInvalidCredential
		}
		return GatewayCredential{}, err
	}

	// Convert domain.GatewayCredential to gatewayv2.GatewayCredential
	gwCred := a.fromDomain(cred)

	// Validate credential state
	if err := a.validateCredential(&gwCred); err != nil {
		return GatewayCredential{}, err
	}

	// Update cache
	if a.cache != nil {
		entry := a.toCacheEntry(gwCred)
		entry.KeyHash = hash
		a.cache.Set(ctx, entry)
	}

	return gwCred, nil
}

// validateCredential checks credential status, scope, and expiry.
func (a *AuthenticatorImpl) validateCredential(cred *GatewayCredential) error {
	now := a.clock()

	if cred.Status != "active" {
		switch cred.Status {
		case "revoked":
			return ErrCredentialRevoked
		case "expired":
			return ErrCredentialExpired
		default:
			return ErrInvalidCredential
		}
	}

	// Gateway only accepts gateway/all scoped credentials
	if cred.Scopes != nil {
		hasGateway := false
		for _, s := range cred.Scopes {
			if s == "gateway" || s == "all" {
				hasGateway = true
				break
			}
		}
		if !hasGateway {
			return ErrWrongScope
		}
	}

	// Check expiry
	if cred.ExpiresAt != nil && cred.ExpiresAt.Before(now) {
		return ErrCredentialExpired
	}

	return nil
}

// validateCached checks if a cached credential is still valid.
func (a *AuthenticatorImpl) validateCached(cached *CredentialCacheEntry) bool {
	now := a.clock()

	if cached.Status != "active" {
		return false
	}

	// Check gateway scope
	hasGateway := false
	for _, s := range cached.Scopes {
		if s == "gateway" || s == "all" {
			hasGateway = true
			break
		}
	}
	if !hasGateway {
		return false
	}

	// Check expiry
	if cached.ExpiresAt != nil && cached.ExpiresAt.Before(now) {
		return false
	}

	return true
}

// fromCached converts a cached credential to GatewayCredential.
func (a *AuthenticatorImpl) fromCached(cached *CredentialCacheEntry) GatewayCredential {
	scopes := make([]string, len(cached.Scopes))
	copy(scopes, cached.Scopes)

	var teamID, projectID, providerAccountID *uuid.UUID
	if cached.TeamID != nil {
		id, _ := uuid.Parse(*cached.TeamID)
		teamID = &id
	}
	if cached.ProjectID != nil {
		id, _ := uuid.Parse(*cached.ProjectID)
		projectID = &id
	}
	if cached.ProviderAccountID != nil {
		id, _ := uuid.Parse(*cached.ProviderAccountID)
		providerAccountID = &id
	}

	allowedProviders := make([]string, len(cached.AllowedProviders))
	copy(allowedProviders, cached.AllowedProviders)

	allowedModels := make([]string, len(cached.AllowedModels))
	copy(allowedModels, cached.AllowedModels)

	subjectID, _ := uuid.Parse(cached.SubjectID)
	orgID, _ := uuid.Parse(cached.OrganizationID)
	id, _ := uuid.Parse(cached.ID)

	return GatewayCredential{
		ID:                id,
		OrganizationID:    orgID,
		SubjectType:       cached.SubjectType,
		SubjectID:         subjectID,
		TeamID:            teamID,
		ProjectID:         projectID,
		Scopes:            scopes,
		ProviderAccountID: providerAccountID,
		AllowedProviders:  allowedProviders,
		AllowedModels:     allowedModels,
		RateLimit:         cached.RateLimit,
		BudgetLimit:       cached.BudgetLimit,
		Status:            cached.Status,
		ExpiresAt:         cached.ExpiresAt,
	}
}

// InvalidateByID evicts a credential from the cache by credential ID.
func (a *AuthenticatorImpl) InvalidateByID(ctx context.Context, id string) {
	if a.cache != nil {
		if inMem, ok := a.cache.(*InMemoryCredentialCache); ok {
			inMem.InvalidateByID(ctx, id)
		}
	}
}

// CredentialCache is a minimal cache interface for the authenticator.
type CredentialCache interface {
	Get(ctx context.Context, keyHash string) *CredentialCacheEntry
	Set(ctx context.Context, entry *CredentialCacheEntry)
	Invalidate(ctx context.Context, keyHash string)
}

// fromDomain converts a domain.GatewayCredential to gatewayv2.GatewayCredential.
func (a *AuthenticatorImpl) fromDomain(d domain.GatewayCredential) GatewayCredential {
	var teamID, projectID, providerAccountID *uuid.UUID
	if d.TeamID != nil {
		id := *d.TeamID
		teamID = &id
	}
	if d.ProjectID != nil {
		id := *d.ProjectID
		projectID = &id
	}
	if d.ProviderAccountID != nil {
		id := *d.ProviderAccountID
		providerAccountID = &id
	}

	scopes := make([]string, len(d.Scopes))
	for i, s := range d.Scopes {
		scopes[i] = string(s)
	}

	allowedProviders := make([]string, len(d.AllowedProviders))
	copy(allowedProviders, d.AllowedProviders)

	allowedModels := make([]string, len(d.AllowedModels))
	copy(allowedModels, d.AllowedModels)

	return GatewayCredential{
		ID:                d.ID,
		OrganizationID:    d.OrganizationID,
		SubjectType:       string(d.SubjectType),
		SubjectID:         d.SubjectID,
		TeamID:            teamID,
		ProjectID:         projectID,
		Scopes:            scopes,
		ProviderAccountID: providerAccountID,
		AllowedProviders:  allowedProviders,
		AllowedModels:     allowedModels,
		RateLimit:         d.RateLimit,
		BudgetLimit:       d.BudgetLimit,
		Status:            string(d.Status),
		ExpiresAt:         d.ExpiresAt,
	}
}

// toCacheEntry converts a gatewayv2.GatewayCredential to CredentialCacheEntry.
func (a *AuthenticatorImpl) toCacheEntry(c GatewayCredential) *CredentialCacheEntry {
	var teamID, projectID, providerAccountID *string
	if c.TeamID != nil {
		s := c.TeamID.String()
		teamID = &s
	}
	if c.ProjectID != nil {
		s := c.ProjectID.String()
		projectID = &s
	}
	if c.ProviderAccountID != nil {
		s := c.ProviderAccountID.String()
		providerAccountID = &s
	}

	scopes := make([]string, len(c.Scopes))
	copy(scopes, c.Scopes)

	allowedProviders := make([]string, len(c.AllowedProviders))
	copy(allowedProviders, c.AllowedProviders)

	allowedModels := make([]string, len(c.AllowedModels))
	copy(allowedModels, c.AllowedModels)

	return &CredentialCacheEntry{
		ID:                c.ID.String(),
		OrganizationID:    c.OrganizationID.String(),
		SubjectType:       c.SubjectType,
		SubjectID:         c.SubjectID.String(),
		TeamID:            teamID,
		ProjectID:         projectID,
		Scopes:            scopes,
		ProviderAccountID: providerAccountID,
		AllowedProviders:  allowedProviders,
		AllowedModels:     allowedModels,
		RateLimit:         c.RateLimit,
		BudgetLimit:       c.BudgetLimit,
		Status:            c.Status,
		ExpiresAt:         c.ExpiresAt,
		UpdatedAt:         time.Now(),
	}
}
