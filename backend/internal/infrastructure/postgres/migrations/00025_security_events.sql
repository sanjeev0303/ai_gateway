-- +goose Up
-- Platform-originated events: provider incidents and pipeline failures.
-- These are deliberately NOT audit records. audit_logs (00009) already owns
-- control-plane changes and has its own read API, so mirroring those rows here
-- would be a second, weaker copy of the same truth. security_events covers only
-- what audit_logs does not: things the platform detects about itself.
--
-- Append-only, like audit_logs: an UPDATE/DELETE trigger hard-rejects mutation.
CREATE TABLE security_events (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    category        text        NOT NULL,
    severity        text        NOT NULL,
    title           text        NOT NULL,
    detail          jsonb,
    source          text        NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX security_events_org_created_at_idx ON security_events (organization_id, created_at);
CREATE INDEX security_events_org_category_created_idx ON security_events (organization_id, category, created_at);

-- +goose StatementBegin
CREATE FUNCTION security_events_protect() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'security_events are append-only (update/delete forbidden)';
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER security_events_append_only
    BEFORE UPDATE OR DELETE ON security_events
    FOR EACH ROW EXECUTE FUNCTION security_events_protect();

-- Created after 00012, so this table declares its own tenant guard rather than
-- joining that migration's backfill array.
-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['security_events'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (organization_id = app.current_org_id()) WITH CHECK (organization_id = app.current_org_id())',
            t || '_org_isolation', t);
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER security_events_append_only ON security_events;
DROP FUNCTION security_events_protect();
DROP TABLE security_events;