-- +goose Up
-- Add attribution fields to llm_requests
ALTER TABLE llm_requests
  ADD COLUMN team_id uuid REFERENCES teams(id) ON DELETE SET NULL;
ALTER TABLE llm_requests
  RENAME COLUMN api_credential_id TO gateway_credential_id;

-- Make sure gateway_credential_id references gateway_credentials
ALTER TABLE llm_requests DROP CONSTRAINT IF EXISTS llm_requests_api_credential_id_fkey;
ALTER TABLE llm_requests ADD CONSTRAINT llm_requests_gateway_credential_id_fkey FOREIGN KEY (gateway_credential_id) REFERENCES gateway_credentials(id) ON DELETE SET NULL;

CREATE INDEX llm_requests_org_team_created_idx ON llm_requests (organization_id, team_id, created_at);
CREATE INDEX llm_requests_org_gateway_cred_created_idx ON llm_requests (organization_id, gateway_credential_id, created_at);

-- Add attribution fields to usage_events
ALTER TABLE usage_events
  ADD COLUMN user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN service_account_id uuid REFERENCES service_accounts(id) ON DELETE SET NULL,
  ADD COLUMN team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
  ADD COLUMN project_id uuid REFERENCES projects(id) ON DELETE SET NULL,
  ADD COLUMN gateway_credential_id uuid REFERENCES gateway_credentials(id) ON DELETE SET NULL;

-- Privacy and Retention configurations at Organization level
ALTER TABLE organizations
  ADD COLUMN privacy_default text NOT NULL DEFAULT 'METADATA_ONLY'
    CHECK (privacy_default IN ('OFF', 'METADATA_ONLY', 'PROMPT_ONLY', 'FULL_CONTENT')),
  ADD COLUMN retention_conversations_days int NOT NULL DEFAULT 30,
  ADD COLUMN retention_usage_days int NOT NULL DEFAULT 90,
  ADD COLUMN retention_audits_days int NOT NULL DEFAULT 365,
  ADD COLUMN retention_billing_days int NOT NULL DEFAULT 2555; -- 7 years

-- +goose Down
ALTER TABLE organizations
  DROP COLUMN retention_billing_days,
  DROP COLUMN retention_audits_days,
  DROP COLUMN retention_usage_days,
  DROP COLUMN retention_conversations_days,
  DROP COLUMN privacy_default;

ALTER TABLE usage_events
  DROP COLUMN gateway_credential_id,
  DROP COLUMN project_id,
  DROP COLUMN team_id,
  DROP COLUMN service_account_id,
  DROP COLUMN user_id;

DROP INDEX IF EXISTS llm_requests_org_gateway_cred_created_idx;
DROP INDEX IF EXISTS llm_requests_org_team_created_idx;

ALTER TABLE llm_requests DROP CONSTRAINT IF EXISTS llm_requests_gateway_credential_id_fkey;
ALTER TABLE llm_requests ADD CONSTRAINT llm_requests_api_credential_id_fkey FOREIGN KEY (gateway_credential_id) REFERENCES api_credentials(id) ON DELETE SET NULL;
ALTER TABLE llm_requests RENAME COLUMN gateway_credential_id TO api_credential_id;

ALTER TABLE llm_requests
  DROP COLUMN team_id;
