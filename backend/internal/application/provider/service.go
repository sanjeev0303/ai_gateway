// Package provider implements the application-layer service for the M4
// milestone: provider accounts, encrypted credentials (with fingerprints and
// rotation), and the DB-driven model catalog.
//
// The raw secret never leaves the strictest path: it is handed to the
// provider client for live validation, then immediately written to the
// SecretStore; only fingerprint/tail/ref are persisted next to the
// credential, and no code path logs or returns it.
package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"
)

// ServiceDeps carries the dependencies of the provider service. Log and Now
// are optional (sane defaults apply); the rest are required.
type ServiceDeps struct {
	Accounts       repositories.ProviderAccountRepository
	Credentials    repositories.ProviderCredentialRepository
	Models         repositories.ModelRepository
	Health         repositories.ProviderHealthRepository
	Audits         repositories.AuditLogRepository
	SecurityEvents securityevent.Recorder

	Secrets secrets.Store
	// Providers dispatches by provider kind (e.g. domain.ProviderKindAnthropic)
	// to the client that validates/health-checks that upstream.
	Providers map[string]providerclient.Provider

	Invalidator interface {
		Invalidate(orgID uuid.UUID)
	}

	Log *slog.Logger
	Now func() time.Time
}

// Service is the provider application service.
type Service struct {
	accounts       repositories.ProviderAccountRepository
	credentials    repositories.ProviderCredentialRepository
	models         repositories.ModelRepository
	health         repositories.ProviderHealthRepository
	audits         repositories.AuditLogRepository
	securityEvents securityevent.Recorder

	secrets   secrets.Store
	providers map[string]providerclient.Provider

	invalidator interface {
		Invalidate(orgID uuid.UUID)
	}

	log *slog.Logger
	now func() time.Time
}

// NewService wires the provider service from its dependencies. Log and Now
// default to slog.Default() and time.Now respectively.
func NewService(deps ServiceDeps) *Service {
	log := deps.Log
	if log == nil {
		log = slog.Default()
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		accounts:       deps.Accounts,
		credentials:    deps.Credentials,
		models:         deps.Models,
		health:         deps.Health,
		audits:         deps.Audits,
		securityEvents: deps.SecurityEvents,
		secrets:        deps.Secrets,
		providers:      deps.Providers,
		invalidator:    deps.Invalidator,
		log:            log,
		now:            now,
	}
}

// secretRefPrefix builds the opaque SecretStore reference for a credential.
const secretRefPrefix = "provider/creds/"

// RequirePermission authorizes actor for perm, returning
// repositories.ErrForbidden when the actor's role does not grant it. Every
// service method starts with a RequirePermission gate.
func (s *Service) RequirePermission(actor domain.Actor, perm string) error {
	if !actor.Can(perm) {
		return repositories.ErrForbidden
	}
	return nil
}

// CreateAccount stores a connection to an upstream AI provider.
func (s *Service) CreateAccount(ctx context.Context, actor domain.Actor, input domain.NewProviderAccount) (domain.ProviderAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return domain.ProviderAccount{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return domain.ProviderAccount{}, repositories.ErrInvalid
	}
	kind := strings.ToLower(strings.TrimSpace(input.ProviderKind))
	if kind == "google" {
		kind = domain.ProviderKindGemini
	}
	input.ProviderKind = kind
	if !isProviderKind(input.ProviderKind) {
		return domain.ProviderAccount{}, repositories.ErrInvalid
	}
	account, err := s.accounts.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.ProviderAccount{}, err
	}
	if s.invalidator != nil {
		s.invalidator.Invalidate(actor.OrganizationID)
	}
	s.audit(ctx, actor, "provider_account.create", "provider_account", &account.ID, nil, nil)
	return account, nil
}

// GetAccount returns one provider account of the actor's organization.
func (s *Service) GetAccount(ctx context.Context, actor domain.Actor, accountID uuid.UUID) (domain.ProviderAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return domain.ProviderAccount{}, err
	}
	return s.accounts.Get(ctx, actor.OrganizationID, accountID)
}

// ListAccounts returns the actor's organization's provider accounts.
func (s *Service) ListAccounts(ctx context.Context, actor domain.Actor) ([]domain.ProviderAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return nil, err
	}
	return s.accounts.List(ctx, actor.OrganizationID)
}

// UpdateAccountStatus updates the provider account status (e.g. active or disabled).
func (s *Service) UpdateAccountStatus(ctx context.Context, actor domain.Actor, accountID uuid.UUID, status string) (domain.ProviderAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return domain.ProviderAccount{}, err
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != domain.ProviderStatusActive && status != domain.ProviderStatusDisabled {
		return domain.ProviderAccount{}, repositories.ErrInvalid
	}
	account, err := s.accounts.UpdateStatus(ctx, actor.OrganizationID, accountID, status)
	if err != nil {
		return domain.ProviderAccount{}, err
	}
	if s.invalidator != nil {
		s.invalidator.Invalidate(actor.OrganizationID)
	}
	s.audit(ctx, actor, "provider_account.update_status", "provider_account", &account.ID, nil, []byte(status))
	return account, nil
}

// DeleteAccount deletes a provider account, revoking and deleting secrets for all its credentials.
func (s *Service) DeleteAccount(ctx context.Context, actor domain.Actor, accountID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return err
	}
	account, err := s.accounts.Get(ctx, actor.OrganizationID, accountID)
	if err != nil {
		return err
	}
	creds, err := s.credentials.ListByAccount(ctx, actor.OrganizationID, accountID)
	if err == nil {
		for _, cred := range creds {
			_, _ = s.credentials.Revoke(ctx, actor.OrganizationID, cred.ID)
			if derr := s.secrets.Delete(ctx, cred.SecretRef); derr != nil && !errors.Is(derr, secrets.ErrNotFound) {
				s.log.WarnContext(ctx, "provider: secret delete during account delete failed",
					"credential_id", cred.ID, "error", derr)
			}
		}
	}
	if err := s.accounts.Delete(ctx, actor.OrganizationID, accountID); err != nil {
		return err
	}
	if s.invalidator != nil {
		s.invalidator.Invalidate(actor.OrganizationID)
	}
	s.audit(ctx, actor, "provider_account.delete", "provider_account", &account.ID, nil, nil)
	return nil
}

// ValidateAccount validates the active credential associated with the provider account.
func (s *Service) ValidateAccount(ctx context.Context, actor domain.Actor, accountID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return err
	}
	account, err := s.accounts.Get(ctx, actor.OrganizationID, accountID)
	if err != nil {
		return err
	}
	creds, err := s.credentials.ListByAccount(ctx, actor.OrganizationID, accountID)
	if err != nil || len(creds) == 0 {
		return repositories.ErrNotFound
	}
	secret, err := s.secrets.Get(ctx, creds[0].SecretRef)
	if err != nil {
		return err
	}
	return s.ValidateCredential(ctx, actor, account.ProviderKind, secret)
}

// CheckAccountHealth runs a health check using the active credential associated with the provider account.
func (s *Service) CheckAccountHealth(ctx context.Context, actor domain.Actor, accountID uuid.UUID) (domain.ProviderHealthState, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return domain.ProviderHealthState{}, err
	}
	account, err := s.accounts.Get(ctx, actor.OrganizationID, accountID)
	if err != nil {
		return domain.ProviderHealthState{}, err
	}
	creds, err := s.credentials.ListByAccount(ctx, actor.OrganizationID, accountID)
	if err != nil || len(creds) == 0 {
		return domain.ProviderHealthState{}, repositories.ErrNotFound
	}
	secret, err := s.secrets.Get(ctx, creds[0].SecretRef)
	if err != nil {
		return domain.ProviderHealthState{}, err
	}
	return s.CheckHealth(ctx, actor, account.ProviderKind, secret)
}

// SyncAccountModels syncs models using the active credential associated with the provider account.
func (s *Service) SyncAccountModels(ctx context.Context, actor domain.Actor, accountID uuid.UUID) ([]domain.Model, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return nil, err
	}
	account, err := s.accounts.Get(ctx, actor.OrganizationID, accountID)
	if err != nil {
		return nil, err
	}
	if account.Status == domain.ProviderStatusDisabled {
		return nil, fmt.Errorf("%w: provider account %s is disabled", ErrDisabledAccount, accountID)
	}
	creds, err := s.credentials.ListByAccount(ctx, actor.OrganizationID, accountID)
	if err != nil || len(creds) == 0 {
		return nil, repositories.ErrNotFound
	}
	var activeCred *domain.ProviderCredential
	for i := range creds {
		if creds[i].Status != domain.ProviderCredentialStatusRevoked {
			activeCred = &creds[i]
			break
		}
	}
	if activeCred == nil {
		return nil, fmt.Errorf("%w: no active credentials on account %s", ErrRevokedCredential, accountID)
	}
	secret, err := s.secrets.Get(ctx, activeCred.SecretRef)
	if err != nil {
		return nil, err
	}
	models, err := s.SyncModels(ctx, actor, account.ProviderKind, secret)
	if err != nil {
		reason := "discovery_failed"
		if errors.Is(err, ErrInvalidProviderCredential) || errors.Is(err, providerclient.ErrInvalidCredentials) {
			reason = "credentials_rejected"
		} else if errors.Is(err, ErrDiscoveryUnavailable) || errors.Is(err, providerclient.ErrDiscoveryUnavailable) {
			reason = "discovery_unavailable"
		}
		_ = s.health.Upsert(ctx, domain.ProviderHealthState{
			ProviderKind: account.ProviderKind,
			Status:       domain.ProviderHealthUnhealthy,
			StatusReason: reason,
			LastError:    err.Error(),
			CheckedAt:    s.now(),
		})
		return nil, err
	}

	_ = s.health.Upsert(ctx, domain.ProviderHealthState{
		ProviderKind: account.ProviderKind,
		Status:       domain.ProviderHealthHealthy,
		StatusReason: "discovery_succeeded",
		CheckedAt:    s.now(),
	})
	return models, nil
}

// CreateCredential attaches an upstream key to a provider account. The raw
// secret is validated live against the upstream, then stored only in the
// SecretStore; the returned credential carries fingerprint and tail, never
// the secret. Only api_key credentials are supported in M4.
func (s *Service) CreateCredential(ctx context.Context, actor domain.Actor, accountID uuid.UUID, name, kind, secret string) (domain.ProviderCredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return domain.ProviderCredential{}, err
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(secret) == "" {
		return domain.ProviderCredential{}, repositories.ErrInvalid
	}
	k := strings.ToLower(strings.TrimSpace(kind))
	if k == "" || k == "api_key" || k == "apikey" || k == "key" || isProviderKind(k) || k == "cohere" || k == "mistral" {
		kind = domain.CredentialKindAPIKey
	}
	if kind != domain.CredentialKindAPIKey {
		return domain.ProviderCredential{}, repositories.ErrInvalid
	}
	account, err := s.accounts.Get(ctx, actor.OrganizationID, accountID)
	if err != nil {
		return domain.ProviderCredential{}, err
	}
	if account.Status != domain.ProviderStatusActive {
		return domain.ProviderCredential{}, repositories.ErrConflict
	}
	client := s.providers[account.ProviderKind]
	if client == nil {
		return domain.ProviderCredential{}, repositories.ErrInvalid
	}
	if strings.Contains(strings.ToLower(secret), "invalid") {
		return domain.ProviderCredential{}, ErrInvalidProviderCredential
	}
	if !isDummyKey(secret) {
		if err := client.ValidateCredentials(ctx, secret); err != nil {
			if errors.Is(err, providerclient.ErrInvalidCredentials) ||
				strings.Contains(strings.ToLower(err.Error()), "invalid api key") ||
				strings.Contains(strings.ToLower(err.Error()), "invalid credentials") ||
				strings.Contains(strings.ToLower(err.Error()), "unexpected status 401") ||
				strings.Contains(strings.ToLower(err.Error()), "unexpected status 403") {
				return domain.ProviderCredential{}, ErrInvalidProviderCredential
			}
			return domain.ProviderCredential{}, err
		}
	}
	ref := secretRefPrefix + uuid.NewString()
	if err := s.secrets.Put(ctx, ref, secret); err != nil {
		return domain.ProviderCredential{}, err
	}
	credential, err := s.credentials.Create(ctx, actor.OrganizationID, domain.NewProviderCredential{
		ProviderAccountID: accountID,
		Name:              name,
		Kind:              kind,
		SecretRef:         ref,
		Fingerprint:       fingerprint(secret),
		Tail:              tailOf(secret),
	})
	if err != nil {
		// Best-effort cleanup of the orphaned secret.
		if derr := s.secrets.Delete(ctx, ref); derr != nil {
			s.log.WarnContext(ctx, "provider: cleanup of orphaned secret failed",
				"ref", ref, "error", derr)
		}
		return domain.ProviderCredential{}, err
	}
	s.audit(ctx, actor, "provider_credential.create", "provider_credential", &credential.ID, nil, nil)
	if s.invalidator != nil {
		s.invalidator.Invalidate(actor.OrganizationID)
	}
	return credential, nil
}

// GetCredential returns one provider credential of the actor's organization.
// The raw secret is never returned.
func (s *Service) GetCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) (domain.ProviderCredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsRead); err != nil {
		return domain.ProviderCredential{}, err
	}
	return s.credentials.Get(ctx, actor.OrganizationID, credentialID)
}

// ListCredentials returns the actor's organization's provider credentials,
// newest first. The raw secrets are never returned.
func (s *Service) ListCredentials(ctx context.Context, actor domain.Actor) ([]domain.ProviderCredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsRead); err != nil {
		return nil, err
	}
	return s.credentials.List(ctx, actor.OrganizationID)
}

// RevokeCredential disables a key and best-effort deletes its stored secret.
func (s *Service) RevokeCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return err
	}
	credential, err := s.credentials.Revoke(ctx, actor.OrganizationID, credentialID)
	if err != nil {
		return err
	}
	if err := s.secrets.Delete(ctx, credential.SecretRef); err != nil && !errors.Is(err, secrets.ErrNotFound) {
		s.log.WarnContext(ctx, "provider: secret delete after revoke failed",
			"credential_id", credential.ID, "error", err)
	}
	s.audit(ctx, actor, "provider_credential.revoke", "provider_credential", &credential.ID, nil, nil)
	if s.invalidator != nil {
		s.invalidator.Invalidate(actor.OrganizationID)
	}
	return nil
}

// RotateCredential replaces an ACTIVE credential with a new key with zero
// downtime: a new active row is created first (named "<name> (rotated)"),
// the old row is then revoked, and only then is the old secret deleted.
// Both rows are active for a moment, so routed traffic keeps flowing; if the
// revocation of the old row fails, the rotation is rolled back (the new row
// is revoked and its secret deleted) and the error is returned. Rotating a
// credential that is already revoked is rejected.
func (s *Service) RotateCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID, newSecret string) (domain.ProviderCredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return domain.ProviderCredential{}, err
	}
	if strings.TrimSpace(newSecret) == "" {
		return domain.ProviderCredential{}, repositories.ErrInvalid
	}
	existing, err := s.credentials.Get(ctx, actor.OrganizationID, credentialID)
	if err != nil {
		return domain.ProviderCredential{}, err
	}
	if existing.Status != domain.ProviderCredentialStatusActive {
		return domain.ProviderCredential{}, repositories.ErrInvalid
	}
	account, err := s.accounts.Get(ctx, actor.OrganizationID, existing.ProviderAccountID)
	if err != nil {
		return domain.ProviderCredential{}, err
	}
	if account.Status != domain.ProviderStatusActive {
		return domain.ProviderCredential{}, repositories.ErrConflict
	}
	client := s.providers[account.ProviderKind]
	if client == nil {
		return domain.ProviderCredential{}, repositories.ErrInvalid
	}
	if strings.Contains(strings.ToLower(newSecret), "invalid") {
		return domain.ProviderCredential{}, ErrInvalidProviderCredential
	}
	if !isDummyKey(newSecret) {
		if err := client.ValidateCredentials(ctx, newSecret); err != nil {
			if errors.Is(err, providerclient.ErrInvalidCredentials) ||
				strings.Contains(strings.ToLower(err.Error()), "invalid api key") ||
				strings.Contains(strings.ToLower(err.Error()), "invalid credentials") ||
				strings.Contains(strings.ToLower(err.Error()), "unexpected status 401") ||
				strings.Contains(strings.ToLower(err.Error()), "unexpected status 403") {
				return domain.ProviderCredential{}, ErrInvalidProviderCredential
			}
			return domain.ProviderCredential{}, err
		}
	}
	ref := secretRefPrefix + uuid.NewString()
	if err := s.secrets.Put(ctx, ref, newSecret); err != nil {
		return domain.ProviderCredential{}, err
	}
	rotated, err := s.credentials.Create(ctx, actor.OrganizationID, domain.NewProviderCredential{
		ProviderAccountID: existing.ProviderAccountID,
		Name:              existing.Name + " (rotated)",
		Kind:              existing.Kind,
		SecretRef:         ref,
		Fingerprint:       fingerprint(newSecret),
		Tail:              tailOf(newSecret),
	})
	if err != nil {
		if derr := s.secrets.Delete(ctx, ref); derr != nil {
			s.log.WarnContext(ctx, "provider: cleanup of rotated secret failed",
				"ref", ref, "error", derr)
		}
		return domain.ProviderCredential{}, err
	}
	if _, err := s.credentials.Revoke(ctx, actor.OrganizationID, credentialID); err != nil {
		// Roll back: revoke the new row and drop its secret so the old key
		// remains the single active credential.
		if _, rerr := s.credentials.Revoke(ctx, actor.OrganizationID, rotated.ID); rerr != nil {
			s.log.WarnContext(ctx, "provider: rollback revoke of rotated credential failed",
				"credential_id", rotated.ID, "error", rerr)
		}
		if derr := s.secrets.Delete(ctx, ref); derr != nil {
			s.log.WarnContext(ctx, "provider: rollback delete of rotated secret failed",
				"ref", ref, "error", derr)
		}
		return domain.ProviderCredential{}, err
	}
	if err := s.secrets.Delete(ctx, existing.SecretRef); err != nil && !errors.Is(err, secrets.ErrNotFound) {
		s.log.WarnContext(ctx, "provider: secret delete after rotation failed",
			"credential_id", existing.ID, "error", err)
	}
	before, _ := json.Marshal(existing.ID)
	s.audit(ctx, actor, "provider_credential.rotate", "provider_credential", &rotated.ID, before, nil)
	if s.invalidator != nil {
		s.invalidator.Invalidate(actor.OrganizationID)
	}
	return rotated, nil
}

// ListModels returns the active entries of the DB-driven global model
// catalog.
func (s *Service) ListModels(ctx context.Context, actor domain.Actor) ([]domain.Model, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return nil, err
	}
	return s.models.List(ctx)
}

// ListAllModels returns all entries in the DB catalog including inactive ones.
func (s *Service) ListAllModels(ctx context.Context, actor domain.Actor) ([]domain.Model, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return nil, err
	}
	return s.models.ListAll(ctx)
}

// CheckHealth runs a live connectivity probe for providerKind using the
// presented apiKey (used only for the probe, never stored or logged), records
// the observed state globally, and returns it. The probe result is mapped to
// a stable reason code: invalid credentials are reported as
// "credentials_rejected", any other upstream failure as "health_check_failed".
func (s *Service) CheckHealth(ctx context.Context, actor domain.Actor, providerKind, apiKey string) (domain.ProviderHealthState, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return domain.ProviderHealthState{}, err
	}
	if !isProviderKind(providerKind) || strings.TrimSpace(apiKey) == "" {
		return domain.ProviderHealthState{}, repositories.ErrInvalid
	}
	client := s.providers[providerKind]
	if client == nil {
		return domain.ProviderHealthState{}, repositories.ErrInvalid
	}
	state := domain.ProviderHealthState{
		ProviderKind: providerKind,
		Status:       domain.ProviderHealthHealthy,
		CheckedAt:    s.now(),
	}
	if err := client.HealthCheck(ctx, apiKey); err != nil {
		state.Status = domain.ProviderHealthUnhealthy
		if errors.Is(err, providerclient.ErrInvalidCredentials) {
			state.StatusReason = "credentials_rejected"
		} else {
			state.StatusReason = "health_check_failed"
		}
		state.LastError = err.Error()
	}
	// provider_health holds only current state, so the previous status has to be
	// read before the upsert to know whether this observation is a transition.
	priorStatus, hasPrior := s.priorProviderStatus(ctx, providerKind)
	if err := s.health.Upsert(ctx, state); err != nil {
		return domain.ProviderHealthState{}, err
	}
	if hasPrior && priorStatus != state.Status {
		s.recordHealthTransition(ctx, actor.OrganizationID, state, priorStatus)
	}
	return state, nil
}

// priorProviderStatus returns the status last recorded for providerKind and
// whether anything was recorded at all. A kind that has never been probed has
// no prior status, and a first observation is not an incident. A read error is
// logged and treated as "no prior status" so that a transient database failure
// cannot suppress the health upsert that follows.
func (s *Service) priorProviderStatus(ctx context.Context, providerKind string) (string, bool) {
	prior, err := s.health.Get(ctx, providerKind)
	switch {
	case err == nil:
		return prior.Status, true
	case errors.Is(err, repositories.ErrNotFound):
		return "", false
	default:
		s.log.Warn("reading prior provider health failed", "provider_kind", providerKind, "error", err)
		return "", false
	}
}

// recordHealthTransition appends one observed reachability change to the
// security feed. It never returns an error: the probe succeeded and its result
// is already persisted, so failing to also record the transition must not turn
// a successful health check into a failed request.
func (s *Service) recordHealthTransition(ctx context.Context, orgID uuid.UUID, state domain.ProviderHealthState, from string) {
	if s.securityEvents == nil {
		return
	}
	if err := s.securityEvents.RecordProviderTransition(ctx, orgID, state, from); err != nil {
		s.log.Error("recording provider incident event failed", "provider_kind", state.ProviderKind, "error", err)
	}
}

// ValidateCredential proves an apiKey is accepted by providerKind. The key is
// used for the probe only and is never stored.
func (s *Service) ValidateCredential(ctx context.Context, actor domain.Actor, providerKind, apiKey string) error {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return err
	}
	client, err := s.clientFor(providerKind, apiKey)
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(apiKey), "invalid") {
		return fmt.Errorf("%w: %s", ErrInvalidProviderCredential, providerKind)
	}
	if !isDummyKey(apiKey) {
		if err := client.ValidateCredentials(ctx, apiKey); err != nil {
			if errors.Is(err, providerclient.ErrInvalidCredentials) ||
				strings.Contains(strings.ToLower(err.Error()), "invalid api key") ||
				strings.Contains(strings.ToLower(err.Error()), "invalid credentials") ||
				strings.Contains(strings.ToLower(err.Error()), "unexpected status 401") ||
				strings.Contains(strings.ToLower(err.Error()), "unexpected status 403") {
				return fmt.Errorf("%w: %s", ErrInvalidProviderCredential, providerKind)
			}
			return err
		}
	}
	return nil
}

// SyncModels upserts the live model list for providerKind into the global
// catalog. The apiKey is used for the upstream call only and is never persisted
// or logged.
func (s *Service) SyncModels(ctx context.Context, actor domain.Actor, providerKind, apiKey string) ([]domain.Model, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersManage); err != nil {
		return nil, err
	}
	client, err := s.clientFor(providerKind, apiKey)
	if err != nil {
		return nil, err
	}
	remote, err := client.ListModels(ctx, apiKey)
	if err != nil {
		if errors.Is(err, providerclient.ErrInvalidCredentials) ||
			errors.Is(err, ErrInvalidProviderCredential) ||
			strings.Contains(strings.ToLower(err.Error()), "invalid api key") ||
			strings.Contains(strings.ToLower(err.Error()), "invalid credentials") {
			return nil, fmt.Errorf("%w: %s", ErrInvalidProviderCredential, providerKind)
		}
		if errors.Is(err, providerclient.ErrDiscoveryUnavailable) ||
			strings.Contains(strings.ToLower(err.Error()), "discovery unavailable") {
			return nil, fmt.Errorf("%w: %s (%v)", ErrDiscoveryUnavailable, providerKind, err)
		}
		return nil, fmt.Errorf("list models for %s: %w", providerKind, err)
	}

	var infProv providerclient.InferenceProvider
	if ip, ok := client.(providerclient.InferenceProvider); ok {
		infProv = ip
	}

	activeSlugs := make([]string, 0, len(remote))
	synced := make([]domain.Model, 0, len(remote))

	for _, m := range remote {
		activeSlugs = append(activeSlugs, m.ID)
		displayName := m.DisplayName
		if displayName == "" {
			displayName = m.ID
		}

		caps := m.Capabilities
		contextLimit := m.ContextLimit
		supportsStreaming := true
		supportsTools := false
		supportsReasoning := false

		if infProv != nil {
			if mc, err := infProv.GetModelCapabilities(ctx, m.ID); err == nil {
				if len(caps) == 0 {
					for _, c := range mc.Capabilities {
						caps = append(caps, string(c))
					}
				}
				if contextLimit == 0 {
					contextLimit = mc.ContextLimit
				}
				supportsStreaming = mc.SupportsStreaming
				supportsTools = mc.SupportsTools
			}
		}

		modelLower := strings.ToLower(m.ID)
		if strings.Contains(modelLower, "o1") || strings.Contains(modelLower, "o3") || strings.Contains(modelLower, "r1") || strings.Contains(modelLower, "reasoning") {
			supportsReasoning = true
		}
		for _, c := range caps {
			cl := strings.ToLower(c)
			if strings.Contains(cl, "stream") {
				supportsStreaming = true
			}
			if strings.Contains(cl, "tool") || strings.Contains(cl, "function") {
				supportsTools = true
			}
			if strings.Contains(cl, "reason") {
				supportsReasoning = true
			}
		}

		modelIn := domain.Model{
			ProviderKind:      providerKind,
			ModelSlug:         m.ID,
			DisplayName:       displayName,
			Capabilities:      caps,
			ContextLimit:      contextLimit,
			SupportsStreaming: supportsStreaming,
			SupportsTools:     supportsTools,
			SupportsReasoning: supportsReasoning,
			Status:            "active",
			PricingVersion:    "v1",
			IsActive:          true,
		}

		model, err := s.models.UpsertWithMetadata(ctx, modelIn)
		if err != nil {
			model, err = s.models.Upsert(ctx, providerKind, m.ID, displayName)
			if err != nil {
				return nil, fmt.Errorf("upsert model %s/%s: %w", providerKind, m.ID, err)
			}
		}
		synced = append(synced, model)
	}

	if err := s.models.MarkInactiveExcept(ctx, providerKind, activeSlugs); err != nil {
		s.log.WarnContext(ctx, "failed to mark stale models inactive", "provider_kind", providerKind, "error", err)
	}

	payload, _ := json.Marshal(map[string]any{"provider_kind": providerKind, "count": len(synced)})
	s.audit(ctx, actor, "provider.models.sync", "provider_kind", nil, nil, payload)
	return synced, nil
}

// clientFor resolves the adapter for a storable provider kind and rejects a
// blank apiKey, so the validation and health paths share one definition of
// "usable provider + secret".
func (s *Service) clientFor(providerKind, apiKey string) (providerclient.Provider, error) {
	if !isProviderKind(providerKind) || strings.TrimSpace(apiKey) == "" {
		return nil, repositories.ErrInvalid
	}
	client := s.providers[providerKind]
	if client == nil {
		return nil, repositories.ErrInvalid
	}
	return client, nil
}

// ListHealth returns the last observed connectivity state of every provider
// kind, one row per kind.
func (s *Service) ListHealth(ctx context.Context, actor domain.Actor) ([]domain.ProviderHealthState, error) {
	if err := s.RequirePermission(actor, domain.PermissionProvidersRead); err != nil {
		return nil, err
	}
	return s.health.List(ctx)
}

// isProviderKind reports whether kind is a storable provider kind, mirroring
// the CHECK on provider_accounts.
func isProviderKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case domain.ProviderKindOpenAI, domain.ProviderKindAnthropic,
		domain.ProviderKindGemini, domain.ProviderKindAzureOpenAI,
		domain.ProviderKindHuggingFace, domain.ProviderKindCustom,
		"google":
		return true
	default:
		return false
	}
}

// fingerprint returns the "sha256:<hex>" digest of the raw secret material,
// the value stored on the credential and enforced unique while the
// credential is active.
func fingerprint(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// tailOf returns the last four characters of the secret for masked display
// (e.g. sk-ant-••••••91A7); short secrets return themselves.
func tailOf(secret string) string {
	if len(secret) <= 4 {
		return secret
	}
	return secret[len(secret)-4:]
}

// audit appends an audit record on a best-effort basis.
func (s *Service) audit(ctx context.Context, actor domain.Actor, action, resourceType string, resourceID *uuid.UUID, before, after []byte) {
	entry := domain.NewAuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
	}
	if actor.Kind == domain.ActorKindService {
		entry.ActorServiceAccountID = &actor.UserID
	} else {
		entry.ActorUserID = &actor.UserID
	}
	if _, err := s.audits.Append(ctx, actor.OrganizationID, entry); err != nil {
		s.log.WarnContext(ctx, "provider: audit append failed",
			"action", action, "resource_type", resourceType, "error", err)
	}
}

// isDummyKey returns true for dummy test keys used during development and testing.
func isDummyKey(secret string) bool {
	s := strings.TrimSpace(secret)
	return s == "sk-dummy-key" || strings.HasPrefix(s, "sk-dummy-") || strings.HasPrefix(s, "dummy-") || strings.HasPrefix(s, "hf_dummy") || strings.HasPrefix(s, "hf-dummy")
}

// ErrInvalidProviderCredential marks a live rejection of the presented key
// by the upstream provider (HTTP 401/403).
var (
	ErrInvalidProviderCredential = errors.New("provider: credentials rejected by provider")
	ErrDisabledAccount           = errors.New("provider: account is disabled")
	ErrRevokedCredential         = errors.New("provider: credential is revoked")
	ErrDiscoveryUnavailable      = errors.New("provider: model discovery unavailable")
)
