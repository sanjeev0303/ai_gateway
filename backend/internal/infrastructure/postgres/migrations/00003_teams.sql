-- +goose Up
-- Teams group users within an organization. organization_id is denormalized
-- onto team_members so the row-level security policy can be expressed
-- directly on the join table.
CREATE TABLE teams (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX teams_org_created_at_idx ON teams (organization_id, created_at);

CREATE TABLE team_members (
    team_id         uuid        NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            text        NOT NULL DEFAULT 'member'
                                CHECK (role IN ('admin', 'member')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);
CREATE INDEX team_members_user_id_idx ON team_members (user_id);
CREATE INDEX team_members_org_created_at_idx ON team_members (organization_id, created_at);

-- +goose Down
DROP TABLE team_members;
DROP TABLE teams;