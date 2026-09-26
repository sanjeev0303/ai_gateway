-- name: AppendAuditLog :one
INSERT INTO audit_logs (
    organization_id, actor_user_id, actor_service_account_id, action,
    resource_type, resource_id, before, after, ip_address, request_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10)
RETURNING *;