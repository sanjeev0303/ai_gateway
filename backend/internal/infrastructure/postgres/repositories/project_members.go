package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type projectMemberRepo struct {
	store *postgres.Store
}

// NewProjectMemberRepository returns a ProjectMemberRepository backed by the
// store. project_members is FORCE RLS, so every operation runs inside the
// tenant context scoped to orgID.
func NewProjectMemberRepository(store *postgres.Store) ports.ProjectMemberRepository {
	return &projectMemberRepo{store: store}
}

func (r *projectMemberRepo) Add(ctx context.Context, orgID, projectID, userID uuid.UUID, role string) (domain.ProjectMember, error) {
	var member domain.ProjectMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.AddProjectMember(ctx, db.AddProjectMemberParams{ProjectID: projectID, OrganizationID: orgID, UserID: userID, Role: role})
		if err != nil {
			return err
		}
		member = toProjectMember(row)
		return nil
	})
	return member, mapUniqueViolation(err)
}

func (r *projectMemberRepo) Get(ctx context.Context, orgID, projectID, userID uuid.UUID) (domain.ProjectMember, error) {
	var member domain.ProjectMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetProjectMember(ctx, db.GetProjectMemberParams{ProjectID: projectID, OrganizationID: orgID, UserID: userID})
		if err != nil {
			return err
		}
		member = toProjectMember(row)
		return nil
	})
	return member, mapNotFound(err)
}

func (r *projectMemberRepo) Remove(ctx context.Context, orgID, projectID, userID uuid.UUID) error {
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.RemoveProjectMember(ctx, db.RemoveProjectMemberParams{ProjectID: projectID, OrganizationID: orgID, UserID: userID})
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

func (r *projectMemberRepo) List(ctx context.Context, orgID, projectID uuid.UUID) ([]domain.ProjectMemberView, error) {
	var members []domain.ProjectMemberView
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListProjectMembers(ctx, db.ListProjectMembersParams{ProjectID: projectID, OrganizationID: orgID})
		if err != nil {
			return err
		}
		members = make([]domain.ProjectMemberView, 0, len(rows))
		for _, row := range rows {
			members = append(members, toProjectMemberView(row))
		}
		return nil
	})
	return members, err
}

func toProjectMember(m db.ProjectMember) domain.ProjectMember {
	return domain.ProjectMember{
		ProjectID:      m.ProjectID,
		OrganizationID: m.OrganizationID,
		UserID:         m.UserID,
		Role:           m.Role,
		CreatedAt:      m.CreatedAt,
	}
}

func toProjectMemberView(r db.ListProjectMembersRow) domain.ProjectMemberView {
	return domain.ProjectMemberView{
		User:        toUserProfile(r.ID, r.Email, r.DisplayName, r.Status, r.CreatedAt, r.UpdatedAt, r.LastLoginAt),
		Role:        r.Role,
		MemberSince: r.MemberSince,
	}
}
