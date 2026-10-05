import type { ListResponse, RequestOptions, Transport } from "./client";

export interface User {
  id: string;
  email: string;
  display_name: string;
  status: string;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

/** A row in the organization member list. */
export interface OrgMember {
  user: User;
  role: string;
  member_since: string;
}

/**
 * A project or team membership. Structurally identical to {@link OrgMember};
 * it is a distinct type because the server models the two separately.
 */
export interface Membership {
  user: User;
  role: string;
  member_since: string;
}

export interface MembersModule {
  list(options?: RequestOptions): Promise<ListResponse<OrgMember>>;
  setRole(userId: string, input: { role: string }, options?: RequestOptions): Promise<OrgMember>;
  remove(userId: string, options?: RequestOptions): Promise<void>;
}

export function createMembersModule(transport: Transport): MembersModule {
  return {
    list: (options) => transport.request<ListResponse<OrgMember>>("/v1/members", options),
    setRole: (userId, input, options) =>
      transport.requestAt<OrgMember>(`/v1/members/${encodeURIComponent(userId)}/role`, {
        ...options,
        method: "PATCH",
        body: input,
      }),
    remove: (userId, options) =>
      transport.requestAt<void>(`/v1/members/${encodeURIComponent(userId)}`, {
        ...options,
        method: "DELETE",
      }),
  };
}