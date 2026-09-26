-- name: CreateConversation :one
INSERT INTO conversations (organization_id, user_id, title, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversations
WHERE organization_id = $1 AND id = $2;

-- name: ListConversations :many
SELECT * FROM conversations
WHERE organization_id = $1 AND user_id = $2
ORDER BY created_at DESC;

-- name: CreateConversationMessage :one
INSERT INTO conversation_messages (conversation_id, organization_id, user_id, role, content, tokens, provider_message_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListConversationMessages :many
SELECT * FROM conversation_messages
WHERE organization_id = $1 AND conversation_id = $2
ORDER BY created_at;