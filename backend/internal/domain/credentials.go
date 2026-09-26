package domain

import (
	"time"

	"github.com/google/uuid"
)

// Provider kinds, mirroring the CHECK in provider_accounts.
const (
	ProviderKindOpenAI      = "openai"
	ProviderKindAnthropic   = "anthropic"
	ProviderKindGemini      = "gemini"
	ProviderKindAzureOpenAI = "azure_openai"
	ProviderKindCustom      = "custom"
)

// IsProviderKind reports whether kind is a supported provider kind.
func IsProviderKind(kind string) bool {
	switch kind {
	case ProviderKindOpenAI, ProviderKindAnthropic, ProviderKindGemini, ProviderKindAzureOpenAI, ProviderKindCustom:
		return true
	default:
		return false
	}
}

const (
	ProviderStatusActive   = "active"
	ProviderStatusDisabled = "disabled"
)

// ProviderCredential kinds.
const (
	CredentialKindAPIKey = "api_key"
	CredentialKindOAuth  = "oauth"
)

const (
	ProviderCredentialStatusActive  = "active"
	ProviderCredentialStatusRevoked = "revoked"
)

// GatewayCredentialScope defines the permission scope for gateway credentials.
type GatewayCredentialScope string

const (
	GatewayCredentialScopeGateway GatewayCredentialScope = "gateway"
	GatewayCredentialScopeControl GatewayCredentialScope = "control"
	GatewayCredentialScopeAll     GatewayCredentialScope = "all"
)

// GatewayCredentialStatus represents the lifecycle state of a gateway credential.
type GatewayCredentialStatus string

const (
	GatewayCredentialStatusActive   GatewayCredentialStatus = "active"
	GatewayCredentialStatusRevoked  GatewayCredentialStatus = "revoked"
	GatewayCredentialStatusExpired  GatewayCredentialStatus = "expired"
)

// APICredentialScope defines the permission scope for API credentials.
type APICredentialScope string

const (
	APICredentialScopeGateway APICredentialScope = "gateway"
	APICredentialScopeControl APICredentialScope = "control"
	APICredentialScopeAll     APICredentialScope = "all"
)

const (
	APICredentialStatusActive  = "active"
	APICredentialStatusRevoked = "revoked"
)

// SubjectType classifies the credential subject.
type SubjectType string

const (
	SubjectTypeUser           SubjectType = "user"
	SubjectTypeServiceAccount SubjectType = "service_account"
)

// NewGatewayCredential carries the caller-supplied fields for creating a
// gateway credential. SubjectType + SubjectID supports both users and
// service accounts. TeamID/ProjectID enable sub-tenant scoping. Scopes is
// a slice for composable permissions. ExpiresAt enables short-lived tokens.
// KeyPrefix and KeyHash are set by the application layer (not caller-supplied).
type NewGatewayCredential struct {
	Name        string
	KeyPrefix   string
	KeyHash     string
	SubjectType SubjectType
	SubjectID   uuid.UUID
	TeamID      *uuid.UUID
	ProjectID   *uuid.UUID
	Scopes      []GatewayCredentialScope
	ExpiresAt   *time.Time
}

// IsValidSubjectType reports whether the subject type is valid.
func IsValidSubjectType(t SubjectType) bool {
	return t == SubjectTypeUser || t == SubjectTypeServiceAccount
}

// IsValidGatewayCredentialScope reports whether the scope is valid.
func IsValidGatewayCredentialScope(s GatewayCredentialScope) bool {
	return s == GatewayCredentialScopeGateway || s == GatewayCredentialScopeControl || s == GatewayCredentialScopeAll
}

// GatewayCredential is the stored credential (NEVER stores raw secret).
// KeyPrefix is the first 7 chars after the "aigw_" prefix for identification.
// KeyHash is the argon2id hash of the full raw key.
type GatewayCredential struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	Name            string
	KeyPrefix       string
	KeyHash         string
	SubjectType     SubjectType
	SubjectID       uuid.UUID
	TeamID          *uuid.UUID
	ProjectID       *uuid.UUID
	Scopes          []GatewayCredentialScope
	Status          GatewayCredentialStatus
	ExpiresAt       *time.Time
	LastUsedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const (
	ServiceAccountStatusActive  = "active"
	ServiceAccountStatusRevoked = "revoked"
)

// NewProviderAccount carries the caller-supplied fields for creating a
// provider account.
type NewProviderAccount struct {
	ProviderKind string
	Name         string
	BaseURL      string
}

// ProviderAccount is a connection to an upstream AI provider.
type ProviderAccount struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProviderKind   string
	Name           string
	BaseURL        string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewProviderCredential carries the caller-supplied fields for creating a
// provider credential. Fingerprint ("sha256:"+hex of the secret material) and
// Tail (last four characters) are computed by the application layer from the
// raw secret before it is stored in the SecretStore; only these two derived
// values are ever persisted next to the credential.
type NewProviderCredential struct {
	ProviderAccountID uuid.UUID
	Name              string
	Kind              string
	SecretRef         string
	Fingerprint       string
	Tail              string
}

// ProviderCredential is an upstream API key or OAuth token of a provider
// account, referenced by secret ref rather than stored inline. Fingerprint
// may be empty for credentials created before M4.
type ProviderCredential struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	ProviderAccountID uuid.UUID
	Name              string
	Kind              string
	SecretRef         string
	Fingerprint       string
	Tail              string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewAPICredential carries the caller-supplied fields for creating an API
// credential scoped to the gateway or control plane. ServiceAccountID must be
// set for control/all scopes: non-gateway keys authenticate a machine
// identity, never a human.
type NewAPICredential struct {
	Name             string
	KeyPrefix        string
	KeyHash          string
	Scope            string
	ServiceAccountID *uuid.UUID
}

// APICredential authenticates gateway or control-plane callers.
type APICredential struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	Name             string
	KeyPrefix        string
	KeyHash          string
	Scope            string
	Status           string
	ServiceAccountID *uuid.UUID
	LastUsedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewServiceAccount carries the caller-supplied fields for creating a
// machine identity for automation.
type NewServiceAccount struct {
	Name        string
	Description string
}

// ServiceAccount is a machine identity inside an organization.
type ServiceAccount struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
