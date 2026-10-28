-- name: GetUserAuthByEmail :one
-- The only read path that projects password_hash into a credential. Ordinary
-- user reads may still select the column, but toUser maps them into
-- domain.User, which has no digest field, so the hash cannot reach a DTO.
SELECT * FROM users
WHERE lower(email) = lower($1);

-- name: SetUserPasswordHash :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: TouchUserLastLogin :exec
UPDATE users
SET last_login_at = now()
WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO auth_refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM auth_refresh_tokens
WHERE token_hash = $1;

-- name: RevokeRefreshToken :execrows
-- Single-use rotation: a row is retired exactly once, and a second attempt
-- reports 0 rows affected so the caller can treat reuse as a stolen token.
UPDATE auth_refresh_tokens
SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: RevokeAllRefreshTokensForUser :execrows
-- Sign-out-everywhere and the response to detected token reuse.
UPDATE auth_refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: PurgeExpiredRefreshTokens :execrows
DELETE FROM auth_refresh_tokens
WHERE expires_at < $1 OR (revoked_at IS NOT NULL AND revoked_at < $1);

