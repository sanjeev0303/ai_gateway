-- +goose Up
-- Audit logs are append-only: an UPDATE/DELETE trigger (added in 00012_rls)
-- hard-rejects any mutation. Content is stored as before/after JSON snapshots.
CREATE TABLE audit_logs (
    id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id         uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    actor_user_id           uuid        REFERENCES users (id) ON DELETE SET NULL,
    actor_service_account_id uuid       REFERENCES service_accounts (id) ON DELETE SET NULL,
    action                  text        NOT NULL,
    resource_type           text        NOT NULL,
    resource_id             uuid,
    before                  jsonb,
    after                   jsonb,
    ip_address              text,
    request_id              text,
    created_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_org_created_at_idx ON audit_logs (organization_id, created_at);
CREATE INDEX audit_logs_org_actor_created_idx ON audit_logs (organization_id, actor_user_id, created_at);
CREATE INDEX audit_logs_org_resource_created_idx ON audit_logs (organization_id, resource_type, created_at);

-- +goose Down
DROP TABLE audit_logs;