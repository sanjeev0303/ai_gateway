-- +goose Up
-- M3: identity & organization management.
--
-- 1. organization_members roles expand from (admin, member) to
--    (owner, admin, member, auditor, billing_admin). 'admin' stays valid, so
--    the existing seed rows remain consistent.
-- 2. invitations: token-gated, RLS-tenant-scoped onboarding records.
-- 3. project_members: RLS-tenant-scoped project-level memberships.
-- 4. api_credentials: non-gateway (control/all) keys MUST belong to a service
--    account, so machine identities can be revoked independently of the key.

ALTER TABLE organization_members DROP CONSTRAINT organization_members_role_check;
ALTER TABLE organization_members
    ADD CONSTRAINT organization_members_role_check
    CHECK (role IN ('owner', 'admin', 'member', 'auditor', 'billing_admin'));

CREATE TABLE invitations (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    email           text        NOT NULL,
    role            text        NOT NULL DEFAULT 'member'
                                CHECK (role IN ('owner', 'admin', 'member', 'auditor', 'billing_admin')),
    token_hash      text        NOT NULL UNIQUE,
    status          text        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    expires_at      timestamptz NOT NULL,
    invited_by      uuid        REFERENCES users (id) ON DELETE SET NULL,
    accepted_by     uuid        REFERENCES users (id) ON DELETE SET NULL,
    accepted_at     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- At most one pending invitation per (organization, email).
CREATE UNIQUE INDEX invitations_org_email_pending_idx
    ON invitations (organization_id, lower(email)) WHERE status = 'pending';

CREATE INDEX invitations_org_created_at_idx
    ON invitations (organization_id, created_at);

-- Invitations are mutable (status transitions from pending to
-- accepted/revoked), so they carry the shared updated_at trigger.
CREATE TRIGGER invitations_set_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE project_members (
    project_id      uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            text        NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX project_members_org_idx ON project_members (organization_id);
CREATE INDEX project_members_user_idx ON project_members (user_id);

-- Service-account binding for control/all-scope API credentials.
ALTER TABLE api_credentials
    ADD COLUMN service_account_id uuid REFERENCES service_accounts (id) ON DELETE CASCADE;

ALTER TABLE api_credentials
    ADD CONSTRAINT api_credentials_service_account_scope_check
    CHECK (scope = 'gateway' OR service_account_id IS NOT NULL);

CREATE INDEX api_credentials_service_account_idx ON api_credentials (service_account_id);

-- Row-level security: the two new tenant-scoped tables get the same
-- org-isolation treatment as every other FORCE RLS table. The DO block only
-- registers NEW tables; re-creating policies for existing tables would fail
-- with duplicate-policy errors.
-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['invitations', 'project_members'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (organization_id = app.current_org_id()) WITH CHECK (organization_id = app.current_org_id())',
            t || '_org_isolation', t);
    END LOOP;
END
$$;
-- +goose StatementEnd

-- Invitation rows are hidden by FORCE RLS, so the accept flow (which only
-- knows the token) cannot read them as the anonymous invitation holder:
-- resolve across tenants via a SECURITY DEFINER function, mirroring the
-- API-credential resolver pattern.
-- +goose StatementBegin
CREATE FUNCTION app.resolve_invitation_by_token(p_token_hash text)
RETURNS SETOF invitations
LANGUAGE sql
STABLE
SECURITY DEFINER
AS $$
    SELECT *
    FROM invitations
    WHERE token_hash = p_token_hash
    LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION app.resolve_invitation_by_token(text);

DROP TRIGGER IF EXISTS invitations_set_updated_at ON invitations;
DROP TABLE project_members;
DROP TABLE invitations;

DROP INDEX IF EXISTS api_credentials_service_account_idx;
ALTER TABLE api_credentials DROP CONSTRAINT api_credentials_service_account_scope_check;
ALTER TABLE api_credentials DROP COLUMN service_account_id;

ALTER TABLE organization_members DROP CONSTRAINT organization_members_role_check;
ALTER TABLE organization_members
    ADD CONSTRAINT organization_members_role_check
    CHECK (role IN ('admin', 'member'));