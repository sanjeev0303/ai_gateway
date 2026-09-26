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

-- name: SetPolicyEnforcement :one
UPDATE policies
SET enforcement = $3
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: CreatePolicyRule :one
INSERT INTO policy_rules (policy_id, organization_id, rule_type, description, conditions, parameters, enabled)
VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
RETURNING *;

-- name: ListPolicyRules :many
SELECT * FROM policy_rules
WHERE organization_id = $1 AND policy_id = $2
ORDER BY created_at;

-- name: SetPolicyRuleEnabled :one
UPDATE policy_rules
SET enabled = $3
WHERE organization_id = $1 AND id = $2
RETURNING *;