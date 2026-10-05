import type { ListResponse, RequestOptions, Transport } from "./client";
import type { Membership } from "./members";

export interface Team {
  id: string;
  organization_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

/** The row returned when adding a member; membership lists nest `user` instead. */
export interface TeamMemberRow {
  team_id: string;
  user_id: string;
  role: string;
  member_since: string;
}

export interface TeamsModule {
  create(input: { name: string; description: string }, options?: RequestOptions): Promise<Team>;
  list(options?: RequestOptions): Promise<ListResponse<Team>>;
  get(teamId: string, options?: RequestOptions): Promise<Team>;
  addMember(
    teamId: string,
    input: { user_id: string; role: string },
    options?: RequestOptions,
  ): Promise<TeamMemberRow>;
  listMembers(teamId: string, options?: RequestOptions): Promise<ListResponse<Membership>>;
  removeMember(teamId: string, userId: string, options?: RequestOptions): Promise<void>;
}

export function createTeamsModule(transport: Transport): TeamsModule {
  return {
    create: (input, options) =>
      transport.request<Team>("/v1/teams", { ...options, method: "POST", body: input }),
    list: (options) => transport.request<ListResponse<Team>>("/v1/teams", options),
    get: (teamId, options) =>
      transport.requestAt<Team>(`/v1/teams/${encodeURIComponent(teamId)}`, options),
    addMember: (teamId, input, options) =>
      transport.requestAt<TeamMemberRow>(`/v1/teams/${encodeURIComponent(teamId)}/members`, {
        ...options,
        method: "POST",
        body: input,
      }),
    listMembers: (teamId, options) =>
      transport.requestAt<ListResponse<Membership>>(`/v1/teams/${encodeURIComponent(teamId)}/members`, options),
    removeMember: (teamId, userId, options) =>
      transport.requestAt<void>(
        `/v1/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(userId)}`,
        { ...options, method: "DELETE" },
      ),
  };
}