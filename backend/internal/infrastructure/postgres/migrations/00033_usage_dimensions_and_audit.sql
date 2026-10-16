-- +goose Up
-- Add reasoning, tool, pricing version, and policy decision dimensions to llm_requests
ALTER TABLE llm_requests
  ADD COLUMN reasoning_tokens int NOT NULL DEFAULT 0,
  ADD COLUMN tool_tokens int NOT NULL DEFAULT 0,
  ADD COLUMN pricing_version int,
  ADD COLUMN cost_status text NOT NULL DEFAULT 'calculated'
    CHECK (cost_status IN ('calculated', 'unavailable', 'free', 'estimated')),
  ADD COLUMN policy_decision text DEFAULT 'allowed'
    CHECK (policy_decision IS NULL OR policy_decision IN ('allowed', 'denied', 'none')),
  ADD COLUMN denial_reason text;

CREATE INDEX IF NOT EXISTS llm_requests_org_status_created_idx
  ON llm_requests (organization_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS llm_requests_org_policy_created_idx
  ON llm_requests (organization_id, policy_decision, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS llm_requests_org_policy_created_idx;
DROP INDEX IF EXISTS llm_requests_org_status_created_idx;

ALTER TABLE llm_requests
  DROP COLUMN denial_reason,
  DROP COLUMN policy_decision,
  DROP COLUMN cost_status,
  DROP COLUMN pricing_version,
  DROP COLUMN tool_tokens,
  DROP COLUMN reasoning_tokens;
