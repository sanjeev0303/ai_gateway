import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";
import type { OrgMember } from "./members";

export type Invitation = components["schemas"]["Invitation"];
/** The invite token is returned once, at creation, so it can be shared. */
export type InviteResult = components["schemas"]["CreateInvitationResponse"];

export interface InvitationsModule {
  invite(input: components["schemas"]["CreateInvitationRequest"], options?: RequestOptions): Promise<InviteResult>;
  accept(input: components["schemas"]["AcceptInvitationRequest"], options?: RequestOptions): Promise<OrgMember>;
  list(options?: RequestOptions): Promise<ListResponse<Invitation>>;
  revoke(invitationId: string, options?: RequestOptions): Promise<void>;
}

export function createInvitationsModule(transport: Transport): InvitationsModule {
  return {
    invite: (input, options) =>
      transport.request<InviteResult>("/v1/invitations", { ...options, method: "POST", body: input }),
    accept: (input, options) =>
      transport.request<OrgMember>("/v1/invitations/accept", { ...options, method: "POST", body: input }),
    list: (options) => transport.request<ListResponse<Invitation>>("/v1/invitations", options),
    revoke: (invitationId, options) =>
      transport.requestAt<void>(`/v1/invitations/${encodeURIComponent(invitationId)}`, {
        ...options,
        method: "DELETE",
      }),
  };
}