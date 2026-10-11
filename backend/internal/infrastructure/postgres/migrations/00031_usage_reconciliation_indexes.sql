-- +goose Up
-- Indexes to accelerate tenant-scoped usage summaries, activity feeds, and timeseries charts
CREATE INDEX IF NOT EXISTS llm_requests_org_status_created_idx ON llm_requests (organization_id, status, created_at);
CREATE INDEX IF NOT EXISTS usage_daily_org_bucket_idx ON usage_daily (organization_id, bucket_start);
CREATE INDEX IF NOT EXISTS usage_hourly_org_bucket_idx ON usage_hourly (organization_id, bucket_start);

-- +goose Down
DROP INDEX IF EXISTS usage_hourly_org_bucket_idx;
DROP INDEX IF EXISTS usage_daily_org_bucket_idx;
DROP INDEX IF EXISTS llm_requests_org_status_created_idx;
