-- +goose Up
-- M4: provider connectivity health snapshot.
--
-- One row per provider kind records the last observed connectivity state of
-- that upstream, as probed by live health checks: a stable machine-readable
-- status (unknown/healthy/degraded/unhealthy), a stable reason code, and the
-- human-readable last error (which must never contain secrets or request
-- payloads). The table deliberately carries no organization_id — upstream
-- connectivity is a global fact, not a tenant-owned one — so it is NOT
-- covered by the RLS loop in 00012 and must only be queried with the
-- non-tenant query path.

CREATE TABLE provider_health (
    provider_kind text        PRIMARY KEY
                             CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom')),
    status        text        NOT NULL DEFAULT 'unknown'
                             CHECK (status IN ('unknown', 'healthy', 'degraded', 'unhealthy')),
    status_reason text        NOT NULL DEFAULT '',
    last_error    text        NOT NULL DEFAULT '',
    checked_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER provider_health_set_updated_at
    BEFORE UPDATE ON provider_health
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS provider_health_set_updated_at ON provider_health;
DROP TABLE provider_health;