-- +goose Up
-- +goose StatementBegin

CREATE TABLE guardrails (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    target_type TEXT NOT NULL,
    target_id uuid,
    detector TEXT NOT NULL, -- 'pii', 'secrets', 'unsafe_keywords', 'request_constraints'
    action TEXT NOT NULL, -- 'allow', 'warn', 'block', 'redact'
    configuration JSONB NOT NULL DEFAULT '{}'::jsonb, -- e.g. { "keywords": ["..."], "max_tokens": 1000 }
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for quick lookups during gateway evaluation
CREATE INDEX idx_guardrails_evaluation ON guardrails(organization_id, target_type, target_id, enabled);

CREATE TABLE guardrail_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    guardrail_id uuid NOT NULL REFERENCES guardrails(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    direction TEXT NOT NULL,
    detector TEXT NOT NULL,
    action TEXT NOT NULL,
    matched_category TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_guardrail_events_org ON guardrail_events(organization_id, timestamp DESC);
CREATE INDEX idx_guardrail_events_guardrail ON guardrail_events(guardrail_id, timestamp DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE guardrail_events;
DROP TABLE guardrails;
-- +goose StatementEnd
