-- name: AppendAuditLog :one
INSERT INTO audit_logs (
    organization_id, actor_user_id, actor_service_account_id, action,
    resource_type, resource_id, before, after, ip_address, request_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10)
RETURNING *;


-- name: ListAuditLogs :many
-- Read-only projection of the append-only ledger. Append-only is preserved
-- structurally: this file is the only place that names audit_logs, and it
-- contains no UPDATE or DELETE statement, so no code path can rewrite history.
-- `actor_id` matches either actor column; `actor_type` then narrows to the
-- column that is actually populated.
SELECT a.id,
       a.organization_id,
       a.actor_user_id,
       a.actor_service_account_id,
       a.action,
       a.resource_type,
       a.resource_id,
       a.before,
       a.after,
       a.ip_address,
       a.request_id,
       a.created_at
FROM audit_logs a
WHERE a.organization_id = sqlc.arg(organization_id)
  AND (sqlc.narg(actor_id)::uuid IS NULL
       OR a.actor_user_id = sqlc.narg(actor_id)::uuid
       OR a.actor_service_account_id = sqlc.narg(actor_id)::uuid)
  AND (sqlc.narg(actor_type)::text IS NULL
       OR (sqlc.narg(actor_type)::text = 'user' AND a.actor_user_id IS NOT NULL)
       OR (sqlc.narg(actor_type)::text = 'service_account' AND a.actor_service_account_id IS NOT NULL))
  AND (sqlc.narg(action)::text IS NULL OR a.action = sqlc.narg(action)::text)
  AND (sqlc.narg(resource_type)::text IS NULL OR a.resource_type = sqlc.narg(resource_type)::text)
  AND (sqlc.narg(resource_id)::uuid IS NULL OR a.resource_id = sqlc.narg(resource_id)::uuid)
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR a.created_at >= sqlc.narg(from_ts)::timestamptz)
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR a.created_at < sqlc.narg(to_ts)::timestamptz)
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(limit_count)::int OFFSET sqlc.arg(offset_count)::int;


-- name: GetAuditLog :one
SELECT a.id,
       a.organization_id,
       a.actor_user_id,
       a.actor_service_account_id,
       a.action,
       a.resource_type,
       a.resource_id,
       a.before,
       a.after,
       a.ip_address,
       a.request_id,
       a.created_at
FROM audit_logs a
WHERE a.organization_id = sqlc.arg(organization_id)
  AND a.id = sqlc.arg(id);

-- name: PurgeExpiredAuditLogs :execrows
DELETE FROM audit_logs
WHERE organization_id = sqlc.arg(organization_id)
  AND created_at < sqlc.arg(older_than)::timestamptz;