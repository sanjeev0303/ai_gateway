import type { ListResponse, RequestOptions, Transport } from "./client";
import type { OrgMember } from "./members";

export interface Invitation {
  id: string;
  organization_id: string;
  email: string;
  role: string;
  status: string;
  expires_at: string;
  invited_by?: string;
  accepted_by?: string;
  accepted_at?: string;
  created_at: string;
  updated_at: string;
}

/** The invite token is returned once, at creation, so it can be shared. */
export interface InviteResult {
  invitation: Invitation;
  token: string;
}

export interface InvitationsModule {
  invite(input: { email: string; role: string }, options?: RequestOptions): Promise<InviteResult>;
  accept(input: { token: string }, options?: RequestOptions): Promise<OrgMember>;
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