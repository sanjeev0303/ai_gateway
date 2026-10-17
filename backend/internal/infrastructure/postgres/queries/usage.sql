-- name: RecordLLMRequest :one
INSERT INTO llm_requests (
    organization_id, project_id, user_id, gateway_credential_id, request_id,
    provider_request_id, provider_kind, model, status, error_code,
    prompt_tokens, completion_tokens, cached_input_tokens, total_tokens,
    latency_ms, cost_micros, created_at,
    request_started_at, first_token_at, completed_at,
    time_to_first_token_ms, total_latency_ms, service_account_id, team_id,
    reasoning_tokens, tool_tokens, pricing_version, cost_status, policy_decision, denial_reason
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)
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
INSERT INTO usage_events (
    organization_id, request_id, event_type, quantity, unit, metadata,
    user_id, service_account_id, team_id, project_id, gateway_credential_id
)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11)
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
           p.version AS pricing_version,
           r.prompt_tokens::bigint * p.input_unit_price_micros
         + r.completion_tokens::bigint * p.output_unit_price_micros AS cost
    FROM llm_requests r
    JOIN pricing_models pm
      ON pm.organization_id = r.organization_id
     AND pm.model_slug = r.model
    CROSS JOIN LATERAL (
        SELECT v.version, v.input_unit_price_micros, v.output_unit_price_micros
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
SET cost_micros = resolvable.cost,
    pricing_version = resolvable.pricing_version,
    cost_status = 'calculated'
FROM resolvable
WHERE r.id = resolvable.id
  AND (r.cost_micros IS DISTINCT FROM resolvable.cost OR r.cost_status IS DISTINCT FROM 'calculated');

-- name: DeleteUsageHourlyInWindow :exec
DELETE FROM usage_hourly
WHERE organization_id = sqlc.arg(organization_id) AND bucket_start >= sqlc.arg(from_ts) AND bucket_start < sqlc.arg(to_ts);

-- name: RecomputeUsageHourly :exec
INSERT INTO usage_hourly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
SELECT r.organization_id,
       r.project_id,
       r.model,
       date_trunc('hour', r.created_at, 'UTC') AS bucket_start,
       count(*)::int,
       sum(r.prompt_tokens)::bigint,
       sum(r.completion_tokens)::bigint,
       sum(r.cached_input_tokens)::bigint,
       sum(r.total_tokens)::bigint,
       sum(r.cost_micros)::bigint,
       now()
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.project_id IS NOT NULL
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
SELECT r.organization_id,
       r.project_id,
       r.model,
       date_trunc('day', r.created_at, 'UTC') AS bucket_start,
       count(*)::int,
       sum(r.prompt_tokens)::bigint,
       sum(r.completion_tokens)::bigint,
       sum(r.cached_input_tokens)::bigint,
       sum(r.total_tokens)::bigint,
       sum(r.cost_micros)::bigint,
       now()
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.project_id IS NOT NULL
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
SELECT r.organization_id,
       r.project_id,
       r.model,
       date_trunc('month', r.created_at, 'UTC') AS bucket_start,
       count(*)::int,
       sum(r.prompt_tokens)::bigint,
       sum(r.completion_tokens)::bigint,
       sum(r.cached_input_tokens)::bigint,
       sum(r.total_tokens)::bigint,
       sum(r.cost_micros)::bigint,
       now()
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.project_id IS NOT NULL
  AND r.created_at >= sqlc.arg(from_ts) AND r.created_at < sqlc.arg(to_ts)
GROUP BY r.organization_id, r.project_id, r.model, date_trunc('month', r.created_at, 'UTC')
ON CONFLICT (organization_id, project_id, model, bucket_start) DO UPDATE SET
    requests = EXCLUDED.requests, prompt_tokens = EXCLUDED.prompt_tokens,
    completion_tokens = EXCLUDED.completion_tokens, cached_input_tokens = EXCLUDED.cached_input_tokens,
    total_tokens = EXCLUDED.total_tokens, cost_micros = EXCLUDED.cost_micros, updated_at = now();


-- name: GetUsageSummary :one
SELECT COUNT(*)::bigint as total_requests,
       COALESCE(SUM(r.prompt_tokens), 0)::bigint as input_tokens,
       COALESCE(SUM(r.completion_tokens), 0)::bigint as output_tokens,
       COALESCE(SUM(r.cached_input_tokens), 0)::bigint as cached_tokens,
       COALESCE(SUM(r.reasoning_tokens), 0)::bigint as reasoning_tokens,
       COALESCE(SUM(r.tool_tokens), 0)::bigint as tool_tokens,
       COALESCE(SUM(r.total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(r.cost_micros), 0)::bigint as total_cost_micros,
       COUNT(*) FILTER (WHERE r.status = 'failed')::bigint as errors,
       COALESCE(AVG(r.latency_ms), 0)::float8 as avg_latency_ms,
       COUNT(DISTINCT r.user_id) FILTER (WHERE r.user_id IS NOT NULL)::bigint as active_users,
       -- An ungrouped aggregate always returns one row, so an empty window makes
       -- MAX() NULL. Coalesce to Go's zero time so the scan cannot fail; the
       -- repository detects "no requests" from TotalRequests.
       COALESCE(MAX(r.created_at)::timestamptz, '0001-01-01 00:00:00+00'::timestamptz)::timestamptz as last_activity_at
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(team_id)::uuid IS NULL OR r.team_id = sqlc.narg(team_id))
  AND (sqlc.narg(service_account_id)::uuid IS NULL OR r.service_account_id = sqlc.narg(service_account_id))
  AND (sqlc.narg(gateway_credential_id)::uuid IS NULL OR r.gateway_credential_id = sqlc.narg(gateway_credential_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status));

-- name: GetUsageTimeSeries :many
SELECT date_trunc('day', r.created_at, 'UTC')::timestamptz as bucket_start,
       COUNT(*)::bigint as requests,
       COALESCE(sum(r.total_tokens), 0)::bigint as total_tokens,
       COALESCE(sum(r.cost_micros), 0)::bigint as cost_micros
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
GROUP BY date_trunc('day', r.created_at, 'UTC')
ORDER BY bucket_start ASC;

-- name: GetUsageByProject :many
SELECT r.project_id,
       p.name,
       COUNT(*)::bigint as requests,
       COALESCE(SUM(r.prompt_tokens), 0)::bigint as input_tokens,
       COALESCE(SUM(r.completion_tokens), 0)::bigint as output_tokens,
       COALESCE(SUM(r.total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(r.cost_micros), 0)::bigint as cost_micros,
       COUNT(*) FILTER (WHERE r.status = 'failed')::bigint as errors,
       COALESCE(AVG(r.latency_ms), 0)::float8 as avg_latency_ms,
       MAX(r.created_at)::timestamptz as last_activity_at
FROM llm_requests r
JOIN projects p ON p.id = r.project_id
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND r.project_id IS NOT NULL
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(team_id)::uuid IS NULL OR r.team_id = sqlc.narg(team_id))
  AND (sqlc.narg(service_account_id)::uuid IS NULL OR r.service_account_id = sqlc.narg(service_account_id))
  AND (sqlc.narg(gateway_credential_id)::uuid IS NULL OR r.gateway_credential_id = sqlc.narg(gateway_credential_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status))
GROUP BY r.project_id, p.name
ORDER BY total_tokens DESC;

-- name: GetUsageByModel :many
SELECT model,
       COUNT(*)::bigint as requests,
       COALESCE(SUM(total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(cost_micros), 0)::bigint as cost_micros
FROM llm_requests
WHERE organization_id = sqlc.arg(organization_id)
  AND created_at >= sqlc.arg(from_ts)
  AND created_at < sqlc.arg(to_ts)
  AND model != ''
GROUP BY model
ORDER BY total_tokens DESC;

-- name: GetUsageByProvider :many
SELECT provider_kind as provider,
       COUNT(*) as requests,
       COALESCE(SUM(total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(cost_micros), 0)::bigint as cost_micros
FROM llm_requests
WHERE organization_id = sqlc.arg(organization_id)
  AND created_at >= sqlc.arg(from_ts)
  AND created_at < sqlc.arg(to_ts)
GROUP BY provider_kind
ORDER BY total_tokens DESC;

-- name: GetUsageByUser :many
SELECT r.user_id,
       u.email,
       u.display_name,
       COUNT(*)::bigint as requests,
       COALESCE(SUM(r.prompt_tokens), 0)::bigint as input_tokens,
       COALESCE(SUM(r.completion_tokens), 0)::bigint as output_tokens,
       COALESCE(SUM(r.total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(r.cost_micros), 0)::bigint as cost_micros,
       COUNT(*) FILTER (WHERE r.status = 'failed')::bigint as errors,
       COALESCE(AVG(r.latency_ms), 0)::float8 as avg_latency_ms,
       MAX(r.created_at)::timestamptz as last_activity_at
FROM llm_requests r
JOIN users u ON u.id = r.user_id
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND r.user_id IS NOT NULL
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(team_id)::uuid IS NULL OR r.team_id = sqlc.narg(team_id))
  AND (sqlc.narg(service_account_id)::uuid IS NULL OR r.service_account_id = sqlc.narg(service_account_id))
  AND (sqlc.narg(gateway_credential_id)::uuid IS NULL OR r.gateway_credential_id = sqlc.narg(gateway_credential_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status))
GROUP BY r.user_id, u.email, u.display_name
ORDER BY total_tokens DESC;

-- name: GetUsageByServiceAccount :many
SELECT r.service_account_id,
       sa.name,
       COUNT(*)::bigint as requests,
       COALESCE(SUM(r.prompt_tokens), 0)::bigint as input_tokens,
       COALESCE(SUM(r.completion_tokens), 0)::bigint as output_tokens,
       COALESCE(SUM(r.total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(r.cost_micros), 0)::bigint as cost_micros,
       COUNT(*) FILTER (WHERE r.status = 'failed')::bigint as errors,
       COALESCE(AVG(r.latency_ms), 0)::float8 as avg_latency_ms,
       MAX(r.created_at)::timestamptz as last_activity_at
FROM llm_requests r
JOIN service_accounts sa ON sa.id = r.service_account_id
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND r.service_account_id IS NOT NULL
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status))
GROUP BY r.service_account_id, sa.name
ORDER BY total_tokens DESC;

-- name: GetUsageByTeam :many
SELECT r.team_id,
       t.name,
       COUNT(r.id)::bigint as requests,
       COALESCE(SUM(r.prompt_tokens), 0)::bigint as input_tokens,
       COALESCE(SUM(r.completion_tokens), 0)::bigint as output_tokens,
       COALESCE(SUM(r.total_tokens), 0)::bigint as total_tokens,
       COALESCE(SUM(r.cost_micros), 0)::bigint as cost_micros,
       COUNT(r.id) FILTER (WHERE r.status = 'failed')::bigint as errors,
       COALESCE(AVG(r.latency_ms), 0)::float8 as avg_latency_ms,
       MAX(r.created_at)::timestamptz as last_activity_at
FROM llm_requests r
JOIN teams t ON t.id = r.team_id
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND r.team_id IS NOT NULL
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(service_account_id)::uuid IS NULL OR r.service_account_id = sqlc.narg(service_account_id))
  AND (sqlc.narg(gateway_credential_id)::uuid IS NULL OR r.gateway_credential_id = sqlc.narg(gateway_credential_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status))
GROUP BY r.team_id, t.name
ORDER BY total_tokens DESC;

-- name: GetRecentUsage :many
SELECT id, request_id, project_id, user_id, provider_kind, model, status,
       prompt_tokens, completion_tokens, cached_input_tokens, reasoning_tokens, tool_tokens,
       total_tokens, latency_ms, cost_micros, pricing_version, cost_status, policy_decision, denial_reason, error_code,
       created_at, request_started_at, first_token_at, completed_at, time_to_first_token_ms, total_latency_ms,
       service_account_id, team_id, gateway_credential_id
FROM llm_requests
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC
LIMIT 50;

-- name: GetUsageRequests :many
SELECT id, request_id, project_id, user_id, provider_kind, model, status,
       prompt_tokens, completion_tokens, cached_input_tokens, reasoning_tokens, tool_tokens,
       total_tokens, latency_ms, cost_micros, pricing_version, cost_status, policy_decision, denial_reason, error_code,
       created_at, request_started_at, first_token_at, completed_at, time_to_first_token_ms, total_latency_ms,
       service_account_id, team_id, gateway_credential_id
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(team_id)::uuid IS NULL OR EXISTS (
        SELECT 1
        FROM team_members tm
        WHERE tm.team_id = sqlc.narg(team_id)
          AND tm.user_id = r.user_id
          AND tm.organization_id = r.organization_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status))
ORDER BY r.created_at DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: GetUsageLatencyOverall :one
SELECT COUNT(r.time_to_first_token_ms)::bigint as ttft_samples,
       COUNT(r.total_latency_ms)::bigint as total_samples,
       COALESCE(AVG(r.time_to_first_token_ms), 0)::float8 as avg_ttft_ms,
       COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY r.time_to_first_token_ms), 0)::float8 as p50_ttft_ms,
       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.time_to_first_token_ms), 0)::float8 as p95_ttft_ms,
       COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY r.time_to_first_token_ms), 0)::float8 as p99_ttft_ms,
       COALESCE(AVG(r.total_latency_ms), 0)::float8 as avg_total_ms,
       COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY r.total_latency_ms), 0)::float8 as p50_total_ms,
       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.total_latency_ms), 0)::float8 as p95_total_ms,
       COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY r.total_latency_ms), 0)::float8 as p99_total_ms
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(team_id)::uuid IS NULL OR EXISTS (
        SELECT 1
        FROM team_members tm
        WHERE tm.team_id = sqlc.narg(team_id)
          AND tm.user_id = r.user_id
          AND tm.organization_id = r.organization_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status));

-- name: GetUsageLatencyByProviderModel :many
SELECT r.provider_kind as provider,
       r.model,
       COUNT(r.time_to_first_token_ms)::bigint as ttft_samples,
       COUNT(r.total_latency_ms)::bigint as total_samples,
       COALESCE(AVG(r.time_to_first_token_ms), 0)::float8 as avg_ttft_ms,
       COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY r.time_to_first_token_ms), 0)::float8 as p50_ttft_ms,
       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.time_to_first_token_ms), 0)::float8 as p95_ttft_ms,
       COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY r.time_to_first_token_ms), 0)::float8 as p99_ttft_ms,
       COALESCE(AVG(r.total_latency_ms), 0)::float8 as avg_total_ms,
       COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY r.total_latency_ms), 0)::float8 as p50_total_ms,
       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.total_latency_ms), 0)::float8 as p95_total_ms,
       COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY r.total_latency_ms), 0)::float8 as p99_total_ms
FROM llm_requests r
WHERE r.organization_id = sqlc.arg(organization_id)
  AND r.created_at >= sqlc.arg(from_ts)
  AND r.created_at < sqlc.arg(to_ts)
  AND (sqlc.narg(project_id)::uuid IS NULL OR r.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(user_id)::uuid IS NULL OR r.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(team_id)::uuid IS NULL OR EXISTS (
        SELECT 1
        FROM team_members tm
        WHERE tm.team_id = sqlc.narg(team_id)
          AND tm.user_id = r.user_id
          AND tm.organization_id = r.organization_id))
  AND (sqlc.narg(provider)::text IS NULL OR r.provider_kind = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR r.model = sqlc.narg(model))
  AND (sqlc.narg(status)::text IS NULL OR r.status = sqlc.narg(status))
GROUP BY r.provider_kind, r.model
ORDER BY total_samples DESC;

-- name: PurgeExpiredUsageEvents :execrows
DELETE FROM usage_events
WHERE organization_id = sqlc.arg(organization_id)
  AND timestamp < sqlc.arg(older_than)::timestamptz;

-- name: PurgeExpiredLlmRequests :execrows
DELETE FROM llm_requests
WHERE organization_id = sqlc.arg(organization_id)
  AND created_at < sqlc.arg(older_than)::timestamptz;

