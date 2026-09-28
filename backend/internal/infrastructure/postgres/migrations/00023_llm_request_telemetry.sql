-- +goose Up
ALTER TABLE llm_requests
  ADD COLUMN request_started_at    timestamptz,
  ADD COLUMN first_token_at        timestamptz,
  ADD COLUMN completed_at          timestamptz,
  ADD COLUMN time_to_first_token_ms int,
  ADD COLUMN total_latency_ms      int;

-- +goose Down
ALTER TABLE llm_requests
  DROP COLUMN request_started_at,
  DROP COLUMN first_token_at,
  DROP COLUMN completed_at,
  DROP COLUMN time_to_first_token_ms,
  DROP COLUMN total_latency_ms;
