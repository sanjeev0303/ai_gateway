-- +goose Up
-- Budgets are spend/token/request caps enforced on the gateway hot path.
-- They are distinct from policy budget rules: a budget is a user-facing
-- configuration that compiles down into one or more policy budget rules.
CREATE TABLE budgets (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    scope           text        NOT NULL
                        CHECK (scope IN ('organization', 'team', 'project', 'gateway_credential', 'service_account')),
    scope_id        uuid        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    limit_type      text        NOT NULL
                        CHECK (limit_type IN ('spend', 'tokens', 'requests')),
    limit_amount    numeric     NOT NULL,
    "window"        text        NOT NULL
                        CHECK ("window" IN ('hourly', 'daily', 'weekly', 'monthly')),
    alert_threshold numeric     NOT NULL DEFAULT 0.8,
    enabled         boolean     NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX budgets_org_scope_scope_id_idx ON budgets (organization_id, scope, scope_id);
CREATE INDEX budgets_org_created_at_idx ON budgets (organization_id, created_at);

-- +goose Down
DROP TABLE budgets;