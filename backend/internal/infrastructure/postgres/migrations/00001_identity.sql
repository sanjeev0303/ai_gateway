-- +goose Up
-- Organizations are the tenant root. They are a global table (no RLS).
CREATE TABLE organizations (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    slug       text        NOT NULL UNIQUE,
    status     text        NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'suspended')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Users are global identities that may belong to many organizations.
CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL,
    display_name  text        NOT NULL DEFAULT '',
    status        text        NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'disabled')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

-- Membership scopes a user to an organization with a role.
CREATE TABLE organization_members (
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            text        NOT NULL DEFAULT 'member'
                                CHECK (role IN ('admin', 'member')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id)
);
CREATE INDEX organization_members_user_id_idx ON organization_members (user_id);
CREATE INDEX organization_members_created_at_idx ON organization_members (organization_id, created_at);

-- +goose Down
DROP TABLE organization_members;
DROP TABLE users;
DROP TABLE organizations;