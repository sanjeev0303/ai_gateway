-- +goose Up
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_alert_type_check;
ALTER TABLE alerts ADD CONSTRAINT alerts_alert_type_check
    CHECK (alert_type IN (
        'spend_limit', 'spend_threshold', 'spend_exceeded',
        'budget_warning', 'budget_exhaustion',
        'rate_limit', 'rate_limit_spike', 'request_rate_spike',
        'error_rate', 'error_rate_spike',
        'anomaly', 'usage_anomaly', 'usage_spike', 'spend_spike',
        'provider_failure', 'credential_failure',
        'policy_violation', 'policy_denial_spike'
    ));

CREATE TABLE IF NOT EXISTS in_app_notifications (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    alert_id        uuid        REFERENCES alerts (id) ON DELETE CASCADE,
    alert_event_id  uuid        REFERENCES alert_events (id) ON DELETE CASCADE,
    title           text        NOT NULL,
    message         text        NOT NULL,
    severity        text        NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'critical')),
    read            boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS in_app_notifications_org_read_idx ON in_app_notifications (organization_id, read, created_at DESC);

ALTER TABLE in_app_notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE in_app_notifications FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS in_app_notifications_org_isolation ON in_app_notifications;
CREATE POLICY in_app_notifications_org_isolation ON in_app_notifications
    USING (organization_id = app.current_org_id())
    WITH CHECK (organization_id = app.current_org_id());

CREATE INDEX IF NOT EXISTS audit_logs_org_created_idx ON audit_logs (organization_id, created_at);
CREATE INDEX IF NOT EXISTS billing_events_org_created_idx ON billing_events (organization_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS in_app_notifications;
