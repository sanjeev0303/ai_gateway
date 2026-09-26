-- name: CreateGatewayCredential :one
INSERT INTO gateway_credentials (organization_id, name, key_prefix, key_hash, subject_type, subject_id, team_id, project_id, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ResolveGatewayCredentialByKeyHash :one
SELECT * FROM app.resolve_gateway_credential($1);

-- name: ResolveGatewayCredentialByKeyPrefix :many
SELECT * FROM app.resolve_gateway_credential_by_prefix($1);

-- name: ListGatewayCredentials :many
SELECT * FROM gateway_credentials
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: GetGatewayCredential :one
SELECT * FROM gateway_credentials
WHERE organization_id = $1 AND id = $2;

-- name: RevokeGatewayCredential :one
UPDATE gateway_credentials
SET status = 'revoked', updated_at = now()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: UpdateGatewayCredentialStatus :one
UPDATE gateway_credentials
SET status = $3, updated_at = now()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: RevokeGatewayCredentialForRotate :one
UPDATE gateway_credentials
SET status = 'revoked', updated_at = now()
WHERE organization_id = $1 AND id = $2
RETURNING organization_id;

-- name: CreateGatewayCredentialAfterRotate :one
INSERT INTO gateway_credentials (organization_id, name, key_prefix, key_hash, subject_type, subject_id, team_id, project_id, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;