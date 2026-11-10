-- +goose Up
-- Platform monitoring for the admin console: resource samples the worker
-- polls from every service, the warning and error log lines services write,
-- alerts raised on them, and cross-tenant reads of audit, security events and
-- payments. These are platform tables (no tenant column), so no RLS.

CREATE TABLE service_samples (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service     text        NOT NULL,
    sampled_at  timestamptz NOT NULL DEFAULT now(),
    up          boolean     NOT NULL,
    error       text        NOT NULL DEFAULT '',
    cpu_percent double precision,
    rss_bytes   bigint,
    heap_bytes  bigint,
    goroutines  integer,
    detail      jsonb       NOT NULL DEFAULT '{}'
);
CREATE INDEX service_samples_service_time_idx ON service_samples (service, sampled_at DESC);
CREATE INDEX service_samples_time_idx ON service_samples (sampled_at);

CREATE TABLE service_logs (
    id        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service   text        NOT NULL,
    level     text        NOT NULL CHECK (level IN ('WARN', 'ERROR')),
    message   text        NOT NULL,
    attrs     jsonb       NOT NULL DEFAULT '{}',
    logged_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX service_logs_time_idx ON service_logs (logged_at DESC);
CREATE INDEX service_logs_service_time_idx ON service_logs (service, logged_at DESC);

CREATE TABLE platform_alerts (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind            text        NOT NULL,
    service         text        NOT NULL,
    severity        text        NOT NULL CHECK (severity IN ('warning', 'critical')),
    title           text        NOT NULL,
    detail          text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'open'
                                CHECK (status IN ('open', 'acknowledged', 'resolved')),
    opened_at       timestamptz NOT NULL DEFAULT now(),
    last_seen_at    timestamptz NOT NULL DEFAULT now(),
    acknowledged_at timestamptz,
    acknowledged_by text        NOT NULL DEFAULT '',
    resolved_at     timestamptz,
    notified_at     timestamptz
);
-- One live alert per problem: a condition that persists updates its alert
-- instead of opening (and emailing) a new one every poll.
CREATE UNIQUE INDEX platform_alerts_live_key ON platform_alerts (kind, service) WHERE status <> 'resolved';
CREATE INDEX platform_alerts_opened_idx ON platform_alerts (opened_at DESC);

-- Every organization's audit log, newest first. p_org NULL means all.
-- +goose StatementBegin
CREATE FUNCTION app.platform_tenant_audit(p_org uuid, p_action text, p_limit int, p_offset int)
RETURNS TABLE (
    id                uuid,
    organization_id   uuid,
    organization_name text,
    actor_email       text,
    action            text,
    resource_type     text,
    resource_id       uuid,
    after             jsonb,
    ip_address        text,
    created_at        timestamptz,
    total_count       bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT a.id, a.organization_id, o.name, coalesce(u.email, ''),
           a.action, a.resource_type, a.resource_id, a.after, a.ip_address, a.created_at,
           count(*) OVER ()
    FROM audit_logs a
    JOIN organizations o ON o.id = a.organization_id
    LEFT JOIN users u ON u.id = a.actor_user_id
    WHERE (p_org IS NULL OR a.organization_id = p_org)
      AND (p_action = '' OR a.action ILIKE p_action || '%')
    ORDER BY a.created_at DESC
    LIMIT p_limit OFFSET p_offset;
$$;
-- +goose StatementEnd

-- Every organization's security events, newest first.
-- +goose StatementBegin
CREATE FUNCTION app.platform_security_events(p_severity text, p_limit int, p_offset int)
RETURNS TABLE (
    id                uuid,
    organization_id   uuid,
    organization_name text,
    category          text,
    severity          text,
    title             text,
    detail            jsonb,
    source            text,
    created_at        timestamptz,
    total_count       bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT e.id, e.organization_id, o.name, e.category, e.severity, e.title,
           coalesce(e.detail, '{}'::jsonb), e.source, e.created_at,
           count(*) OVER ()
    FROM security_events e
    JOIN organizations o ON o.id = e.organization_id
    WHERE p_severity = '' OR e.severity = p_severity
    ORDER BY e.created_at DESC
    LIMIT p_limit OFFSET p_offset;
$$;
-- +goose StatementEnd

-- Earnings: every paid invoice is one payment taken.
-- +goose StatementBegin
CREATE FUNCTION app.platform_revenue_summary()
RETURNS TABLE (
    lifetime_micros   bigint,
    last_30d_micros   bigint,
    this_month_micros bigint,
    payments          bigint,
    paying_orgs       bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT coalesce(sum(total_micros), 0)::bigint,
           coalesce(sum(total_micros) FILTER (WHERE coalesce(paid_at, issued_at) >= now() - interval '30 days'), 0)::bigint,
           coalesce(sum(total_micros) FILTER (WHERE coalesce(paid_at, issued_at) >= date_trunc('month', now())), 0)::bigint,
           count(*),
           count(DISTINCT organization_id)
    FROM invoices
    WHERE status = 'paid';
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_revenue_monthly(p_months int)
RETURNS TABLE (month date, revenue_micros bigint, payments bigint)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT m::date,
           coalesce(sum(i.total_micros), 0)::bigint,
           count(i.id)
    FROM generate_series(date_trunc('month', now()) - make_interval(months => p_months - 1),
                         date_trunc('month', now()), interval '1 month') AS m
    LEFT JOIN invoices i
      ON i.status = 'paid'
     AND date_trunc('month', coalesce(i.paid_at, i.issued_at)) = m
    GROUP BY m
    ORDER BY m;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app.platform_payments(p_org uuid, p_limit int, p_offset int)
RETURNS TABLE (
    id                uuid,
    organization_id   uuid,
    organization_name text,
    number            text,
    total_micros      bigint,
    currency          text,
    payment_provider  text,
    payment_id        text,
    paid_at           timestamptz,
    total_count       bigint
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT i.id, i.organization_id, o.name, i.number, i.total_micros, i.currency,
           coalesce(i.payment_provider, ''), coalesce(i.payment_id, ''),
           coalesce(i.paid_at, i.issued_at), count(*) OVER ()
    FROM invoices i
    JOIN organizations o ON o.id = i.organization_id
    WHERE i.status = 'paid' AND (p_org IS NULL OR i.organization_id = p_org)
    ORDER BY coalesce(i.paid_at, i.issued_at) DESC
    LIMIT p_limit OFFSET p_offset;
$$;
-- +goose StatementEnd

-- Every organization that has paid at least once.
-- +goose StatementBegin
CREATE FUNCTION app.platform_paying_customers(p_limit int)
RETURNS TABLE (
    organization_id   uuid,
    organization_name text,
    plan_name         text,
    lifetime_micros   bigint,
    payments          bigint,
    last_paid_at      timestamptz
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
    SELECT o.id, o.name,
           coalesce((SELECT p.name FROM subscriptions s JOIN plans p ON p.id = s.plan_id
                     WHERE s.organization_id = o.id ORDER BY s.created_at DESC LIMIT 1), ''),
           sum(i.total_micros)::bigint, count(*), max(coalesce(i.paid_at, i.issued_at))
    FROM invoices i
    JOIN organizations o ON o.id = i.organization_id
    WHERE i.status = 'paid'
    GROUP BY o.id, o.name
    ORDER BY 4 DESC
    LIMIT p_limit;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.platform_paying_customers(int);
DROP FUNCTION IF EXISTS app.platform_payments(uuid, int, int);
DROP FUNCTION IF EXISTS app.platform_revenue_monthly(int);
DROP FUNCTION IF EXISTS app.platform_revenue_summary();
DROP FUNCTION IF EXISTS app.platform_security_events(text, int, int);
DROP FUNCTION IF EXISTS app.platform_tenant_audit(uuid, text, int, int);
DROP TABLE IF EXISTS platform_alerts;
DROP TABLE IF EXISTS service_logs;
DROP TABLE IF EXISTS service_samples;
