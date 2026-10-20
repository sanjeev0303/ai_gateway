-- +goose Up
-- Add cache_read_input_token_price_micros and effective_to to pricing_versions
ALTER TABLE pricing_versions
    ADD COLUMN IF NOT EXISTS cache_read_input_token_price_micros bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS effective_to timestamptz;

-- Add check constraints to reject negative prices and ensure valid effective window
ALTER TABLE pricing_versions
    DROP CONSTRAINT IF EXISTS pricing_versions_input_price_check,
    ADD CONSTRAINT pricing_versions_input_price_check CHECK (input_unit_price_micros >= 0);

ALTER TABLE pricing_versions
    DROP CONSTRAINT IF EXISTS pricing_versions_output_price_check,
    ADD CONSTRAINT pricing_versions_output_price_check CHECK (output_unit_price_micros >= 0);

ALTER TABLE pricing_versions
    DROP CONSTRAINT IF EXISTS pricing_versions_cache_price_check,
    ADD CONSTRAINT pricing_versions_cache_price_check CHECK (cache_read_input_token_price_micros >= 0);

ALTER TABLE pricing_versions
    DROP CONSTRAINT IF EXISTS pricing_versions_effective_window_check,
    ADD CONSTRAINT pricing_versions_effective_window_check CHECK (effective_to IS NULL OR effective_to > effective_from);

CREATE INDEX IF NOT EXISTS pricing_versions_effective_idx 
    ON pricing_versions (organization_id, pricing_model_id, effective_from DESC, version DESC);

-- +goose Down
DROP INDEX IF EXISTS pricing_versions_effective_idx;
ALTER TABLE pricing_versions
    DROP CONSTRAINT IF EXISTS pricing_versions_effective_window_check,
    DROP CONSTRAINT IF EXISTS pricing_versions_cache_price_check,
    DROP CONSTRAINT IF EXISTS pricing_versions_output_price_check,
    DROP CONSTRAINT IF EXISTS pricing_versions_input_price_check,
    DROP COLUMN IF EXISTS effective_to,
    DROP COLUMN IF EXISTS cache_read_input_token_price_micros;
