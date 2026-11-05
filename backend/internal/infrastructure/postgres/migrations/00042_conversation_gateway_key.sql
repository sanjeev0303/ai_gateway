-- +goose Up
-- Each conversation remembers the gateway key (aigw_...) it came through, so an
-- employee's history can be read per key.
ALTER TABLE conversations
  ADD COLUMN credential_id uuid REFERENCES gateway_credentials (id) ON DELETE SET NULL;
CREATE INDEX conversations_org_user_credential_idx ON conversations (organization_id, user_id, credential_id);

-- Earlier conversations did not record their key. The request log did: take
-- the same employee's request closest to the conversation's first reply, and
-- only when it is within 10 seconds (the reply is recorded right after it).
UPDATE conversations c
SET credential_id = match.gateway_credential_id
FROM (
  SELECT DISTINCT ON (f.id) f.id, r.gateway_credential_id
  FROM (
    SELECT conv.id, conv.organization_id, conv.user_id,
           MIN(m.created_at) FILTER (WHERE m.role = 'assistant') AS replied_at
    FROM conversations conv
    JOIN conversation_messages m ON m.conversation_id = conv.id
    GROUP BY conv.id
  ) f
  JOIN llm_requests r
    ON r.organization_id = f.organization_id
   AND r.user_id = f.user_id
   AND r.gateway_credential_id IS NOT NULL
   AND r.created_at BETWEEN f.replied_at - interval '10 seconds' AND f.replied_at + interval '10 seconds'
  ORDER BY f.id, abs(extract(epoch FROM r.created_at - f.replied_at))
) match
WHERE c.id = match.id AND c.credential_id IS NULL;

-- +goose Down
DROP INDEX conversations_org_user_credential_idx;
ALTER TABLE conversations DROP COLUMN credential_id;
