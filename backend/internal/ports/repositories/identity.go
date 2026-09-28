package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// OrganizationRepository manages tenant roots. Organizations form a global
// table, so reads are not tenant-scoped.
type OrganizationRepository interface {
	Create(ctx context.Context, input domain.NewOrganization) (domain.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (domain.Organization, error)
	// ListAllIDs returns every organization ID. organizations is a global table
	// outside the tenant RLS set, so this crosses tenant boundaries by design.
	ListAllIDs(ctx context.Context) ([]uuid.UUID, error)
}

// UserRepository manages the global identity registry.
type UserRepository interface {
	Create(ctx context.Context, input domain.NewUser) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	// ListOrganizationsForUser reads across tenants via the SECURITY DEFINER
	// app.organizations_for_user function (login / org switcher path).
	ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error)
}

// AuthRepository is the only persistence surface that can read or write a
// password hash. Keeping it separate from UserRepository is what guarantees
// the credential cannot ride along on an ordinary user read: the sqlc User
// model does carry the column, so the protection is the narrow interface,
// not the generated struct.
type AuthRepository interface {
	// Register creates the user, their password digest, the organization they
	// own, the owner membership, and the initial refresh token as a single
	// transaction. Nothing is persisted unless all five writes land, so a
	// failure cannot leave an account that can never sign in, an organization
	// with no owner, or a user with no session. It returns ErrConflict when
	// the email or the organization slug is already taken.
	Register(ctx context.Context, reg domain.Registration) (domain.User, domain.Organization, error)
	// GetCredentialByEmail returns the identity plus its stored password
	// hash. Returns ErrNotFound when the email is unknown; a nil
	// PasswordHash means the account exists but has no password yet (users
	// provisioned through the dev-login path).
	GetCredentialByEmail(ctx context.Context, email string) (domain.UserCredential, error)
	// SetPasswordHash writes a new password digest for an existing user.
	SetPasswordHash(ctx context.Context, userID uuid.UUID, encoded string) error
	// TouchLastLogin records that an authentication attempt succeeded.
	TouchLastLogin(ctx context.Context, userID uuid.UUID) error
}

// RefreshTokenRepository persists refresh credentials by digest. Rows are
// single-use: a rotation retires the presented row, so presenting one again
// is detectable reuse rather than a silent re-authorization.
type RefreshTokenRepository interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (domain.RefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	// Revoke retires an active token. It returns ErrNotFound when the token is
	// unknown or already revoked.
	Revoke(ctx context.Context, tokenHash string) error
	// RevokeAllForUser signs the user out everywhere. It is the response to
	// confirmed reuse and the primitive behind "log out all devices".
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// OrgMemberRepository manages organization_members rows. The join table is
// FORCE RLS; Add/Remove/List all run inside the tenant context.
type OrgMemberRepository interface {
	Add(ctx context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error)
	Get(ctx context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error)
	Remove(ctx context.Context, orgID, userID uuid.UUID) error
	// SetRole changes a member's role; returns ErrNotFound when the member
	// does not exist.
	SetRole(ctx context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error)
	CountByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error)
	List(ctx context.Context, orgID uuid.UUID) ([]domain.OrgMember, error)
}
