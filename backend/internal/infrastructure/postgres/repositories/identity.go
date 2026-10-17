package repositories

import (
	"context"
	"fmt"
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

func (r *organizationRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateOrganization) (domain.Organization, error) {
	var org domain.Organization
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.UpdateOrganization(ctx, db.UpdateOrganizationParams{
			ID:                         id,
			Name:                       input.Name,
			Slug:                       input.Slug,
			PrivacyDefault:             input.PrivacyDefault,
			RetentionConversationsDays: input.RetentionConversationsDays,
			RetentionUsageDays:         input.RetentionUsageDays,
			RetentionAuditsDays:        input.RetentionAuditsDays,
			RetentionBillingDays:       input.RetentionBillingDays,
		})
		if err != nil {
			return err
		}
		org = toOrganization(row)
		return nil
	})
	return org, mapUniqueViolation(mapNotFound(err))
}

func (r *organizationRepo) ListAllIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		var err error
		ids, err = q.ListAllOrganizationIDs(ctx)
		return err
	})
	return ids, err
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

type authRepo struct {
	store *postgres.Store
}

// NewAuthRepository returns an AuthRepository backed by the store. Users are a
// global identity registry, so credential reads run without a tenant binding.
func NewAuthRepository(store *postgres.Store) ports.AuthRepository {
	return &authRepo{store: store}
}

func (r *authRepo) Register(ctx context.Context, reg domain.Registration) (domain.User, domain.Organization, error) {
	var (
		user domain.User
		org  domain.Organization
	)
	// The org ID is supplied by the caller, so the tenant context is bound
	// before any row exists. organization_members is FORCE RLS and compares
	// organization_id to app.current_org_id(), so the binding must be in
	// place for the membership insert; users and organizations are global
	// tables and ignore it.
	hash := reg.PasswordHash
	err := r.store.WithTenantTx(ctx, reg.OrganizationID, func(q *db.Queries) error {
		u, err := q.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
			Email:        reg.User.Email,
			DisplayName:  reg.User.DisplayName,
			PasswordHash: &hash,
		})
		if err != nil {
			return err
		}
		o, err := q.CreateOrganizationWithID(ctx, db.CreateOrganizationWithIDParams{
			ID:   reg.OrganizationID,
			Name: reg.OrganizationName,
			Slug: reg.OrganizationSlug,
		})
		if err != nil {
			return err
		}
		if _, err := q.AddOrgMember(ctx, db.AddOrgMemberParams{
			OrganizationID: o.ID,
			UserID:         u.ID,
			Role:           domain.RoleOwner,
		}); err != nil {
			return err
		}
		if _, err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
			UserID:    u.ID,
			TokenHash: reg.RefreshTokenHash,
			ExpiresAt: reg.RefreshTokenExpiresAt,
		}); err != nil {
			return err
		}
		user, org = toUser(u), toOrganization(o)
		return nil
	})
	if err != nil {
		if uniqueViolationOn(err, "users_email_lower_key") {
			return domain.User{}, domain.Organization{}, fmt.Errorf("%w: %v", ports.ErrEmailTaken, err)
		}
		return domain.User{}, domain.Organization{}, mapUniqueViolation(err)
	}
	return user, org, nil
}

func (r *authRepo) GetCredentialByEmail(ctx context.Context, email string) (domain.UserCredential, error) {
	var cred domain.UserCredential
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetUserAuthByEmail(ctx, email)
		if err != nil {
			return err
		}
		cred = domain.UserCredential{User: toUser(row), PasswordHash: row.PasswordHash}
		return nil
	})
	return cred, mapNotFound(err)
}

func (r *authRepo) SetPasswordHash(ctx context.Context, userID uuid.UUID, encoded string) error {
	return r.store.WithTx(ctx, func(q *db.Queries) error {
		return q.SetUserPasswordHash(ctx, db.SetUserPasswordHashParams{ID: userID, PasswordHash: &encoded})
	})
}

func (r *authRepo) TouchLastLogin(ctx context.Context, userID uuid.UUID) error {
	return r.store.WithTx(ctx, func(q *db.Queries) error {
		return q.TouchUserLastLogin(ctx, userID)
	})
}

type refreshTokenRepo struct {
	store *postgres.Store
}

// NewRefreshTokenRepository returns a RefreshTokenRepository backed by the
// store. The table carries no org RLS: refresh runs before any tenant context
// exists, and a row is reachable only by its high-entropy digest.
func NewRefreshTokenRepository(store *postgres.Store) ports.RefreshTokenRepository {
	return &refreshTokenRepo{store: store}
}

func (r *refreshTokenRepo) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}
		token = toRefreshToken(row)
		return nil
	})
	return token, err
}

func (r *refreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetRefreshTokenByHash(ctx, tokenHash)
		if err != nil {
			return err
		}
		token = toRefreshToken(row)
		return nil
	})
	return token, mapNotFound(err)
}

func (r *refreshTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	return r.store.WithTx(ctx, func(q *db.Queries) error {
		n, err := q.RevokeRefreshToken(ctx, tokenHash)
		if err != nil {
			return err
		}
		// Single-use rotation: 0 rows means the row was already retired, so the
		// caller is looking at a replayed token rather than a live one.
		if n == 0 {
			return ports.ErrNotFound
		}
		return nil
	})
}

func (r *refreshTokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.store.WithTx(ctx, func(q *db.Queries) error {
		_, err := q.RevokeAllRefreshTokensForUser(ctx, userID)
		return err
	})
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
		ID:                         o.ID,
		Name:                       o.Name,
		Slug:                       o.Slug,
		Status:                     o.Status,
		PrivacyDefault:             domain.PrivacyMode(o.PrivacyDefault),
		RetentionConversationsDays: int(o.RetentionConversationsDays),
		RetentionUsageDays:         int(o.RetentionUsageDays),
		RetentionAuditsDays:        int(o.RetentionAuditsDays),
		RetentionBillingDays:       int(o.RetentionBillingDays),
		CreatedAt:                  o.CreatedAt,
		UpdatedAt:                  o.UpdatedAt,
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

func toRefreshToken(t db.AuthRefreshToken) domain.RefreshToken {
	return domain.RefreshToken{
		ID:         t.ID,
		UserID:     t.UserID,
		TokenHash:  t.TokenHash,
		ExpiresAt:  t.ExpiresAt,
		RevokedAt:  t.RevokedAt,
		CreatedAt:  t.CreatedAt,
		LastUsedAt: t.LastUsedAt,
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
