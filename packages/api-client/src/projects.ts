import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";
import type { Membership } from "./members";

export type Project = components["schemas"]["Project"];
/** The row returned when adding a member; membership lists nest `user` instead. */
export type ProjectMemberRow = components["schemas"]["ProjectMemberRow"];
export type ProjectUpdateInput = components["schemas"]["UpdateProjectRequest"];
export type ProjectStatusInput = components["schemas"]["SetProjectStatusRequest"];

export interface ProjectsModule {
  create(input: components["schemas"]["CreateProjectRequest"], options?: RequestOptions): Promise<Project>;
  list(options?: RequestOptions): Promise<ListResponse<Project>>;
  get(projectId: string, options?: RequestOptions): Promise<Project>;
  update(projectId: string, input: ProjectUpdateInput, options?: RequestOptions): Promise<Project>;
  setStatus(projectId: string, input: ProjectStatusInput, options?: RequestOptions): Promise<Project>;
  addMember(
    projectId: string,
    input: components["schemas"]["AddMembershipRequest"],
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
    update: (projectId, input, options) =>
      transport.requestAt<Project>(`/v1/projects/${encodeURIComponent(projectId)}`, {
        ...options,
        method: "PATCH",
        body: input,
      }),
    setStatus: (projectId, input, options) =>
      transport.requestAt<Project>(`/v1/projects/${encodeURIComponent(projectId)}/status`, {
        ...options,
        method: "PATCH",
        body: input,
      }),

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