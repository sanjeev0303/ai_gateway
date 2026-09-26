-- Demo seed data for local development, applied by cmd/seed (postgres.Seed).
--
-- The file is idempotent: re-running cmd/seed on an already-seeded database
-- is a no-op. Fixed UUIDs make the tenant binding (organization_members and
-- projects are FORCE RLS) deterministic, and the WHERE NOT EXISTS guards keep
-- every insert safe to replay.
--
-- Seeded tenant root: 11111111-1111-4111-8111-111111111111 (Acme)
BEGIN;

INSERT INTO organizations (id, name, slug, status)
SELECT '11111111-1111-4111-8111-111111111111', 'Acme', 'acme', 'active'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'acme');

-- Bind the tenant context for the RLS-protected inserts below. set_config
-- with is_local=true scopes the value to this transaction (SET LOCAL
-- semantics), exactly like Store.WithTenantTx does for application code.
SELECT set_config('app.org_id', '11111111-1111-4111-8111-111111111111', true);

INSERT INTO users (id, email, display_name, status)
SELECT '22222222-2222-4222-8222-222222222222', 'admin@acme.example', 'Acme Admin', 'active'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE lower(email) = lower('admin@acme.example'));

INSERT INTO organization_members (organization_id, user_id, role)
SELECT '11111111-1111-4111-8111-111111111111', '22222222-2222-4222-8222-222222222222', 'admin'
WHERE NOT EXISTS (
    SELECT 1 FROM organization_members
    WHERE organization_id = '11111111-1111-4111-8111-111111111111'
      AND user_id = '22222222-2222-4222-8222-222222222222'
);

INSERT INTO projects (id, organization_id, name, slug, description, status)
SELECT '33333333-3333-4333-8333-333333333333', '11111111-1111-4111-8111-111111111111',
       'Default', 'default', 'Demo project for local development', 'active'
WHERE NOT EXISTS (
    SELECT 1 FROM projects
    WHERE organization_id = '11111111-1111-4111-8111-111111111111'
      AND slug = 'default'
);

COMMIT;