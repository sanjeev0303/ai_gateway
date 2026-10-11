-- Demo seed data for local development, applied by cmd/seed (postgres.Seed).
--
-- The file is idempotent: re-running cmd/seed on an already-seeded database
-- is a no-op. Fixed UUIDs make the tenant binding deterministic, and the
-- WHERE NOT EXISTS / ON CONFLICT guards keep every insert safe to replay.
--
-- Seeded tenant root: 11111111-1111-4111-8111-111111111111 (Acme)
BEGIN;

-- 1. Organizations
INSERT INTO organizations (id, name, slug, status)
VALUES ('11111111-1111-4111-8111-111111111111', 'Acme', 'acme', 'active')
ON CONFLICT (id) DO NOTHING;

-- Bind the tenant context for RLS-protected inserts below.
SELECT set_config('app.org_id', '11111111-1111-4111-8111-111111111111', true);

-- 2. Users & Memberships
UPDATE users SET email = 'previous_admin@acme.example' WHERE lower(email) = 'admin@acme.example' AND id != '22222222-2222-4222-8222-222222222222';

INSERT INTO users (id, email, display_name, status)
VALUES ('22222222-2222-4222-8222-222222222222', 'admin@acme.example', 'Acme Admin', 'active')
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, status = EXCLUDED.status;

INSERT INTO organization_members (organization_id, user_id, role)
VALUES ('11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'admin')
ON CONFLICT (organization_id, user_id) DO NOTHING;

-- 3. Projects
INSERT INTO projects (id, organization_id, name, slug, description, status)
VALUES
    ('33333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', 'Default', 'default', 'Demo project for local development', 'active'),
    ('33333333-3333-4333-8333-444444444444', '11111111-1111-4111-8111-111111111111', 'Research', 'research', 'AI research and prototyping', 'active')
ON CONFLICT (organization_id, slug) DO NOTHING;

-- 4. Teams
INSERT INTO teams (id, organization_id, name, description)
VALUES ('44444444-4444-4444-8444-111111111111', '11111111-1111-4111-8111-111111111111', 'Core Engineering', 'Platform and backend team')
ON CONFLICT (organization_id, name) DO NOTHING;

INSERT INTO team_members (team_id, organization_id, user_id, role)
VALUES ('44444444-4444-4444-8444-111111111111', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'admin')
ON CONFLICT (team_id, user_id) DO NOTHING;

-- 5. Plans and Subscriptions
INSERT INTO plans (id, code, name, description, status)
VALUES
    ('99999999-9999-4999-8999-000000000001', 'free', 'Free Plan', 'Starter tier with 10k requests/mo', 'active'),
    ('99999999-9999-4999-8999-000000000002', 'pro', 'Pro Plan', 'Production tier with 1M requests/mo', 'active'),
    ('99999999-9999-4999-8999-000000000003', 'enterprise', 'Enterprise', 'Custom high volume with dedicated throughput', 'active')
ON CONFLICT (code) DO NOTHING;

INSERT INTO subscriptions (id, organization_id, plan_id, status, current_period_start, current_period_end)
SELECT '99999999-9999-4999-8999-111111111111',
       '11111111-1111-4111-8111-111111111111',
       '99999999-9999-4999-8999-000000000002',
       'active',
       now() - interval '15 days',
       now() + interval '15 days'
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions
    WHERE organization_id = '11111111-1111-4111-8111-111111111111'
);

-- 6. Global Model Catalog
INSERT INTO models (id, provider_kind, model_slug, display_name, is_active)
VALUES
    ('88888888-8888-4888-8888-000000000001', 'anthropic', 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet', true),
    ('88888888-8888-4888-8888-000000000002', 'anthropic', 'claude-3-haiku-20240307', 'Claude 3 Haiku', true),
    ('88888888-8888-4888-8888-000000000003', 'openai', 'gpt-4o', 'GPT-4o', true),
    ('88888888-8888-4888-8888-000000000004', 'openai', 'gpt-4o-mini', 'GPT-4o Mini', true)
ON CONFLICT (provider_kind, model_slug) DO NOTHING;

-- 7. Provider Accounts & Credentials
INSERT INTO provider_accounts (id, organization_id, provider_kind, name, base_url, status)
VALUES
    ('77777777-7777-4777-8777-111111111111', '11111111-1111-4111-8111-111111111111', 'anthropic', 'Primary Anthropic', 'https://api.anthropic.com', 'active')
ON CONFLICT (organization_id, name) DO NOTHING;

INSERT INTO provider_credentials (id, organization_id, provider_account_id, name, kind, secret_ref, status)
VALUES
    ('77777777-7777-4777-8777-222222222222', '11111111-1111-4111-8111-111111111111', '77777777-7777-4777-8777-111111111111', 'Default Anthropic Key', 'api_key', 'ANTHROPIC_API_KEY', 'active')
ON CONFLICT (organization_id, name) DO NOTHING;

-- 8. Policies and Rules
INSERT INTO policies (id, organization_id, name, description, enforcement, priority)
VALUES
    ('55555555-5555-4555-8555-111111111111', '11111111-1111-4111-8111-111111111111', 'Default Engineering Policy', 'Production AI governance policies for engineering', 'active', 1)
ON CONFLICT (organization_id, name) DO NOTHING;

INSERT INTO policy_rules (id, policy_id, organization_id, rule_type, description, conditions, parameters, enabled)
VALUES
    ('55555555-5555-4555-8555-222222222222', '55555555-5555-4555-8555-111111111111', '11111111-1111-4111-8111-111111111111', 'allowlist', 'Allow vetted production models', '{}'::jsonb, '{"models": ["claude-3-5-sonnet-20241022", "claude-3-haiku-20240307", "gpt-4o", "gpt-4o-mini"]}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

-- 9. Budgets
INSERT INTO budgets (id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled)
VALUES
    ('66666666-6666-4666-8666-111111111111', '11111111-1111-4111-8111-111111111111', 'Acme Monthly Budget', 'Global organization monthly spend limit', 'organization', '11111111-1111-4111-8111-111111111111', 'spend', 1000.00, 'monthly', 0.85, true)
ON CONFLICT (organization_id, name) DO NOTHING;

-- 10. Alerts
INSERT INTO alerts (id, organization_id, alert_type, name, severity, configuration, status)
VALUES
    ('aaaa1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'spend_limit', 'Spend Spike Watchdog', 'warning', '{"threshold_usd": 500, "window_hours": 24}'::jsonb, 'active')
ON CONFLICT (id) DO NOTHING;

-- 11. Guardrails
INSERT INTO guardrails (id, organization_id, name, description, target_type, detector, action, configuration, enabled)
VALUES
    ('bbbb1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'Global PII Detector', 'Detect and alert on customer PII in model prompts', 'global', 'pii', 'warn', '{"entities": ["email", "phone", "ssn"]}'::jsonb, true)
ON CONFLICT (id) DO NOTHING;

-- 12. Seeded Usage Analytics Rollups (Last 7 Days)
INSERT INTO usage_daily (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
SELECT
    '11111111-1111-4111-8111-111111111111',
    '33333333-3333-4333-8333-333333333333',
    m.model,
    (date_trunc('day', now()) - (d.day || ' days')::interval)::timestamptz,
    (120 + d.day * 15 + m.multiplier * 30),
    (85000 + d.day * 4000 + m.multiplier * 15000),
    (24000 + d.day * 1200 + m.multiplier * 6000),
    (10000 + d.day * 500),
    (119000 + d.day * 5700 + m.multiplier * 21000),
    (350000 + d.day * 18000 + m.multiplier * 60000),
    now()
FROM
    generate_series(0, 6) AS d(day),
    (VALUES
        ('claude-3-5-sonnet-20241022', 2),
        ('claude-3-haiku-20240307', 1),
        ('gpt-4o', 3)
    ) AS m(model, multiplier)
ON CONFLICT (organization_id, project_id, model, bucket_start) DO NOTHING;

INSERT INTO usage_hourly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
SELECT
    '11111111-1111-4111-8111-111111111111',
    '33333333-3333-4333-8333-333333333333',
    'claude-3-5-sonnet-20241022',
    (date_trunc('hour', now()) - (h.hour || ' hours')::interval)::timestamptz,
    (15 + (h.hour % 7) * 4),
    (12000 + (h.hour % 7) * 1500),
    (3500 + (h.hour % 7) * 400),
    1200,
    (16700 + (h.hour % 7) * 1900),
    (48000 + (h.hour % 7) * 5500),
    now()
FROM
    generate_series(0, 23) AS h(hour)
ON CONFLICT (organization_id, project_id, model, bucket_start) DO NOTHING;

-- 13. Pricing Models & Versioned Rate Sheets
INSERT INTO pricing_models (id, organization_id, model_slug, provider_kind, display_name, currency, status)
VALUES
    ('eeee1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'claude-3-5-sonnet-20241022', 'anthropic', 'Claude 3.5 Sonnet', 'usd', 'active'),
    ('eeee1111-1111-4111-8111-222222222222', '11111111-1111-4111-8111-111111111111', 'claude-3-haiku-20240307', 'anthropic', 'Claude 3 Haiku', 'usd', 'active'),
    ('eeee1111-1111-4111-8111-333333333333', '11111111-1111-4111-8111-111111111111', 'gpt-4o', 'openai', 'GPT-4o', 'usd', 'active'),
    ('eeee1111-1111-4111-8111-444444444444', '11111111-1111-4111-8111-111111111111', 'gpt-4o-mini', 'openai', 'GPT-4o Mini', 'usd', 'active')
ON CONFLICT (organization_id, model_slug) DO NOTHING;

INSERT INTO pricing_versions (id, pricing_model_id, organization_id, version, input_unit_price_micros, output_unit_price_micros, effective_from)
VALUES
    ('ffff1111-1111-4111-8111-111111111111', 'eeee1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 1, 3, 15, now() - interval '90 days'),
    ('ffff1111-1111-4111-8111-222222222222', 'eeee1111-1111-4111-8111-222222222222', '11111111-1111-4111-8111-111111111111', 1, 1, 2, now() - interval '90 days'),
    ('ffff1111-1111-4111-8111-333333333333', 'eeee1111-1111-4111-8111-333333333333', '11111111-1111-4111-8111-111111111111', 1, 3, 10, now() - interval '90 days'),
    ('ffff1111-1111-4111-8111-444444444444', 'eeee1111-1111-4111-8111-444444444444', '11111111-1111-4111-8111-111111111111', 1, 1, 1, now() - interval '90 days')
ON CONFLICT (pricing_model_id, version) DO NOTHING;

COMMIT;