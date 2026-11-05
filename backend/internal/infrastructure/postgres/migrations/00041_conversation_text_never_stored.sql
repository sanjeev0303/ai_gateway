-- +goose Up
-- Conversation text is never stored. The gateway keeps it in memory for a short
-- while and the control API reads it from there; the database keeps only each
-- message's role, token count and time.
UPDATE conversation_messages SET content = '' WHERE content <> '';

-- Gateway conversations were titled with their opening prompt, which is
-- conversation text too. Conversations created through the API (no model or
-- provider) keep the title their caller chose.
UPDATE conversations SET title = ''
WHERE title <> '' AND (model IS NOT NULL OR provider IS NOT NULL);

ALTER TABLE conversation_messages
  ADD CONSTRAINT conversation_messages_content_never_stored CHECK (content = '');

-- +goose Down
-- The wiped text cannot be restored; only the guard is removed.
ALTER TABLE conversation_messages
  DROP CONSTRAINT conversation_messages_content_never_stored;
