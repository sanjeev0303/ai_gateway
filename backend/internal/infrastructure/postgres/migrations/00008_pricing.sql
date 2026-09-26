-- +goose Up
-- Pricing models are per-organization model/cost entries. Prices are stored
-- as integer micros (never floats) in the model's currency.
CREATE TABLE pricing_models (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    model_slug      text        NOT NULL,
    provider_kind   text        NOT NULL
                                CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom')),
    display_name    text        NOT NULL DEFAULT '',
    currency        text        NOT NULL DEFAULT 'usd',
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'inactive')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, model_slug)
);
CREATE INDEX pricing_models_org_created_at_idx ON pricing_models (organization_id, created_at);

-- Versioned price sheets let billing move to a new price without rewriting
-- historical usage rows.
CREATE TABLE pricing_versions (
    id                       uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    pricing_model_id         uuid        NOT NULL REFERENCES pricing_models (id) ON DELETE CASCADE,
    organization_id          uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    version                  int         NOT NULL,
    input_unit_price_micros  bigint      NOT NULL DEFAULT 0,
    output_unit_price_micros bigint      NOT NULL DEFAULT 0,
    effective_from           timestamptz NOT NULL,
    created_at               timestamptz NOT NULL DEFAULT now(),
    UNIQUE (pricing_model_id, version)
);
CREATE INDEX pricing_versions_org_model_idx ON pricing_versions (organization_id, pricing_model_id);
CREATE INDEX pricing_versions_org_created_idx ON pricing_versions (organization_id, created_at);

-- +goose Down
DROP TABLE pricing_versions;
DROP TABLE pricing_models;