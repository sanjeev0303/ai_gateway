-- +goose Up
-- M4: DB-driven model catalog.
--
-- The set of models the platform can route to is a global registry: adding a
-- model is a row, not a code change. The table deliberately carries no
-- organization_id — every tenant sees the same catalog — so it is NOT covered
-- by the RLS loop in 00012 and must only be queried outside a tenant-scoped
-- transaction (or via app.supports_global()). Application code reads it with
-- the non-tenant query path.

CREATE TABLE models (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_kind text        NOT NULL
                            CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom')),
    model_slug    text        NOT NULL,
    display_name  text        NOT NULL,
    is_active     boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider_kind, model_slug)
);

CREATE INDEX models_provider_active_idx ON models (provider_kind, is_active);

CREATE TRIGGER models_set_updated_at
    BEFORE UPDATE ON models
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS models_set_updated_at ON models;
DROP TABLE models;