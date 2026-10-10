-- name: CreatePolicy :one
INSERT INTO policies (organization_id, name, description, enforcement, priority)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPolicy :one
SELECT * FROM policies
WHERE organization_id = $1 AND id = $2;

-- name: ListPolicies :many
SELECT * FROM policies
WHERE organization_id = $1
ORDER BY priority, created_at;

-- name: UpdatePolicy :one
UPDATE policies
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    enforcement = COALESCE(sqlc.narg('enforcement'), enforcement),
    priority = COALESCE(sqlc.narg('priority'), priority)
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: DeletePolicy :exec
DELETE FROM policies
WHERE organization_id = $1 AND id = $2;

-- name: CreatePolicyRule :one
INSERT INTO policy_rules (policy_id, organization_id, rule_type, description, conditions, parameters, enabled)
VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
RETURNING *;

-- name: ListPolicyRules :many
SELECT * FROM policy_rules
WHERE organization_id = $1 AND policy_id = $2
ORDER BY created_at;

-- name: GetPolicyRule :one
SELECT * FROM policy_rules
WHERE organization_id = $1 AND policy_id = $2 AND id = $3;

-- name: UpdatePolicyRule :one
UPDATE policy_rules
SET description = COALESCE(sqlc.narg('description'), description),
    conditions = COALESCE(sqlc.narg('conditions')::jsonb, conditions),
    parameters = COALESCE(sqlc.narg('parameters')::jsonb, parameters),
    enabled = COALESCE(sqlc.narg('enabled'), enabled)
WHERE organization_id = $1 AND policy_id = $2 AND id = $3
RETURNING *;

-- name: DeletePolicyRule :exec
DELETE FROM policy_rules
WHERE organization_id = $1 AND policy_id = $2 AND id = $3;