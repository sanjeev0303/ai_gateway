-- +goose Up
-- Expand alerts constraints to allow all supported alert types and resolved status
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_alert_type_check;
ALTER TABLE alerts ADD CONSTRAINT alerts_alert_type_check
    CHECK (alert_type IN (
        'spend_limit', 'spend_threshold', 'spend_exceeded',
        'rate_limit', 'rate_limit_spike', 'request_rate_spike',
        'error_rate', 'error_rate_spike',
        'anomaly', 'usage_anomaly',
        'provider_failure', 'credential_failure', 'policy_violation'
    ));

ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_status_check;
ALTER TABLE alerts ADD CONSTRAINT alerts_status_check
    CHECK (status IN ('active', 'muted', 'resolved'));

-- Alert events record the historical trigger instances and resolution events
CREATE TABLE alert_events (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    alert_id         uuid        NOT NULL REFERENCES alerts (id) ON DELETE CASCADE,
    alert_type       text        NOT NULL,
    severity         text        NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    status           text        NOT NULL DEFAULT 'triggered' CHECK (status IN ('triggered', 'resolved', 'acknowledged')),
    message          text        NOT NULL,
    context          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    triggered_at     timestamptz NOT NULL DEFAULT now(),
    resolved_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX alert_events_org_alert_idx ON alert_events (organization_id, alert_id, triggered_at DESC);
CREATE INDEX alert_events_org_triggered_idx ON alert_events (organization_id, triggered_at DESC);

ALTER TABLE alert_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE alert_events FORCE ROW LEVEL SECURITY;
CREATE POLICY alert_events_org_isolation ON alert_events
    USING (organization_id = app.current_org_id())
    WITH CHECK (organization_id = app.current_org_id());

-- +goose Down
DROP TABLE IF EXISTS alert_events;

ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_status_check;
ALTER TABLE alerts ADD CONSTRAINT alerts_status_check
    CHECK (status IN ('active', 'muted'));

ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_alert_type_check;
ALTER TABLE alerts ADD CONSTRAINT alerts_alert_type_check
    CHECK (alert_type IN ('spend_limit', 'rate_limit', 'error_rate', 'anomaly'));
