-- name: RecordLLMRequest :one
INSERT INTO llm_requests (
    organization_id, project_id, user_id, api_credential_id, request_id,
    provider_request_id, provider_kind, model, status, error_code,
    prompt_tokens, completion_tokens, cached_input_tokens, total_tokens,
    latency_ms, cost_micros, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
ON CONFLICT (organization_id, request_id) DO NOTHING
RETURNING *;

-- name: GetLLMRequestByRequestID :one
SELECT * FROM llm_requests
WHERE organization_id = $1 AND request_id = $2;

-- name: ListLLMRequests :many
SELECT * FROM llm_requests
WHERE organization_id = $1 AND created_at >= $2 AND created_at < $3
ORDER BY created_at DESC;

-- name: RecordUsageEvent :one
INSERT INTO usage_events (organization_id, request_id, event_type, quantity, unit, metadata)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
ON CONFLICT (organization_id, request_id, event_type) DO NOTHING
RETURNING *;

-- name: GetUsageEvent :one
SELECT * FROM usage_events
WHERE organization_id = $1 AND request_id = $2 AND event_type = $3;

-- name: ListUsageEventsByRequest :many
SELECT * FROM usage_events
WHERE organization_id = $1 AND request_id = $2
ORDER BY created_at;

-- name: BackfillLLMRequestCost :exec
WITH resolvable AS (
    SELECT r.id,
           r.prompt_tokens::bigint * p.input_unit_price_micros
         + r.completion_tokens::bigint * p.output_unit_price_micros AS cost
    FROM llm_requests r
    JOIN pricing_models pm
      ON pm.organization_id = r.organization_id
     AND pm.model_slug = r.model
    CROSS JOIN LATERAL (
        SELECT v.input_unit_price_micros, v.output_unit_price_micros
        FROM pricing_versions v
        WHERE v.pricing_model_id = pm.id
          AND v.organization_id = r.organization_id
          AND v.effective_from <= r.created_at
        ORDER BY v.effective_from DESC, v.version DESC
        LIMIT 1
    ) p
    WHERE r.organization_id = sqlc.arg(organization_id)
      AND r.created_at >= sqlc.arg(from_ts)
      AND r.created_at < sqlc.arg(to_ts)
)
UPDATE llm_requests r
SET cost_micros = resolvable.cost
FROM resolvable
WHERE r.id = resolvable.id
  AND r.cost_micros IS DISTINCT FROM resolvable.cost;

-- name: DeleteUsageHourlyInWindow :exec
DELETE FROM usage_hourly
WHERE organization_id = sqlc.arg(organization_id) AND bucket_start >= sqlc.arg(from_ts) AND bucket_start < sqlc.arg(to_ts);

-- name: RecomputeUsageHourly :exec
INSERT INTO usage_hourly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
SELECT r.organization_id, r.project_id, r.model,
       date_trunc('hour', r.created_at, 'UTC') AS bucket_start,
       count(*)::int,
       sum(r.prompt_tokens)::bigint,
       sum(r.completion_tokens)::bigint,
       sum(r.cached_input_tokens)::bigint,
       sum(r.total_tokens)::bigint,
       sum(r.cost_micros)::bigint,
       now()
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id) AND r.project_id IS NOT NULL
  AND r.created_at >= sqlc.arg(from_ts) AND r.created_at < sqlc.arg(to_ts)
GROUP BY r.organization_id, r.project_id, r.model, date_trunc('hour', r.created_at, 'UTC')
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests = EXCLUDED.requests, prompt_tokens = EXCLUDED.prompt_tokens,
    completion_tokens = EXCLUDED.completion_tokens, cached_input_tokens = EXCLUDED.cached_input_tokens,
    total_tokens = EXCLUDED.total_tokens, cost_micros = EXCLUDED.cost_micros, updated_at = now();

-- name: DeleteUsageDailyInWindow :exec
DELETE FROM usage_daily
WHERE organization_id = sqlc.arg(organization_id) AND bucket_start >= sqlc.arg(from_ts) AND bucket_start < sqlc.arg(to_ts);

-- name: RecomputeUsageDaily :exec
INSERT INTO usage_daily (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
SELECT r.organization_id, r.project_id, r.model,
       date_trunc('day', r.created_at, 'UTC') AS bucket_start,
       count(*)::int,
       sum(r.prompt_tokens)::bigint,
       sum(r.completion_tokens)::bigint,
       sum(r.cached_input_tokens)::bigint,
       sum(r.total_tokens)::bigint,
       sum(r.cost_micros)::bigint,
       now()
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id) AND r.project_id IS NOT NULL
  AND r.created_at >= sqlc.arg(from_ts) AND r.created_at < sqlc.arg(to_ts)
GROUP BY r.organization_id, r.project_id, r.model, date_trunc('day', r.created_at, 'UTC')
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests = EXCLUDED.requests, prompt_tokens = EXCLUDED.prompt_tokens,
    completion_tokens = EXCLUDED.completion_tokens, cached_input_tokens = EXCLUDED.cached_input_tokens,
    total_tokens = EXCLUDED.total_tokens, cost_micros = EXCLUDED.cost_micros, updated_at = now();

-- name: DeleteUsageMonthlyInWindow :exec
DELETE FROM usage_monthly
WHERE organization_id = sqlc.arg(organization_id) AND bucket_start >= sqlc.arg(from_ts) AND bucket_start < sqlc.arg(to_ts);

-- name: RecomputeUsageMonthly :exec
INSERT INTO usage_monthly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
SELECT r.organization_id, r.project_id, r.model,
       date_trunc('month', r.created_at, 'UTC') AS bucket_start,
       count(*)::int,
       sum(r.prompt_tokens)::bigint,
       sum(r.completion_tokens)::bigint,
       sum(r.cached_input_tokens)::bigint,
       sum(r.total_tokens)::bigint,
       sum(r.cost_micros)::bigint,
       now()
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id) AND r.project_id IS NOT NULL
  AND r.created_at >= sqlc.arg(from_ts) AND r.created_at < sqlc.arg(to_ts)
GROUP BY r.organization_id, r.project_id, r.model, date_trunc('month', r.created_at, 'UTC')
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests = EXCLUDED.requests, prompt_tokens = EXCLUDED.prompt_tokens,
    completion_tokens = EXCLUDED.completion_tokens, cached_input_tokens = EXCLUDED.cached_input_tokens,
    total_tokens = EXCLUDED.total_tokens, cost_micros = EXCLUDED.cost_micros, updated_at = now();
