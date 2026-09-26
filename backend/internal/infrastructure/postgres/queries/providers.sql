-- name: CreateProviderAccount :one
INSERT INTO provider_accounts (organization_id, provider_kind, name, base_url, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProviderAccount :one
SELECT * FROM provider_accounts
WHERE organization_id = $1 AND id = $2;

-- name: ListProviderAccounts :many
SELECT * FROM provider_accounts
WHERE organization_id = $1
ORDER BY created_at;

-- name: CreateProviderCredential :one
INSERT INTO provider_credentials (provider_account_id, organization_id, name, kind, secret_ref, fingerprint, tail, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetProviderCredential :one
SELECT * FROM provider_credentials
WHERE organization_id = $1 AND id = $2;

-- name: ListProviderCredentials :many
SELECT * FROM provider_credentials
WHERE organization_id = $1
ORDER BY created_at;

-- name: ListProviderCredentialsByAccount :many
SELECT * FROM provider_credentials
WHERE organization_id = $1 AND provider_account_id = $2 AND status = 'active'
ORDER BY created_at DESC;

-- name: RevokeProviderCredential :one
UPDATE provider_credentials
SET status = 'revoked'
WHERE organization_id = $1 AND id = $2
RETURNING *;