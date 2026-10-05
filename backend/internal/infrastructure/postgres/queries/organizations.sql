-- name: CreateOrganization :one
INSERT INTO organizations (name, slug)
VALUES ($1, $2)
RETURNING *;

-- name: CreateOrganizationWithID :one
INSERT INTO organizations (id, name, slug)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations
WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations
WHERE slug = $1;

-- name: ListAllOrganizationIDs :many
SELECT id FROM organizations
ORDER BY created_at;
-- name: UpdateOrganization :one
UPDATE organizations
SET
  name = COALESCE(sqlc.narg('name')::text, name),
  slug = COALESCE(sqlc.narg('slug')::text, slug),
  updated_at = now()
WHERE id = $1
RETURNING *;
