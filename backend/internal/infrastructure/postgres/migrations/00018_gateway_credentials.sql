-- +goose Up
CREATE TYPE gateway_credential_status AS ENUM ('active', 'revoked', 'expired');
CREATE TYPE gateway_credential_scope AS ENUM ('gateway', 'control', 'all');
CREATE TYPE gateway_subject_type AS ENUM ('user', 'service_account');

CREATE TABLE gateway_credentials (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            text NOT NULL,
    key_prefix      text NOT NULL,
    key_hash        text NOT NULL,
    subject_type    gateway_subject_type NOT NULL,
    subject_id      uuid NOT NULL,
    team_id         uuid REFERENCES teams(id) ON DELETE SET NULL,
    project_id      uuid REFERENCES projects(id) ON DELETE SET NULL,
    scopes          gateway_credential_scope[] NOT NULL,
    status          gateway_credential_status NOT NULL DEFAULT 'active',
    expires_at      timestamptz,
    last_used_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_gateway_credentials_org ON gateway_credentials(organization_id);
CREATE INDEX idx_gateway_credentials_key_prefix ON gateway_credentials(key_prefix);
CREATE INDEX idx_gateway_credentials_key_hash ON gateway_credentials(key_hash);
CREATE INDEX idx_gateway_credentials_subject ON gateway_credentials(subject_type, subject_id);

-- Cross-tenant resolver (SECURITY DEFINER, returns SETOF for ErrNotFound semantics)
-- +goose StatementBegin
CREATE FUNCTION app.resolve_gateway_credential(p_hash text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_hash = p_hash AND status = 'active' AND (expires_at IS NULL OR expires_at > now()) LIMIT 1;
$$;
-- +goose StatementEnd

-- Prefix-based resolver for cache warm-up and candidate narrowing
-- +goose StatementBegin
CREATE FUNCTION app.resolve_gateway_credential_by_prefix(p_prefix text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_prefix = p_prefix AND status = 'active' AND (expires_at IS NULL OR expires_at > now());
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.resolve_gateway_credential_by_prefix(text);
DROP FUNCTION IF EXISTS app.resolve_gateway_credential(text);
DROP TABLE IF EXISTS gateway_credentials;
DROP TYPE IF EXISTS gateway_subject_type;
DROP TYPE IF EXISTS gateway_credential_scope;
DROP TYPE IF EXISTS gateway_credential_status;