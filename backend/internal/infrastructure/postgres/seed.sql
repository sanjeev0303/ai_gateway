-- Demo seed data for local development, applied by cmd/seed (postgres.Seed).
--
-- The file is idempotent: re-running cmd/seed on an already-seeded database
-- is a no-op. Fixed UUIDs make the tenant binding deterministic, and the
-- WHERE NOT EXISTS / ON CONFLICT guards keep every insert safe to replay.
--
-- Seeded tenant root: 11111111-1111-4111-8111-111111111111 (Acme)
-- Synthetic seed records are explicitly distinguished with '[Synthetic Seed]' prefixes/metadata.
BEGIN;

-- 1. Organizations
INSERT INTO organizations (id, name, slug, status, privacy_default, retention_conversations_days, retention_usage_days, retention_audits_days, retention_billing_days)
VALUES ('11111111-1111-4111-8111-111111111111', 'Acme', 'acme', 'active', 'METADATA_ONLY', 30, 90, 365, 2555)
ON CONFLICT (id) DO UPDATE SET
    privacy_default = EXCLUDED.privacy_default,
    retention_conversations_days = EXCLUDED.retention_conversations_days,
    retention_usage_days = EXCLUDED.retention_usage_days,
    retention_audits_days = EXCLUDED.retention_audits_days,
    retention_billing_days = EXCLUDED.retention_billing_days;

-- Bind the tenant context for RLS-protected inserts below.
SELECT set_config('app.org_id', '11111111-1111-4111-8111-111111111111', true);

-- 2. Users & Memberships
UPDATE users SET email = 'previous_admin@acme.example' WHERE lower(email) = 'admin@acme.example' AND id != '22222222-2222-4222-8222-222222222222';
UPDATE users SET email = 'previous_alice@acme.example' WHERE lower(email) = 'alice@acme.example' AND id != '22222222-2222-4222-8222-333333333333';
UPDATE users SET email = 'previous_bob@acme.example' WHERE lower(email) = 'bob@acme.example' AND id != '22222222-2222-4222-8222-444444444444';

INSERT INTO users (id, email, display_name, status, password_hash)
VALUES
    ('22222222-2222-4222-8222-222222222222', 'admin@acme.example', '[Synthetic Seed] Acme Admin', 'active', '$pbkdf2-sha256$600000$MTIzNDU2Nzg5MDEyMzQ1Ng$/d//SBWUawB2F9x5Je9ya9p1Fyb/rA3zccmZusD1LWk'),
    ('22222222-2222-4222-8222-333333333333', 'alice@acme.example', '[Synthetic Seed] Alice Engineer', 'active', '$pbkdf2-sha256$600000$MTIzNDU2Nzg5MDEyMzQ1Ng$/d//SBWUawB2F9x5Je9ya9p1Fyb/rA3zccmZusD1LWk'),
    ('22222222-2222-4222-8222-444444444444', 'bob@acme.example', '[Synthetic Seed] Bob Researcher', 'active', '$pbkdf2-sha256$600000$MTIzNDU2Nzg5MDEyMzQ1Ng$/d//SBWUawB2F9x5Je9ya9p1Fyb/rA3zccmZusD1LWk')
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, status = EXCLUDED.status, password_hash = EXCLUDED.password_hash;

INSERT INTO organization_members (organization_id, user_id, role)
VALUES
    ('11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'admin'),
    ('11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-333333333333', 'member'),
    ('11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-444444444444', 'member')
ON CONFLICT (organization_id, user_id) DO NOTHING;

-- 3. Projects
INSERT INTO projects (id, organization_id, name, slug, description, status)
VALUES
    ('33333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Default Project', 'default', 'Demo project for local development', 'active'),
    ('33333333-3333-4333-8333-444444444444', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Research Project', 'research', 'AI research and prototyping', 'active')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, status = EXCLUDED.status;

INSERT INTO project_members (project_id, organization_id, user_id, role)
VALUES
    ('33333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'admin'),
    ('33333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-333333333333', 'member'),
    ('33333333-3333-4333-8333-444444444444', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-444444444444', 'member')
ON CONFLICT (project_id, user_id) DO NOTHING;

-- 4. Teams
INSERT INTO teams (id, organization_id, name, description)
VALUES
    ('44444444-4444-4444-8444-111111111111', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Core Engineering', 'Platform and backend team'),
    ('44444444-4444-4444-8444-222222222222', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Applied Research', 'AI model evaluation and applied ML research')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description;

INSERT INTO team_members (team_id, organization_id, user_id, role)
VALUES
    ('44444444-4444-4444-8444-111111111111', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'admin'),
    ('44444444-4444-4444-8444-111111111111', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-333333333333', 'member'),
    ('44444444-4444-4444-8444-222222222222', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-444444444444', 'member')
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
    ('77777777-7777-4777-8777-111111111111', '11111111-1111-4111-8111-111111111111', 'anthropic', '[Synthetic Seed] Primary Anthropic', 'https://api.anthropic.com', 'active'),
    ('77777777-7777-4777-8777-333333333333', '11111111-1111-4111-8111-111111111111', 'openai', '[Synthetic Seed] Primary OpenAI', 'https://api.openai.com', 'active')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, provider_kind = EXCLUDED.provider_kind, base_url = EXCLUDED.base_url, status = EXCLUDED.status;

INSERT INTO provider_credentials (id, organization_id, provider_account_id, name, kind, secret_ref, status)
VALUES
    ('77777777-7777-4777-8777-222222222222', '11111111-1111-4111-8111-111111111111', '77777777-7777-4777-8777-111111111111', '[Synthetic Seed] Default Anthropic Key', 'api_key', 'ANTHROPIC_API_KEY', 'active'),
    ('77777777-7777-4777-8777-444444444444', '11111111-1111-4111-8111-111111111111', '77777777-7777-4777-8777-333333333333', '[Synthetic Seed] Default OpenAI Key', 'api_key', 'OPENAI_API_KEY', 'active')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, kind = EXCLUDED.kind, secret_ref = EXCLUDED.secret_ref, status = EXCLUDED.status;

-- 8. Gateway Credentials
INSERT INTO gateway_credentials (id, organization_id, name, key_prefix, key_hash, subject_type, subject_id, team_id, project_id, scopes, status)
VALUES
    ('aaaa2222-2222-4222-8222-111111111111', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Admin Gateway Key', 'aigw_seed_ad', 'sha256:36fbd759fb5468e4a1a49570a8755b5a85fdf50b08a7d2fc71ee0d7ae0801469', 'user', '22222222-2222-4222-8222-222222222222', '44444444-4444-4444-8444-111111111111', '33333333-3333-4333-8333-333333333333', ARRAY['gateway'::gateway_credential_scope, 'control'::gateway_credential_scope, 'all'::gateway_credential_scope], 'active'),
    ('aaaa2222-2222-4222-8222-222222222222', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Alice Gateway Key', 'aigw_seed_al', 'sha256:e374afbd97d2270b1317f61cdaf079c977c0bfb0a0500a42ec11ce6285217dbd', 'user', '22222222-2222-4222-8222-333333333333', '44444444-4444-4444-8444-111111111111', '33333333-3333-4333-8333-333333333333', ARRAY['gateway'::gateway_credential_scope], 'active'),
    ('aaaa2222-2222-4222-8222-333333333333', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Bob Gateway Key', 'aigw_seed_bo', 'sha256:13bf255e847d094ba0c814e7eeda88de4f0481ba6f61f9b89e72f5e1eba688e0', 'user', '22222222-2222-4222-8222-444444444444', '44444444-4444-4444-8444-222222222222', '33333333-3333-4333-8333-444444444444', ARRAY['gateway'::gateway_credential_scope], 'active')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, key_prefix = EXCLUDED.key_prefix, key_hash = EXCLUDED.key_hash, scopes = EXCLUDED.scopes, status = EXCLUDED.status;

-- 9. Policies and Rules
INSERT INTO policies (id, organization_id, name, description, enforcement, priority)
VALUES
    ('55555555-5555-4555-8555-111111111111', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Default Engineering Policy', 'Production AI governance policies for engineering', 'active', 1)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, enforcement = EXCLUDED.enforcement, priority = EXCLUDED.priority;

INSERT INTO policy_rules (id, policy_id, organization_id, rule_type, description, conditions, parameters, enabled)
VALUES
    ('55555555-5555-4555-8555-222222222222', '55555555-5555-4555-8555-111111111111', '11111111-1111-4111-8111-111111111111', 'allowlist', 'Allow vetted production models', '{}'::jsonb, '{"models": ["claude-3-5-sonnet-20241022", "claude-3-haiku-20240307", "gpt-4o", "gpt-4o-mini"]}'::jsonb, true),
    ('55555555-5555-4555-8555-333333333333', '55555555-5555-4555-8555-111111111111', '11111111-1111-4111-8111-111111111111', 'rate_limit', 'Global rate limit per client', '{}'::jsonb, '{"limit": 120, "window_seconds": 60}'::jsonb, true)
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, conditions = EXCLUDED.conditions, parameters = EXCLUDED.parameters, enabled = EXCLUDED.enabled;

-- 10. Budgets
INSERT INTO budgets (id, organization_id, name, description, scope, scope_id, limit_type, limit_amount, "window", alert_threshold, enabled)
VALUES
    ('66666666-6666-4666-8666-111111111111', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Acme Monthly Budget', 'Global organization monthly spend limit', 'organization', '11111111-1111-4111-8111-111111111111', 'spend', 1000.00, 'monthly', 0.85, true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, limit_amount = EXCLUDED.limit_amount, alert_threshold = EXCLUDED.alert_threshold, enabled = EXCLUDED.enabled;

-- 11. Guardrails
INSERT INTO guardrails (id, organization_id, name, description, target_type, detector, action, configuration, enabled)
VALUES
    ('bbbb1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', '[Synthetic Seed] Global PII Detector', 'Detect and alert on customer PII in model prompts', 'global', 'pii', 'warn', '{"entities": ["email", "phone", "ssn"]}'::jsonb, true)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, detector = EXCLUDED.detector, action = EXCLUDED.action, configuration = EXCLUDED.configuration, enabled = EXCLUDED.enabled;

-- 12. Alerts & Alert Events
INSERT INTO alerts (id, organization_id, alert_type, name, severity, configuration, status)
VALUES
    ('aaaa1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'spend_limit', '[Synthetic Seed] Spend Spike Watchdog', 'warning', '{"threshold_usd": 500, "window_hours": 24}'::jsonb, 'active'),
    ('aaaa1111-1111-4111-8111-222222222222', '11111111-1111-4111-8111-111111111111', 'error_rate', '[Synthetic Seed] Upstream Error Watchdog', 'critical', '{"error_rate_threshold": 0.05, "window_minutes": 15}'::jsonb, 'active')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, severity = EXCLUDED.severity, configuration = EXCLUDED.configuration, status = EXCLUDED.status;

INSERT INTO alert_events (id, organization_id, alert_id, alert_type, severity, status, message, context, triggered_at)
VALUES
    ('ccaa1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'aaaa1111-1111-4111-8111-111111111111', 'spend_limit', 'warning', 'triggered', '[Synthetic Seed] Organization spend exceeded 85% of monthly budget', '{"current_spend_usd": 855.20, "budget_usd": 1000.00, "synthetic": true}'::jsonb, now() - interval '2 hours')
ON CONFLICT (id) DO NOTHING;

INSERT INTO in_app_notifications (id, organization_id, alert_id, alert_event_id, title, message, severity, read, created_at)
VALUES
    ('ddaa1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'aaaa1111-1111-4111-8111-111111111111', 'ccaa1111-1111-4111-8111-111111111111', '[Synthetic Seed] Spend Budget Warning', 'Organization spend has reached $855.20 (85.5% of $1000.00 limit)', 'warning', false, now() - interval '2 hours')
ON CONFLICT (id) DO NOTHING;

-- 13. Audit Logs & Security Events
INSERT INTO audit_logs (id, organization_id, actor_user_id, action, resource_type, resource_id, before, after, created_at)
VALUES
    ('eeaa1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'organization.initialize', 'organization', '11111111-1111-4111-8111-111111111111', null, '{"name": "Acme", "slug": "acme", "synthetic": true}'::jsonb, now() - interval '30 days'),
    ('eeaa1111-1111-4111-8111-222222222222', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'member.invite', 'user', '22222222-2222-4222-8222-333333333333', null, '{"email": "alice@acme.example", "role": "member", "synthetic": true}'::jsonb, now() - interval '20 days'),
    ('eeaa1111-1111-4111-8111-333333333333', '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'policy.create', 'policy', '55555555-5555-4555-8555-111111111111', null, '{"name": "Default Engineering Policy", "synthetic": true}'::jsonb, now() - interval '10 days')
ON CONFLICT (id) DO NOTHING;

INSERT INTO security_events (id, organization_id, category, severity, title, detail, source, created_at)
VALUES
    ('ffaa1111-1111-4111-8111-111111111111', '11111111-1111-4111-8111-111111111111', 'rate_limit', 'warning', '[Synthetic Seed] Gateway Rate Limit Throttled', '{"client_ip": "127.0.0.1", "model": "gpt-4o", "synthetic": true}'::jsonb, 'gateway-v2', now() - interval '1 day')
ON CONFLICT (id) DO NOTHING;

-- 14. Seeded LLM Requests and Usage Events
INSERT INTO llm_requests (id, organization_id, project_id, team_id, user_id, gateway_credential_id, request_id, provider_request_id, provider_kind, model, status, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, latency_ms, cost_micros, created_at)
VALUES
    ('10000000-0000-4000-8000-000000000001', '11111111-1111-4111-8111-111111111111', '33333333-3333-4333-8333-333333333333', '44444444-4444-4444-8444-111111111111', '22222222-2222-4222-8222-333333333333', 'aaaa2222-2222-4222-8222-222222222222', 'req_synth_seed_001', 'synth_anthropic_msg_001', 'anthropic', 'claude-3-5-sonnet-20241022', 'succeeded', 840, 260, 100, 1200, 412, 5400, now() - interval '3 hours'),
    ('10000000-0000-4000-8000-000000000002', '11111111-1111-4111-8111-111111111111', '33333333-3333-4333-8333-444444444444', '44444444-4444-4444-8444-222222222222', '22222222-2222-4222-8222-444444444444', 'aaaa2222-2222-4222-8222-333333333333', 'req_synth_seed_002', 'synth_openai_chat_002', 'openai', 'gpt-4o', 'succeeded', 520, 180, 0, 700, 280, 3100, now() - interval '1 hour')
ON CONFLICT (organization_id, request_id) DO NOTHING;

INSERT INTO usage_events (id, organization_id, request_id, event_type, quantity, unit, metadata, user_id, team_id, project_id, gateway_credential_id, created_at)
VALUES
    ('20000000-0000-4000-8000-000000000001', '11111111-1111-4111-8111-111111111111', 'req_synth_seed_001', 'token_usage', 1200, 'tokens', '{"synthetic": true, "model": "claude-3-5-sonnet-20241022"}'::jsonb, '22222222-2222-4222-8222-333333333333', '44444444-4444-4444-8444-111111111111', '33333333-3333-4333-8333-333333333333', 'aaaa2222-2222-4222-8222-222222222222', now() - interval '3 hours'),
    ('20000000-0000-4000-8000-000000000002', '11111111-1111-4111-8111-111111111111', 'req_synth_seed_001', 'cost', 5400, 'micros', '{"synthetic": true, "currency": "usd"}'::jsonb, '22222222-2222-4222-8222-333333333333', '44444444-4444-4444-8444-111111111111', '33333333-3333-4333-8333-333333333333', 'aaaa2222-2222-4222-8222-222222222222', now() - interval '3 hours'),
    ('20000000-0000-4000-8000-000000000003', '11111111-1111-4111-8111-111111111111', 'req_synth_seed_002', 'token_usage', 700, 'tokens', '{"synthetic": true, "model": "gpt-4o"}'::jsonb, '22222222-2222-4222-8222-444444444444', '44444444-4444-4444-8444-222222222222', '33333333-3333-4333-8333-444444444444', 'aaaa2222-2222-4222-8222-333333333333', now() - interval '1 hour'),
    ('20000000-0000-4000-8000-000000000004', '11111111-1111-4111-8111-111111111111', 'req_synth_seed_002', 'cost', 3100, 'micros', '{"synthetic": true, "currency": "usd"}'::jsonb, '22222222-2222-4222-8222-444444444444', '44444444-4444-4444-8444-222222222222', '33333333-3333-4333-8333-444444444444', 'aaaa2222-2222-4222-8222-333333333333', now() - interval '1 hour')
ON CONFLICT (organization_id, request_id, event_type) DO NOTHING;

-- 15. Seeded Usage Analytics Rollups (Last 7 Days)
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

-- 16. Pricing Models & Versioned Rate Sheets
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