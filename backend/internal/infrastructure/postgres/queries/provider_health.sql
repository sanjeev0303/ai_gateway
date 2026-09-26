-- name: GetProviderHealth :one
SELECT * FROM provider_health
WHERE provider_kind = $1;

-- name: ListProviderHealth :many
SELECT * FROM provider_health
ORDER BY provider_kind;

-- name: UpsertProviderHealth :one
INSERT INTO provider_health (provider_kind, status, status_reason, last_error, checked_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider_kind) DO UPDATE SET
    status        = EXCLUDED.status,
    status_reason = EXCLUDED.status_reason,
    last_error    = EXCLUDED.last_error,
    checked_at    = EXCLUDED.checked_at
RETURNING *;