package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// OrganizationRepository manages tenant roots. Organizations form a global
// table, so reads are not tenant-scoped.
type OrganizationRepository interface {
	Create(ctx context.Context, input domain.NewOrganization) (domain.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (domain.Organization, error)
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
