-- +goose Up
-- event_outbox is the transactional outbox used by the "postgres" EventBus
-- (AIGW_EVENTBUS_KIND=postgres): the data plane inserts a row here instead of
-- publishing to an external broker, and the worker drains it. It is an internal
-- transport table (no organization_id, no RLS) — like the billing catalog
-- tables — because only the worker (a system actor) reads it, and rows are
-- deleted once processed.
CREATE TABLE event_outbox (
    id              bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind            text        NOT NULL CHECK (kind IN ('usage', 'security')),
    payload         jsonb       NOT NULL,
    attempts        int         NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error      text
);

-- The claim query orders by (kind, next_attempt_at, id) under FOR UPDATE SKIP
-- LOCKED, so this index serves the drain directly.
CREATE INDEX event_outbox_claim_idx ON event_outbox (kind, next_attempt_at, id);

-- Terminal failures (non-retryable, or attempts exhausted) are moved here so
-- the live table stays small and a poison message never blocks the queue.
CREATE TABLE event_outbox_dead_letter (
    id        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind      text        NOT NULL,
    payload   jsonb       NOT NULL,
    attempts  int         NOT NULL,
    error     text,
    failed_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS event_outbox_dead_letter;
DROP TABLE IF EXISTS event_outbox;
