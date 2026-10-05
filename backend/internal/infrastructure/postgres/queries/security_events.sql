-- name: AppendSecurityEvent :one
INSERT INTO security_events (
    organization_id, category, severity, title, detail, source
)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING *;


-- name: ListSecurityEvents :many
-- Read-only projection of the append-only event ledger. Append-only holds
-- structurally: this file is the only place that names security_events and it
-- contains no UPDATE or DELETE, so no code path can rewrite a recorded event.
SELECT e.id,
       e.organization_id,
       e.category,
       e.severity,
       e.title,
       e.detail,
       e.source,
       e.created_at
FROM security_events e
WHERE e.organization_id = sqlc.arg(organization_id)
  AND (sqlc.narg(category)::text IS NULL OR e.category = sqlc.narg(category)::text)
  AND (sqlc.narg(severity)::text IS NULL OR e.severity = sqlc.narg(severity)::text)
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR e.created_at >= sqlc.narg(from_ts)::timestamptz)
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR e.created_at < sqlc.narg(to_ts)::timestamptz)
ORDER BY e.created_at DESC, e.id DESC
LIMIT sqlc.arg(limit_count)::int OFFSET sqlc.arg(offset_count)::int;