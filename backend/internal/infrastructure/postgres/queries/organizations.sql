-- name: CreateOrganization :one
INSERT INTO organizations (name, slug)
VALUES ($1, $2)
RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations
WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations
WHERE slug = $1;