"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from "@tanstack/react-query";

import * as api from "@/lib/control-api";
import type {
  ApiCredential,
  GatewayCredential,
  Invitation,
  Member,
  Model,
  Project,
  ProjectMember,
  ProviderAccount,
  ProviderCredential,
  ProviderHealth,
  ServiceAccount,
  Team,
  TeamMember,
} from "@/lib/control-api";

type Items<T> = readonly T[];

export const queryKeys = {
  session: ["session"] as const,
  members: (org: string) => ["org", org, "members"] as const,
  invitations: (org: string) => ["org", org, "invitations"] as const,
  projects: (org: string) => ["org", org, "projects"] as const,
  projectMembers: (org: string, id: string) => ["org", org, "projects", id, "members"] as const,
  teams: (org: string) => ["org", org, "teams"] as const,
  teamMembers: (org: string, id: string) => ["org", org, "teams", id, "members"] as const,
  serviceAccounts: (org: string) => ["org", org, "service-accounts"] as const,
  apiCredentials: (org: string) => ["org", org, "api-credentials"] as const,
  gatewayCredentials: (org: string) => ["org", org, "gateway-credentials"] as const,
  providerAccounts: (org: string) => ["org", org, "provider-accounts"] as const,
  providerCredentials: (org: string) => ["org", org, "provider-credentials"] as const,
  models: (org: string) => ["org", org, "models"] as const,
  providerHealth: (org: string) => ["org", org, "provider-health"] as const,
};

export function useSession(): UseQueryResult<api.Session> {
  return useQuery({
    queryKey: queryKeys.session,
    queryFn: api.getSession,
    staleTime: 60_000,
    retry: false,
  });
}

export function useCsrfToken(): string {
  const { data } = useSession();
  return data?.csrf_token ?? "";
}

function orgOf(session: api.Session | undefined): string {
  return session?.claims.organization_id ?? "";
}

function useOrg(): string {
  return orgOf(useSession().data);
}

export function useMembers(): UseQueryResult<Items<Member>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.members(org),
    queryFn: async () => (await api.listMembers()).items,
    enabled: org !== "",
  });
}

export function useInvitations(): UseQueryResult<Items<Invitation>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.invitations(org),
    queryFn: async () => (await api.listInvitations()).items,
    enabled: org !== "",
  });
}

export function useProjects(): UseQueryResult<Items<Project>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.projects(org),
    queryFn: async () => (await api.listProjects()).items,
    enabled: org !== "",
  });
}

export function useProjectMembers(id: string | null): UseQueryResult<Items<ProjectMember>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.projectMembers(org, id ?? ""),
    queryFn: async () => (await api.listProjectMembers(id as string)).items,
    enabled: org !== "" && id !== null,
  });
}

export function useTeams(): UseQueryResult<Items<Team>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.teams(org),
    queryFn: async () => (await api.listTeams()).items,
    enabled: org !== "",
  });
}

export function useTeamMembers(id: string | null): UseQueryResult<Items<TeamMember>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.teamMembers(org, id ?? ""),
    queryFn: async () => (await api.listTeamMembers(id as string)).items,
    enabled: org !== "" && id !== null,
  });
}

export function useServiceAccounts(): UseQueryResult<Items<ServiceAccount>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.serviceAccounts(org),
    queryFn: async () => (await api.listServiceAccounts()).items,
    enabled: org !== "",
  });
}

export function useApiCredentials(): UseQueryResult<Items<ApiCredential>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.apiCredentials(org),
    queryFn: async () => (await api.listApiCredentials()).items,
    enabled: org !== "",
  });
}

export function useGatewayCredentials(): UseQueryResult<Items<GatewayCredential>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.gatewayCredentials(org),
    queryFn: async () => (await api.listGatewayCredentials()).items,
    enabled: org !== "",
  });
}

export function useProviderAccounts(): UseQueryResult<Items<ProviderAccount>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.providerAccounts(org),
    queryFn: async () => (await api.listProviderAccounts()).items,
    enabled: org !== "",
  });
}

export function useProviderCredentials(): UseQueryResult<Items<ProviderCredential>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.providerCredentials(org),
    queryFn: async () => (await api.listProviderCredentials()).items,
    enabled: org !== "",
  });
}

export function useModels(): UseQueryResult<Items<Model>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.models(org),
    queryFn: async () => (await api.listModels()).items,
    enabled: org !== "",
  });
}

export function useProviderHealth(): UseQueryResult<Items<ProviderHealth>> {
  const org = useOrg();
  return useQuery({
    queryKey: queryKeys.providerHealth(org),
    queryFn: async () => (await api.listProviderHealth()).items,
    enabled: org !== "",
  });
}

export type Mutation<TData, TVars> = UseMutationResult<TData, Error, TVars>;

function useInvalidate(keys: readonly (readonly unknown[])[]): () => Promise<void> {
  const client = useQueryClient();
  return async () => {
    await Promise.all(keys.map((key) => client.invalidateQueries({ queryKey: key })));
  };
}

export function useSetMemberRole(): Mutation<unknown, { userId: string; role: string }> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.members(org), queryKeys.invitations(org)]);
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      api.setMemberRole(userId, role),
    onSuccess: invalidate,
  });
}

export function useRemoveMember(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.members(org)]);
  return useMutation({
    mutationFn: (userId: string) => api.removeMember(userId),
    onSuccess: invalidate,
  });
}

export function useInvite(): Mutation<unknown, { email: string; role: string }> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.invitations(org), queryKeys.members(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { email: string; role: string }) =>
      api.invite(input.email, input.role, csrf),
    onSuccess: invalidate,
  });
}

export function useRevokeInvitation(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.invitations(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (id: string) => api.revokeInvitation(id, csrf),
    onSuccess: invalidate,
  });
}

export function useCreateProject(): Mutation<Project, { name: string; description?: string }> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.projects(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { name: string; description?: string }) => api.createProject(input, csrf),
    onSuccess: invalidate,
  });
}

export function useCreateTeam(): Mutation<Team, { name: string; description?: string }> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.teams(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { name: string; description?: string }) => api.createTeam(input, csrf),
    onSuccess: invalidate,
  });
}

export function useRemoveTeamMember(): Mutation<unknown, { teamId: string; userId: string }> {
  const org = useOrg();
  const client = useQueryClient();
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: ({ teamId, userId }: { teamId: string; userId: string }) =>
      api.removeTeamMember(teamId, userId, csrf),
    onSuccess: (_result, variables) =>
      client.invalidateQueries({ queryKey: queryKeys.teamMembers(org, variables.teamId) }),
  });
}

export function useCreateApiCredential(): Mutation<
  api.CreatedApiCredential,
  { name: string; scope?: string; service_account_id?: string }
> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.apiCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { name: string; scope?: string; service_account_id?: string }) =>
      api.createApiCredential(input, csrf),
    onSuccess: invalidate,
  });
}

export function useRevokeApiCredential(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.apiCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (id: string) => api.revokeApiCredential(id, csrf),
    onSuccess: invalidate,
  });
}

export function useCreateGatewayCredential(): Mutation<
  GatewayCredential,
  {
    name: string;
    subject_type: string;
    subject_id: string;
    team_id?: string;
    project_id?: string;
    scopes: string[];
    expires_at?: string;
  }
> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.gatewayCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: {
      name: string;
      subject_type: string;
      subject_id: string;
      team_id?: string;
      project_id?: string;
      scopes: string[];
      expires_at?: string;
    }) => api.createGatewayCredential(input, csrf),
    onSuccess: invalidate,
  });
}

export function useRevokeGatewayCredential(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.gatewayCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (id: string) => api.revokeGatewayCredential(id, csrf),
    onSuccess: invalidate,
  });
}

export function useRotateGatewayCredential(): Mutation<
  GatewayCredential,
  { id: string; name?: string; scopes?: string[]; expires_at?: string }
> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.gatewayCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { id: string; name?: string; scopes?: string[]; expires_at?: string }) =>
      api.rotateGatewayCredential(input.id, input, csrf),
    onSuccess: invalidate,
  });
}

export function useExpireGatewayCredential(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.gatewayCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (id: string) => api.expireGatewayCredential(id, csrf),
    onSuccess: invalidate,
  });
}

export function useRestoreGatewayCredential(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.gatewayCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (id: string) => api.restoreGatewayCredential(id, csrf),
    onSuccess: invalidate,
  });
}

export function useCreateProviderAccount(): Mutation<
  ProviderAccount,
  { provider_kind: string; name: string; base_url?: string }
> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.providerAccounts(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { provider_kind: string; name: string; base_url?: string }) =>
      api.createProviderAccount(input, csrf),
    onSuccess: invalidate,
  });
}

export function useCreateProviderCredential(): Mutation<
  ProviderCredential,
  { accountId: string; name: string; kind: string; secret: string }
> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.providerCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { accountId: string; name: string; kind: string; secret: string }) =>
      api.createProviderCredential(input.accountId, input, csrf),
    onSuccess: invalidate,
  });
}

export function useRevokeProviderCredential(): Mutation<unknown, string> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.providerCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (id: string) => api.revokeProviderCredential(id, csrf),
    onSuccess: invalidate,
  });
}

export function useRotateProviderCredential(): Mutation<
  ProviderCredential,
  { id: string; secret: string }
> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.providerCredentials(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: (input: { id: string; secret: string }) =>
      api.rotateProviderCredential(input.id, input.secret, csrf),
    onSuccess: invalidate,
  });
}

export function useSyncModels(): Mutation<Model[], { providerKind: string; apiKey: string }> {
  const org = useOrg();
  const invalidate = useInvalidate([queryKeys.models(org)]);
  const csrf = useCsrfToken();
  return useMutation({
    mutationFn: async (input: { providerKind: string; apiKey: string }) =>
      (await api.syncProviderModels(input.providerKind, input.apiKey, csrf)).items,
    onSuccess: invalidate,
  });
}
