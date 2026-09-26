-- name: AddProjectMember :one
INSERT INTO project_members (project_id, organization_id, user_id, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetProjectMember :one
SELECT * FROM project_members
WHERE project_id = $1 AND organization_id = $2 AND user_id = $3;

-- name: RemoveProjectMember :execrows
DELETE FROM project_members
WHERE project_id = $1 AND organization_id = $2 AND user_id = $3;

-- name: ListProjectMembers :many
SELECT u.id, u.email, u.display_name, u.status, u.created_at, u.updated_at, u.last_login_at,
       m.role, m.created_at AS member_since
FROM project_members m
JOIN users u ON u.id = m.user_id
WHERE m.project_id = $1 AND m.organization_id = $2
ORDER BY m.created_at;