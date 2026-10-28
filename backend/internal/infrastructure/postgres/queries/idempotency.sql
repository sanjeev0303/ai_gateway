-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (organization_id, scope_key, idempotency_key, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (organization_id, scope_key, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys
WHERE organization_id = $1 AND scope_key = $2 AND idempotency_key = $3;

-- name: PurgeExpiredIdempotencyKeys :execrows
DELETE FROM idempotency_keys
WHERE organization_id = $1 AND expires_at < $2;