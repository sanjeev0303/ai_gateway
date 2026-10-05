import type { ListResponse, RequestOptions, Transport } from "./client";
import type { Membership } from "./members";

export interface Project {
  id: string;
  organization_id: string;
  name: string;
  slug: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
}

/** The row returned when adding a member; membership lists nest `user` instead. */
export interface ProjectMemberRow {
  project_id: string;
  user_id: string;
  role: string;
  member_since: string;
}

export interface ProjectsModule {
  create(
    input: { name: string; slug: string; description: string },
    options?: RequestOptions,
  ): Promise<Project>;
  list(options?: RequestOptions): Promise<ListResponse<Project>>;
  get(projectId: string, options?: RequestOptions): Promise<Project>;
  addMember(
    projectId: string,
    input: { user_id: string; role: string },
    options?: RequestOptions,
  ): Promise<ProjectMemberRow>;
  listMembers(projectId: string, options?: RequestOptions): Promise<ListResponse<Membership>>;
  removeMember(projectId: string, userId: string, options?: RequestOptions): Promise<void>;
}

export function createProjectsModule(transport: Transport): ProjectsModule {
  return {
    create: (input, options) =>
      transport.request<Project>("/v1/projects", { ...options, method: "POST", body: input }),
    list: (options) => transport.request<ListResponse<Project>>("/v1/projects", options),
    get: (projectId, options) =>
      transport.requestAt<Project>(`/v1/projects/${encodeURIComponent(projectId)}`, options),
    addMember: (projectId, input, options) =>
      transport.requestAt<ProjectMemberRow>(
        `/v1/projects/${encodeURIComponent(projectId)}/members`,
        { ...options, method: "POST", body: input },
      ),
    listMembers: (projectId, options) =>
      transport.requestAt<ListResponse<Membership>>(
        `/v1/projects/${encodeURIComponent(projectId)}/members`,
        options,
      ),
    removeMember: (projectId, userId, options) =>
      transport.requestAt<void>(
        `/v1/projects/${encodeURIComponent(projectId)}/members/${encodeURIComponent(userId)}`,
        { ...options, method: "DELETE" },
      ),
  };
}