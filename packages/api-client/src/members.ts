import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

/** A row in the organization member list. */
export type OrgMember = components["schemas"]["OrgMember"];
/** A project or team membership; same shape as OrgMember but with a narrower role. */
export type Membership = components["schemas"]["Membership"];

export interface MembersModule {
  list(options?: RequestOptions): Promise<ListResponse<OrgMember>>;
  setRole(userId: string, input: components["schemas"]["SetMemberRoleRequest"], options?: RequestOptions): Promise<OrgMember>;
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