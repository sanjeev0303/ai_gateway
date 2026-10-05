-- +goose Up
-- Conversation read API support. Every column added here is nullable: the
-- existing write path does not populate them, so historical rows read back as
-- NULL and the API reports them as absent rather than inventing a value.
ALTER TABLE conversations
  ADD COLUMN project_id    uuid REFERENCES projects (id) ON DELETE SET NULL,
  ADD COLUMN model         text,
  ADD COLUMN provider      text,
  -- NULL means "inherit the server default", not "no restriction". The server
  -- resolves NULL through config and fails closed to METADATA_ONLY.
  ADD COLUMN privacy_state text
    CHECK (privacy_state IS NULL
           OR privacy_state IN ('OFF', 'METADATA_ONLY', 'PROMPT_ONLY', 'FULL_CONTENT'));

ALTER TABLE conversation_messages
  ADD COLUMN tool_name text;

-- ponytail: no new indexes. The list endpoint is org-scoped and ordered by
-- created_at, which conversations_org_created_at_idx already serves; the
-- project/provider/model predicates filter that scan. Add indexes only if a
-- real query plan shows the scan going sequential.

-- +goose Down
ALTER TABLE conversation_messages
  DROP COLUMN tool_name;

ALTER TABLE conversations
  DROP COLUMN privacy_state,
  DROP COLUMN provider,
  DROP COLUMN model,
  DROP COLUMN project_id;