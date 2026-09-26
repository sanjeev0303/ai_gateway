-- +goose Up
-- Projects scope gateway usage inside an organization.
CREATE TABLE projects (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            text        NOT NULL,
    slug            text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'archived')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, slug)
);
CREATE INDEX projects_org_created_at_idx ON projects (organization_id, created_at);

-- +goose Down
DROP TABLE projects;