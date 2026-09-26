-- name: RecordLLMRequest :one
INSERT INTO llm_requests (
    organization_id, project_id, user_id, api_credential_id, request_id,
    provider_request_id, provider_kind, model, status, error_code,
    prompt_tokens, completion_tokens, cached_input_tokens, total_tokens,
    latency_ms, cost_micros
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
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

-- name: RollupUsageHourly :execrows
INSERT INTO usage_hourly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests            = usage_hourly.requests + EXCLUDED.requests,
    prompt_tokens       = usage_hourly.prompt_tokens + EXCLUDED.prompt_tokens,
    completion_tokens   = usage_hourly.completion_tokens + EXCLUDED.completion_tokens,
    cached_input_tokens = usage_hourly.cached_input_tokens + EXCLUDED.cached_input_tokens,
    total_tokens        = usage_hourly.total_tokens + EXCLUDED.total_tokens,
    cost_micros         = usage_hourly.cost_micros + EXCLUDED.cost_micros,
    updated_at          = now();

-- name: RollupUsageDaily :execrows
INSERT INTO usage_daily (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests            = usage_daily.requests + EXCLUDED.requests,
    prompt_tokens       = usage_daily.prompt_tokens + EXCLUDED.prompt_tokens,
    completion_tokens   = usage_daily.completion_tokens + EXCLUDED.completion_tokens,
    cached_input_tokens = usage_daily.cached_input_tokens + EXCLUDED.cached_input_tokens,
    total_tokens        = usage_daily.total_tokens + EXCLUDED.total_tokens,
    cost_micros         = usage_daily.cost_micros + EXCLUDED.cost_micros,
    updated_at          = now();

-- name: RollupUsageMonthly :execrows
INSERT INTO usage_monthly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests            = usage_monthly.requests + EXCLUDED.requests,
    prompt_tokens       = usage_monthly.prompt_tokens + EXCLUDED.prompt_tokens,
    completion_tokens   = usage_monthly.completion_tokens + EXCLUDED.completion_tokens,
    cached_input_tokens = usage_monthly.cached_input_tokens + EXCLUDED.cached_input_tokens,
    total_tokens        = usage_monthly.total_tokens + EXCLUDED.total_tokens,
    cost_micros         = usage_monthly.cost_micros + EXCLUDED.cost_micros,
    updated_at          = now();