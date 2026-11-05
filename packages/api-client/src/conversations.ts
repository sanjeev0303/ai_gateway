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
  /** The gateway key (aigw_…) the conversation came through, when recorded. */
  credential_id?: string | null;
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

/**
 * Conversation text is never stored. The gateway keeps it in memory until the
 * conversation is deleted; this says whether the turns carry it. `expired`:
 * the gateway restarted since, so only counts and times remain.
 */
export type TranscriptState = "available" | "expired";

export interface ConversationMessages extends ListResponse<ConversationMessage> {
  transcript?: TranscriptState;
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
  /** One gateway key (aigw_…). */
  credential_id?: string;
}

/** How many conversations one employee has through one gateway key. */
export interface ConversationKeyCount {
  user_id: string;
  /** Null for conversations that recorded no key. */
  credential_id: string | null;
  conversations: number;
  /** RFC3339. */
  last_activity_at: string;
}

export interface ConversationListQuery extends ConversationQuery {
  /** 1 to 200; the server defaults to 50. */
  limit?: number;
  offset?: number;
}

export interface ConversationsModule {
  list(query?: ConversationListQuery, options?: RequestOptions): Promise<ListResponse<ConversationSummary>>;
  get(conversationId: string, options?: RequestOptions): Promise<ConversationSummary>;
  messages(conversationId: string, options?: RequestOptions): Promise<ConversationMessages>;
  /** Conversation counts per employee and gateway key. Members see only their own. */
  byKey(options?: RequestOptions): Promise<ListResponse<ConversationKeyCount>>;
  /** Deletes the whole conversation, its text included. Owners and admins only. */
  remove(conversationId: string, options?: RequestOptions): Promise<void>;
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
          credential_id: query?.credential_id,
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
      transport.requestAt<ConversationMessages>(
        `/v1/conversations/${encodeURIComponent(conversationId)}/messages`,
        options,
      ),
    byKey: (options) => transport.requestAt<ListResponse<ConversationKeyCount>>("/v1/conversations/by-key", options),
    remove: (conversationId, options) =>
      transport.requestAt<void>(`/v1/conversations/${encodeURIComponent(conversationId)}`, {
        ...options,
        method: "DELETE",
      }),
  };
}