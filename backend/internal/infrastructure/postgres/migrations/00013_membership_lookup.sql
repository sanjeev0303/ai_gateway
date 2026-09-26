-- +goose Up
-- List every organization a user belongs to, across tenants.
--
-- organization_members is FORCE RLS scoped to app.current_org_id(), so a
-- tenant-context session cannot see another tenant's membership rows. The
-- control plane (login, org switcher) needs this cross-tenant read, so the
-- lookup runs as the migration owner (superuser) via SECURITY DEFINER, the
-- same pattern as app.resolve_api_credential. Callers only receive rows they
-- already own (membership of the given user), and org-scoped membership
-- access still goes through RLS.
-- The view exists solely as a named return type: sqlc cannot infer columns
-- from an anonymous RETURNS TABLE, but it resolves RETURNS SETOF <relation>
-- (view or table), exactly as it resolves the api_credentials rowtype.
CREATE VIEW app.user_organizations AS
SELECT o.id, o.name, o.slug, o.status, o.created_at, o.updated_at, m.role
FROM organization_members m
JOIN organizations o ON o.id = m.organization_id;

-- +goose StatementBegin
CREATE FUNCTION app.organizations_for_user(p_user_id uuid)
RETURNS SETOF app.user_organizations
LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT uo.*
    FROM app.user_organizations uo
    JOIN organization_members m ON m.organization_id = uo.id
    WHERE m.user_id = p_user_id
    ORDER BY uo.created_at;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.organizations_for_user(uuid);
DROP VIEW IF EXISTS app.user_organizations;