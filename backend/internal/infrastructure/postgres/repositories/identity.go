package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type organizationRepo struct {
	store *postgres.Store
}

// NewOrganizationRepository returns an OrganizationRepository backed by the
// store. Organizations are a global table, so writes run without a tenant
// binding.
func NewOrganizationRepository(store *postgres.Store) ports.OrganizationRepository {
	return &organizationRepo{store: store}
}

func (r *organizationRepo) Create(ctx context.Context, input domain.NewOrganization) (domain.Organization, error) {
	var org domain.Organization
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.CreateOrganization(ctx, db.CreateOrganizationParams{Name: input.Name, Slug: input.Slug})
		if err != nil {
			return err
		}
		org = toOrganization(row)
		return nil
	})
	return org, mapUniqueViolation(err)
}

func (r *organizationRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	var org domain.Organization
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetOrganizationByID(ctx, id)
		if err != nil {
			return err
		}
		org = toOrganization(row)
		return nil
	})
	return org, mapNotFound(err)
}

func (r *organizationRepo) GetBySlug(ctx context.Context, slug string) (domain.Organization, error) {
	var org domain.Organization
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetOrganizationBySlug(ctx, slug)
		if err != nil {
			return err
		}
		org = toOrganization(row)
		return nil
	})
	return org, mapNotFound(err)
}

type userRepo struct {
	store *postgres.Store
}

// NewUserRepository returns a UserRepository backed by the store. Users are a
// global identity registry, so reads/writes run without a tenant binding.
func NewUserRepository(store *postgres.Store) ports.UserRepository {
	return &userRepo{store: store}
}

func (r *userRepo) Create(ctx context.Context, input domain.NewUser) (domain.User, error) {
	var user domain.User
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.CreateUser(ctx, db.CreateUserParams{Email: input.Email, DisplayName: input.DisplayName})
		if err != nil {
			return err
		}
		user = toUser(row)
		return nil
	})
	return user, mapUniqueViolation(err)
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	var user domain.User
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetUserByID(ctx, id)
		if err != nil {
			return err
		}
		user = toUser(row)
		return nil
	})
	return user, mapNotFound(err)
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetUserByEmail(ctx, email)
		if err != nil {
			return err
		}
		user = toUser(row)
		return nil
	})
	return user, mapNotFound(err)
}

func (r *userRepo) ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error) {
	var orgs []domain.UserOrganization
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		rows, err := q.ListOrganizationsForUser(ctx, userID)
		if err != nil {
			return err
		}
		orgs = make([]domain.UserOrganization, 0, len(rows))
		for _, row := range rows {
			orgs = append(orgs, toUserOrganization(row))
		}
		return nil
	})
	return orgs, err
}

type orgMemberRepo struct {
	store *postgres.Store
}

// NewOrgMemberRepository returns an OrgMemberRepository backed by the store.
// organization_members is FORCE RLS, so every operation runs inside a tenant
// transaction scoped to orgID.
func NewOrgMemberRepository(store *postgres.Store) ports.OrgMemberRepository {
	return &orgMemberRepo{store: store}
}

func (r *orgMemberRepo) Add(ctx context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
	var member domain.OrganizationMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.AddOrgMember(ctx, db.AddOrgMemberParams{OrganizationID: orgID, UserID: userID, Role: role})
		if err != nil {
			return err
		}
		member = domain.OrganizationMember{
			OrganizationID: row.OrganizationID,
			UserID:         row.UserID,
			Role:           row.Role,
			CreatedAt:      row.CreatedAt,
		}
		return nil
	})
	return member, mapUniqueViolation(err)
}

func (r *orgMemberRepo) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.RemoveOrgMember(ctx, db.RemoveOrgMemberParams{OrganizationID: orgID, UserID: userID})
		if err != nil {
			return err
		}
		if n == 0 {
			return ports.ErrNotFound
		}
		return nil
	})
	return err
}

func (r *orgMemberRepo) Get(ctx context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
	var member domain.OrgMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetOrgMember(ctx, db.GetOrgMemberParams{OrganizationID: orgID, UserID: userID})
		if err != nil {
			return err
		}
		member = toOrgMemberRow(row)
		return nil
	})
	return member, mapNotFound(err)
}

func (r *orgMemberRepo) SetRole(ctx context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
	var member domain.OrganizationMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.SetOrgMemberRole(ctx, db.SetOrgMemberRoleParams{OrganizationID: orgID, UserID: userID, Role: role})
		if err != nil {
			return err
		}
		member = toOrganizationMember(row)
		return nil
	})
	return member, mapNotFound(err)
}

func (r *orgMemberRepo) CountByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	var count int64
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		var err error
		count, err = q.CountOrgMembersByRole(ctx, db.CountOrgMembersByRoleParams{OrganizationID: orgID, Role: role})
		return err
	})
	return count, err
}

func (r *orgMemberRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
	var members []domain.OrgMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListOrgMembers(ctx, orgID)
		if err != nil {
			return err
		}
		members = make([]domain.OrgMember, 0, len(rows))
		for _, row := range rows {
			members = append(members, toOrgMember(row))
		}
		return nil
	})
	return members, err
}

func toOrganization(o db.Organization) domain.Organization {
	return domain.Organization{
		ID:        o.ID,
		Name:      o.Name,
		Slug:      o.Slug,
		Status:    o.Status,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

func toUser(u db.User) domain.User {
	return domain.User{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

func toUserProfile(id uuid.UUID, email, displayName, status string, createdAt, updatedAt time.Time, lastLoginAt *time.Time) domain.User {
	return domain.User{
		ID:          id,
		Email:       email,
		DisplayName: displayName,
		Status:      status,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		LastLoginAt: lastLoginAt,
	}
}

func toOrgMember(r db.ListOrgMembersRow) domain.OrgMember {
	return domain.OrgMember{
		User:        toUserProfile(r.ID, r.Email, r.DisplayName, r.Status, r.CreatedAt, r.UpdatedAt, r.LastLoginAt),
		Role:        r.Role,
		MemberSince: r.MemberSince,
	}
}

func toOrgMemberRow(r db.GetOrgMemberRow) domain.OrgMember {
	return domain.OrgMember{
		User:        toUserProfile(r.ID, r.Email, r.DisplayName, r.Status, r.CreatedAt, r.UpdatedAt, r.LastLoginAt),
		Role:        r.Role,
		MemberSince: r.MemberSince,
	}
}

func toOrganizationMember(m db.OrganizationMember) domain.OrganizationMember {
	return domain.OrganizationMember{
		OrganizationID: m.OrganizationID,
		UserID:         m.UserID,
		Role:           m.Role,
		CreatedAt:      m.CreatedAt,
	}
}

func toUserOrganization(r db.AppUserOrganization) domain.UserOrganization {
	return domain.UserOrganization{
		Organization: domain.Organization{
			ID:        r.ID,
			Name:      r.Name,
			Slug:      r.Slug,
			Status:    r.Status,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		},
		Role: r.Role,
	}
}
