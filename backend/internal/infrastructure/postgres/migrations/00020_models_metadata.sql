-- +goose Up
-- M4: governance-owned model catalog metadata.
--
-- Providers advertise only facts (providerclient.Model); the platform layers
-- governance fields -- capability flags, status, pricing version -- on top
-- when seeding the catalog. The status CHECK mirrors domain.ModelStatus*.
-- is_active is kept as the query-side activation flag.

ALTER TABLE models
    ADD COLUMN capabilities       text[]   NOT NULL DEFAULT '{}',
    ADD COLUMN context_limit      integer  NOT NULL DEFAULT 0,
    ADD COLUMN supports_streaming boolean  NOT NULL DEFAULT true,
    ADD COLUMN supports_tools     boolean  NOT NULL DEFAULT true,
    ADD COLUMN supports_reasoning boolean  NOT NULL DEFAULT false,
    ADD COLUMN status             text     NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'inactive')),
    ADD COLUMN pricing_version    text     NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE models
    DROP COLUMN pricing_version,
    DROP COLUMN status,
    DROP COLUMN supports_reasoning,
    DROP COLUMN supports_tools,
    DROP COLUMN supports_streaming,
    DROP COLUMN context_limit,
    DROP COLUMN capabilities;