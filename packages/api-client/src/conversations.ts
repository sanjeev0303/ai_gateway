import type { ListResponse, RequestOptions, Transport } from "./client";

/**
 * Conversation payloads are the snake_case DTOs declared in
 * `backend/api/openapi/openapi.yaml`, mirrored here so a consumer of this
 * package does not need `@ai-governance/contracts` at the type level.
 *
 * Content is governed by the server, never by the caller: no query parameter
 * widens what a conversation returns.
 */

export type PrivacyMode = "OFF" | "METADATA_ONLY" | "PROMPT_ONLY" | "FULL_CONTENT";

export interface ConversationSummary {
  id: string;
  title: string;
  user_id: string;
  project_id?: string | null;
  provider?: string | null;
  model?: string | null;
  /** RFC3339. */
  started_at: string;
  /** RFC3339, or null/undefined when the conversation has no messages. */
  last_activity_at?: string | null;
  /** Assistant turns, so it can be lower than `message_count`. */
  request_count: number;
  message_count: number;
  total_tokens: number;
  /**
   * Micro-dollars. Always null today: `conversation_messages` has no link to
   * `llm_requests`, so per-conversation spend cannot be derived. The field is
   * present so it can be filled in without a breaking change.
   */
  cost_micros?: number | null;
  /** The effective mode after applying the conversation override, then the server default. */
  privacy_state: PrivacyMode;
}

export interface ConversationMessage {
  id: string;
  conversation_id: string;
  /** `system`, `user`, `assistant`, or `tool`. */
  role: string;
  /**
   * Absent — not empty — when the effective privacy mode withholds it, so a
   * redacted turn is distinguishable from a genuinely empty one. Under
   * `PROMPT_ONLY` only `system` and `user` turns keep content.
   */
  content?: string;
  tokens: number;
  tool_name?: string | null;
  provider_message_id?: string | null;
  /** RFC3339. */
  created_at: string;
}

export interface ConversationQuery {
  /** RFC3339; a `Date` is serialised as ISO 8601. */
  from?: string | Date;
  to?: string | Date;
  user_id?: string;
  team_id?: string;
  project_id?: string;
  provider?: string;
  model?: string;
}

export interface ConversationListQuery extends ConversationQuery {
  /** 1 to 200; the server defaults to 50. */
  limit?: number;
  offset?: number;
}

export interface ConversationsModule {
  list(query?: ConversationListQuery, options?: RequestOptions): Promise<ListResponse<ConversationSummary>>;
  get(conversationId: string, options?: RequestOptions): Promise<ConversationSummary>;
  messages(conversationId: string, options?: RequestOptions): Promise<ListResponse<ConversationMessage>>;
}

export function createConversationsModule(transport: Transport): ConversationsModule {
  return {
    list: (query, options) =>
      transport.request<ListResponse<ConversationSummary>>("/v1/conversations", {
        ...options,
        query: {
          from: query?.from,
          to: query?.to,
          user_id: query?.user_id,
          team_id: query?.team_id,
          project_id: query?.project_id,
          provider: query?.provider,
          model: query?.model,
          limit: query?.limit,
          offset: query?.offset,
        },
      }),
    get: (conversationId, options) =>
      transport.requestAt<ConversationSummary>(
        `/v1/conversations/${encodeURIComponent(conversationId)}`,
        options,
      ),
    messages: (conversationId, options) =>
      transport.requestAt<ListResponse<ConversationMessage>>(
        `/v1/conversations/${encodeURIComponent(conversationId)}/messages`,
        options,
      ),
  };
}