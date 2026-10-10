-- name: CreateBudget :one
INSERT INTO budgets (organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled, created_at, updated_at;

-- name: GetBudget :one
SELECT id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled, created_at, updated_at FROM budgets
WHERE organization_id = $1 AND id = $2;

-- name: ListBudgets :many
SELECT id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled, created_at, updated_at FROM budgets
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: ListBudgetsByScopeType :many
SELECT id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled, created_at, updated_at FROM budgets
WHERE organization_id = $1 AND scope = $2
ORDER BY created_at DESC;

-- name: ListBudgetsByScope :many
SELECT id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled, created_at, updated_at FROM budgets
WHERE organization_id = $1 AND scope = $2 AND scope_id = $3
ORDER BY created_at DESC;

-- name: UpdateBudget :one
UPDATE budgets
SET name = COALESCE(sqlc.narg('name')::text, name),
    description = COALESCE(sqlc.narg('description')::text, description),
    limit_amount = COALESCE(sqlc.narg('limit_amount')::numeric, limit_amount),
    alert_threshold = COALESCE(sqlc.narg('alert_threshold')::numeric, alert_threshold),
    enabled = COALESCE(sqlc.narg('enabled')::boolean, enabled),
    updated_at = now()
WHERE organization_id = $1 AND id = $2
RETURNING id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled, created_at, updated_at;

-- name: DeleteBudget :exec
DELETE FROM budgets WHERE organization_id = $1 AND id = $2;