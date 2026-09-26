-- +goose Up
-- Policies group governance rules (rate limits, budgets, blocklists) that are
-- evaluated in the gateway hot path.
CREATE TABLE policies (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            text        NOT NULL,
    description     text        NOT NULL DEFAULT '',
    enforcement     text        NOT NULL DEFAULT 'active'
                                CHECK (enforcement IN ('permissive', 'active', 'disabled')),
    priority        int         NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, name)
);
CREATE INDEX policies_org_created_at_idx ON policies (organization_id, created_at);

CREATE TABLE policy_rules (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id       uuid        NOT NULL REFERENCES policies (id) ON DELETE CASCADE,
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    rule_type       text        NOT NULL
                                CHECK (rule_type IN ('rate_limit', 'token_budget', 'blocklist', 'allowlist', 'content_filter', 'budget')),
    description     text        NOT NULL DEFAULT '',
    conditions      jsonb       NOT NULL DEFAULT '{}'::jsonb,
    parameters      jsonb       NOT NULL DEFAULT '{}'::jsonb,
    enabled         boolean     NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX policy_rules_policy_id_idx ON policy_rules (policy_id);
CREATE INDEX policy_rules_org_created_at_idx ON policy_rules (organization_id, created_at);

-- +goose Down
DROP TABLE policy_rules;
DROP TABLE policies;