-- name: CreateProject :one
INSERT INTO projects (organization_id, name, slug, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects
WHERE organization_id = $1 AND id = $2;

-- name: ListProjects :many
SELECT * FROM projects
WHERE organization_id = $1
ORDER BY created_at;

-- name: SetProjectStatus :one
UPDATE projects
SET status = $3
WHERE organization_id = $1 AND id = $2
RETURNING *;-- name: UpdateProject :one
UPDATE projects
SET 
  name = COALESCE(sqlc.narg('name'), name),
  description = COALESCE(sqlc.narg('description'), description)
WHERE organization_id = $1 AND id = $2
RETURNING *;
