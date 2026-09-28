package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// ProjectRepository manages projects within an organization.
type ProjectRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewProject) (domain.Project, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Project, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Project, error)
	SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) (domain.Project, error)
}

// TeamRepository manages teams within an organization.
type TeamRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewTeam) (domain.Team, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Team, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Team, error)
}

// TeamMemberRepository manages team_members rows, joining the user profile
// on List.
type TeamMemberRepository interface {
	Add(ctx context.Context, orgID, teamID, userID uuid.UUID, role string) (domain.TeamMember, error)
	Remove(ctx context.Context, orgID, teamID, userID uuid.UUID) error
	List(ctx context.Context, orgID, teamID uuid.UUID) ([]domain.TeamMemberView, error)
}

// ProviderAccountRepository manages upstream provider connections.
type ProviderAccountRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewProviderAccount) (domain.ProviderAccount, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error)
}

// ProviderCredentialRepository manages upstream keys/tokens by secret ref.
type ProviderCredentialRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewProviderCredential) (domain.ProviderCredential, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error)
	// ListByAccount returns the ACTIVE credentials of one account, newest
	// first. This is the per-tenant resolve path for gateway request routing.
	ListByAccount(ctx context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error)
	Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error)
}

// ModelRepository is the DB-driven global model catalog. It is global by
// design (models are not tenant-owned), so operations are never org-scoped.
type ModelRepository interface {
	List(ctx context.Context) ([]domain.Model, error)
	Upsert(ctx context.Context, providerKind, modelSlug, displayName string) (domain.Model, error)
}

// ProviderHealthRepository records the last observed connectivity state of
// each upstream provider kind. Like the model catalog it is global by design
// (upstream connectivity is a platform fact, not a tenant-owned one), so
// operations are never org-scoped.
type ProviderHealthRepository interface {
	Get(ctx context.Context, providerKind string) (domain.ProviderHealthState, error)
	List(ctx context.Context) ([]domain.ProviderHealthState, error)
	// Upsert records the outcome of a live health probe for one provider
	// kind, replacing any previously observed state.
	Upsert(ctx context.Context, state domain.ProviderHealthState) error
}

// APICredentialRepository manages gateway/control-plane API keys.
type APICredentialRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewAPICredential) (domain.APICredential, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.APICredential, error)
	// ResolveByKeyHash is cross-tenant (gateway hot path): it resolves the
	// owning organization via the SECURITY DEFINER app.resolve_api_credential.
	ResolveByKeyHash(ctx context.Context, keyHash string) (domain.APICredential, error)
	Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.APICredential, error)
}

// ServiceAccountRepository manages machine identities.
type ServiceAccountRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewServiceAccount) (domain.ServiceAccount, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.ServiceAccount, error)
	Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error)
}

// GatewayCredentialRepository manages employee/service-account gateway credentials.
type GatewayCredentialRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewGatewayCredential) (domain.GatewayCredential, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error)
	// ResolveByKeyHash is cross-tenant (gateway hot path): it resolves the
	// owning organization via the SECURITY DEFINER app.resolve_gateway_credential.
	ResolveByKeyHash(ctx context.Context, keyHash string) (domain.GatewayCredential, error)
	// ResolveByKeyPrefix narrows candidates by prefix before hash verification.
	ResolveByKeyPrefix(ctx context.Context, keyPrefix string) ([]domain.GatewayCredential, error)
	Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error)
	// UpdateStatus atomically transitions status (active→revoked/expired, revoked→active for restore).
	UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status domain.GatewayCredentialStatus) (domain.GatewayCredential, error)
	// Rotate creates a new credential and revokes the old one in one transaction.
	Rotate(ctx context.Context, orgID, id uuid.UUID, input domain.NewGatewayCredential) (domain.GatewayCredential, string, error) // returns new cred + raw key
}

// PolicyRepository manages the compiled policy groups fronting the gateway.
type PolicyRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewPolicy) (domain.Policy, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (domain.Policy, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error)
	SetEnforcement(ctx context.Context, orgID, id uuid.UUID, enforcement string) (domain.Policy, error)
}

// PolicyRuleRepository manages the individual rules of a policy.
type PolicyRuleRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input domain.NewPolicyRule) (domain.PolicyRule, error)
	List(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error)
	SetEnabled(ctx context.Context, orgID, id uuid.UUID, enabled bool) (domain.PolicyRule, error)
}
