-- +goose Up
-- Indexes supporting automated background retention and garbage data pruning:
-- 1. Expired idempotency keys cleanup per tenant
-- 2. Expired / accepted / revoked invitations cleanup per tenant
CREATE INDEX IF NOT EXISTS idempotency_keys_org_expires_at_idx ON idempotency_keys (organization_id, expires_at);
CREATE INDEX IF NOT EXISTS invitations_org_expires_at_idx ON invitations (organization_id, expires_at);

-- +goose Down
DROP INDEX IF EXISTS invitations_org_expires_at_idx;
DROP INDEX IF EXISTS idempotency_keys_org_expires_at_idx;
