-- +goose Up
-- gateway_credentials (00018) carries organization_id but was created after
-- the 00012 RLS sweep, so it has no row-level security. Close the gap with
-- the same org-isolation policy every other tenant-scoped table carries.
--
-- Safety: the CRUD path is org-scoped and tenant-bound (WithTenantTx), and
-- the hot-path resolvers (resolve_gateway_credential[_by_prefix]) are
-- SECURITY DEFINER owned by the migration user, which bypasses RLS — so the
-- data plane is unaffected.

ALTER TABLE gateway_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE gateway_credentials FORCE ROW LEVEL SECURITY;
CREATE POLICY gateway_credentials_org_isolation ON gateway_credentials
    USING (organization_id = app.current_org_id())
    WITH CHECK (organization_id = app.current_org_id());

-- +goose Down
DROP POLICY IF EXISTS gateway_credentials_org_isolation ON gateway_credentials;
ALTER TABLE gateway_credentials NO FORCE ROW LEVEL SECURITY;
ALTER TABLE gateway_credentials DISABLE ROW LEVEL SECURITY;