package gatewayv2

import (
	"context"
	"errors"
	"fmt"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"
)

type stubAccountRepo struct {
	listFn func(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error)
}

var _ repositories.ProviderAccountRepository = (*stubAccountRepo)(nil)

func (s *stubAccountRepo) Create(context.Context, uuid.UUID, domain.NewProviderAccount) (domain.ProviderAccount, error) {
	panic("stubAccountRepo.Create not configured")
}
func (s *stubAccountRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
	panic("stubAccountRepo.Get not configured")
}
func (s *stubAccountRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
	if s.listFn == nil {
		panic("stubAccountRepo.List not configured")
	}
	return s.listFn(ctx, orgID)
}

type stubCredentialRepo struct {
	listByAccountFn func(ctx context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error)
}

var _ repositories.ProviderCredentialRepository = (*stubCredentialRepo)(nil)

func (s *stubCredentialRepo) Create(context.Context, uuid.UUID, domain.NewProviderCredential) (domain.ProviderCredential, error) {
	panic("stubCredentialRepo.Create not configured")
}
func (s *stubCredentialRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
	panic("stubCredentialRepo.Get not configured")
}
func (s *stubCredentialRepo) List(context.Context, uuid.UUID) ([]domain.ProviderCredential, error) {
	panic("stubCredentialRepo.List not configured")
}
func (s *stubCredentialRepo) ListByAccount(ctx context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
	if s.listByAccountFn == nil {
		panic("stubCredentialRepo.ListByAccount not configured")
	}
	return s.listByAccountFn(ctx, orgID, accountID)
}
func (s *stubCredentialRepo) Revoke(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
	panic("stubCredentialRepo.Revoke not configured")
}

type stubPolicyRepo struct {
	listFn func(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error)
}

var _ repositories.PolicyRepository = (*stubPolicyRepo)(nil)

func (s *stubPolicyRepo) Create(context.Context, uuid.UUID, domain.NewPolicy) (domain.Policy, error) {
	panic("stubPolicyRepo.Create not configured")
}
func (s *stubPolicyRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Policy, error) {
	panic("stubPolicyRepo.Get not configured")
}
func (s *stubPolicyRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	if s.listFn == nil {
		panic("stubPolicyRepo.List not configured")
	}
	return s.listFn(ctx, orgID)
}
func (s *stubPolicyRepo) SetEnforcement(context.Context, uuid.UUID, uuid.UUID, string) (domain.Policy, error) {
	panic("stubPolicyRepo.SetEnforcement not configured")
}

type stubPolicyRuleRepo struct {
	listFn func(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error)
}

var _ repositories.PolicyRuleRepository = (*stubPolicyRuleRepo)(nil)

func (s *stubPolicyRuleRepo) Create(context.Context, uuid.UUID, domain.NewPolicyRule) (domain.PolicyRule, error) {
	panic("stubPolicyRuleRepo.Create not configured")
}
func (s *stubPolicyRuleRepo) List(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	if s.listFn == nil {
		panic("stubPolicyRuleRepo.List not configured")
	}
	return s.listFn(ctx, orgID, policyID)
}
func (s *stubPolicyRuleRepo) SetEnabled(context.Context, uuid.UUID, uuid.UUID, bool) (domain.PolicyRule, error) {
	panic("stubPolicyRuleRepo.SetEnabled not configured")
}

type stubSecretStore struct {
	values map[string]string
	getErr error
}

var _ secrets.Store = (*stubSecretStore)(nil)

func (s *stubSecretStore) Get(_ context.Context, ref string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	if v, ok := s.values[ref]; ok {
		return v, nil
	}
	return "", fmt.Errorf("%w: %s", secrets.ErrNotFound, ref)
}
func (s *stubSecretStore) Put(_ context.Context, ref, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[ref] = value
	return nil
}
func (s *stubSecretStore) Delete(_ context.Context, ref string) error {
	delete(s.values, ref)
	return nil
}

// errStub is a sentinel for repository error-pass-through tests.
var errStub = errors.New("stub failure")

// stubHasher hashes deterministically; same input always yields same output.
type stubHasher struct {
	err error
}

var _ ports.TokenHasher = (*stubHasher)(nil)

func (s *stubHasher) Hash(_ context.Context, secret string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "hash:" + secret, nil
}

// stubGatewayCredRepo resolves gateway credentials by key hash.
type stubGatewayCredRepo struct {
	resolveFn   func(ctx context.Context, keyHash string) (domain.GatewayCredential, error)
	resolveHits int
}

var _ repositories.GatewayCredentialRepository = (*stubGatewayCredRepo)(nil)

func (s *stubGatewayCredRepo) Create(context.Context, uuid.UUID, domain.NewGatewayCredential) (domain.GatewayCredential, error) {
	panic("stubGatewayCredRepo.Create not configured")
}
func (s *stubGatewayCredRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.GatewayCredential, error) {
	panic("stubGatewayCredRepo.Get not configured")
}
func (s *stubGatewayCredRepo) List(context.Context, uuid.UUID) ([]domain.GatewayCredential, error) {
	panic("stubGatewayCredRepo.List not configured")
}
func (s *stubGatewayCredRepo) ResolveByKeyHash(ctx context.Context, keyHash string) (domain.GatewayCredential, error) {
	if s.resolveFn == nil {
		panic("stubGatewayCredRepo.ResolveByKeyHash not configured")
	}
	s.resolveHits++
	return s.resolveFn(ctx, keyHash)
}
func (s *stubGatewayCredRepo) ResolveByKeyPrefix(context.Context, string) ([]domain.GatewayCredential, error) {
	panic("stubGatewayCredRepo.ResolveByKeyPrefix not configured")
}
func (s *stubGatewayCredRepo) Revoke(context.Context, uuid.UUID, uuid.UUID) (domain.GatewayCredential, error) {
	panic("stubGatewayCredRepo.Revoke not configured")
}
func (s *stubGatewayCredRepo) UpdateStatus(context.Context, uuid.UUID, uuid.UUID, domain.GatewayCredentialStatus) (domain.GatewayCredential, error) {
	panic("stubGatewayCredRepo.UpdateStatus not configured")
}
func (s *stubGatewayCredRepo) Rotate(context.Context, uuid.UUID, uuid.UUID, domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
	panic("stubGatewayCredRepo.Rotate not configured")
}

// stubCredentialCache is an in-memory CredentialCache keyed by hash.
type stubCredentialCache struct {
	entries map[string]*CredentialCacheEntry
	lastSet *CredentialCacheEntry
}

func (c *stubCredentialCache) Get(_ context.Context, keyHash string) *CredentialCacheEntry {
	if c.entries == nil {
		return nil
	}
	return c.entries[keyHash]
}
func (c *stubCredentialCache) Set(_ context.Context, entry *CredentialCacheEntry) {
	if entry == nil {
		return
	}
	c.lastSet = entry
	if c.entries == nil {
		c.entries = make(map[string]*CredentialCacheEntry)
	}
	c.entries[entry.ID] = entry
}
func (c *stubCredentialCache) Invalidate(_ context.Context, keyHash string) {
	if c.entries == nil {
		return
	}
	delete(c.entries, keyHash)
}
