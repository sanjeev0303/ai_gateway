-- name: CreateConversation :one
INSERT INTO conversations (organization_id, user_id, title, status)
VALUES (sqlc.arg(organization_id), sqlc.arg(user_id), sqlc.arg(title), sqlc.arg(status))
RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversations
WHERE organization_id = sqlc.arg(organization_id) AND id = sqlc.arg(id);

-- name: ListConversations :many
-- Aggregates come from conversation_messages because llm_requests has no
-- conversation key: an assistant turn is the only evidence of a provider
-- request. GROUP BY the primary key so the ungrouped conversation columns can
-- still be selected.
SELECT c.id,
       c.title,
       c.user_id,
       c.project_id,
       c.model,
       c.provider,
       c.privacy_state,
       c.created_at,
       COUNT(m.id)::bigint                                        AS message_count,
       COUNT(m.id) FILTER (WHERE m.role = 'assistant')::bigint    AS request_count,
       COALESCE(SUM(m.tokens), 0)::bigint                         AS total_tokens,
       -- Coalesce because the LEFT JOIN yields NULL for a conversation with no
       -- messages; the repository detects that case from MessageCount == 0.
       COALESCE(MAX(m.created_at)::timestamptz, '0001-01-01 00:00:00+00'::timestamptz)::timestamptz AS last_activity_at
FROM conversations c
LEFT JOIN conversation_messages m
       ON m.conversation_id = c.id
      AND m.organization_id = c.organization_id
WHERE c.organization_id = sqlc.arg(organization_id)
  AND (sqlc.narg(id)::uuid IS NULL OR c.id = sqlc.narg(id))
  AND (sqlc.narg(user_id)::uuid IS NULL OR c.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(project_id)::uuid IS NULL OR c.project_id = sqlc.narg(project_id))
  AND (sqlc.narg(provider)::text IS NULL OR c.provider = sqlc.narg(provider))
  AND (sqlc.narg(model)::text IS NULL OR c.model = sqlc.narg(model))
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR c.created_at >= sqlc.narg(from_ts)::timestamptz)
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR c.created_at < sqlc.narg(to_ts)::timestamptz)
  AND (sqlc.narg(team_id)::uuid IS NULL OR EXISTS (
        SELECT 1
        FROM team_members tm
        WHERE tm.team_id = sqlc.narg(team_id)
          AND tm.user_id = c.user_id
          AND tm.organization_id = c.organization_id))
GROUP BY c.id
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(limit_count)::int OFFSET sqlc.arg(offset_count)::int;


-- name: CreateConversationMessage :one
INSERT INTO conversation_messages (conversation_id, organization_id, user_id, role, content, tokens, tool_name, provider_message_id)
VALUES (sqlc.arg(conversation_id), sqlc.arg(organization_id), sqlc.arg(user_id), sqlc.arg(role),
        sqlc.arg(content), sqlc.arg(tokens), sqlc.narg(tool_name), sqlc.narg(provider_message_id))
RETURNING *;

-- name: ListConversationMessages :many
SELECT * FROM conversation_messages
WHERE organization_id = sqlc.arg(organization_id) AND conversation_id = sqlc.arg(conversation_id)
ORDER BY created_at, id;

-- name: PurgeExpiredConversations :execrows
DELETE FROM conversations
WHERE organization_id = sqlc.arg(organization_id)
  AND created_at < sqlc.arg(older_than)::timestamptz;