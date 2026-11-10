-- +goose Up
-- Platform administration: cross-tenant reads for the super-admin console,
-- and enforcement of organization suspension and user disabling at the
-- gateway.
--
-- Every tenant table is FORCE RLS, so a normal session cannot read across
-- organizations. These reads run SECURITY DEFINER (as the migration owner),
-- the same pattern as app.organizations_for_user. Only the control API's
-- /v1/admin routes call them, and those routes require a platform admin.

-- Platform-wide time ranges scan llm_requests by created_at alone; every
-- existing index leads with organization_id.
CREATE INDEX IF NOT EXISTS llm_requests_created_at_idx ON llm_requests (created_at);

-- +goose StatementBegin
CREATE FUNCTION app.platform_overview(p_since timestamptz)
RETURNS TABLE (
    orgs_total         bigint,
    orgs_suspended     bigint,
    users_total        bigint,
    users_disabled     bigint,
    requests           bigint,
    failed_requests    bigint,
    tokens             bigint,
    cost_micros        bigint,
    active_keys        bigint,
    paid_subscriptions bigint,
    mrr_micros         bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT
        (SELECT count(*) FROM organizations),
        (SELECT count(*) FROM organizations WHERE status = 'suspended'),
        (SELECT count(*) FROM users),
        (SELECT count(*) FROM users WHERE status = 'disabled'),
        r.requests, r.failed, r.tokens, r.cost,
        (SELECT count(*) FROM gateway_credentials WHERE status = 'active'),
        s.paid, s.mrr
    FROM (
        SELECT count(*) AS requests,
               count(*) FILTER (WHERE status <> 'succeeded') AS failed,
               coalesce(sum(total_tokens), 0)::bigint AS tokens,
               coalesce(sum(cost_micros), 0)::bigint AS cost
        FROM llm_requests
        WHERE created_at >= p_since
    ) r,
    (
        SELECT count(*) FILTER (WHERE p.price_micros > 0) AS paid,
               coalesce(sum(p.price_micros), 0)::bigint AS mrr
        FROM subscriptions sub
        JOIN plans p ON p.id = sub.plan_id
        WHERE sub.status IN ('active', 'trialing')
    ) s;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_organizations(p_search text, p_status text, p_since timestamptz, p_limit int, p_offset int)
RETURNS TABLE (
    id                  uuid,
    name                text,
    slug                text,
    status              text,
    created_at          timestamptz,
    plan_code           text,
    plan_name           text,
    subscription_status text,
    members             bigint,
    active_keys         bigint,
    requests            bigint,
    cost_micros         bigint,
    last_request_at     timestamptz,
    total_count         bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT o.id, o.name, o.slug, o.status, o.created_at,
           sub.code, sub.name, sub.status,
           (SELECT count(*) FROM organization_members m WHERE m.organization_id = o.id),
           (SELECT count(*) FROM gateway_credentials g WHERE g.organization_id = o.id AND g.status = 'active'),
           u.requests, u.cost, u.last_at,
           count(*) OVER ()
    FROM organizations o
    LEFT JOIN LATERAL (
        SELECT p.code, p.name, s.status
        FROM subscriptions s
        JOIN plans p ON p.id = s.plan_id
        WHERE s.organization_id = o.id
        ORDER BY s.created_at DESC
        LIMIT 1
    ) sub ON true
    LEFT JOIN LATERAL (
        SELECT count(*) AS requests,
               coalesce(sum(r.cost_micros), 0)::bigint AS cost,
               max(r.created_at) AS last_at
        FROM llm_requests r
        WHERE r.organization_id = o.id AND r.created_at >= p_since
    ) u ON true
    WHERE (p_search = '' OR o.name ILIKE '%' || p_search || '%' OR o.slug ILIKE '%' || p_search || '%' OR o.id::text = p_search)
      AND (p_status = '' OR o.status = p_status)
    ORDER BY o.created_at DESC
    LIMIT p_limit OFFSET p_offset;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_organization_members(p_org uuid)
RETURNS TABLE (
    user_id       uuid,
    email         text,
    display_name  text,
    role          text,
    status        text,
    joined_at     timestamptz,
    last_login_at timestamptz
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT u.id, u.email, u.display_name, m.role, u.status, m.created_at, u.last_login_at
    FROM organization_members m
    JOIN users u ON u.id = m.user_id
    WHERE m.organization_id = p_org
    ORDER BY m.created_at;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_users(p_search text, p_status text, p_limit int, p_offset int)
RETURNS TABLE (
    id            uuid,
    email         text,
    display_name  text,
    status        text,
    created_at    timestamptz,
    last_login_at timestamptz,
    org_count     bigint,
    org_names     text,
    total_count   bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT u.id, u.email, u.display_name, u.status, u.created_at, u.last_login_at,
           coalesce(mo.n, 0), coalesce(mo.names, ''),
           count(*) OVER ()
    FROM users u
    LEFT JOIN LATERAL (
        SELECT count(*) AS n, string_agg(o.name, ', ' ORDER BY o.name) AS names
        FROM organization_members m
        JOIN organizations o ON o.id = m.organization_id
        WHERE m.user_id = u.id
    ) mo ON true
    WHERE (p_search = '' OR u.email ILIKE '%' || p_search || '%' OR u.display_name ILIKE '%' || p_search || '%' OR u.id::text = p_search)
      AND (p_status = '' OR u.status = p_status)
    ORDER BY u.created_at DESC
    LIMIT p_limit OFFSET p_offset;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_usage_daily(p_since timestamptz)
RETURNS TABLE (day date, requests bigint, failed_requests bigint, tokens bigint, cost_micros bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT (created_at AT TIME ZONE 'UTC')::date AS day,
           count(*),
           count(*) FILTER (WHERE status <> 'succeeded'),
           coalesce(sum(total_tokens), 0)::bigint,
           coalesce(sum(cost_micros), 0)::bigint
    FROM llm_requests
    WHERE created_at >= p_since
    GROUP BY 1
    ORDER BY 1;
$$;
-- +goose StatementEnd

-- p_dimension is one of 'provider', 'model', 'organization'.
-- +goose StatementBegin
CREATE FUNCTION app.platform_usage_breakdown(p_since timestamptz, p_dimension text, p_limit int)
RETURNS TABLE (key text, label text, requests bigint, tokens bigint, cost_micros bigint, avg_latency_ms bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT CASE p_dimension WHEN 'provider' THEN r.provider_kind WHEN 'model' THEN r.model ELSE r.organization_id::text END,
           CASE p_dimension WHEN 'provider' THEN r.provider_kind WHEN 'model' THEN r.model ELSE coalesce(o.name, r.organization_id::text) END,
           count(*),
           coalesce(sum(r.total_tokens), 0)::bigint,
           coalesce(sum(r.cost_micros), 0)::bigint,
           coalesce(avg(r.latency_ms), 0)::bigint
    FROM llm_requests r
    LEFT JOIN organizations o ON o.id = r.organization_id
    WHERE r.created_at >= p_since
    GROUP BY 1, 2
    ORDER BY 5 DESC, 3 DESC
    LIMIT p_limit;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_plan_summary()
RETURNS TABLE (plan_code text, plan_name text, price_micros bigint, currency text, subscriptions bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT p.code, p.name, p.price_micros, p.currency,
           count(s.id) FILTER (WHERE s.status IN ('active', 'trialing'))
    FROM plans p
    LEFT JOIN subscriptions s ON s.plan_id = p.id
    GROUP BY p.id
    ORDER BY p.price_micros;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_invoices(p_limit int)
RETURNS TABLE (
    id                uuid,
    organization_id   uuid,
    organization_name text,
    number            text,
    status            text,
    currency          text,
    total_micros      bigint,
    issued_at         timestamptz
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT i.id, i.organization_id, o.name, i.number, i.status, i.currency, i.total_micros, i.issued_at
    FROM invoices i
    JOIN organizations o ON o.id = i.organization_id
    ORDER BY i.issued_at DESC
    LIMIT p_limit;
$$;
-- +goose StatementEnd

-- Every platform admin action, append-only. It is a platform table, not a
-- tenant one (an action can target a user or the model catalog, which belong
-- to no organization), so it carries no RLS. Actions aimed at an organization
-- are also written to that organization's own audit_logs so its owners see
-- them.
CREATE TABLE platform_audit_log (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Plain IDs, no foreign keys: audit rows must outlive the user or
    -- organization they name, and the append-only trigger would reject the
    -- UPDATE an ON DELETE SET NULL performs.
    actor_user_id   uuid,
    actor_email     text        NOT NULL,
    action          text        NOT NULL,
    target_type     text        NOT NULL,
    target_id       uuid,
    organization_id uuid,
    reason          text        NOT NULL DEFAULT '',
    detail          jsonb       NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX platform_audit_log_created_at_idx ON platform_audit_log (created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION platform_audit_log_protect() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'platform_audit_log is append-only (update/delete forbidden)';
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER platform_audit_log_no_update
    BEFORE UPDATE OR DELETE ON platform_audit_log
    FOR EACH ROW EXECUTE FUNCTION platform_audit_log_protect();

-- Active gateway keys a user holds as their subject, in any organization, so
-- disabling the user can evict them from gateway caches.
-- +goose StatementBegin
CREATE FUNCTION app.platform_user_gateway_credentials(p_user uuid)
RETURNS TABLE (id uuid, organization_id uuid)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT g.id, g.organization_id
    FROM gateway_credentials g
    WHERE g.subject_type = 'user' AND g.subject_id = p_user AND g.status = 'active';
$$;
-- +goose StatementEnd

-- The gateway resolves keys through these. A key now stops working when its
-- organization is suspended or, for a user-subject key, its user is disabled.
-- A user-subject key whose subject is not a user row keeps resolving as it did
-- before: subject IDs were never validated, and failing those keys silently on
-- deploy would be an outage, not a fix.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.resolve_gateway_credential(p_hash text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT g.* FROM gateway_credentials g
    JOIN organizations o ON o.id = g.organization_id AND o.status = 'active'
    LEFT JOIN users u ON g.subject_type = 'user' AND u.id = g.subject_id
    WHERE g.key_hash = p_hash AND g.status = 'active'
      AND (g.expires_at IS NULL OR g.expires_at > now())
      AND (u.id IS NULL OR u.status = 'active')
    LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.resolve_gateway_credential_by_prefix(p_prefix text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT g.* FROM gateway_credentials g
    JOIN organizations o ON o.id = g.organization_id AND o.status = 'active'
    LEFT JOIN users u ON g.subject_type = 'user' AND u.id = g.subject_id
    WHERE g.key_prefix = p_prefix AND g.status = 'active'
      AND (g.expires_at IS NULL OR g.expires_at > now())
      AND (u.id IS NULL OR u.status = 'active');
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.resolve_gateway_credential(p_hash text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_hash = p_hash AND status = 'active' AND (expires_at IS NULL OR expires_at > now()) LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.resolve_gateway_credential_by_prefix(p_prefix text) RETURNS SETOF gateway_credentials
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT * FROM gateway_credentials WHERE key_prefix = p_prefix AND status = 'active' AND (expires_at IS NULL OR expires_at > now());
$$;
-- +goose StatementEnd

DROP FUNCTION IF EXISTS app.platform_user_gateway_credentials(uuid);
DROP TABLE IF EXISTS platform_audit_log;
DROP FUNCTION IF EXISTS platform_audit_log_protect();
DROP FUNCTION IF EXISTS app.platform_invoices(int);
DROP FUNCTION IF EXISTS app.platform_plan_summary();
DROP FUNCTION IF EXISTS app.platform_usage_breakdown(timestamptz, text, int);
DROP FUNCTION IF EXISTS app.platform_usage_daily(timestamptz);
DROP FUNCTION IF EXISTS app.platform_users(text, text, int, int);
DROP FUNCTION IF EXISTS app.platform_organization_members(uuid);
DROP FUNCTION IF EXISTS app.platform_organizations(text, text, timestamptz, int, int);
DROP FUNCTION IF EXISTS app.platform_overview(timestamptz);
DROP INDEX IF EXISTS llm_requests_created_at_idx;
