-- +goose Up
-- Provider accounts are the organization's configured LLM providers.
CREATE TABLE provider_accounts (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    provider_kind   text        NOT NULL
                                CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom')),
    name            text        NOT NULL,
    base_url        text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'disabled')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX provider_accounts_org_created_at_idx ON provider_accounts (organization_id, created_at);
CREATE INDEX provider_accounts_org_kind_idx ON provider_accounts (organization_id, provider_kind);

-- Provider credentials reference where the secret material lives (env var
-- name today; vault path in the security milestone). The secret itself is
-- never stored in this table.
CREATE TABLE provider_credentials (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    provider_account_id uuid        NOT NULL REFERENCES provider_accounts (id) ON DELETE CASCADE,
    name                text        NOT NULL,
    kind                text        NOT NULL DEFAULT 'api_key'
                                    CHECK (kind IN ('api_key', 'oauth')),
    secret_ref          text        NOT NULL,
    status              text        NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active', 'revoked')),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX provider_credentials_org_created_at_idx ON provider_credentials (organization_id, created_at);
CREATE INDEX provider_credentials_account_idx ON provider_credentials (provider_account_id);

-- API credentials authenticate gateway clients. Only a key prefix and a
-- SHA-256 hash of the key are stored; the raw key is shown once at creation.
CREATE TABLE api_credentials (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            text        NOT NULL,
    key_prefix      text        NOT NULL,
    key_hash        text        NOT NULL UNIQUE,
    scope           text        NOT NULL DEFAULT 'gateway'
                                CHECK (scope IN ('gateway', 'control', 'all')),
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'revoked')),
    last_used_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX api_credentials_org_status_idx ON api_credentials (organization_id, status);
CREATE INDEX api_credentials_org_created_at_idx ON api_credentials (organization_id, created_at);

-- Service accounts are machine identities that own audit actions.
CREATE TABLE service_accounts (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'revoked')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX service_accounts_org_created_at_idx ON service_accounts (organization_id, created_at);

-- +goose Down
DROP TABLE service_accounts;
DROP TABLE api_credentials;
DROP TABLE provider_credentials;
DROP TABLE provider_accounts;