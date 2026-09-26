-- +goose Up
-- Every LLM request the gateway proxies lands here. request_id is the
-- client-supplied idempotency anchor; provider_request_id (when present) is
-- the provider's own identifier, also deduplicated per organization.
CREATE TABLE llm_requests (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id       uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    project_id            uuid        REFERENCES projects (id) ON DELETE SET NULL,
    user_id               uuid        REFERENCES users (id) ON DELETE SET NULL,
    api_credential_id     uuid        REFERENCES api_credentials (id) ON DELETE SET NULL,
    request_id            text        NOT NULL,
    provider_request_id   text,
    provider_kind         text        NOT NULL,
    model                 text        NOT NULL,
    status                text        NOT NULL DEFAULT 'succeeded'
                                      CHECK (status IN ('succeeded', 'failed', 'cancelled')),
    error_code            text,
    prompt_tokens         int         NOT NULL DEFAULT 0,
    completion_tokens     int         NOT NULL DEFAULT 0,
    cached_input_tokens   int         NOT NULL DEFAULT 0,
    total_tokens          int         NOT NULL DEFAULT 0,
    latency_ms            int         NOT NULL DEFAULT 0,
    cost_micros           bigint      NOT NULL DEFAULT 0,
    created_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, request_id)
);
CREATE UNIQUE INDEX llm_requests_org_provider_req_key
    ON llm_requests (organization_id, provider_request_id)
    WHERE provider_request_id IS NOT NULL;
CREATE INDEX llm_requests_org_created_at_idx ON llm_requests (organization_id, created_at);
CREATE INDEX llm_requests_org_model_created_idx ON llm_requests (organization_id, model, created_at);
CREATE INDEX llm_requests_org_project_created_idx ON llm_requests (organization_id, project_id, created_at);
CREATE INDEX llm_requests_org_user_created_idx ON llm_requests (organization_id, user_id, created_at);
CREATE INDEX llm_requests_project_id_idx ON llm_requests (project_id);
CREATE INDEX llm_requests_credential_id_idx ON llm_requests (api_credential_id);

-- Normalized usage events feed rollups, alerts and billing. One row per
-- (request, event_type); the unique key makes ingest idempotent.
CREATE TABLE usage_events (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    request_id      text        NOT NULL,
    event_type      text        NOT NULL
                                CHECK (event_type IN ('llm_request', 'token_usage', 'cost')),
    quantity        bigint      NOT NULL DEFAULT 0,
    unit            text        NOT NULL DEFAULT 'count'
                                CHECK (unit IN ('count', 'tokens', 'micros')),
    metadata        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, request_id, event_type)
);
CREATE INDEX usage_events_org_created_at_idx ON usage_events (organization_id, created_at);
CREATE INDEX usage_events_org_event_created_idx ON usage_events (organization_id, event_type, created_at);

-- Pre-aggregated usage per project/model per time bucket. Project-scoped by
-- design: organization-wide totals are computed from usage_events at read
-- time. Rollups are recomputable projections of usage_events.
CREATE TABLE usage_hourly (
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    project_id          uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    model               text        NOT NULL,
    bucket_start        timestamptz NOT NULL,
    requests            int         NOT NULL DEFAULT 0,
    prompt_tokens       bigint      NOT NULL DEFAULT 0,
    completion_tokens   bigint      NOT NULL DEFAULT 0,
    cached_input_tokens bigint      NOT NULL DEFAULT 0,
    total_tokens        bigint      NOT NULL DEFAULT 0,
    cost_micros         bigint      NOT NULL DEFAULT 0,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, project_id, model, bucket_start)
);

CREATE TABLE usage_daily (
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    project_id          uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    model               text        NOT NULL,
    bucket_start        timestamptz NOT NULL,
    requests            int         NOT NULL DEFAULT 0,
    prompt_tokens       bigint      NOT NULL DEFAULT 0,
    completion_tokens   bigint      NOT NULL DEFAULT 0,
    cached_input_tokens bigint      NOT NULL DEFAULT 0,
    total_tokens        bigint      NOT NULL DEFAULT 0,
    cost_micros         bigint      NOT NULL DEFAULT 0,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, project_id, model, bucket_start)
);

CREATE TABLE usage_monthly (
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    project_id          uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    model               text        NOT NULL,
    bucket_start        timestamptz NOT NULL,
    requests            int         NOT NULL DEFAULT 0,
    prompt_tokens       bigint      NOT NULL DEFAULT 0,
    completion_tokens   bigint      NOT NULL DEFAULT 0,
    cached_input_tokens bigint      NOT NULL DEFAULT 0,
    total_tokens        bigint      NOT NULL DEFAULT 0,
    cost_micros         bigint      NOT NULL DEFAULT 0,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, project_id, model, bucket_start)
);

-- Idempotency keys let API clients retry safely: a unique
-- (organization, scope, key) wins the race exactly once.
CREATE TABLE idempotency_keys (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    scope_key       text        NOT NULL,
    idempotency_key text        NOT NULL,
    expires_at      timestamptz NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, scope_key, idempotency_key)
);
CREATE INDEX idempotency_keys_org_scope_idx ON idempotency_keys (organization_id, scope_key);

-- +goose Down
DROP TABLE idempotency_keys;
DROP TABLE usage_monthly;
DROP TABLE usage_daily;
DROP TABLE usage_hourly;
DROP TABLE usage_events;
DROP TABLE llm_requests;