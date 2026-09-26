-- +goose Up
-- Conversations record user-facing chat exchanges.
CREATE TABLE conversations (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title           text        NOT NULL DEFAULT '',
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'archived')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX conversations_org_created_at_idx ON conversations (organization_id, created_at);
CREATE INDEX conversations_org_user_created_at_idx ON conversations (organization_id, user_id, created_at);
CREATE INDEX conversations_user_id_idx ON conversations (user_id);

-- Conversation messages are append-only; no updated_at.
CREATE TABLE conversation_messages (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id     uuid        NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    organization_id     uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id             uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role                text        NOT NULL
                                    CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content             text        NOT NULL,
    tokens              int         NOT NULL DEFAULT 0,
    provider_message_id text,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX conversation_messages_conversation_created_idx ON conversation_messages (conversation_id, created_at);
CREATE INDEX conversation_messages_org_created_at_idx ON conversation_messages (organization_id, created_at);
CREATE INDEX conversation_messages_org_user_created_idx ON conversation_messages (organization_id, user_id, created_at);
CREATE INDEX conversation_messages_user_id_idx ON conversation_messages (user_id);

-- +goose Down
DROP TABLE conversation_messages;
DROP TABLE conversations;