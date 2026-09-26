-- +goose Up
-- Row-level security for every tenant-owned table.
--
-- app.current_org_id() reads the session GUC 'app.org_id'. Application code
-- sets the GUC inside an explicit transaction before touching tenant data
-- (SET LOCAL semantics via set_config(..., true)); every query also passes
-- organization_id explicitly, so RLS is defense-in-depth rather than the
-- only guard. FORCE RLS makes even the table owner subject to the policy, so
-- a forgotten GUC fails loudly instead of leaking rows across tenants.

CREATE SCHEMA app;

CREATE FUNCTION app.current_org_id() RETURNS uuid AS $$
    SELECT NULLIF(current_setting('app.org_id', TRUE), '')::uuid
$$ LANGUAGE sql STABLE;

-- Keep updated_at fresh for tables that carry it.
-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Audit logs are immutable: any attempt to UPDATE or DELETE one fails hard.
-- +goose StatementBegin
CREATE FUNCTION audit_logs_protect() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs are append-only (update/delete forbidden)';
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Apply the tenant guard to each table owning an organization_id column.
-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'organization_members',
        'projects',
        'teams',
        'team_members',
        'provider_accounts',
        'provider_credentials',
        'api_credentials',
        'service_accounts',
        'policies',
        'policy_rules',
        'conversations',
        'conversation_messages',
        'llm_requests',
        'usage_events',
        'usage_hourly',
        'usage_daily',
        'usage_monthly',
        'idempotency_keys',
        'pricing_models',
        'pricing_versions',
        'audit_logs',
        'subscriptions',
        'billing_events',
        'alerts'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (organization_id = app.current_org_id()) WITH CHECK (organization_id = app.current_org_id())',
            t || '_org_isolation', t);
    END LOOP;
END
$$;
-- +goose StatementEnd

CREATE TRIGGER organizations_set_updated_at BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER projects_set_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER teams_set_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER provider_accounts_set_updated_at BEFORE UPDATE ON provider_accounts FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER provider_credentials_set_updated_at BEFORE UPDATE ON provider_credentials FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER api_credentials_set_updated_at BEFORE UPDATE ON api_credentials FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER service_accounts_set_updated_at BEFORE UPDATE ON service_accounts FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER policies_set_updated_at BEFORE UPDATE ON policies FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER conversations_set_updated_at BEFORE UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER pricing_models_set_updated_at BEFORE UPDATE ON pricing_models FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER pricing_versions_set_updated_at BEFORE UPDATE ON pricing_versions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER plans_set_updated_at BEFORE UPDATE ON plans FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER subscriptions_set_updated_at BEFORE UPDATE ON subscriptions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER billing_events_set_updated_at BEFORE UPDATE ON billing_events FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER alerts_set_updated_at BEFORE UPDATE ON alerts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER audit_logs_append_only BEFORE UPDATE OR DELETE ON audit_logs FOR EACH ROW EXECUTE FUNCTION audit_logs_protect();

-- The gateway validates API keys in the request hot path, where no tenant
-- context exists yet: the key itself identifies the organization. RLS would
-- block a cross-tenant lookup by key_hash, so the lookup runs as the migration
-- owner (superuser) via SECURITY DEFINER. Callers see only the one matching
-- credential; org-scoped access to api_credentials still goes through RLS.
-- +goose StatementBegin
CREATE FUNCTION app.resolve_api_credential(p_hash text) RETURNS api_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM api_credentials WHERE key_hash = p_hash LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER audit_logs_append_only ON audit_logs;
DROP TRIGGER alerts_set_updated_at ON alerts;
DROP TRIGGER billing_events_set_updated_at ON billing_events;
DROP TRIGGER subscriptions_set_updated_at ON subscriptions;
DROP TRIGGER plans_set_updated_at ON plans;
DROP TRIGGER pricing_versions_set_updated_at ON pricing_versions;
DROP TRIGGER pricing_models_set_updated_at ON pricing_models;
DROP TRIGGER conversations_set_updated_at ON conversations;
DROP TRIGGER policies_set_updated_at ON policies;
DROP TRIGGER service_accounts_set_updated_at ON service_accounts;
DROP TRIGGER api_credentials_set_updated_at ON api_credentials;
DROP TRIGGER provider_credentials_set_updated_at ON provider_credentials;
DROP TRIGGER provider_accounts_set_updated_at ON provider_accounts;
DROP TRIGGER teams_set_updated_at ON teams;
DROP TRIGGER projects_set_updated_at ON projects;
DROP TRIGGER users_set_updated_at ON users;
DROP TRIGGER organizations_set_updated_at ON organizations;

-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'organization_members',
        'projects',
        'teams',
        'team_members',
        'provider_accounts',
        'provider_credentials',
        'api_credentials',
        'service_accounts',
        'policies',
        'policy_rules',
        'conversations',
        'conversation_messages',
        'llm_requests',
        'usage_events',
        'usage_hourly',
        'usage_daily',
        'usage_monthly',
        'idempotency_keys',
        'pricing_models',
        'pricing_versions',
        'audit_logs',
        'subscriptions',
        'billing_events',
        'alerts'
    ] LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', t || '_org_isolation', t);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
    END LOOP;
END
$$;
-- +goose StatementEnd

DROP FUNCTION audit_logs_protect();
DROP FUNCTION set_updated_at();
DROP FUNCTION app.current_org_id();
DROP SCHEMA app;