import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

export type ConversationSummary = components["schemas"]["ConversationSummary"];
export type ConversationMessage = components["schemas"]["ConversationMessage"];
export type ConversationMessages = components["schemas"]["ConversationMessagesResponse"];
export type TranscriptState = ConversationMessages["transcript"];
export type ConversationKeyCount = components["schemas"]["ConversationKeyCount"];

export interface ConversationQuery {
  /** RFC3339; a `Date` is serialised as ISO 8601. */
  from?: string | Date;
  to?: string | Date;
  user_id?: string;
  team_id?: string;
  project_id?: string;
  provider?: string;
  model?: string;
  /** A gateway key's id, or "none" for conversations that recorded no key. */
  credential_id?: string;
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