-- +goose Up
-- Plans are the global billing catalog (no organization_id; no RLS).
CREATE TABLE plans (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    code        text        NOT NULL UNIQUE,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    status      text        NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'inactive')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Subscriptions attach an organization to a plan. A partial unique index
-- guarantees at most one active subscription per organization.
CREATE TABLE subscriptions (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id       uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    plan_id               uuid        NOT NULL REFERENCES plans (id),
    status                text        NOT NULL DEFAULT 'active'
                                      CHECK (status IN ('active', 'trialing', 'canceled', 'past_due', 'incomplete')),
    current_period_start  timestamptz,
    current_period_end    timestamptz,
    cancel_at_period_end  boolean     NOT NULL DEFAULT false,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX subscriptions_one_active_per_org
    ON subscriptions (organization_id)
    WHERE status IN ('active', 'trialing');
CREATE INDEX subscriptions_org_created_at_idx ON subscriptions (organization_id, created_at);

-- Billing events are provider/webhook records. The unique
-- (organization, billing_event_id) key makes ingest idempotent.
CREATE TABLE billing_events (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    billing_event_id text       NOT NULL,
    event_type      text        NOT NULL DEFAULT 'charge',
    amount_micros   bigint,
    currency        text        NOT NULL DEFAULT 'usd',
    status          text        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'succeeded', 'failed')),
    payload         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, billing_event_id)
);
CREATE INDEX billing_events_org_created_at_idx ON billing_events (organization_id, created_at);

-- +goose Down
DROP TABLE billing_events;
DROP TABLE subscriptions;
DROP TABLE plans;