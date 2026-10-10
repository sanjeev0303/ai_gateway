-- +goose Up
ALTER TABLE llm_requests ADD COLUMN service_account_id uuid REFERENCES service_accounts(id) ON DELETE SET NULL;
CREATE INDEX llm_requests_service_account_created_idx ON llm_requests (organization_id, service_account_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS llm_requests_service_account_created_idx;
ALTER TABLE llm_requests DROP COLUMN service_account_id;
