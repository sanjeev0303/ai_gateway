-- +goose Up
-- Alerts are the organization's configured governance/watchdog rules.
CREATE TABLE alerts (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    alert_type       text        NOT NULL
                                 CHECK (alert_type IN ('spend_limit', 'rate_limit', 'error_rate', 'anomaly')),
    name             text        NOT NULL,
    severity         text        NOT NULL DEFAULT 'warning'
                                 CHECK (severity IN ('info', 'warning', 'critical')),
    configuration    jsonb       NOT NULL DEFAULT '{}'::jsonb,
    status           text        NOT NULL DEFAULT 'active'
                                 CHECK (status IN ('active', 'muted')),
    last_triggered_at timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX alerts_org_created_at_idx ON alerts (organization_id, created_at);

-- +goose Down
DROP TABLE alerts;