-- +goose Up
ALTER TABLE gateway_credentials
    ADD COLUMN IF NOT EXISTS provider_account_id uuid REFERENCES provider_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS allowed_providers text[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS allowed_models text[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS rate_limit integer,
    ADD COLUMN IF NOT EXISTS budget_limit numeric(12, 6);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.resolve_gateway_credential(p_hash text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_hash = p_hash AND status = 'active' AND (expires_at IS NULL OR expires_at > now()) LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.resolve_gateway_credential_by_prefix(p_prefix text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_prefix = p_prefix AND status = 'active' AND (expires_at IS NULL OR expires_at > now());
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.resolve_gateway_credential_by_prefix(text);
DROP FUNCTION IF EXISTS app.resolve_gateway_credential(text);

ALTER TABLE gateway_credentials
    DROP COLUMN IF EXISTS budget_limit,
    DROP COLUMN IF EXISTS rate_limit,
    DROP COLUMN IF EXISTS allowed_models,
    DROP COLUMN IF EXISTS allowed_providers,
    DROP COLUMN IF EXISTS provider_account_id;

-- +goose StatementBegin
CREATE FUNCTION app.resolve_gateway_credential(p_hash text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_hash = p_hash AND status = 'active' AND (expires_at IS NULL OR expires_at > now()) LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.resolve_gateway_credential_by_prefix(p_prefix text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_prefix = p_prefix AND status = 'active' AND (expires_at IS NULL OR expires_at > now());
$$;
-- +goose StatementEnd
