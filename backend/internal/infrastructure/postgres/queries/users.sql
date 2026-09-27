-- name: CreateUser :one
INSERT INTO users (email, display_name)
VALUES ($1, $2)
RETURNING *;

-- name: CreateUserWithPassword :one
INSERT INTO users (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE lower(email) = lower($1);

-- name: AddOrgMember :one
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveOrgMember :execrows
DELETE FROM organization_members
WHERE organization_id = $1 AND user_id = $2;

-- name: ListOrgMembers :many
SELECT u.id, u.email, u.display_name, u.status, u.created_at, u.updated_at, u.last_login_at,
       m.role, m.created_at AS member_since
FROM organization_members m
JOIN users u ON u.id = m.user_id
WHERE m.organization_id = $1
ORDER BY m.created_at;

-- name: ListOrganizationsForUser :many
-- Cross-tenant read (login / org switcher): organization_members is FORCE RLS,
-- so the join runs inside the SECURITY DEFINER app.organizations_for_user.
SELECT *
FROM app.organizations_for_user($1);

-- name: GetOrgMember :one
SELECT u.id, u.email, u.display_name, u.status, u.created_at, u.updated_at, u.last_login_at,
       m.role, m.created_at AS member_since
FROM organization_members m
JOIN users u ON u.id = m.user_id
WHERE m.organization_id = $1 AND m.user_id = $2;

-- name: SetOrgMemberRole :one
UPDATE organization_members
SET role = $3
WHERE organization_id = $1 AND user_id = $2
RETURNING organization_id, user_id, role, created_at;

-- name: CountOrgMembersByRole :one
SELECT count(*)
FROM organization_members
WHERE organization_id = $1 AND role = $2;