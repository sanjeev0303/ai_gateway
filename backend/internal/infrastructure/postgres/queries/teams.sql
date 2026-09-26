-- name: CreateTeam :one
INSERT INTO teams (organization_id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams
WHERE organization_id = $1 AND id = $2;

-- name: ListTeams :many
SELECT * FROM teams
WHERE organization_id = $1
ORDER BY created_at;

-- name: AddTeamMember :one
INSERT INTO team_members (team_id, organization_id, user_id, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: RemoveTeamMember :execrows
DELETE FROM team_members
WHERE team_id = $1 AND organization_id = $2 AND user_id = $3;

-- name: ListTeamMembers :many
SELECT u.id, u.email, u.display_name, u.status, u.created_at, u.updated_at, u.last_login_at,
       m.role, m.created_at AS member_since
FROM team_members m
JOIN users u ON u.id = m.user_id
WHERE m.team_id = $1 AND m.organization_id = $2
ORDER BY m.created_at;