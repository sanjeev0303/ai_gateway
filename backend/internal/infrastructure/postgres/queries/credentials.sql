-- name: CreateAPICredential :one
INSERT INTO api_credentials (organization_id, name, key_prefix, key_hash, scope, status, service_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ResolveAPICredentialByKeyHash :one
SELECT * FROM app.resolve_api_credential($1);

-- name: ListAPICredentials :many
SELECT * FROM api_credentials
WHERE organization_id = $1
ORDER BY created_at;

-- name: RevokeAPICredential :one
UPDATE api_credentials
SET status = 'revoked'
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: CreateServiceAccount :one
INSERT INTO service_accounts (organization_id, name, description, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListServiceAccounts :many
SELECT * FROM service_accounts
WHERE organization_id = $1
ORDER BY created_at;

-- name: RevokeServiceAccount :one
UPDATE service_accounts
SET status = 'revoked'
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: GetServiceAccount :one
SELECT * FROM service_accounts
WHERE organization_id = $1 AND id = $2;